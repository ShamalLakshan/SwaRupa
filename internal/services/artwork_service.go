package services

import (
	"context"

	"github.com/ShamalLakshan/SwaRupa/internal/database"
	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ArtworkService provides business logic for artwork and artwork source operations.
type ArtworkService struct {
	db *pgxpool.Pool
}

// NewArtworkService creates a new artwork service instance.
func NewArtworkService(db *pgxpool.Pool) *ArtworkService {
	return &ArtworkService{db: db}
}

// CreateArtworkWithSource creates a new artwork and its initial source in a transaction.
// The source is automatically marked as primary and discovered by "system".
func (s *ArtworkService) CreateArtworkWithSource(ctx context.Context, albumID, imageURL, sourceName string, sourcePageURL string, isOfficial bool, submittedBy string) (*models.Artwork, error) {
	// Set default confidence/quality scores based on source type
	sourceType := "storefront"
	confidenceScore := 0.7
	qualityScore := 0.7

	if sourceName == "official" {
		sourceType = "official"
		confidenceScore = 1.0
		qualityScore = 1.0
	} else if sourceName == "community" {
		sourceType = "community"
		confidenceScore = 0.5
		qualityScore = 0.5
	}

	// Insert artwork
	artworkID, err := database.InsertArtwork(ctx, s.db, albumID, "", imageURL, "", submittedBy, isOfficial)
	if err != nil {
		return nil, err
	}

	// Insert initial source as primary
	_, err = database.InsertArtworkSource(ctx, s.db, artworkID, sourceName, sourcePageURL, imageURL, sourceType, "system", confidenceScore, qualityScore, true)
	if err != nil {
		return nil, err
	}

	// Fetch and return complete artwork with sources
	return s.GetArtworkByID(ctx, artworkID)
}

// GetArtworkByID retrieves a single artwork with all its sources.
func (s *ArtworkService) GetArtworkByID(ctx context.Context, artworkID string) (*models.Artwork, error) {
	return database.GetArtworkByIDWithSources(ctx, s.db, artworkID)
}

// GetArtworksByAlbum retrieves all artworks for an album with optional filtering.
func (s *ArtworkService) GetArtworksByAlbum(ctx context.Context, albumID, status string, onlyOfficial bool, sortByPriority bool) ([]models.Artwork, error) {
	return database.GetArtworksByAlbumIDWithSources(ctx, s.db, albumID, status, onlyOfficial, sortByPriority)
}

// GetAllArtworks retrieves all artworks across all albums with optional filtering.
func (s *ArtworkService) GetAllArtworks(ctx context.Context, status string, onlyOfficial bool, sortByPriority bool) ([]models.Artwork, error) {
	return database.GetAllArtworksWithSources(ctx, s.db, status, onlyOfficial, sortByPriority)
}

// ListArtworkSources retrieves all sources for a specific artwork.
func (s *ArtworkService) ListArtworkSources(ctx context.Context, artworkID string) ([]models.ArtworkSource, error) {
	return database.GetArtworkSourcesByArtworkID(ctx, s.db, artworkID)
}

// AddSource adds a new source to an existing artwork.
func (s *ArtworkService) AddSource(ctx context.Context, artworkID, sourceName, sourcePageURL, imageURL string) (*models.ArtworkSource, error) {
	// Determine source type and score based on source name
	sourceType := "storefront"
	confidenceScore := 0.7
	qualityScore := 0.7

	if sourceName == "official" {
		sourceType = "official"
		confidenceScore = 1.0
		qualityScore = 1.0
	} else if sourceName == "community" {
		sourceType = "community"
		confidenceScore = 0.5
		qualityScore = 0.5
	}

	// Insert the new source (not primary by default)
	sourceID, err := database.InsertArtworkSource(ctx, s.db, artworkID, sourceName, sourcePageURL, imageURL, sourceType, "", confidenceScore, qualityScore, false)
	if err != nil {
		return nil, err
	}

	return database.GetArtworkSourceByID(ctx, s.db, sourceID)
}

// UpdateSourceScore updates the confidence and quality scores for a source.
func (s *ArtworkService) UpdateSourceScore(ctx context.Context, sourceID string, confidenceScore, qualityScore float64) error {
	return database.UpdateArtworkSourceScores(ctx, s.db, sourceID, confidenceScore, qualityScore)
}

// SetPrimarySource sets a source as the primary/canonical source for an artwork.
func (s *ArtworkService) SetPrimarySource(ctx context.Context, artworkID, sourceID string) error {
	return database.UpdateArtworkSourcePrimary(ctx, s.db, artworkID, sourceID)
}

// DeleteSource removes a source from an artwork.
func (s *ArtworkService) DeleteSource(ctx context.Context, sourceID string) error {
	return database.DeleteArtworkSource(ctx, s.db, sourceID)
}

// GetSourceByID retrieves a single source by ID.
func (s *ArtworkService) GetSourceByID(ctx context.Context, sourceID string) (*models.ArtworkSource, error) {
	return database.GetArtworkSourceByID(ctx, s.db, sourceID)
}

// ApproveArtwork updates artwork approval status to "approved".
func (s *ArtworkService) ApproveArtwork(ctx context.Context, artworkID string) error {
	return s.UpdateApprovalStatus(ctx, artworkID, "approved")
}

// RejectArtwork updates artwork approval status to "rejected".
func (s *ArtworkService) RejectArtwork(ctx context.Context, artworkID string) error {
	return s.UpdateApprovalStatus(ctx, artworkID, "rejected")
}

// UpdateApprovalStatus updates artwork approval status.
func (s *ArtworkService) UpdateApprovalStatus(ctx context.Context, artworkID, status string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE artworks SET approval_status = $1 WHERE id = $2`,
		status, artworkID,
	)
	return err
}
