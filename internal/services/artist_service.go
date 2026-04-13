package services

import (
	"context"

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

// CreateArtist creates a new artist record.
func (s *ArtistService) CreateArtist(ctx context.Context, name, ArtistBio, imageURL, submittedBy string) (*models.Artist, error) {
	id := uuid.New().String()

	_, err := s.db.Exec(ctx,
		`INSERT INTO artists (id, name, artist_bio, image_url, submitted_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, now())`,
		id, name, nullableString(ArtistBio), nullableString(imageURL), nullableString(submittedBy),
	)
	if err != nil {
		return nil, err
	}

	return s.GetArtistByID(ctx, id)
}

// GetArtistByID retrieves a single artist by ID.
func (s *ArtistService) GetArtistByID(ctx context.Context, artistID string) (*models.Artist, error) {
	var artist models.Artist
	var ArtistBio, imageURL, submittedBy *string

	err := s.db.QueryRow(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, created_at
		 FROM artists WHERE id = $1`,
		artistID,
	).Scan(&artist.ID, &artist.Name, &ArtistBio, &imageURL, &submittedBy, &artist.CreatedAt)
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

	return &artist, nil
}

// GetAllArtists retrieves all artists.
func (s *ArtistService) GetAllArtists(ctx context.Context) ([]models.Artist, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, created_at
		 FROM artists
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var artist models.Artist
		var ArtistBio, imageURL, submittedBy *string

		if err := rows.Scan(
			&artist.ID, &artist.Name, &ArtistBio, &imageURL, &submittedBy, &artist.CreatedAt,
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

		artists = append(artists, artist)
	}

	if artists == nil {
		artists = []models.Artist{}
	}

	return artists, nil
}
