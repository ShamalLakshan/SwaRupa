package services

import (
	"context"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
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
func (s *UserService) CreateUser(ctx context.Context, id, displayName string) (*models.User, error) {
	_, err := s.db.Exec(ctx,
		`INSERT INTO users (id, display_name, role, created_at)
		 VALUES ($1, $2, 'contributor', now())
		 ON CONFLICT (id) DO NOTHING`,
		id, nullableString(displayName),
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

	err := s.db.QueryRow(ctx,
		`SELECT id, display_name, role, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &displayName, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	if displayName != nil {
		user.DisplayName = *displayName
	}

	return &user, nil
}

// GetAllUsers retrieves all users.
func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, display_name, role, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var displayName *string

		if err := rows.Scan(&user.ID, &displayName, &user.Role, &user.CreatedAt); err != nil {
			continue
		}

		if displayName != nil {
			user.DisplayName = *displayName
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
