package services

import (
	"context"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AlbumService provides business logic for album operations.
type AlbumService struct {
	db *pgxpool.Pool
}

// NewAlbumService creates a new album service instance.
func NewAlbumService(db *pgxpool.Pool) *AlbumService {
	return &AlbumService{db: db}
}

// CreateAlbum creates a new album and associates it with artists in a transaction.
func (s *AlbumService) CreateAlbum(ctx context.Context, title string, releaseYear int, artistIDs []string, submittedBy string) (*models.Album, error) {
	id := uuid.New().String()

	// Begin transaction for atomic album + artist association
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Insert album record
	_, err = tx.Exec(ctx,
		`INSERT INTO albums (id, title, release_year, submitted_by, created_at)
		 VALUES ($1, $2, $3, $4, now())`,
		id, title, releaseYear, nullableString(submittedBy),
	)
	if err != nil {
		return nil, err
	}

	// Insert album_artists junction records
	for _, artistID := range artistIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO album_artists (album_id, artist_id) VALUES ($1, $2)`,
			id, artistID,
		)
		if err != nil {
			return nil, err
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Fetch and return complete album with artists
	return s.GetAlbumByID(ctx, id)
}

// GetAlbumByID retrieves a single album with its associated artists.
func (s *AlbumService) GetAlbumByID(ctx context.Context, albumID string) (*models.Album, error) {
	var album models.Album
	var submittedBy *string

	// Fetch album record
	err := s.db.QueryRow(ctx,
		`SELECT id, title, release_year, submitted_by, created_at
		 FROM albums WHERE id = $1`,
		albumID,
	).Scan(&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.CreatedAt)
	if err != nil {
		return nil, err
	}

	if submittedBy != nil {
		album.SubmittedBy = *submittedBy
	}

	// Fetch associated artists
	rows, err := s.db.Query(ctx,
		`SELECT a.id, a.name, a.artist_bio, a.image_url, a.submitted_by, a.created_at
		 FROM artists a
		 INNER JOIN album_artists aa ON aa.artist_id = a.id
		 WHERE aa.album_id = $1`,
		albumID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var artist models.Artist
		var artist_bio, imgURL, artistSubBy *string

		if err := rows.Scan(
			&artist.ID, &artist.Name, &artist_bio, &imgURL, &artistSubBy, &artist.CreatedAt,
		); err != nil {
			continue
		}

		if artist_bio != nil {
			artist.ArtistBio = *artist_bio
		}
		if imgURL != nil {
			artist.ImageURL = *imgURL
		}
		if artistSubBy != nil {
			artist.SubmittedBy = *artistSubBy
		}

		artists = append(artists, artist)
	}

	album.Artists = artists
	return &album, nil
}

// GetAllAlbums retrieves all albums with their associated artists.
func (s *AlbumService) GetAllAlbums(ctx context.Context) ([]models.Album, error) {
	// Fetch all albums
	albumRows, err := s.db.Query(ctx,
		`SELECT id, title, release_year, submitted_by, created_at
		 FROM albums
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer albumRows.Close()

	var albums []models.Album

	for albumRows.Next() {
		var album models.Album
		var submittedBy *string

		if err := albumRows.Scan(
			&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.CreatedAt,
		); err != nil {
			continue
		}

		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}

		// Fetch artists for this album
		artistRows, err := s.db.Query(ctx,
			`SELECT a.id, a.name, a.artist_bio, a.image_url, a.submitted_by, a.created_at
			 FROM artists a
			 INNER JOIN album_artists aa ON aa.artist_id = a.id
			 WHERE aa.album_id = $1`,
			album.ID,
		)
		if err == nil {
			var artists []models.Artist

			for artistRows.Next() {
				var artist models.Artist
				var artist_bio, imgURL, artistSubBy *string

				if err := artistRows.Scan(
					&artist.ID, &artist.Name, &artist_bio, &imgURL, &artistSubBy, &artist.CreatedAt,
				); err != nil {
					continue
				}

				if artist_bio != nil {
					artist.ArtistBio = *artist_bio
				}
				if imgURL != nil {
					artist.ImageURL = *imgURL
				}
				if artistSubBy != nil {
					artist.SubmittedBy = *artistSubBy
				}

				artists = append(artists, artist)
			}

			album.Artists = artists
			artistRows.Close()
		}

		albums = append(albums, album)
	}

	if albums == nil {
		albums = []models.Album{}
	}

	return albums, nil
}

// nullableString helper
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
