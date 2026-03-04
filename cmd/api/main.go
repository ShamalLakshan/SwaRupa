package main

import (
	"log"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/database"
	"github.com/ShamalLakshan/SwaRupa/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system env")
	}

	// Connect to Supabase
	database.Connect()
	defer database.Close()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Pass database.DB (the connection pool) to handlers
	r.POST("/artists", handlers.CreateArtist(database.DB))
	r.GET("/artists/:id", handlers.GetArtist(database.DB))

	r.POST("/albums", handlers.CreateAlbum(database.DB))
	r.GET("/albums/:id", handlers.GetAlbum(database.DB))

	r.POST("/albums/:id/artworks", handlers.CreateArtwork(database.DB))
	r.GET("/albums/:id/artworks", handlers.GetArtworksByAlbum(database.DB))

	log.Println("Server running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
