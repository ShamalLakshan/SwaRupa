package services

import (
	"context"
	"errors"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserService provides business logic for user operations.
type UserService struct {
	db *pgxpool.Pool
}

// NewUserService creates a new user service instance.
func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user or returns existing (idempotent).
// If displayName or email are provided, they will be set for the user if missing.
func (s *UserService) CreateUser(ctx context.Context, id, displayName, email string) (*models.User, error) {
	_, err := s.db.Exec(ctx,
		`INSERT INTO users (id, display_name, email, role, created_at)
		 VALUES ($1, $2, $3, 'contributor', now())
		 ON CONFLICT (id) DO UPDATE
		   SET display_name = COALESCE(NULLIF($2, ''), users.display_name),
			   email = COALESCE(NULLIF($3, ''), users.email)`,
		id, displayName, email,
	)
	if err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, id)
}

// GetUserByID retrieves a single user by ID.
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	var displayName *string
	var contactEmail *string

	err := s.db.QueryRow(ctx,
		`SELECT id, display_name, contact_email, role, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &displayName, &contactEmail, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	if displayName != nil {
		user.DisplayName = *displayName
	}

	if contactEmail != nil {
		user.ContactEmail = *contactEmail
	}

	return &user, nil
}

// GetAllUsers retrieves all users.
func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, display_name, contact_email, role, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var displayName *string
		var contactEmail *string

		if err := rows.Scan(&user.ID, &displayName, &contactEmail, &user.Role, &user.CreatedAt); err != nil {
			continue
		}

		if displayName != nil {
			user.DisplayName = *displayName
		}
		if contactEmail != nil {
			user.ContactEmail = *contactEmail
		}

		users = append(users, user)
	}

	if users == nil {
		users = []models.User{}
	}

	return users, nil
}

// IsAdmin checks if a user has admin role.
func (s *UserService) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var role string
	err := s.db.QueryRow(ctx,
		`SELECT role FROM users WHERE id = $1`,
		userID,
	).Scan(&role)
	if err != nil {
		return false, err
	}
	return role == "admin", nil
}

// LinkProviderAccount migrates all user-linked data from providerUserID into primaryUserID
// and then deletes providerUserID from users.
func (s *UserService) LinkProviderAccount(ctx context.Context, primaryUserID, providerUserID string) error {
	if primaryUserID == "" || providerUserID == "" {
		return errors.New("primary and provider user IDs are required")
	}
	if primaryUserID == providerUserID {
		return errors.New("cannot link the same user account")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Validate both users exist before merge.
	var tmp string
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1`, primaryUserID).Scan(&tmp); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1`, providerUserID).Scan(&tmp); err != nil {
		return err
	}

	// Prefer existing primary profile fields; fill from provider when missing.
	_, err = tx.Exec(ctx,
		`UPDATE users u
		 SET display_name = COALESCE(NULLIF(u.display_name, ''), NULLIF(p.display_name, '')),
		     email = COALESCE(NULLIF(u.email, ''), NULLIF(p.email, ''))
		 FROM users p
		 WHERE u.id = $1 AND p.id = $2`,
		primaryUserID, providerUserID,
	)
	if err != nil {
		return err
	}

	// Re-point all known user references.
	if _, err = tx.Exec(ctx, `UPDATE artists SET submitted_by = $1 WHERE submitted_by = $2`, primaryUserID, providerUserID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE albums SET submitted_by = $1 WHERE submitted_by = $2`, primaryUserID, providerUserID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE artworks SET submitted_by = $1 WHERE submitted_by = $2`, primaryUserID, providerUserID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE artwork_sources SET discovered_by = $1 WHERE discovered_by = $2`, primaryUserID, providerUserID); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, providerUserID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// CreateOrUpdateUserFromGitHub handles GitHub OAuth callback - idempotent upsert
// Creates a new user or updates existing one with latest GitHub profile info
func (s *UserService) CreateOrUpdateUserFromGitHub(ctx context.Context, gitHubID, gitHubUsername, displayName, profileURL string, contactEmail *string) (*models.User, error) {
	var email *string
	if contactEmail != nil && *contactEmail != "" {
		email = contactEmail
	}

	// UPSERT: insert if new, update if exists
	_, err := s.db.Exec(ctx,
		`INSERT INTO users (id, github_id, github_username, github_profile_url, display_name, contact_email, role, oauth_provider, last_login, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'contributor', 'github', now(), now())
		 ON CONFLICT (id) DO UPDATE
		   SET github_username = $3,
		       github_profile_url = $4,
		       display_name = COALESCE(NULLIF($5, ''), users.display_name),
		       contact_email = COALESCE(NULLIF($6, ''), users.contact_email),
		       last_login = now()`,
		gitHubID, gitHubID, gitHubUsername, profileURL, displayName, email,
	)
	if err != nil {
		return nil, err
	}

	return s.GetUserByGitHubID(ctx, gitHubID)
}

// GetUserByGitHubID retrieves a user by their GitHub ID
func (s *UserService) GetUserByGitHubID(ctx context.Context, gitHubID string) (*models.User, error) {
	var user models.User
	var displayName, contactEmail, gitHubUsername, gitHubProfileURL, oauthProvider *string
	var lastLogin *time.Time

	err := s.db.QueryRow(ctx,
		`SELECT id, github_id, github_username, github_profile_url, display_name, contact_email, oauth_provider, role, last_login, created_at
		 FROM users WHERE github_id = $1`,
		gitHubID,
	).Scan(&user.ID, &user.GitHubID, &gitHubUsername, &gitHubProfileURL, &displayName, &contactEmail, &oauthProvider, &user.Role, &lastLogin, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	if displayName != nil {
		user.DisplayName = *displayName
	}
	if contactEmail != nil {
		user.ContactEmail = *contactEmail
	}
	if gitHubUsername != nil {
		user.GitHubUsername = *gitHubUsername
	}
	if gitHubProfileURL != nil {
		user.GitHubProfileURL = *gitHubProfileURL
	}
	if oauthProvider != nil {
		user.OAuthProvider = *oauthProvider
	}
	user.LastLogin = lastLogin

	return &user, nil
}

// GetUserByGitHubUsername retrieves a user by their GitHub username
func (s *UserService) GetUserByGitHubUsername(ctx context.Context, gitHubUsername string) (*models.User, error) {
	var user models.User
	var displayName, contactEmail, gitHubID, gitHubProfileURL, oauthProvider *string
	var lastLogin *time.Time

	err := s.db.QueryRow(ctx,
		`SELECT id, github_id, github_username, github_profile_url, display_name, contact_email, oauth_provider, role, last_login, created_at
		 FROM users WHERE github_username = $1`,
		gitHubUsername,
	).Scan(&user.ID, &gitHubID, &user.GitHubUsername, &gitHubProfileURL, &displayName, &contactEmail, &oauthProvider, &user.Role, &lastLogin, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	if displayName != nil {
		user.DisplayName = *displayName
	}
	if contactEmail != nil {
		user.ContactEmail = *contactEmail
	}
	if gitHubID != nil {
		user.GitHubID = *gitHubID
	}
	if gitHubProfileURL != nil {
		user.GitHubProfileURL = *gitHubProfileURL
	}
	if oauthProvider != nil {
		user.OAuthProvider = *oauthProvider
	}
	user.LastLogin = lastLogin

	return &user, nil
}

// IsUserNotFoundError identifies pgx not-found errors from QueryRow scans.
func IsUserNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
