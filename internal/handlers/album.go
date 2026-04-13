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

// CreateAlbum handles POST /api/albums requests to create a new album and associate it with one or more artists.
// The handler implements a transactional create pattern to ensure atomic insertion of both the album record
// and all album_artists cross-reference records, maintaining referential integrity.
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
// SQL Operations:
// 1. Begins a database transaction: BEGIN TRANSACTION
// 2. Inserts album record: INSERT INTO albums (id, title, release_year, submitted_by) VALUES ($1, $2, $3, $4)
// 3. For each artist_id, inserts junction record: INSERT INTO album_artists (album_id, artist_id) VALUES ($1, $2)
// 4. Commits transaction: COMMIT (or rolls back on any error)
//
// The transaction ensures all-or-nothing semantics: if any INSERT fails (e.g., invalid artist_id foreign key),
// the entire transaction is rolled back, preventing orphaned album records without artist associations.
// Optional fields are normalized through nullableString() to convert empty strings to SQL NULL.
//
// Response:
// - 201 Created: Album and artist associations successfully created; returns Album model with ID and artists
// - 400 Bad Request: Missing 'title' or 'artist_ids' fields, or artist_ids is empty
// - 500 Internal Server Error: Database error (transaction failure, constraint violation, etc.)
func CreateAlbum(db *pgxpool.Pool) gin.HandlerFunc {
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

		// Generate UUID for the album record.
		id := uuid.New().String()

		// --- Transaction: insert album + album_artists atomically ---
		// Begin() starts a database transaction, which allows multiple SQL statements to execute
		// with ACID semantics (Atomicity, Consistency, Isolation, Durability).
		// If any statement fails, the transaction is rolled back, ensuring no partial updates.
		tx, err := db.Begin(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
			return
		}
		// Defer ensures the transaction is rolled back if not explicitly committed.
		// This is a safety mechanism preventing leaked transactions on early returns.
		defer tx.Rollback(context.Background())

		// 1. Insert the album record into the albums table.
		// This establishes the primary album entity with title, year, and submission provenance.
		// The transaction context (tx) ensures this statement participates in the atomic unit.
		_, err = tx.Exec(
			context.Background(),
			`INSERT INTO albums (id, title, release_year, submitted_by)
			 VALUES ($1, $2, $3, $4)`,
			id, req.Title, req.ReleaseYear, nullableString(req.SubmittedBy),
		)
		if err != nil {
			// Exec errors (column type mismatches, constraint violations) fail the transaction.
			// The deferred Rollback() cleans up automatically; no explicit cleanup needed.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert album"})
			return
		}

		// 2. Insert album_artists junction records for many-to-many relationship.
		// Each iteration inserts a row linking the album to one artist.
		// Using a loop with individual INSERTs is simpler than a multi-value INSERT
		// but less efficient. In production, consider batch insert or multi-value syntax.
		for _, artistID := range req.ArtistIDs {
			_, err = tx.Exec(
				context.Background(),
				`INSERT INTO album_artists (album_id, artist_id) VALUES ($1, $2)`,
				id, artistID,
			)
			if err != nil {
				// Foreign key constraint violations (invalid artistID) will error here.
				// The transaction is rolled back, keeping the database in a consistent state.
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to link artist: " + artistID})
				return
			}
		}

		// 3. Commit the transaction.
		// This makes all previous INSERTs durable and visible to other database clients.
		// If Commit fails (rare but possible due to serialization issues), return 500.
		if err = tx.Commit(context.Background()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
			return
		}

		// Return HTTP 201 Created with the newly created album.
		// Note: Artists slice is empty here; clients can call GetAlbum to retrieve populated artists.
		c.JSON(http.StatusCreated, models.Album{
			ID:          id,
			Title:       req.Title,
			ReleaseYear: req.ReleaseYear,
			SubmittedBy: req.SubmittedBy,
			CreatedAt:   time.Now(),
		})
	}
}

// GetAlbum handles GET /api/albums/:id requests to retrieve a single album record with its associated artists.
// The :id path parameter is the album's UUID as returned from CreateAlbum or stored in the database.
//
// SQL Operations:
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
// Nullable columns (submitted_by, artist_bio, etc.) are scanned into pointer types.
// If NULL in the database, the field is omitted from the JSON response.
//
// Response:
// - 200 OK: Album found; returns Album model with populated Artists slice
// - 404 Not Found: No album with the given ID exists in the database
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the album ID from the URL path parameter.
		id := c.Param("id")

		// Initialize Album model and nullable pointer for optional columns.
		var album models.Album
		var submittedBy *string

		// Fetch the album record by ID using a simple indexed lookup.
		// QueryRow is optimized for singleton results and implicitly checks for zero rows.
		err := db.QueryRow(
			context.Background(),
			`SELECT id, title, release_year, submitted_by, created_at
			 FROM albums WHERE id = $1`,
			id,
		).Scan(&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.CreatedAt)
		if err != nil {
			// Album not found: return 404 Not Found.
			c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
			return
		}
		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}

		// Fetch associated artists via INNER JOIN on the album_artists junction table.
		// Query() returns an iterator over multiple rows, unlike QueryRow which expects one row.
		// The INNER JOIN ensures only artists explicitly linked to this album are returned.
		rows, err := db.Query(
			context.Background(),
			`SELECT a.id, a.name, a.artist_bio, a.image_url, a.submitted_by, a.created_at
			 FROM artists a
			 INNER JOIN album_artists aa ON aa.artist_id = a.id
			 WHERE aa.album_id = $1`,
			id,
		)
		if err != nil {
			// Query errors indicate database connectivity or syntax issues.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artists"})
			return
		}
		// Always defer rows.Close() to release database resources (connection back to pool).
		// Failure to close rows can lead to connection pool exhaustion in high-traffic systems.
		defer rows.Close()

		// Initialize the artists slice to hold query results.
		// Starting with nil (not empty slice) ensures json.Marshal returns null for no artists
		// (though both are semantically equivalent in REST APIs, convention varies).
		var artists []models.Artist
		// Iterate over rows.Next() to process each result row.
		// rows.Next() returns false when iteration completes or an error occurs.
		for rows.Next() {
			var artist models.Artist
			// Nullable column pointers for optional artist fields.
			var mbID, imgURL, subBy *string

			// Scan unpacks the current row into destination variables.
			// Order must match the SELECT column order; type mismatches cause errors.
			if err := rows.Scan(
				&artist.ID, &artist.Name, &mbID, &imgURL, &subBy, &artist.CreatedAt,
			); err != nil {
				// Row-level scan errors (e.g., type mismatches) are logged silently and skipped.
				// In production, log these errors for debugging; continue to process remaining rows.
				continue
			}
			// Dereference nullable pointers and populate the artist model.
			if mbID != nil {
				artist.MusicBrainzID = *mbID
			}
			if imgURL != nil {
				artist.ImageURL = *imgURL
			}
			if subBy != nil {
				artist.SubmittedBy = *subBy
			}
			// Append the populated artist to the slice.
			artists = append(artists, artist)
		}

		// Assign the artists slice to the album model.
		// If no artists were associated with the album, the slice is non-nil but empty.
		album.Artists = artists
		// Marshal the Album with associated Artists and return HTTP 200 OK.
		c.JSON(http.StatusOK, album)
	}
}

// GetAllAlbums handles GET /api/albums requests to retrieve all album records with their associated artists.
// This endpoint is useful for populating album directories, galleries, or performing metadata analysis.
// Each album in the result includes a populated Artists slice with all linked artist records.
//
// SQL Operations:
//
//  1. Fetches all albums: SELECT id, title, release_year, submitted_by, created_at FROM albums
//     Ordered by created_at DESC to show newest albums first.
//
//  2. For each album, fetches associated artists via INNER JOIN (same as GetAlbum):
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
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetAllAlbums(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute multi-row SELECT to fetch all albums, ordered by creation date descending.
		// Query() returns a Rows iterator for variable-length result sets.
		albumRows, err := db.Query(
			context.Background(),
			`SELECT id, title, release_year, submitted_by, created_at
			 FROM albums
			 ORDER BY created_at DESC`,
		)
		if err != nil {
			// Query errors indicate database connectivity or syntax issues.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query albums"})
			return
		}
		// Always defer rows.Close() to release database resources and prevent connection pool exhaustion.
		defer albumRows.Close()

		// Initialize slice to accumulate albums from query results.
		var albums []models.Album

		// Iterate over each album row using albumRows.Next().
		// This loop processes all albums sequentially from the database result set.
		for albumRows.Next() {
			var album models.Album
			var submittedBy *string

			// Scan the current album row into variables.
			if err := albumRows.Scan(
				&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.CreatedAt,
			); err != nil {
				// Scan errors should trigger a 500 response; abort the operation.
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan album row"})
				return
			}

			// Dereference nullable submittedBy pointer.
			if submittedBy != nil {
				album.SubmittedBy = *submittedBy
			}

			// Fetch artists associated with this specific album using the same INNER JOIN pattern as GetAlbum.
			// This requires a separate database round-trip per album; see comments in GetAlbum for join details.
			artistRows, err := db.Query(
				context.Background(),
				`SELECT a.id, a.name, a.artist_bio, a.image_url, a.submitted_by, a.created_at
				 FROM artists a
				 INNER JOIN album_artists aa ON aa.artist_id = a.id
				 WHERE aa.album_id = $1`,
				album.ID,
			)
			if err != nil {
				// Artist query failure should not abort all album retrieval; log and continue with empty artists.
				// In production, log this error with the album ID for debugging.
				artistRows = nil
			} else {
				// Initialize artists slice for this album.
				var artists []models.Artist

				// Iterate over artist rows for this specific album.
				for artistRows.Next() {
					var artist models.Artist
					var mbID, imgURL, subBy *string

					if err := artistRows.Scan(
						&artist.ID, &artist.Name, &mbID, &imgURL, &subBy, &artist.CreatedAt,
					); err != nil {
						// Skip rows with scan errors; continue processing remaining artist rows.
						continue
					}

					// Dereference nullable pointers for optional artist fields.
					if mbID != nil {
						artist.MusicBrainzID = *mbID
					}
					if imgURL != nil {
						artist.ImageURL = *imgURL
					}
					if subBy != nil {
						artist.SubmittedBy = *subBy
					}

					// Append the populated artist to this album's artists slice.
					artists = append(artists, artist)
				}

				// Assign the populated artists slice to the album model.
				album.Artists = artists
				// Close the artist rows iterator to release database resources.
				artistRows.Close()
			}

			// Append the album (with or without artists) to the result slice.
			albums = append(albums, album)
		}

		// Check for errors that occurred during row iteration.
		if err := albumRows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error iterating albums"})
			return
		}

		// Ensure non-nil JSON output: initialize to empty slice if no albums exist.
		// This provides a consistent API contract: empty results are always [].
		if albums == nil {
			albums = []models.Album{}
		}

		// Marshal the albums slice with populated artists to JSON and return HTTP 200 OK.
		c.JSON(http.StatusOK, albums)
	}
}
