package handlers

import (
	"context"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateArtist handler
func CreateArtist(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		id := uuid.New().String()

		_, err := db.Exec(context.Background(), "INSERT INTO artists (id, name) VALUES ($1, $2)", id, req.Name)
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

// GetArtist handler
func GetArtist(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var artist models.Artist
		err := db.QueryRow(context.Background(), "SELECT id, name FROM artists WHERE id=$1", id).
			Scan(&artist.ID, &artist.Name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}

		c.JSON(http.StatusOK, artist)
	}
}
