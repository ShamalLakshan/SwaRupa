package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
// SQL Operation:
// Generates a UUID v4 (RFC 4122) for the new artist record and executes:
// INSERT INTO artists (id, name, artist_bio, image_url, submitted_by) VALUES ($1, $2, $3, $4, $5)
// The optional fields are normalized through nullableString(), converting empty strings to SQL NULL
// for proper database semantics. This ensures nullable TEXT columns store NULL rather than empty strings.
// All values are parameterized to prevent SQL injection attacks.
//
// Response:
// - 201 Created: Artist successfully created; returns Artist model with generated ID and server timestamp
// - 400 Bad Request: Missing required 'name' field in JSON body
// - 500 Internal Server Error: Database error (connection, constraint violations, etc.)
func CreateArtist(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Define a request struct with validation tags for automatic input validation.
		// binding:"required" enforces non-empty values; unmarshaling failures automatically return 400.
		// JSON tags establish bidirectional mapping between Go struct fields and JSON keys.
		var req struct {
			Name          string `json:"name"           binding:"required"`
			MusicBrainzID string `json:"artist_bio"`
			ImageURL      string `json:"image_url"`
			SubmittedBy   string `json:"submitted_by"`
		}

		// Unmarshal and validate the JSON request body.
		// Gin's validator uses struct tags to enforce business logic (e.g., required fields).
		// This is preferable to manual validation as it's declarative and reusable.
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		// Generate a UUID v4 (RFC 4122) for the primary key.
		// UUIDs provide globally unique identifiers without requiring database sequences,
		// enabling client-side key generation and distributed system compatibility.
		// uuid.New() uses cryptographic randomness; .String() formats as a 36-character string.
		id := uuid.New().String()

		// Execute parameterized INSERT query to prevent SQL injection.
		// All parameters ($1, $2, etc.) are placeholder values replaced by pgx at execution time.
		// Optional fields are wrapped in nullableString() to convert empty strings to SQL NULL.
		// The pgx driver automatically handles type conversion and encoding for PostgreSQL protocol.
		_, err := db.Exec(
			context.Background(),
			`INSERT INTO artists (id, name, artist_bio, image_url, submitted_by)
			 VALUES ($1, $2, $3, $4, $5)`,
			id, req.Name, nullableString(req.MusicBrainzID), nullableString(req.ImageURL), nullableString(req.SubmittedBy),
		)
		if err != nil {
			// Database errors include connection failures, constraint violations (e.g., unique constraints),
			// and query syntax errors. All are treated as 500 Internal Server Error.
			// In production systems, log the error and error type for operational visibility.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert artist"})
			return
		}

		// Return HTTP 201 Created with the newly created artist record.
		// The response includes all provided fields and the server-generated ID.
		// Clients use the returned ID for subsequent requests (e.g., GET /artists/{id}).
		c.JSON(http.StatusCreated, models.Artist{
			ID:            id,
			Name:          req.Name,
			MusicBrainzID: req.MusicBrainzID,
			ImageURL:      req.ImageURL,
			SubmittedBy:   req.SubmittedBy,
			CreatedAt:     time.Now(),
		})
	}
}

// GetArtist handles GET /api/artists/:id requests to retrieve a single artist record by ID.
// The :id path parameter is the artist's UUID as returned from CreateArtist or stored in the database.
//
// SQL Operation:
// Executes SELECT id, name, artist_bio, image_url, submitted_by, created_at FROM artists WHERE id = $1
// using an indexed primary key lookup for O(1) retrieval performance.
// Nullable columns (artist_bio, image_url, submitted_by) are scanned into pointer types (*string).
// If a NULL value is encountered in the database, the pointer is set to nil and the field is omitted
// from the JSON response due to the omitempty struct tag annotation.
//
// Response:
// - 200 OK: Artist found; returns complete Artist model with all fields
// - 404 Not Found: No artist with the given ID exists in the database
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetArtist(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the artist ID from the URL path parameter.
		// The :id placeholder in route registration (e.g., GET /artists/:id) maps to Param("id").
		id := c.Param("id")

		// Initialize the Artist model and pointer variables for nullable columns.
		// Using pointers allows us to distinguish between database NULL (nil) and empty strings ("").
		// This is critical for accurate data representation and API contract clarity.
		var artist models.Artist
		var musicBrainzID, imageURL, submittedBy *string

		// Execute a single-row SELECT query using QueryRow.
		// QueryRow is optimized for single-row results and raises an error if no rows match.
		// Parameterized queries ($1) prevent SQL injection by escaping special characters.
		err := db.QueryRow(
			context.Background(),
			`SELECT id, name, artist_bio, image_url, submitted_by, created_at
			 FROM artists WHERE id = $1`,
			id,
		// Scan maps query results to destination variables in the same column order as the SELECT clause.
		// Slice destinations must be addressable (prefixed with &), scalar pointers accept nil for NULL values.
		).Scan(
			&artist.ID,
			&artist.Name,
			&musicBrainzID,
			&imageURL,
			&submittedBy,
			&artist.CreatedAt,
		)
		if err != nil {
			// QueryRow returns pgx.ErrNoRows when the WHERE clause matches no rows.
			// This should result in HTTP 404 Not Found per REST conventions.
			// Other errors (connection failures, scan type mismatches) indicate server problems (500).
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}

		// Dereference nullable pointers and assign to the model.
		// This pattern handles the Go type system's inability to represent NULL in non-pointer types.
		// Only dereference if the pointer is non-nil; omitting this check causes runtime nil dereference panic.
		if musicBrainzID != nil {
			artist.MusicBrainzID = *musicBrainzID
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
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
// SQL Operation:
// Executes SELECT id, name, artist_bio, image_url, submitted_by, created_at FROM artists
// to retrieve all artist records. Results are ordered by created_at descending to show newest artists first.
// Nullable columns (artist_bio, image_url, submitted_by) are scanned into pointer types (*string)
// and omitted from JSON responses if NULL per the struct tag annotations.
//
// Response:
// - 200 OK: Query successful; returns an array of Artist models (empty array if no artists exist)
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetAllArtists(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Initialize a slice to hold all artist records from the database.
		// Using make with zero length and capacity allows dynamic growth as rows are scanned.
		var artists []models.Artist

		// Execute a multi-row SELECT query to retrieve all artists, ordered by creation timestamp.
		// Query() returns a Rows iterator that must be explicitly iterated with Next() calls.
		// Compared to QueryRow, Query() is optimized for variable-length result sets.
		rows, err := db.Query(
			context.Background(),
			`SELECT id, name, artist_bio, image_url, submitted_by, created_at
			 FROM artists
			 ORDER BY created_at DESC`,
		)
		if err != nil {
			// Query errors indicate database connectivity or syntax issues (this should not happen in production).
			// Return 500 and log the error for operational troubleshooting.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query artists"})
			return
		}
		// Defer ensures the result iterator is properly closed and any connection resources are released.
		// Failure to close rows can leak server-side cursor resources over time.
		defer rows.Close()

		// Iterate over all result rows using the Rows iterator.
		// Next() advances the iterator; it returns false when all rows have been consumed.
		for rows.Next() {
			var artist models.Artist
			var musicBrainzID, imageURL, submittedBy *string

			// Scan the current row's values into Go variables, same pattern as GetArtist single-row queries.
			// Pointers allow NULL values to be represented as nil; scalar fields get zero values if NULL.
			err := rows.Scan(
				&artist.ID,
				&artist.Name,
				&musicBrainzID,
				&imageURL,
				&submittedBy,
				&artist.CreatedAt,
			)
			if err != nil {
				// Scan errors indicate type mismatches or corrupted data and should trigger a 500 response.
				// In production, log this error with row information for debugging.
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan artist row"})
				return
			}

			// Dereference nullable pointers and assign to the model, using the same NULL-handling pattern
			// as GetArtist to distinguish between unset fields (NULL) and default/empty values ("").
			if musicBrainzID != nil {
				artist.MusicBrainzID = *musicBrainzID
			}
			if imageURL != nil {
				artist.ImageURL = *imageURL
			}
			if submittedBy != nil {
				artist.SubmittedBy = *submittedBy
			}

			// Append the populated artist to the result slice.
			// Go's slice append operation automatically handles growth and reallocation.
			artists = append(artists, artist)
		}

		// Check the Rows iterator error status after the loop completes.
		// This catches any errors that occurred during iteration but were not raised in Next().
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error iterating artists"})
			return
		}

		// If no artists exist, artists remains as a nil slice or empty slice.
		// JSON encoding treats both as empty arrays: [], presenting a consistent API contract.
		// Initialize to empty slice explicitly to ensure non-nil JSON output for tool consistency.
		if artists == nil {
			artists = []models.Artist{}
		}

		// Marshal the artists slice to JSON and return HTTP 200 OK with the array.
		// Gin's JSON() method handles encoding; the Content-Type is automatically set to application/json.
		c.JSON(http.StatusOK, artists)
	}
}
