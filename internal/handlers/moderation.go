package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// moderateArtwork is the shared logic for ApproveArtwork and RejectArtwork.
//
// Flow:
// 1. Extract artwork ID from URL path
// 2. Extract requested_by from request body (will be replaced by Supabase JWT in Phase 5)
// 3. Look up the user's role in the database
// 4. Reject with 403 if not admin
// 5. Update approval_status on the artwork record
// 6. Return 404 if the artwork ID does not exist
func moderateArtwork(db *pgxpool.Pool, status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		artworkID := c.Param("id")

		// requested_by is the user attempting the moderation action.
		// In Phase 5, this will be extracted automatically from the Supabase Auth JWT token
		// via middleware, and this field will be removed from the request body.
		var req struct {
			RequestedBy string `json:"requested_by" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "requested_by is required"})
			return
		}

		// Look up the requesting user's role.
		// Only users with role = "admin" are permitted to approve or reject artworks.
		var role string
		err := db.QueryRow(
			context.Background(),
			`SELECT role FROM users WHERE id = $1`,
			req.RequestedBy,
		).Scan(&role)
		if err != nil {
			// User not found in the database — reject the request.
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "only admins can moderate artworks"})
			return
		}

		// Update the artwork's approval_status to the target status (approved or rejected).
		// RowsAffected() is checked to detect the case where the artwork ID does not exist.
		result, err := db.Exec(
			context.Background(),
			`UPDATE artworks SET approval_status = $1 WHERE id = $2`,
			status, artworkID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update artwork status"})
			return
		}

		if result.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "artwork not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":              artworkID,
			"approval_status": status,
		})
	}
}

// ApproveArtwork handles PATCH /api/artworks/:id/approve
// Sets the artwork's approval_status to "approved". Admin only.
func ApproveArtwork(db *pgxpool.Pool) gin.HandlerFunc {
	return moderateArtwork(db, "approved")
}

// RejectArtwork handles PATCH /api/artworks/:id/reject
// Sets the artwork's approval_status to "rejected". Admin only.
func RejectArtwork(db *pgxpool.Pool) gin.HandlerFunc {
	return moderateArtwork(db, "rejected")
}
