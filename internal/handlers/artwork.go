package handlers

import (
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateArtwork(c *gin.Context) {
	albumID := c.Param("id")
	var req struct {
		SourceID    string `json:"source_id" binding:"required"`
		ImageURL    string `json:"image_url" binding:"required"`
		SubmittedBy string `json:"submitted_by,omitempty"`
		IsOfficial  bool   `json:"is_official"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Check if album exists
	row := DB.QueryRow("SELECT id FROM albums WHERE id = ?", albumID)
	var aID string
	if err := row.Scan(&aID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "album not found"})
		return
	}

	id := uuid.New().String()
	approvalStatus := "pending"
	priority := 10
	if req.IsOfficial {
		approvalStatus = "approved"
		priority = 100
	}

	_, err := DB.Exec(`INSERT INTO artworks
		(id, album_id, source_id, image_url, is_official, submitted_by, approval_status, priority_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, albumID, req.SourceID, req.ImageURL, req.IsOfficial, req.SubmittedBy, approvalStatus, priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert artwork"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":              id,
		"album_id":        albumID,
		"source_id":       req.SourceID,
		"image_url":       req.ImageURL,
		"is_official":     req.IsOfficial,
		"approval_status": approvalStatus,
	})
}

func GetArtworks(c *gin.Context) {
	albumID := c.Param("id")
	rows, err := DB.Query(`
		SELECT id, album_id, source_id, image_url, is_official, submitted_by, approval_status, priority_score
		FROM artworks
		WHERE album_id = ?
		ORDER BY is_official DESC, priority_score DESC, created_at DESC
	`, albumID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query artworks"})
		return
	}
	defer rows.Close()

	var artworks []models.Artwork
	for rows.Next() {
		var a models.Artwork
		if err := rows.Scan(&a.ID, &a.AlbumID, &a.SourceID, &a.ImageURL, &a.IsOfficial, &a.SubmittedBy, &a.ApprovalStatus, &a.PriorityScore); err != nil {
			continue
		}
		artworks = append(artworks, a)
	}

	c.JSON(http.StatusOK, artworks)
}
