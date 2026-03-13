package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateArtwork handles POST /api/albums/:id/artworks requests to create a new artwork record
// for an album. Artwork records represent album cover images or promotional artwork.
// Each artwork has metadata about approval status, priority ranking, and submission attribution.
//
// Request body structure:
//
//	{
//	  "source_id": "external-source-id" (optional),
//	  "image_url": "https://example.com/image.jpg" (required),
//	  "thumbnail_url": "https://example.com/thumb.jpg" (optional),
//	  "is_official": false,
//	  "submitted_by": "user-id" (optional)
//	}
//
// SQL Operation:
// Generates a UUID v4 for the artwork record and executes:
// INSERT INTO artworks (id, album_id, source_id, image_url, thumbnail_url, is_official, submitted_by, approval_status, priority_score)
// VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', 0)
//
// All new artworks are initialized with approval_status='pending' and priority_score=0,
// indicating they require moderation before being displayed in query results.
// Optional fields are normalized through nullableString() for proper NULL handling.
//
// Response:
// - 201 Created: Artwork successfully created; returns Artwork model with status 'pending'
// - 400 Bad Request: Missing required 'image_url' field
// - 500 Internal Server Error: Database error (foreign key violation on album_id, connection issues, etc.)
func CreateArtwork(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the album ID from the URL path (e.g., POST /albums/{id}/artworks).
		// This establishes the foreign key relationship to the parent album.
		albumID := c.Param("id")

		// Parse JSON request body with validation.
		// binding:"required" on image_url ensures artworks always have a valid image reference.
		var req struct {
			SourceID     string `json:"source_id"`
			ImageURL     string `json:"image_url"     binding:"required"`
			ThumbnailURL string `json:"thumbnail_url"`
			IsOfficial   bool   `json:"is_official"`
			SubmittedBy  string `json:"submitted_by"`
		}

		// Unmarshal and validate the JSON request body.
		// Gin's validator uses struct tags to enforce business logic constraints.
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image_url is required"})
			return
		}

		// Generate UUID v4 for the artwork record primary key.
		id := uuid.New().String()

		// Execute INSERT with hardcoded approval_status='pending' and priority_score=0.
		// This ensures all new artwork submissions start in pending state requiring moderation.
		// Parameterized query prevents SQL injection attacks through user input.
		_, err := db.Exec(
			context.Background(),
			`INSERT INTO artworks
				(id, album_id, source_id, image_url, thumbnail_url, is_official, submitted_by, approval_status, priority_score)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', 0)`,
			id, albumID,
			nullableString(req.SourceID),
			req.ImageURL,
			nullableString(req.ThumbnailURL),
			req.IsOfficial,
			nullableString(req.SubmittedBy),
		)
		if err != nil {
			// INSERT errors include foreign key constraint violations (invalid album_id).
			// These should ideally return 409 Conflict, but generic 500 is used here.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert artwork"})
			return
		}

		// Return HTTP 201 Created with the newly created artwork record.
		// The response includes the auto-initialized approval_status and priority_score.
		c.JSON(http.StatusCreated, models.Artwork{
			ID:             id,
			AlbumID:        albumID,
			SourceID:       req.SourceID,
			ImageURL:       req.ImageURL,
			ThumbnailURL:   req.ThumbnailURL,
			IsOfficial:     req.IsOfficial,
			SubmittedBy:    req.SubmittedBy,
			ApprovalStatus: "pending",
			PriorityScore:  0,
			CreatedAt:      time.Now(),
		})
	}
}

// GetArtworksByAlbum handles GET /api/albums/:id/artworks requests to retrieve artwork records
// for a specific album with optional filtering and sorting.
//
// Supported query parameters:
//   - status: Filter by approval_status ("pending", "approved", or "rejected")
//   - official: If "true", return only official artwork (is_official = true)
//   - sort: If "priority", sort by priority_score DESC; otherwise sort by created_at DESC
//
// SQL Operations:
// Base query constructs: SELECT id, album_id, source_id, image_url, thumbnail_url,
//
//	      is_official, submitted_by, approval_status, priority_score, created_at
//	FROM artworks WHERE album_id = $1
//
// Dynamic WHERE clauses are appended based on query parameters:
// - status filter: AND approval_status = $N (parameterized to prevent SQL injection)
// - official filter: AND is_official = true
//
// Sorting:
// - If sort=priority: ORDER BY priority_score DESC (highest priority first)
// - Otherwise: ORDER BY created_at DESC (newest first)
//
// Nullable columns are scanned into pointer types; NULL values are omitted from JSON responses.
// If no artworks match the query criteria, returns an empty array (never null).
//
// Response:
// - 200 OK: Returns artworks array (may be empty if no matches)
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetArtworksByAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the album ID from the URL path.
		albumID := c.Param("id")

		// --- Build query dynamically based on filters ---
		// Start with a base SELECT and WHERE clause that all requests share.
		// Dynamic query construction allows flexible filtering without multiple handler functions.
		// Note: In production systems, prepared statements or query builders (sqlc, sqlx) are preferred.
		query := `SELECT id, album_id, source_id, image_url, thumbnail_url,
		                 is_official, submitted_by, approval_status, priority_score, created_at
		          FROM artworks
		          WHERE album_id = $1`
		// Initialize parameter list with the albumID, which is always the first ($1) parameter.
		args := []any{albumID}
		// Track the next parameter index for adding dynamic WHERE clauses.
		// After $1 (albumID), the next parameter will be $2, then $3, etc.
		argIdx := 2

		// Parse the optional 'status' query parameter (e.g., ?status=approved).
		// Query() safely extracts query string parameters from the request URL.
		if status := c.Query("status"); status != "" {
			// Append a WHERE clause that uses the next available parameter index.
			// Using fmt.Sprintf with $N placeholders ensures proper parameter binding.
			query += fmt.Sprintf(" AND approval_status = $%d", argIdx)
			// Append the status value to the args list in the same order as parameters in query.
			args = append(args, status)
			// Increment argIdx for the next parameter.
			argIdx++
		}

		// Parse the optional 'official' query parameter (e.g., ?official=true).
		// This is a boolean filter; a simple string comparison to "true" enables flexible parsing.
		if c.Query("official") == "true" {
			// This WHERE clause uses a literal true value, not a parameter.
			// Since we control the literal "true" string, SQL injection is not a concern.
			query += " AND is_official = true"
		}

		// Parse the optional 'sort' query parameter (e.g., ?sort=priority).
		// Sorting is determined by query string values, not database parameters.
		if c.Query("sort") == "priority" {
			// Sort by priority_score descending (highest priority first).
			// Moderators use priority scores to rank competing artwork options.
			query += " ORDER BY priority_score DESC"
		} else {
			// Default: sort by created_at descending (newest first).
			// This is a sensible default for chronological ordering of submissions.
			query += " ORDER BY created_at DESC"
		}

		// Execute the dynamically constructed query with all compiled parameters.
		// Query() returns iterator for multiple rows; args... unpacks the slice as individual arguments.
		rows, err := db.Query(context.Background(), query, args...)
		if err != nil {
			// Query errors indicate database connectivity, syntax errors, or parameter type mismatches.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artworks"})
			return
		}
		// Defer rows.Close() to release the database cursor and return connection to the pool.
		defer rows.Close()

		// Initialize artwork slice to accumulate results.
		var artworks []models.Artwork
		// Iterate over result rows.
		for rows.Next() {
			var aw models.Artwork
			// Nullable pointer variables for optional columns.
			var sourceID, thumbnailURL, submittedBy *string

			// Scan the current row into the artwork struct and nullable pointers.
			// Column order must match the SELECT clause order.
			if err := rows.Scan(
				&aw.ID, &aw.AlbumID, &sourceID, &aw.ImageURL, &thumbnailURL,
				&aw.IsOfficial, &submittedBy, &aw.ApprovalStatus, &aw.PriorityScore, &aw.CreatedAt,
			); err != nil {
				// Row scan errors are silently skipped; iteration continues with next row.
				// In production, log these errors for visibility into potential data consistency issues.
				continue
			}
			// Dereference nullable pointers and populate the artwork model.
			if sourceID != nil {
				aw.SourceID = *sourceID
			}
			if thumbnailURL != nil {
				aw.ThumbnailURL = *thumbnailURL
			}
			if submittedBy != nil {
				aw.SubmittedBy = *submittedBy
			}
			// Append the populated artwork to the result slice.
			artworks = append(artworks, aw)
		}

		// Ensure the response is a non-nil empty array rather than nil if no artworks match.
		// JSON convention: null implies missing/unset, [] implies empty collection.
		// Explicitly setting a non-nil empty slice ensures consistent API behavior.
		if artworks == nil {
			artworks = []models.Artwork{}
		}

		// Marshal the artworks slice to JSON and return HTTP 200 OK.
		// The slice may be empty if no artworks match the filter criteria.
		c.JSON(http.StatusOK, artworks)
	}
}
