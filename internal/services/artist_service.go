package services

import (
	"context"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/database"
	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ArtistService provides business logic for artist operations.
type ArtistService struct {
	db *pgxpool.Pool
}

// NewArtistService creates a new artist service instance.
func NewArtistService(db *pgxpool.Pool) *ArtistService {
	return &ArtistService{db: db}
}

// CreateArtist creates a new artist record with approval_status = 'pending'.
func (s *ArtistService) CreateArtist(ctx context.Context, name, ArtistBio, imageURL, submittedBy string) (*models.Artist, error) {
	id := uuid.New().String()

	_, err := s.db.Exec(ctx,
		`INSERT INTO artists (id, name, artist_bio, image_url, submitted_by, approval_status, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', now())`,
		id, name, nullableString(ArtistBio), nullableString(imageURL), nullableString(submittedBy),
	)
	if err != nil {
		return nil, err
	}

	return s.GetArtistByID(ctx, id)
}

// GetArtistByID retrieves a single artist by ID with approval details.
func (s *ArtistService) GetArtistByID(ctx context.Context, artistID string) (*models.Artist, error) {
	var artist models.Artist
	var ArtistBio, imageURL, submittedBy, approvedBy, rejectionReason *string
	var approvedAt *time.Time

	err := s.db.QueryRow(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM artists WHERE id = $1`,
		artistID,
	).Scan(&artist.ID, &artist.Name, &ArtistBio, &imageURL, &submittedBy, &artist.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &artist.CreatedAt)
	if err != nil {
		return nil, err
	}

	if ArtistBio != nil {
		artist.ArtistBio = *ArtistBio
	}
	if imageURL != nil {
		artist.ImageURL = *imageURL
	}
	if submittedBy != nil {
		artist.SubmittedBy = *submittedBy
	}
	artist.ApprovedBy = approvedBy
	artist.ApprovedAt = approvedAt
	artist.RejectionReason = rejectionReason

	return &artist, nil
}

// GetAllArtists retrieves all approved artists (public endpoint).
func (s *ArtistService) GetAllArtists(ctx context.Context) ([]models.Artist, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM artists
		 WHERE approval_status = 'approved'
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var artist models.Artist
		var ArtistBio, imageURL, submittedBy, approvedBy, rejectionReason *string
		var approvedAt *time.Time

		if err := rows.Scan(
			&artist.ID, &artist.Name, &ArtistBio, &imageURL, &submittedBy, &artist.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &artist.CreatedAt,
		); err != nil {
			continue
		}

		if ArtistBio != nil {
			artist.ArtistBio = *ArtistBio
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
		}
		artist.ApprovedBy = approvedBy
		artist.ApprovedAt = approvedAt
		artist.RejectionReason = rejectionReason

		artists = append(artists, artist)
	}

	if artists == nil {
		artists = []models.Artist{}
	}

	return artists, nil
}

// GetAllArtistsWithPagination retrieves artists with pagination support.
func (s *ArtistService) GetAllArtistsWithPagination(ctx context.Context, page, limit int) ([]models.Artist, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)
	return database.GetAllArtistsWithPagination(ctx, s.db, limit, offset)
}

// SearchArtistsByName performs fuzzy search on artist names (approved artists only).
func (s *ArtistService) SearchArtistsByName(ctx context.Context, query string, page, limit int) ([]models.Artist, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)
	return database.SearchArtistsByName(ctx, s.db, query, limit, offset)
}

// ApproveArtist marks an artist as approved and records the approver.
func (s *ArtistService) ApproveArtist(ctx context.Context, artistID, approvedBy string) (*models.Artist, error) {
	_, err := s.db.Exec(ctx,
		`UPDATE artists SET approval_status = 'approved', approved_by = $1, approved_at = now()
		 WHERE id = $2`,
		approvedBy, artistID,
	)
	if err != nil {
		return nil, err
	}
	return s.GetArtistByID(ctx, artistID)
}

// RejectArtist marks an artist as rejected with an optional reason.
func (s *ArtistService) RejectArtist(ctx context.Context, artistID, approvedBy, rejectionReason string) (*models.Artist, error) {
	_, err := s.db.Exec(ctx,
		`UPDATE artists SET approval_status = 'rejected', approved_by = $1, rejection_reason = $2, approved_at = now()
		 WHERE id = $3`,
		approvedBy, nullableString(rejectionReason), artistID,
	)
	if err != nil {
		return nil, err
	}
	return s.GetArtistByID(ctx, artistID)
}

// GetApprovedArtists retrieves only approved artists with pagination (public endpoint).
func (s *ArtistService) GetApprovedArtists(ctx context.Context, page, limit int) ([]models.Artist, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)

	// Get total count
	var total int64
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM artists WHERE approval_status = 'approved'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	rows, err := s.db.Query(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM artists WHERE approval_status = 'approved'
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var artist models.Artist
		var ArtistBio, imageURL, submittedBy, approvedBy, rejectionReason *string
		var approvedAt *time.Time

		if err := rows.Scan(&artist.ID, &artist.Name, &ArtistBio, &imageURL, &submittedBy, &artist.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &artist.CreatedAt); err != nil {
			continue
		}

		if ArtistBio != nil {
			artist.ArtistBio = *ArtistBio
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
		}
		artist.ApprovedBy = approvedBy
		artist.ApprovedAt = approvedAt
		artist.RejectionReason = rejectionReason
		artists = append(artists, artist)
	}

	if artists == nil {
		artists = []models.Artist{}
	}

	return artists, total, nil
}

// GetPendingArtists retrieves pending approval artists (admin only).
func (s *ArtistService) GetPendingArtists(ctx context.Context, page, limit int) ([]models.Artist, int64, error) {
	page, limit = models.ValidatePaginationParams(page, limit)
	offset := models.CalculateOffset(page, limit)

	// Get total count
	var total int64
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM artists WHERE approval_status = 'pending'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	rows, err := s.db.Query(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, approval_status, approved_by, approved_at, rejection_reason, created_at
		 FROM artists WHERE approval_status = 'pending'
		 ORDER BY created_at ASC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var artist models.Artist
		var ArtistBio, imageURL, submittedBy, approvedBy, rejectionReason *string
		var approvedAt *time.Time

		if err := rows.Scan(&artist.ID, &artist.Name, &ArtistBio, &imageURL, &submittedBy, &artist.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason, &artist.CreatedAt); err != nil {
			continue
		}

		if ArtistBio != nil {
			artist.ArtistBio = *ArtistBio
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
		}
		artist.ApprovedBy = approvedBy
		artist.ApprovedAt = approvedAt
		artist.RejectionReason = rejectionReason
		artists = append(artists, artist)
	}

	if artists == nil {
		artists = []models.Artist{}
	}

	return artists, total, nil
}
