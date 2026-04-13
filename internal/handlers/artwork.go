package handlers

import (
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
)

// CreateArtwork returns a handler for POST /api/albums/:id/artworks
// Creates a new artwork for an album with an initial source
func CreateArtwork(artworkService *services.ArtworkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		albumID := c.Param("id")

		var req struct {
			SourceName       string  `json:"source_name" binding:"required"`
			SourcePage       string  `json:"source_page"`
			ImageURL         string  `json:"image_url" binding:"required"`
			SourceType       string  `json:"source_type"`
			ConfidenceScore  float64 `json:"confidence_score"`
			QualityScore     float64 `json:"quality_score"`
			DiscoveredBy     string  `json:"discovered_by"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Determine isOfficial based on source type
		isOfficial := req.SourceType == "official"

		artwork, err := artworkService.CreateArtworkWithSource(
			c.Request.Context(),
			albumID,
			req.ImageURL,
			req.SourceName,
			req.SourcePage,
			isOfficial,
			req.DiscoveredBy,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, artwork)
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
// - 500 Internal Server Error: Service or database error
func GetArtworksByAlbum(artworkService *services.ArtworkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the album ID from the URL path.
		albumID := c.Param("id")

		// Parse query parameters for filtering and sorting
		status := c.Query("status")
		onlyOfficial := c.Query("official") == "true"
		sortByPriority := c.Query("sort") == "priority"

		// Call service method to get artworks with based on filters
		artworks, err := artworkService.GetArtworksByAlbum(c.Request.Context(), albumID, status, onlyOfficial, sortByPriority)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artworks"})
			return
		}

		// Ensure the response is a non-nil empty array rather than nil if no artworks match.
		// JSON convention: null implies missing/unset, [] implies empty collection.
		if artworks == nil {
			artworks = []models.Artwork{}
		}

		// Marshal the artworks slice to JSON and return HTTP 200 OK.
		c.JSON(http.StatusOK, artworks)
	}
}

// GetAllArtworks handles GET /api/artworks requests to retrieve all artwork records from the database.
// This endpoint returns all artwork across all albums with optional filtering and sorting.
// Useful for administrative interfaces, approval dashboards, and comprehensive metadata audits.
//
// Supported query parameters:
//   - status: Filter by approval_status ("pending", "approved", or "rejected")
//   - official: If "true", return only official artwork (is_official = true)
//   - sort: If "priority", sort by priority_score DESC; otherwise sort by created_at DESC
//
// SQL Operations:
// Base query: SELECT id, album_id, source_id, image_url, thumbnail_url,
//
//	      is_official, submitted_by, approval_status, priority_score, created_at
//	FROM artworks
//
// Dynamic WHERE clauses are appended based on query parameters:
// - status filter: WHERE approval_status = $1 (parameterized to prevent SQL injection)
// - official filter: WHERE is_official = true
//
// Sorting:
// - If sort=priority: ORDER BY priority_score DESC (highest priority first)
// - Otherwise: ORDER BY created_at DESC (newest first)
//
// Nullable columns are scanned into pointer types; NULL values are omitted from JSON responses.
// If no artworks match the query criteria, returns an empty array (never null).
//
// Response:
// - 200 OK: Returns artworks array (may be empty if no matches or no artworks exist)
// - 500 Internal Server Error: Service or database error
func GetAllArtworks(artworkService *services.ArtworkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse query parameters for filtering and sorting
		status := c.Query("status")
		onlyOfficial := c.Query("official") == "true"
		sortByPriority := c.Query("sort") == "priority"

		// Call service method to get all artworks with filters
		artworks, err := artworkService.GetAllArtworks(c.Request.Context(), status, onlyOfficial, sortByPriority)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artworks"})
			return
		}

		// Ensure the response is a non-nil empty array rather than nil if no artworks exist or match.
		// JSON convention: null implies missing/unset, [] implies empty collection.
		if artworks == nil {
			artworks = []models.Artwork{}
		}

		// Marshal the artworks slice to JSON and return HTTP 200 OK.
		c.JSON(http.StatusOK, artworks)
	}
}
