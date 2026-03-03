package handlers

import (
	"database/sql"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateArtist returns a handler with the DB injected
func CreateArtist(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		// Generate a new UUID for the artist
		id := uuid.New().String()

		// Insert into database
		_, err := db.Exec("INSERT INTO artists (id, name) VALUES (?, ?)", id, req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert artist"})
			return
		}

		artist := models.Artist{
			ID:   id,
			Name: req.Name,
		}

		c.JSON(http.StatusCreated, artist)
	}
}

// GetArtist returns a handler with the DB injected
func GetArtist(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		row := db.QueryRow("SELECT id, name FROM artists WHERE id = ?", id)

		var artist models.Artist
		if err := row.Scan(&artist.ID, &artist.Name); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}

		c.JSON(http.StatusOK, artist)
	}
}
