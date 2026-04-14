// Package main is the entry point for the SwaRupa Music Metadata API server.
// SwaRupa is a community-driven music metadata platform built with Go and PostgreSQL.
// It provides REST API endpoints for managing artist, album, and artwork metadata with
// community submission and moderation workflows.
package main

import (
	"log"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/database"
	"github.com/ShamalLakshan/SwaRupa/internal/handlers"
	"github.com/ShamalLakshan/SwaRupa/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// main initializes the SwaRupa API server with database connections and HTTP routes.
// The server uses the Gin web framework for HTTP request handling and pgx for PostgreSQL connectivity.
//
// Initialization Steps:
// 1. Load environment variables from .env file (if present; uses system env variables otherwise)
// 2. Initialize PostgreSQL connection pool via database.Connect()
// 3. Initialize service instances for data access and business logic
// 4. Create Gin router and register HTTP route handlers
// 5. Start HTTP server on port 8080
//
// Database Connection:
// PostgreSQL is connected via database.Connect(), which reads POOLER_DATABASE_URL from environment.
// The connection pool is maintained globally and shared across all request handlers.
// The pool is gracefully closed on server shutdown via database.Close() deferred call.
//
// Service Layer:
// Service instances encapsulate business logic and database queries for each domain entity.
// Services provide a cleaner separation of concerns and enable better testability.
// Each handler receives the appropriate service instance instead of direct database access.
//
// Route Structure:
// - Health Check: GET /api/health - Simple connection test
// - Welcome: GET /api/welcome - API information and documentation link
// - Users: POST /api/users, GET /api/users/:id
// - Artists: POST /api/artists, GET /api/artists/:id
// - Albums: POST /api/albums, GET /api/albums/:id
// - Artwork: POST /api/albums/:id/artworks, GET /api/albums/:id/artworks
// - Artwork Sources: POST/GET/PATCH/DELETE /api/artworks/:artwork_id/sources
//
// All endpoints accept and return JSON-formatted data with appropriate HTTP status codes.
func main() {
	// Load environment variables from .env file for local development.
	// Settings like POOLER_DATABASE_URL are read from this file or from system environment.
	// godotenv.Load() is non-fatal; if .env is missing, system environment variables are used.
	// This is useful for development; production typically uses deployment secrets management.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	// Initialize the PostgreSQL connection pool.
	// This establishes a pooled connection to the database and verifies connectivity.
	// If the database is unreachable, Connect() will call log.Fatal() and terminate the server.
	database.Connect()
	// Defer the Close() call to ensure graceful cleanup when the server exits.
	// This flushes in-flight queries and releases all connection pool resources.
	defer database.Close()

	// Initialize service instances for handling business logic and data access.
	// Services provide a cleaner separation of concerns between handlers and database queries.
	artworkService := services.NewArtworkService(database.DB)
	albumService := services.NewAlbumService(database.DB)
	artistService := services.NewArtistService(database.DB)
	userService := services.NewUserService(database.DB)

	// Sets gin to Release mode instead of the default debug mode.
	gin.SetMode(gin.ReleaseMode)

	// Create a new Gin router instance.
	// gin.Default() includes default middleware for logging and error recovery.
	// This router will handle all HTTP requests for the API.
	r := gin.Default()

	// Register health check endpoint: GET /api/health
	// Returns {"status": "ok"} with HTTP 200 if the server is running.
	// Used by load balancers and monitoring systems to verify server availability.
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Register welcome/discovery endpoint: GET /api/welcome
	// Returns general API information and documentation links for API consumers.
	// Serves as an entry point for client applications discovering the API.
	r.GET("/api/welcome", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"api-health": "ok", "docs": "documentation-link-ToBeUpdated"})
	})

	// ── Users ─────────────────────────────────────────────
	// User endpoints for authentication and user profile management.
	r.POST("/api/users", handlers.CreateUser(userService)) // Create new user from auth provider UID
	r.GET("/api/users/:id", handlers.GetUser(userService)) // Retrieve user profile by ID
	r.GET("/api/users", handlers.GetAllUsers(userService)) // Retrieve all users

	// ── Artists ───────────────────────────────────────────
	// Artist CRUD endpoints for managing music artists.
	r.POST("/api/artists", handlers.CreateArtist(artistService)) // Create new artist record
	r.GET("/api/artists/:id", handlers.GetArtist(artistService)) // Retrieve artist by ID
	r.GET("/api/artists", handlers.GetAllArtists(artistService)) // Retrieve all artists

	// ── Albums ────────────────────────────────────────────
	// Album CRUD endpoints for managing music albums and their artist associations.
	r.POST("/api/albums", handlers.CreateAlbum(albumService)) // Create new album with artists
	r.GET("/api/albums/:id", handlers.GetAlbum(albumService)) // Retrieve album with populated artists
	r.GET("/api/albums", handlers.GetAllAlbums(albumService)) // Retrieve all albums with artists

	// ── Artworks ──────────────────────────────────────────
	// Artwork submission and retrieval endpoints for album cover images and promotional images.
	r.POST("/api/albums/:id/artworks", handlers.CreateArtwork(artworkService))     // Submit new artwork for album
	r.GET("/api/albums/:id/artworks", handlers.GetArtworksByAlbum(artworkService)) // Retrieve artworks with filtering
	r.GET("/api/artworks", handlers.GetAllArtworks(artworkService))                // Retrieve all artworks with filtering

	// ── Artwork Sources ───────────────────────────────────
	// Artwork source management endpoints for tracking and updating artwork metadata sources.
	r.POST("/api/artworks/:artwork_id/sources", handlers.AddArtworkSource(artworkService))                 // Add new source for artwork
	r.GET("/api/artworks/:artwork_id/sources", handlers.ListArtworkSources(artworkService))                // List all sources for artwork
	r.PATCH("/api/artworks/:artwork_id/sources/:source_id", handlers.UpdateArtworkSource(artworkService))  // Update source details
	r.DELETE("/api/artworks/:artwork_id/sources/:source_id", handlers.DeleteArtworkSource(artworkService)) // Remove source for artwork

	// ── Moderation (Phase 2) ──────────────────────────────
	r.PATCH("/api/artworks/:artwork_id/approve", handlers.ApproveArtwork(artworkService, userService))
	r.PATCH("/api/artworks/:artwork_id/reject", handlers.RejectArtwork(artworkService, userService))

	// Log server startup message for operational visibility.
	// Indicates that the server is ready to accept requests.
	log.Println("Server running on http://localhost:8080")
	// Start the HTTP server on port 8080, blocking until the server stops or encounters an error.
	// r.Run() is a convenience method that creates a net/http server and calls ListenAndServe().
	// Errors typically occur due to port conflicts or permission issues binding to the port.
	if err := r.Run(":8080"); err != nil {
		// If the server fails to start, log the error and exit with a non-zero exit code.
		log.Fatal("Failed to start server:", err)
	}
}
