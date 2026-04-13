package handlers

import (
	"context"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
)

// CreateAlbum handles POST /api/albums requests to create a new album and associate it with one or more artists.
// The handler delegates business logic to the AlbumService, which implements a transactional create pattern
// to ensure atomic insertion of both the album record and all album_artists cross-reference records,
// maintaining referential integrity.
//
// Request body structure:
//
//	{
//	  "title": "Album Title",
//	  "release_year": 2023,
//	  "artist_ids": ["artist-uuid-1", "artist-uuid-2"],
//	  "submitted_by": "user-id" (optional)
//	}
//
// Service Operations:
// The AlbumService.CreateAlbum() method:
// 1. Generates a UUID for the album
// 2. Begins a database transaction
// 3. Inserts the album record: INSERT INTO albums (id, title, release_year, submitted_by, created_at)
// 4. For each artist_id, inserts a junction record: INSERT INTO album_artists (album_id, artist_id)
// 5. Commits the transaction
// 6. Fetches and returns the complete album with its associated artists
//
// The transaction ensures all-or-nothing semantics: if any INSERT fails (e.g., invalid artist_id foreign key),
// the entire transaction is rolled back, preventing orphaned album records without artist associations.
// Optional fields (submitted_by) are normalized within the service through nullableString() to convert
// empty strings to SQL NULL.
//
// Response:
// - 201 Created: Album and artist associations successfully created; returns Album model with ID and artists
// - 400 Bad Request: Missing 'title' or 'artist_ids' fields, or artist_ids is empty
// - 500 Internal Server Error: Service error (transaction failure, constraint violation, etc.)
func CreateAlbum(albumService *services.AlbumService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse request body with validation.
		// binding:"required,min=1" enforces that artist_ids is not empty,
		// preventing orphaned album records without associated artists.
		var req struct {
			Title       string   `json:"title"       binding:"required"`
			ReleaseYear int      `json:"release_year"`
			ArtistIDs   []string `json:"artist_ids"  binding:"required,min=1"`
			SubmittedBy string   `json:"submitted_by"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title and at least one artist_id are required"})
			return
		}

		// Delegate album creation to the service layer.
		// The service handles transaction management, UUID generation, and artist association.
		album, err := albumService.CreateAlbum(
			context.Background(),
			req.Title,
			req.ReleaseYear,
			req.ArtistIDs,
			req.SubmittedBy,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create album"})
			return
		}

		// Return HTTP 201 Created with the newly created album.
		c.JSON(http.StatusCreated, album)
	}
}

// GetAlbum handles GET /api/albums/:id requests to retrieve a single album record with its associated artists.
// The :id path parameter is the album's UUID as returned from CreateAlbum or stored in the database.
// The handler delegates data fetching to the AlbumService, which executes the necessary queries.
//
// Service Operations:
// The AlbumService.GetAlbumByID() method:
//
//  1. Fetches album record: SELECT id, title, release_year, submitted_by, created_at FROM albums WHERE id = $1
//     This is an indexed primary key lookup with O(1) retrieval complexity.
//
//  2. Fetches associated artists via INNER JOIN:
//     SELECT a.id, a.name, a.artist_bio, a.image_url, a.submitted_by, a.created_at
//     FROM artists a
//     INNER JOIN album_artists aa ON aa.artist_id = a.id
//     WHERE aa.album_id = $1
//     This query joins the artists table with the album_artists junction table, returning only
//     artists explicitly linked to the specified album. The INNER JOIN excludes unassociated artists.
//
// Nullable columns (submitted_by, artist_bio, etc.) are handled within the service and scanned into pointer types.
// If NULL in the database, the field is omitted from the JSON response.
//
// Response:
// - 200 OK: Album found; returns Album model with populated Artists slice
// - 404 Not Found: No album with the given ID exists in the database
// - 500 Internal Server Error: Service error (connection, query failure, etc.)
func GetAlbum(albumService *services.AlbumService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the album ID from the URL path parameter.
		id := c.Param("id")

		// Delegate album retrieval to the service layer.
		// The service handles all database queries and artist association population.
		album, err := albumService.GetAlbumByID(context.Background(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
			return
		}

		// Marshal the Album with associated Artists and return HTTP 200 OK.
		c.JSON(http.StatusOK, album)
	}
}

// GetAllAlbums handles GET /api/albums requests to retrieve all album records with their associated artists.
// This endpoint is useful for populating album directories, galleries, or performing metadata analysis.
// Each album in the result includes a populated Artists slice with all linked artist records.
// The handler delegates data fetching to the AlbumService.
//
// Service Operations:
// The AlbumService.GetAllAlbums() method:
//
//  1. Fetches all albums: SELECT id, title, release_year, submitted_by, created_at FROM albums
//     Ordered by created_at DESC to show newest albums first.
//
//  2. For each album, fetches associated artists via INNER JOIN:
//     SELECT a.id, a.name, a.artist_bio, a.image_url, a.submitted_by, a.created_at
//     FROM artists a
//     INNER JOIN album_artists aa ON aa.artist_id = a.id
//     WHERE aa.album_id = $1
//
// This implementation retrieves all albums first, then queries artists for each album.
// In future optimizations, a single SQL query with grouping could reduce round-trips.
//
// Response:
// - 200 OK: Query successful; returns array of Album models, each with populated Artists
// - 500 Internal Server Error: Service error (connection, query failure, etc.)
func GetAllAlbums(albumService *services.AlbumService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Delegate album retrieval to the service layer.
		// The service handles all database queries and artist association population.
		albums, err := albumService.GetAllAlbums(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve albums"})
			return
		}

		// Ensure non-nil JSON output: return empty slice if no albums exist.
		// This provides a consistent API contract: empty results are always [].
		if albums == nil {
			albums = []models.Album{}
		}

		// Marshal the albums slice with populated artists to JSON and return HTTP 200 OK.
		c.JSON(http.StatusOK, albums)
	}
}
