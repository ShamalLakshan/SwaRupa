package services

import (
	"context"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/database"
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
// The album is created with approval_status = 'pending'.
func (s *AlbumService) CreateAlbum(ctx context.Context, title string, releaseYear int, artistIDs []string, submittedBy string) (*models.Album, error) {
	id := uuid.New().String()

	// Begin transaction for atomic album + artist association
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Insert album record with approval_status = 'pending'
	_, err = tx.Exec(ctx,
		`INSERT INTO albums (id, title, release_year, submitted_by, approval_status, created_at)
		 VALUES ($1, $2, $3, $4, 'pending', now())`,
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

// GetAlbumByID retrieves a single album with its associated artists and approval details.
func (s *AlbumService) GetAlbumByID(ctx context.Context, albumID string) (*models.Album, error) {
	var album models.Album
	var submittedBy, approvedBy, rejectionReason *string
	var approvedAt *time.Time

	// Fetch album record
	err := s.db.QueryRow(ctx,
		`SELECT id, title, release_year, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM albums WHERE id = $1`,
		albumID,
	).Scan(&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &album.CreatedAt)
	if err != nil {
		return nil, err
	}

	if submittedBy != nil {
		album.SubmittedBy = *submittedBy
	}
	album.ApprovedBy = approvedBy
	album.ApprovedAt = approvedAt
	album.RejectionReason = rejectionReason

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

// GetAllAlbums retrieves all approved albums with their associated artists (public endpoint).
func (s *AlbumService) GetAllAlbums(ctx context.Context) ([]models.Album, error) {
	// Fetch approved albums only
	albumRows, err := s.db.Query(ctx,
		`SELECT id, title, release_year, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM albums WHERE approval_status = 'approved'
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer albumRows.Close()

	var albums []models.Album

	for albumRows.Next() {
		var album models.Album
		var submittedBy, approvedBy, rejectionReason *string
		var approvedAt *time.Time

		if err := albumRows.Scan(
			&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &album.CreatedAt,
		); err != nil {
			continue
		}

		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}
		album.ApprovedBy = approvedBy
		album.ApprovedAt = approvedAt
		album.RejectionReason = rejectionReason

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

// GetAllAlbumsWithPagination retrieves albums with pagination support and their associated artists.
func (s *AlbumService) GetAllAlbumsWithPagination(ctx context.Context, page, limit int) ([]models.Album, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)
	return database.GetAllAlbumsWithPagination(ctx, s.db, limit, offset)
}

// SearchAlbumsByName performs fuzzy search on album titles (approved albums only).
func (s *AlbumService) SearchAlbumsByName(ctx context.Context, query string, page, limit int) ([]models.Album, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)
	return database.SearchAlbumsByName(ctx, s.db, query, limit, offset)
}

// ApproveAlbum marks an album as approved and records the approver.
func (s *AlbumService) ApproveAlbum(ctx context.Context, albumID, approvedBy string) (*models.Album, error) {
	_, err := s.db.Exec(ctx,
		`UPDATE albums SET approval_status = 'approved', approved_by = $1, approved_at = now()
		 WHERE id = $2`,
		approvedBy, albumID,
	)
	if err != nil {
		return nil, err
	}
	return s.GetAlbumByID(ctx, albumID)
}

// RejectAlbum marks an album as rejected with an optional reason.
func (s *AlbumService) RejectAlbum(ctx context.Context, albumID, approvedBy, rejectionReason string) (*models.Album, error) {
	_, err := s.db.Exec(ctx,
		`UPDATE albums SET approval_status = 'rejected', approved_by = $1, rejection_reason = $2, approved_at = now()
		 WHERE id = $3`,
		approvedBy, nullableString(rejectionReason), albumID,
	)
	if err != nil {
		return nil, err
	}
	return s.GetAlbumByID(ctx, albumID)
}

// GetApprovedAlbums retrieves only approved albums with pagination (public endpoint).
func (s *AlbumService) GetApprovedAlbums(ctx context.Context, page, limit int) ([]models.Album, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)

	// Get total count
	var total int64
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM albums WHERE approval_status = 'approved'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated approved albums
	albumRows, err := s.db.Query(ctx,
		`SELECT id, title, release_year, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM albums WHERE approval_status = 'approved'
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer albumRows.Close()

	var albums []models.Album

	for albumRows.Next() {
		var album models.Album
		var submittedBy, approvedBy, rejectionReason *string
		var approvedAt *time.Time

		if err := albumRows.Scan(
			&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &album.CreatedAt,
		); err != nil {
			continue
		}

		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}
		album.ApprovedBy = approvedBy
		album.ApprovedAt = approvedAt
		album.RejectionReason = rejectionReason

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

	return albums, total, nil
}

// GetPendingAlbums retrieves pending approval albums (admin only).
func (s *AlbumService) GetPendingAlbums(ctx context.Context, page, limit int) ([]models.Album, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)

	// Get total count
	var total int64
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM albums WHERE approval_status = 'pending'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated pending albums
	albumRows, err := s.db.Query(ctx,
		`SELECT id, title, release_year, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM albums WHERE approval_status = 'pending'
		 ORDER BY created_at ASC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer albumRows.Close()

	var albums []models.Album

	for albumRows.Next() {
		var album models.Album
		var submittedBy, approvedBy, rejectionReason *string
		var approvedAt *time.Time

		if err := albumRows.Scan(
			&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &album.CreatedAt,
		); err != nil {
			continue
		}

		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}
		album.ApprovedBy = approvedBy
		album.ApprovedAt = approvedAt
		album.RejectionReason = rejectionReason

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

	return albums, total, nil
}

// nullableString helper
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
