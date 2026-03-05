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

// CreateAlbum handler
// Accepts artist_ids (required, at least one) and inserts into albums + album_artists atomically.
func CreateAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		id := uuid.New().String()

		// --- Transaction: insert album + album_artists atomically ---
		tx, err := db.Begin(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
			return
		}
		defer tx.Rollback(context.Background())

		// 1. Insert album
		_, err = tx.Exec(
			context.Background(),
			`INSERT INTO albums (id, title, release_year, submitted_by)
			 VALUES ($1, $2, $3, $4)`,
			id, req.Title, req.ReleaseYear, nullableString(req.SubmittedBy),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert album"})
			return
		}

		// 2. Insert album_artists rows
		for _, artistID := range req.ArtistIDs {
			_, err = tx.Exec(
				context.Background(),
				`INSERT INTO album_artists (album_id, artist_id) VALUES ($1, $2)`,
				id, artistID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to link artist: " + artistID})
				return
			}
		}

		if err = tx.Commit(context.Background()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
			return
		}

		c.JSON(http.StatusCreated, models.Album{
			ID:          id,
			Title:       req.Title,
			ReleaseYear: req.ReleaseYear,
			SubmittedBy: req.SubmittedBy,
			CreatedAt:   time.Now(),
		})
	}
}

// GetAlbum handler
// Returns the album with its full list of artists.
func GetAlbum(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Fetch album row
		var album models.Album
		var submittedBy *string

		err := db.QueryRow(
			context.Background(),
			`SELECT id, title, release_year, submitted_by, created_at
			 FROM albums WHERE id = $1`,
			id,
		).Scan(&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.CreatedAt)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
			return
		}
		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}

		// Fetch linked artists via album_artists join
		rows, err := db.Query(
			context.Background(),
			`SELECT a.id, a.name, a.musicbrainz_id, a.image_url, a.submitted_by, a.created_at
			 FROM artists a
			 INNER JOIN album_artists aa ON aa.artist_id = a.id
			 WHERE aa.album_id = $1`,
			id,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artists"})
			return
		}
		defer rows.Close()

		var artists []models.Artist
		for rows.Next() {
			var artist models.Artist
			var mbID, imgURL, subBy *string

			if err := rows.Scan(
				&artist.ID, &artist.Name, &mbID, &imgURL, &subBy, &artist.CreatedAt,
			); err != nil {
				continue
			}
			if mbID != nil {
				artist.MusicBrainzID = *mbID
			}
			if imgURL != nil {
				artist.ImageURL = *imgURL
			}
			if subBy != nil {
				artist.SubmittedBy = *subBy
			}
			artists = append(artists, artist)
		}

		album.Artists = artists
		c.JSON(http.StatusOK, album)
	}
}
