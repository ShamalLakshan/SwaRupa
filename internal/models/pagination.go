package models

// PaginationParams holds common pagination parameters.
type PaginationParams struct {
	Page  int
	Limit int
}

// PaginatedResponse wraps paginated data with metadata.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
}

// ValidatePaginationParams validates and normalizes pagination parameters.
// Returns default values (page=1, limit=20) if invalid parameters are provided.
// Enforces minimum limit=1 and maximum limit=100 to prevent abuse.
func ValidatePaginationParams(page, limit int) (int, int) {
	// Default page to 1 if invalid
	if page < 1 {
		page = 1
	}

	// Default limit to 20 if invalid
	if limit < 1 {
		limit = 20
	}

	// Cap limit at 100 to prevent excessive database load
	if limit > 100 {
		limit = 100
	}

	return page, limit
}

// CalculateOffset converts page and limit to SQL OFFSET.
// Offset = (page - 1) * limit
func CalculateOffset(page, limit int) int {
	return (page - 1) * limit
}

// CalculateTotalPages computes total pages given total count and limit.
func CalculateTotalPages(total int64, limit int) int {
	if total == 0 {
		return 1
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
