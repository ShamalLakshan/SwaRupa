package handlers

import (
	"database/sql"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var DB *sql.DB // Assign DB from main.go

func CreateAlbum(c *gin.Context) {
	var req struct {
		ArtistID    string `json:"artist_id" binding:"required"`
		Title       string `json:"title" binding:"required"`
		ReleaseYear int    `json:"release_year"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Check if artist exists
	row := DB.QueryRow("SELECT id FROM artists WHERE id = ?", req.ArtistID)
	var artistID string
	if err := row.Scan(&artistID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artist not found"})
		return
	}

	id := uuid.New().String()
	_, err := DB.Exec("INSERT INTO albums (id, artist_id, title, release_year) VALUES (?, ?, ?, ?)",
		id, req.ArtistID, req.Title, req.ReleaseYear)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert album"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           id,
		"artist_id":    req.ArtistID,
		"title":        req.Title,
		"release_year": req.ReleaseYear,
	})
}

func GetAlbum(c *gin.Context) {
	id := c.Param("id")
	row := DB.QueryRow("SELECT id, artist_id, title, release_year FROM albums WHERE id = ?", id)

	var album models.Album
	if err := row.Scan(&album.ID, &album.ArtistID, &album.Title, &album.ReleaseYear); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}

	c.JSON(http.StatusOK, album)
}
