package database

import (
	"context"
	"fmt"

	// "time"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetArtworkByIDWithSources retrieves a single artwork by ID with all its sources.
// Returns the artwork with nested sources array, or error if not found.
func GetArtworkByIDWithSources(ctx context.Context, db *pgxpool.Pool, artworkID string) (*models.Artwork, error) {
	var aw models.Artwork
	var sourceID, thumbnailURL, submittedBy, canonicalImageURL *string

	err := db.QueryRow(ctx,
		`SELECT id, album_id, source_id, image_url, thumbnail_url, is_official,
		        submitted_by, approval_status, priority_score, created_at, canonical_image_url
		 FROM artworks
		 WHERE id = $1`,
		artworkID,
	).Scan(
		&aw.ID, &aw.AlbumID, &sourceID, &aw.ImageURL, &thumbnailURL, &aw.IsOfficial,
		&submittedBy, &aw.ApprovalStatus, &aw.PriorityScore, &aw.CreatedAt, &canonicalImageURL,
	)
	if err != nil {
		return nil, err
	}

	// Dereference nullable pointers
	if sourceID != nil {
		aw.SourceID = *sourceID
	}
	if thumbnailURL != nil {
		aw.ThumbnailURL = *thumbnailURL
	}
	if submittedBy != nil {
		aw.SubmittedBy = *submittedBy
	}

	// Fetch all sources for this artwork
	sources, err := GetArtworkSourcesByArtworkID(ctx, db, artworkID)
	if err == nil && sources != nil {
		aw.Sources = sources
	} else if sources == nil {
		aw.Sources = []models.ArtworkSource{} // Ensure non-nil empty slice
	}

	return &aw, nil
}

// GetArtworksByAlbumIDWithSources retrieves all artworks for an album with their sources.
// Supports filtering by status and official flag, and sorting.
func GetArtworksByAlbumIDWithSources(ctx context.Context, db *pgxpool.Pool, albumID, status string, onlyOfficial bool, sortByPriority bool) ([]models.Artwork, error) {
	query := `SELECT id, album_id, source_id, image_url, thumbnail_url, is_official,
	                 submitted_by, approval_status, priority_score, created_at, canonical_image_url
	          FROM artworks
	          WHERE album_id = $1`
	args := []any{albumID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(" AND approval_status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if onlyOfficial {
		query += " AND is_official = true"
	}

	if sortByPriority {
		query += " ORDER BY priority_score DESC"
	} else {
		query += " ORDER BY created_at DESC"
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artworks []models.Artwork
	for rows.Next() {
		var aw models.Artwork
		var sourceID, thumbnailURL, submittedBy, canonicalImageURL *string

		if err := rows.Scan(
			&aw.ID, &aw.AlbumID, &sourceID, &aw.ImageURL, &thumbnailURL, &aw.IsOfficial,
			&submittedBy, &aw.ApprovalStatus, &aw.PriorityScore, &aw.CreatedAt, &canonicalImageURL,
		); err != nil {
			continue
		}

		if sourceID != nil {
			aw.SourceID = *sourceID
		}
		if thumbnailURL != nil {
			aw.ThumbnailURL = *thumbnailURL
		}
		if submittedBy != nil {
			aw.SubmittedBy = *submittedBy
		}

		// Fetch sources for this artwork
		sources, err := GetArtworkSourcesByArtworkID(ctx, db, aw.ID)
		if err == nil && sources != nil {
			aw.Sources = sources
		} else if sources == nil {
			aw.Sources = []models.ArtworkSource{}
		}

		artworks = append(artworks, aw)
	}

	if artworks == nil {
		artworks = []models.Artwork{}
	}

	return artworks, nil
}

// GetAllArtworksWithSources retrieves all artworks across all albums with their sources.
// Supports filtering by status and official flag, and sorting.
func GetAllArtworksWithSources(ctx context.Context, db *pgxpool.Pool, status string, onlyOfficial bool, sortByPriority bool) ([]models.Artwork, error) {
	query := `SELECT id, album_id, source_id, image_url, thumbnail_url, is_official,
	                 submitted_by, approval_status, priority_score, created_at, canonical_image_url
	          FROM artworks
	          WHERE 1=1`
	args := []any{}
	argIdx := 1

	if status != "" {
		query += fmt.Sprintf(" AND approval_status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if onlyOfficial {
		query += " AND is_official = true"
	}

	if sortByPriority {
		query += " ORDER BY priority_score DESC"
	} else {
		query += " ORDER BY created_at DESC"
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artworks []models.Artwork
	for rows.Next() {
		var aw models.Artwork
		var sourceID, thumbnailURL, submittedBy, canonicalImageURL *string

		if err := rows.Scan(
			&aw.ID, &aw.AlbumID, &sourceID, &aw.ImageURL, &thumbnailURL, &aw.IsOfficial,
			&submittedBy, &aw.ApprovalStatus, &aw.PriorityScore, &aw.CreatedAt, &canonicalImageURL,
		); err != nil {
			continue
		}

		if sourceID != nil {
			aw.SourceID = *sourceID
		}
		if thumbnailURL != nil {
			aw.ThumbnailURL = *thumbnailURL
		}
		if submittedBy != nil {
			aw.SubmittedBy = *submittedBy
		}

		// Fetch sources for this artwork
		sources, err := GetArtworkSourcesByArtworkID(ctx, db, aw.ID)
		if err == nil && sources != nil {
			aw.Sources = sources
		} else if sources == nil {
			aw.Sources = []models.ArtworkSource{}
		}

		artworks = append(artworks, aw)
	}

	if artworks == nil {
		artworks = []models.Artwork{}
	}

	return artworks, nil
}

// InsertArtwork inserts a new artwork record and returns the ID.
func InsertArtwork(ctx context.Context, db *pgxpool.Pool, albumID, sourceID, imageURL, thumbnailURL, submittedBy string, isOfficial bool) (string, error) {
	var id string
	err := db.QueryRow(ctx,
		`INSERT INTO artworks (id, album_id, source_id, image_url, thumbnail_url, is_official, submitted_by, approval_status, priority_score, created_at)
		 VALUES (gen_random_uuid()::text, $1, $2, $3, $4, $5, $6, 'pending', 0, now())
		 RETURNING id`,
		albumID,
		nullableString(sourceID),
		imageURL,
		nullableString(thumbnailURL),
		isOfficial,
		nullableString(submittedBy),
	).Scan(&id)
	return id, err
}

// GetArtworkSourcesByArtworkID retrieves all sources for a given artwork ID.
func GetArtworkSourcesByArtworkID(ctx context.Context, db *pgxpool.Pool, artworkID string) ([]models.ArtworkSource, error) {
	rows, err := db.Query(ctx,
		`SELECT id, artwork_id, source_name, source_page, image_url, source_type,
		        confidence_score, quality_score, is_primary, discovered_by, created_at
		 FROM artwork_sources
		 WHERE artwork_id = $1
		 ORDER BY is_primary DESC, quality_score DESC`,
		artworkID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []models.ArtworkSource
	for rows.Next() {
		var s models.ArtworkSource
		var sourcePage, discoveredBy *string

		if err := rows.Scan(
			&s.ID, &s.ArtworkID, &s.SourceName, &sourcePage, &s.ImageURL, &s.SourceType,
			&s.ConfidenceScore, &s.QualityScore, &s.IsPrimary, &discoveredBy, &s.CreatedAt,
		); err != nil {
			continue
		}

		if sourcePage != nil {
			s.SourcePage = *sourcePage
		}
		if discoveredBy != nil {
			s.DiscoveredBy = *discoveredBy
		}

		sources = append(sources, s)
	}

	if sources == nil {
		sources = []models.ArtworkSource{}
	}

	return sources, nil
}

// InsertArtworkSource inserts a new artwork source record.
func InsertArtworkSource(ctx context.Context, db *pgxpool.Pool, artworkID, sourceName, sourcePageURL, imageURL, sourceType, discoveredBy string, confidenceScore, qualityScore float64, isPrimary bool) (string, error) {
	var id string
	err := db.QueryRow(ctx,
		`INSERT INTO artwork_sources (id, artwork_id, source_name, source_page, image_url, source_type, confidence_score, quality_score, is_primary, discovered_by, created_at)
		 VALUES (gen_random_uuid()::text, $1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		 RETURNING id`,
		artworkID,
		sourceName,
		nullableString(sourcePageURL),
		imageURL,
		sourceType,
		confidenceScore,
		qualityScore,
		isPrimary,
		nullableString(discoveredBy),
	).Scan(&id)
	return id, err
}

// UpdateArtworkSourceScores updates the confidence and quality scores of a source.
func UpdateArtworkSourceScores(ctx context.Context, db *pgxpool.Pool, sourceID string, confidenceScore, qualityScore float64) error {
	_, err := db.Exec(ctx,
		`UPDATE artwork_sources SET confidence_score = $1, quality_score = $2 WHERE id = $3`,
		confidenceScore, qualityScore, sourceID,
	)
	return err
}

// UpdateArtworkSourcePrimary sets which source is primary for an artwork.
func UpdateArtworkSourcePrimary(ctx context.Context, db *pgxpool.Pool, artworkID, sourceID string) error {
	// First, unset all other sources as primary
	_, err := db.Exec(ctx,
		`UPDATE artwork_sources SET is_primary = false WHERE artwork_id = $1`,
		artworkID,
	)
	if err != nil {
		return err
	}

	// Then set the specified source as primary
	_, err = db.Exec(ctx,
		`UPDATE artwork_sources SET is_primary = true WHERE id = $1`,
		sourceID,
	)
	return err
}

// DeleteArtworkSource deletes an artwork source by ID.
func DeleteArtworkSource(ctx context.Context, db *pgxpool.Pool, sourceID string) error {
	_, err := db.Exec(ctx, `DELETE FROM artwork_sources WHERE id = $1`, sourceID)
	return err
}

// GetArtworkSourceByID retrieves a single artwork source by ID.
func GetArtworkSourceByID(ctx context.Context, db *pgxpool.Pool, sourceID string) (*models.ArtworkSource, error) {
	var s models.ArtworkSource
	var sourcePage, discoveredBy *string

	err := db.QueryRow(ctx,
		`SELECT id, artwork_id, source_name, source_page, image_url, source_type,
		        confidence_score, quality_score, is_primary, discovered_by, created_at
		 FROM artwork_sources
		 WHERE id = $1`,
		sourceID,
	).Scan(
		&s.ID, &s.ArtworkID, &s.SourceName, &sourcePage, &s.ImageURL, &s.SourceType,
		&s.ConfidenceScore, &s.QualityScore, &s.IsPrimary, &discoveredBy, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if sourcePage != nil {
		s.SourcePage = *sourcePage
	}
	if discoveredBy != nil {
		s.DiscoveredBy = *discoveredBy
	}

	return &s, nil
}

// nullableString converts a string to a pointer, returning nil for empty strings.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
