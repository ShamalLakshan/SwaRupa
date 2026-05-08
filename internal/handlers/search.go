package handlers

import (
	"net/http"
	"strconv"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
)

// SearchArtists handles GET /search/artists?q=query&page=1&limit=20
// Performs fuzzy search on artist names using trigram similarity (pg_trgm).
// Supports pagination through query parameters.
//
// Query Parameters:
//   - q: Required search query string (artist name or partial name)
//   - page: Optional page number (default: 1)
//   - limit: Optional results per page (default: 20, max: 100)
//
// Response:
// - 200 OK: Returns paginated search results with metadata
// - 400 Bad Request: Missing required 'q' parameter
// - 500 Internal Server Error: Database or service error
func SearchArtists(artistService *services.ArtistService) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "search query 'q' is required"})
			return
		}

		// Parse pagination parameters from query string
		page := 1
		limit := 20
		if p := c.Query("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				page = parsed
			}
		}
		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil {
				limit = parsed
			}
		}

		// Perform the search
		artists, total, err := artistService.SearchArtistsByName(c.Request.Context(), query, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
			return
		}

		// Ensure non-nil empty slice
		if artists == nil {
			artists = []models.Artist{}
		}

		// Build paginated response
		page, limit = models.ValidatePaginationParams(page, limit)
		totalPages := models.CalculateTotalPages(total, limit)

		response := models.PaginatedResponse{
			Data:       artists,
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}

// SearchAlbums handles GET /search/albums?q=query&page=1&limit=20
// Performs fuzzy search on album titles using trigram similarity (pg_trgm).
// Includes associated artists for each album and supports pagination.
//
// Query Parameters:
//   - q: Required search query string (album title or partial title)
//   - page: Optional page number (default: 1)
//   - limit: Optional results per page (default: 20, max: 100)
//
// Response:
// - 200 OK: Returns paginated search results with metadata and associated artists
// - 400 Bad Request: Missing required 'q' parameter
// - 500 Internal Server Error: Database or service error
func SearchAlbums(albumService *services.AlbumService) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "search query 'q' is required"})
			return
		}

		// Parse pagination parameters from query string
		page := 1
		limit := 20
		if p := c.Query("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				page = parsed
			}
		}
		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil {
				limit = parsed
			}
		}

		// Perform the search
		albums, total, err := albumService.SearchAlbumsByName(c.Request.Context(), query, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
			return
		}

		// Ensure non-nil empty slice
		if albums == nil {
			albums = []models.Album{}
		}

		// Build paginated response
		page, limit = models.ValidatePaginationParams(page, limit)
		totalPages := models.CalculateTotalPages(total, limit)

		response := models.PaginatedResponse{
			Data:       albums,
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}
