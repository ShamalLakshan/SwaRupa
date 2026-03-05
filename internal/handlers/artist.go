package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateArtist handler
func CreateArtist(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name          string `json:"name"           binding:"required"`
			MusicBrainzID string `json:"musicbrainz_id"`
			ImageURL      string `json:"image_url"`
			SubmittedBy   string `json:"submitted_by"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		id := uuid.New().String()

		_, err := db.Exec(
			context.Background(),
			`INSERT INTO artists (id, name, musicbrainz_id, image_url, submitted_by)
			 VALUES ($1, $2, $3, $4, $5)`,
			id, req.Name, nullableString(req.MusicBrainzID), nullableString(req.ImageURL), nullableString(req.SubmittedBy),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert artist"})
			return
		}

		c.JSON(http.StatusCreated, models.Artist{
			ID:            id,
			Name:          req.Name,
			MusicBrainzID: req.MusicBrainzID,
			ImageURL:      req.ImageURL,
			SubmittedBy:   req.SubmittedBy,
			CreatedAt:     time.Now(),
		})
	}
}

// GetArtist handler
func GetArtist(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var artist models.Artist
		var musicBrainzID, imageURL, submittedBy *string

		err := db.QueryRow(
			context.Background(),
			`SELECT id, name, musicbrainz_id, image_url, submitted_by, created_at
			 FROM artists WHERE id = $1`,
			id,
		).Scan(
			&artist.ID,
			&artist.Name,
			&musicBrainzID,
			&imageURL,
			&submittedBy,
			&artist.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}

		if musicBrainzID != nil {
			artist.MusicBrainzID = *musicBrainzID
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
		}

		c.JSON(http.StatusOK, artist)
	}
}
