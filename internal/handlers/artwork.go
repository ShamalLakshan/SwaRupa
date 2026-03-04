package handlers

import (
	"context"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateArtwork handler
func CreateArtwork(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		albumID := c.Param("id")

		var req struct {
			SourceID     string `json:"source_id"`
			ImageURL     string `json:"image_url" binding:"required"`
			ThumbnailURL string `json:"thumbnail_url"`
			IsOfficial   bool   `json:"is_official"`
			SubmittedBy  string `json:"submitted_by"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image_url is required"})
			return
		}

		id := uuid.New().String()

		_, err := db.Exec(
			context.Background(),
			`INSERT INTO artworks (
				id, album_id, source_id, image_url, thumbnail_url, is_official, submitted_by, approval_status, priority_score
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			id, albumID, req.SourceID, req.ImageURL, req.ThumbnailURL, req.IsOfficial, req.SubmittedBy, "pending", 0,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert artwork"})
			return
		}

		artwork := models.Artwork{
			ID:             id,
			AlbumID:        albumID,
			SourceID:       req.SourceID,
			ImageURL:       req.ImageURL,
			ThumbnailURL:   req.ThumbnailURL,
			IsOfficial:     req.IsOfficial,
			SubmittedBy:    req.SubmittedBy,
			ApprovalStatus: "pending",
			PriorityScore:  0,
		}

		c.JSON(http.StatusCreated, artwork)
	}
}

// GetArtworksByAlbum handler
func GetArtworksByAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		albumID := c.Param("id")

		rows, err := db.Query(
			context.Background(),
			`SELECT id, album_id, source_id, image_url, thumbnail_url, is_official,
			        submitted_by, approval_status, priority_score
			 FROM artworks
			 WHERE album_id=$1`,
			albumID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artworks"})
			return
		}
		defer rows.Close()

		var artworks []models.Artwork
		for rows.Next() {
			var artwork models.Artwork
			if err := rows.Scan(
				&artwork.ID,
				&artwork.AlbumID,
				&artwork.SourceID,
				&artwork.ImageURL,
				&artwork.ThumbnailURL,
				&artwork.IsOfficial,
				&artwork.SubmittedBy,
				&artwork.ApprovalStatus,
				&artwork.PriorityScore,
			); err != nil {
				continue
			}
			artworks = append(artworks, artwork)
		}

		c.JSON(http.StatusOK, artworks)
	}
}
