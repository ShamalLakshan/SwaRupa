package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateUser handler
// The ID comes from the client (Firebase/Supabase Auth UID).
func CreateUser(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID          string `json:"id"           binding:"required"`
			DisplayName string `json:"display_name"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}

		_, err := db.Exec(
			context.Background(),
			`INSERT INTO users (id, display_name, role)
			 VALUES ($1, $2, 'contributor')
			 ON CONFLICT (id) DO NOTHING`,
			req.ID, nullableString(req.DisplayName),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		c.JSON(http.StatusCreated, models.User{
			ID:          req.ID,
			DisplayName: req.DisplayName,
			Role:        "contributor",
			CreatedAt:   time.Now(),
		})
	}
}

// GetUser handler
func GetUser(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var user models.User
		var displayName *string

		err := db.QueryRow(
			context.Background(),
			`SELECT id, display_name, role, created_at FROM users WHERE id = $1`,
			id,
		).Scan(&user.ID, &displayName, &user.Role, &user.CreatedAt)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		if displayName != nil {
			user.DisplayName = *displayName
		}

		c.JSON(http.StatusOK, user)
	}
}
