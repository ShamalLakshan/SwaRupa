package database

import (
	"context"
	"fmt"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetArtistsByAlbumID retrieves all artists associated with an album.
func GetArtistsByAlbumID(ctx context.Context, db *pgxpool.Pool, albumID string) ([]models.Artist, error) {
	rows, err := db.Query(ctx,
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

	artists := []models.Artist{}
	for rows.Next() {
		var artist models.Artist
		var artistBio, imageURL, submittedBy *string

		err := rows.Scan(
			&artist.ID, &artist.Name, &artistBio, &imageURL, &submittedBy, &artist.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if artistBio != nil {
			artist.ArtistBio = *artistBio
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
		}

		artists = append(artists, artist)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return artists, nil
}

// SearchArtistsByName performs fuzzy search on artist names using pg_trgm trigram similarity.
// Returns paginated results sorted by similarity score (highest first).
// Requires: pg_trgm extension enabled on the database (CREATE EXTENSION IF NOT EXISTS pg_trgm)
func SearchArtistsByName(ctx context.Context, db *pgxpool.Pool, query string, limit, offset int) ([]models.Artist, int64, error) {
	// First, get the total count of matching artists
	var total int64
	countErr := db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM artists
		 WHERE name % $1 OR name ILIKE $2`,
		query, "%"+query+"%",
	).Scan(&total)
	if countErr != nil {
		return nil, 0, countErr
	}

	// Fetch paginated results sorted by similarity
	rows, err := db.Query(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, created_at
		 FROM artists
		 WHERE name % $1 OR name ILIKE $2
		 ORDER BY similarity(name, $1) DESC, created_at DESC
		 LIMIT $3 OFFSET $4`,
		query, "%"+query+"%", limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	artists := []models.Artist{}
	for rows.Next() {
		var artist models.Artist
		var artistBio, imageURL, submittedBy *string

		err := rows.Scan(
			&artist.ID, &artist.Name, &artistBio, &imageURL, &submittedBy, &artist.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if artistBio != nil {
			artist.ArtistBio = *artistBio
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
		}

		artists = append(artists, artist)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return artists, total, nil
}

// SearchAlbumsByName performs fuzzy search on album titles using pg_trgm trigram similarity.
// Returns paginated results sorted by similarity score (highest first).
// Includes associated artists for each album.
// Requires: pg_trgm extension enabled on the database (CREATE EXTENSION IF NOT EXISTS pg_trgm)
func SearchAlbumsByName(ctx context.Context, db *pgxpool.Pool, query string, limit, offset int) ([]models.Album, int64, error) {
	// First, get the total count of matching albums
	var total int64
	countErr := db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM albums
		 WHERE title % $1 OR title ILIKE $2`,
		query, "%"+query+"%",
	).Scan(&total)
	if countErr != nil {
		return nil, 0, countErr
	}

	// Fetch paginated album results sorted by similarity
	rows, err := db.Query(ctx,
		`SELECT id, title, release_year, submitted_by, created_at
		 FROM albums
		 WHERE title % $1 OR title ILIKE $2
		 ORDER BY similarity(title, $1) DESC, created_at DESC
		 LIMIT $3 OFFSET $4`,
		query, "%"+query+"%", limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	albums := []models.Album{}
	for rows.Next() {
		var album models.Album
		var submittedBy *string

		err := rows.Scan(
			&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}

		// Fetch associated artists for this album
		artists, err := GetArtistsByAlbumID(ctx, db, album.ID)
		if err == nil && artists != nil {
			album.Artists = artists
		} else if artists == nil {
			album.Artists = []models.Artist{}
		}

		albums = append(albums, album)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return albums, total, nil
}

// GetAllArtistsWithPagination retrieves all artists with pagination support.
func GetAllArtistsWithPagination(ctx context.Context, db *pgxpool.Pool, limit, offset int) ([]models.Artist, int64, error) {
	// Get total count
	var total int64
	countErr := db.QueryRow(ctx, `SELECT COUNT(*) FROM artists`).Scan(&total)
	if countErr != nil {
		return nil, 0, countErr
	}

	// Fetch paginated results
	rows, err := db.Query(ctx,
		`SELECT id, name, artist_bio, image_url, submitted_by, created_at
		 FROM artists
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	artists := []models.Artist{}
	for rows.Next() {
		var artist models.Artist
		var artistBio, imageURL, submittedBy *string

		err := rows.Scan(
			&artist.ID, &artist.Name, &artistBio, &imageURL, &submittedBy, &artist.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if artistBio != nil {
			artist.ArtistBio = *artistBio
		}
		if imageURL != nil {
			artist.ImageURL = *imageURL
		}
		if submittedBy != nil {
			artist.SubmittedBy = *submittedBy
		}

		artists = append(artists, artist)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return artists, total, nil
}

// GetAllAlbumsWithPagination retrieves all albums with pagination support and associated artists.
func GetAllAlbumsWithPagination(ctx context.Context, db *pgxpool.Pool, limit, offset int) ([]models.Album, int64, error) {
	// Get total count
	var total int64
	countErr := db.QueryRow(ctx, `SELECT COUNT(*) FROM albums`).Scan(&total)
	if countErr != nil {
		return nil, 0, countErr
	}

	// Fetch paginated album results
	rows, err := db.Query(ctx,
		`SELECT id, title, release_year, submitted_by, created_at
		 FROM albums
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	albums := []models.Album{}
	for rows.Next() {
		var album models.Album
		var submittedBy *string

		err := rows.Scan(
			&album.ID, &album.Title, &album.ReleaseYear, &submittedBy, &album.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if submittedBy != nil {
			album.SubmittedBy = *submittedBy
		}

		// Fetch associated artists for this album
		artists, err := GetArtistsByAlbumID(ctx, db, album.ID)
		if err == nil && artists != nil {
			album.Artists = artists
		} else if artists == nil {
			album.Artists = []models.Artist{}
		}

		albums = append(albums, album)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return albums, total, nil
}

// GetAllArtworksWithPagination retrieves all artworks with pagination support.
// Supports filtering by status and official flag, and sorting.
func GetAllArtworksWithPagination(ctx context.Context, db *pgxpool.Pool, status string, onlyOfficial bool, sortByPriority bool, limit, offset int) ([]models.Artwork, int64, error) {
	// Get total count
	whereClause := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if status != "" {
		whereClause += fmt.Sprintf(" AND approval_status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if onlyOfficial {
		whereClause += " AND is_official = true"
	}

	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM artworks %s`, whereClause)
	countErr := db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if countErr != nil {
		return nil, 0, countErr
	}

	orderClause := "ORDER BY created_at DESC"
	if sortByPriority {
		orderClause = "ORDER BY priority_score DESC"
	}

	query := fmt.Sprintf(
		`SELECT id, album_id, source_id, image_url, thumbnail_url, is_official,
		        submitted_by, approval_status, priority_score, created_at
		 FROM artworks
		 %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		whereClause, orderClause, argIdx, argIdx+1,
	)

	args = append(args, limit, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	artworks := []models.Artwork{}
	for rows.Next() {
		var aw models.Artwork
		var sourceID, thumbnailURL, submittedBy *string

		if err := rows.Scan(
			&aw.ID, &aw.AlbumID, &sourceID, &aw.ImageURL, &thumbnailURL, &aw.IsOfficial,
			&submittedBy, &aw.ApprovalStatus, &aw.PriorityScore, &aw.CreatedAt,
		); err != nil {
			return nil, 0, err
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

		aw.Sources = []models.ArtworkSource{}
		artworks = append(artworks, aw)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return artworks, total, nil
}

// GetArtworksByAlbumIDWithPagination retrieves paginated artworks for an album with optional filtering.
func GetArtworksByAlbumIDWithPagination(ctx context.Context, db *pgxpool.Pool, albumID, status string, onlyOfficial bool, sortByPriority bool, limit, offset int) ([]models.Artwork, int64, error) {
	// Build the WHERE clause dynamically
	whereClause := "WHERE album_id = $1"
	args := []interface{}{albumID}
	argIndex := 2

	if status != "" {
		whereClause += fmt.Sprintf(" AND approval_status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if onlyOfficial {
		whereClause += " AND is_official = true"
	}

	// Get total count
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM artworks %s`, whereClause)
	countErr := db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if countErr != nil {
		return nil, 0, countErr
	}

	// Determine sort order
	orderClause := "ORDER BY created_at DESC"
	if sortByPriority {
		orderClause = "ORDER BY priority_score DESC"
	}

	// Fetch paginated results
	query := fmt.Sprintf(
		`SELECT id, album_id, source_id, image_url, thumbnail_url, is_official,
		        submitted_by, approval_status, priority_score, created_at
		 FROM artworks
		 %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		whereClause, orderClause, argIndex, argIndex+1,
	)

	args = append(args, limit, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	artworks := []models.Artwork{}
	for rows.Next() {
		var aw models.Artwork
		var sourceID, thumbnailURL, submittedBy *string

		err := rows.Scan(
			&aw.ID, &aw.AlbumID, &sourceID, &aw.ImageURL, &thumbnailURL, &aw.IsOfficial,
			&submittedBy, &aw.ApprovalStatus, &aw.PriorityScore, &aw.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
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

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return artworks, total, nil
}
