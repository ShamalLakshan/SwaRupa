package handlers

import (
	"context"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateAlbum handler
func CreateAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ArtistID    string `json:"artist_id" binding:"required"`
			Title       string `json:"title" binding:"required"`
			ReleaseYear int    `json:"release_year"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "artist_id and title are required"})
			return
		}

		id := uuid.New().String()

		_, err := db.Exec(
			context.Background(),
			"INSERT INTO albums (id, artist_id, title, release_year) VALUES ($1, $2, $3, $4)",
			id, req.ArtistID, req.Title, req.ReleaseYear,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert album"})
			return
		}

		album := models.Album{
			ID:          id,
			ArtistID:    req.ArtistID,
			Title:       req.Title,
			ReleaseYear: req.ReleaseYear,
		}

		c.JSON(http.StatusCreated, album)
	}
}

// GetAlbum handler
func GetAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var album models.Album
		err := db.QueryRow(
			context.Background(),
			"SELECT id, artist_id, title, release_year FROM albums WHERE id=$1",
			id,
		).Scan(&album.ID, &album.ArtistID, &album.Title, &album.ReleaseYear)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
			return
		}

		c.JSON(http.StatusOK, album)
	}
}
