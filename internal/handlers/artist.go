package handlers

import (
	"context"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
)

// CreateArtist handles POST /api/artists requests to create a new artist record.
// The handler accepts a JSON payload with the artist's name (required) and optional metadata
// such as MusicBrainz identifier, profile image URL, and submission attribution.
//
// Request body structure:
//
//	{
//	  "name": "Artist Name",
//	  "artist_bio": "mbz-uuid" (optional),
//	  "image_url": "https://example.com/image.jpg" (optional),
//	  "submitted_by": "user-id" (optional)
//	}
//
// Operation:
// Calls ArtistService.CreateArtist() which generates a UUID v4 (RFC 4122) for the new artist record
// and executes: INSERT INTO artists (id, name, artist_bio, image_url, submitted_by, created_at)
// VALUES ($1, $2, $3, $4, $5, now())
// The optional fields are normalized through nullableString(), converting empty strings to SQL NULL
// for proper database semantics. This ensures nullable TEXT columns store NULL rather than empty strings.
// All values are parameterized to prevent SQL injection attacks.
//
// Response:
// - 201 Created: Artist successfully created; returns Artist model with generated ID and server timestamp
// - 400 Bad Request: Missing required 'name' field in JSON body
// - 500 Internal Server Error: Database error (connection, constraint violations, etc.)
func CreateArtist(artistService *services.ArtistService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Define a request struct with validation tags for automatic input validation.
		// binding:"required" enforces non-empty values; unmarshaling failures automatically return 400.
		// JSON tags establish bidirectional mapping between Go struct fields and JSON keys.
		var req struct {
			Name        string `json:"name"           binding:"required"`
			ArtistBio   string `json:"artist_bio"`
			ImageURL    string `json:"image_url"`
			SubmittedBy string `json:"submitted_by"`
		}

		// Unmarshal and validate the JSON request body.
		// Gin's validator uses struct tags to enforce business logic (e.g., required fields).
		// This is preferable to manual validation as it's declarative and reusable.
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		// Call the service to create the artist.
		// The service handles UUID generation, database insertion, and returning the created artist.
		artist, err := artistService.CreateArtist(
			context.Background(),
			req.Name,
			req.ArtistBio,
			req.ImageURL,
			req.SubmittedBy,
		)
		if err != nil {
			// Database errors include connection failures, constraint violations (e.g., unique constraints),
			// and query syntax errors. All are treated as 500 Internal Server Error.
			// In production systems, log the error and error type for operational visibility.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create artist"})
			return
		}

		// Return HTTP 201 Created with the newly created artist record.
		// The response includes all provided fields and the server-generated ID.
		// Clients use the returned ID for subsequent requests (e.g., GET /artists/{id}).
		c.JSON(http.StatusCreated, artist)
	}
}

// GetArtist handles GET /api/artists/:id requests to retrieve a single artist record by ID.
// The :id path parameter is the artist's UUID as returned from CreateArtist or stored in the database.
//
// Operation:
// Calls ArtistService.GetArtistByID() which executes:
// SELECT id, name, artist_bio, image_url, submitted_by, created_at FROM artists WHERE id = $1
// using an indexed primary key lookup for O(1) retrieval performance.
// Nullable columns (artist_bio, image_url, submitted_by) are scanned into pointer types (*string).
// If a NULL value is encountered in the database, the pointer is set to nil and the field is omitted
// from the JSON response due to the omitempty struct tag annotation.
//
// Response:
// - 200 OK: Artist found; returns complete Artist model with all fields
// - 404 Not Found: No artist with the given ID exists in the database
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetArtist(artistService *services.ArtistService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the artist ID from the URL path parameter.
		// The :id placeholder in route registration (e.g., GET /artists/:id) maps to Param("id").
		id := c.Param("id")

		// Call the service to retrieve the artist by ID.
		// The service handles the database query and NULL pointer dereferencing.
		artist, err := artistService.GetArtistByID(context.Background(), id)
		if err != nil {
			// GetArtistByID returns an error if the artist is not found or a database error occurs.
			// Per REST conventions, treat not found as HTTP 404.
			// Other errors (connection failures, scan type mismatches) indicate server problems (500).
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}

		// Marshal the Artist struct to JSON and return HTTP 200 OK.
		// Gin's JSON() method automatically calls the encoding/json marshaler.
		// Fields tagged with omitempty are excluded if empty (zero value or empty slice).
		c.JSON(http.StatusOK, artist)
	}
}

// GetAllArtists handles GET /api/artists requests to retrieve all artist records from the database.
// The endpoint returns a paginated or complete list of artists depending on query parameters.
// This endpoint is useful for populating artist directories, dropdowns, or full metadata exports.
//
// Operation:
// Calls ArtistService.GetAllArtists() which executes:
// SELECT id, name, artist_bio, image_url, submitted_by, created_at FROM artists
// to retrieve all artist records. Results are ordered by created_at descending to show newest artists first.
// Nullable columns (artist_bio, image_url, submitted_by) are scanned into pointer types (*string)
// and omitted from JSON responses if NULL per the struct tag annotations.
//
// Response:
// - 200 OK: Query successful; returns an array of Artist models (empty array if no artists exist)
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetAllArtists(artistService *services.ArtistService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call the service to retrieve all artists.
		// The service handles the database query and returns a properly formatted slice.
		artists, err := artistService.GetAllArtists(context.Background())
		if err != nil {
			// Query errors indicate database connectivity or syntax issues (this should not happen in production).
			// Return 500 and log the error for operational troubleshooting.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query artists"})
			return
		}

		// If no artists exist, artists is an empty slice.
		// JSON encoding treats this as an empty array: [], presenting a consistent API contract.
		// Ensure non-nil JSON output for tool consistency.
		if artists == nil {
			artists = []models.Artist{}
		}

		// Marshal the artists slice to JSON and return HTTP 200 OK with the array.
		// Gin's JSON() method handles encoding; the Content-Type is automatically set to application/json.
		c.JSON(http.StatusOK, artists)
	}
}
