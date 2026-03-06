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
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	// Connect to Supabase
	database.Connect()
	defer database.Close()

	r := gin.Default()

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Welcome
	r.GET("/api/welcome", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"api-health": "ok", "docs": "documentation-link-ToBeUpdated"})
	})

	// ── Users ─────────────────────────────────────────────
	r.POST("/api/users", handlers.CreateUser(database.DB))
	r.GET("/api/users/:id", handlers.GetUser(database.DB))

	// ── Artists ───────────────────────────────────────────
	r.POST("/api/artists", handlers.CreateArtist(database.DB))
	r.GET("/api/artists/:id", handlers.GetArtist(database.DB))

	// ── Albums ────────────────────────────────────────────
	r.POST("/api/albums", handlers.CreateAlbum(database.DB))
	r.GET("/api/albums/:id", handlers.GetAlbum(database.DB))

	// ── Artworks ──────────────────────────────────────────
	r.POST("/api/albums/:id/artworks", handlers.CreateArtwork(database.DB))
	r.GET("/api/albums/:id/artworks", handlers.GetArtworksByAlbum(database.DB))

	log.Println("Server running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
