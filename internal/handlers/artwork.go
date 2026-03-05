package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
			ImageURL     string `json:"image_url"     binding:"required"`
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
			`INSERT INTO artworks
				(id, album_id, source_id, image_url, thumbnail_url, is_official, submitted_by, approval_status, priority_score)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', 0)`,
			id, albumID,
			nullableString(req.SourceID),
			req.ImageURL,
			nullableString(req.ThumbnailURL),
			req.IsOfficial,
			nullableString(req.SubmittedBy),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert artwork"})
			return
		}

		c.JSON(http.StatusCreated, models.Artwork{
			ID:             id,
			AlbumID:        albumID,
			SourceID:       req.SourceID,
			ImageURL:       req.ImageURL,
			ThumbnailURL:   req.ThumbnailURL,
			IsOfficial:     req.IsOfficial,
			SubmittedBy:    req.SubmittedBy,
			ApprovalStatus: "pending",
			PriorityScore:  0,
			CreatedAt:      time.Now(),
		})
	}
}

// GetArtworksByAlbum handler
// Supports query params:
//
//	?status=approved|pending|rejected
//	?official=true
//	?sort=priority
func GetArtworksByAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		albumID := c.Param("id")

		// --- Build query dynamically based on filters ---
		query := `SELECT id, album_id, source_id, image_url, thumbnail_url,
		                 is_official, submitted_by, approval_status, priority_score, created_at
		          FROM artworks
		          WHERE album_id = $1`
		args := []any{albumID}
		argIdx := 2

		if status := c.Query("status"); status != "" {
			query += fmt.Sprintf(" AND approval_status = $%d", argIdx)
			args = append(args, status)
			argIdx++
		}

		if c.Query("official") == "true" {
			query += " AND is_official = true"
		}

		if c.Query("sort") == "priority" {
			query += " ORDER BY priority_score DESC"
		} else {
			query += " ORDER BY created_at DESC"
		}

		rows, err := db.Query(context.Background(), query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artworks"})
			return
		}
		defer rows.Close()

		var artworks []models.Artwork
		for rows.Next() {
			var aw models.Artwork
			var sourceID, thumbnailURL, submittedBy *string

			if err := rows.Scan(
				&aw.ID, &aw.AlbumID, &sourceID, &aw.ImageURL, &thumbnailURL,
				&aw.IsOfficial, &submittedBy, &aw.ApprovalStatus, &aw.PriorityScore, &aw.CreatedAt,
			); err != nil {
				continue
			}
			if sourceID != nil {
				aw.SourceID = *sourceID
			}
			if thumbnailURL != nil {
				aw.ThumbnailURL = *thumbnailURL
			}
			if submittedBy != nil {
				aw.SubmittedBy = *submittedBy
			}
			artworks = append(artworks, aw)
		}

		if artworks == nil {
			artworks = []models.Artwork{}
		}

		c.JSON(http.StatusOK, artworks)
	}
}
