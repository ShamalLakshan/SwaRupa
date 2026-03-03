package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/db"
	"github.com/ShamalLakshan/SwaRupa/internal/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize DB
	database := db.Connect()
	defer database.Close()

	// Assign DB to handlers
	handlers.DB = database

	r := gin.Default()

	// Health endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Artist endpoints
	r.POST("/artists", handlers.CreateArtist(database))
	r.GET("/artists/:id", handlers.GetArtist(database))

	// Album endpoints
	r.POST("/albums", handlers.CreateAlbum)
	r.GET("/albums/:id", handlers.GetAlbum)

	// Artwork endpoints
	r.POST("/albums/:id/artworks", handlers.CreateArtwork)
	r.GET("/albums/:id/artworks", handlers.GetArtworks)

	fmt.Println("Server running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
