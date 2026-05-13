package services

import (
	"context"
	"errors"

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
	var email *string

	err := s.db.QueryRow(ctx,
		`SELECT id, display_name, email, role, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &displayName, &email, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	if displayName != nil {
		user.DisplayName = *displayName
	}

	if email != nil {
		user.Email = *email
	}

	return &user, nil
}

// GetAllUsers retrieves all users.
func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, display_name, email, role, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var displayName *string
		var email *string

		if err := rows.Scan(&user.ID, &displayName, &email, &user.Role, &user.CreatedAt); err != nil {
			continue
		}

		if displayName != nil {
			user.DisplayName = *displayName
		}
		if email != nil {
			user.Email = *email
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

// IsUserNotFoundError identifies pgx not-found errors from QueryRow scans.
func IsUserNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
