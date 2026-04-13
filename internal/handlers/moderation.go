package handlers

import (
	"context"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
)

// moderate Artwork is the shared logic for ApproveArtwork and RejectArtwork.
//
// Flow:
// 1. Extract artwork ID from URL path
// 2. Extract requested_by from request body (will be replaced by Supabase JWT in Phase 5)
// 3. Check if the user is an admin using UserService
// 4. Reject with 403 if not admin
// 5. Update approval_status on the artwork record using ArtworkService
// 6. Return 404 if the artwork ID does not exist
func moderateArtworkWithService(artworkService *services.ArtworkService, userService *services.UserService, status string) gin.HandlerFunc {
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

		// Check the requesting user's role using UserService.
		// Only users with role = "admin" are permitted to approve or reject artworks.
		isAdmin, err := userService.IsAdmin(context.Background(), req.RequestedBy)
		if err != nil {
			// User not found in the database — reject the request.
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "only admins can moderate artworks"})
			return
		}

		// Update the artwork's approval_status using ArtworkService.
		err = artworkService.UpdateApprovalStatus(context.Background(), artworkID, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update artwork status"})
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
func ApproveArtwork(artworkService *services.ArtworkService, userService *services.UserService) gin.HandlerFunc {
	return moderateArtworkWithService(artworkService, userService, "approved")
}

// RejectArtwork handles PATCH /api/artworks/:id/reject
// Sets the artwork's approval_status to "rejected". Admin only.
func RejectArtwork(artworkService *services.ArtworkService, userService *services.UserService) gin.HandlerFunc {
	return moderateArtworkWithService(artworkService, userService, "rejected")
}
