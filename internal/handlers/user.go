// Package handlers implements HTTP request handlers for the SwaRupa music metadata API.
// Each handler receives a PostgreSQL connection pool, constructs appropriate SQL queries,
// parses JSON request bodies, and returns structured JSON responses with proper HTTP status codes.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateUser handles POST /api/users requests to create a new user account or register an existing authentication ID.
// The handler expects a JSON payload with the user's authentication ID (from Firebase/Supabase) and optional display name.
//
// Request body structure:
//
//	{
//	  "id": "auth-provider-uid",
//	  "display_name": "User Display Name" (optional)
//	}
//
// SQL Operation:
// The handler executes INSERT INTO users (id, display_name, role) VALUES ($1, $2, 'contributor')
// with ON CONFLICT (id) DO NOTHING to ensure idempotency. If a user with the given ID already exists,
// the INSERT is silently ignored (no error raised), allowing safe retry semantics.
// The role is hardcoded to 'contributor' for new users, indicating basic submission permissions.
//
// Response:
// - 201 Created: User successfully created or already exists; returns User model with timestamp
// - 400 Bad Request: Missing required 'id' field in JSON body
// - 500 Internal Server Error: Database error (connection, constraints, etc.)
func CreateUser(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Define a struct to capture JSON request body fields.
		// The `binding:"required"` tags enforce validation; missing fields trigger automatic 400 response.
		// The `json:"id"` tags map JSON field names to struct fields, enabling unmarshaling.
		var req struct {
			ID          string `json:"id"           binding:"required"`
			DisplayName string `json:"display_name"`
		}

		// ShouldBindJSON unmarshals the HTTP request body into the req struct.
		// It automatically validates binding tags and returns an error if validation fails.
		// The error check here is defensive; binding tags already trigger a response,
		// but explicit error handling is a best practice for clarity.
		if err := c.ShouldBindJSON(&req); err != nil {
			// Return HTTP 400 Bad Request with JSON error details when validation fails.
			// Using gin.H (a map shorthand) allows flexible JSON response construction.
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}

		// Execute PostgreSQL INSERT with ON CONFLICT clause for idempotency.
		// This prevents duplicate key errors if the same user ID is submitted multiple times.
		// The query uses parameterized arguments ($1, $2, $3) to prevent SQL injection attacks.
		// context.Background() provides a non-cancellable context; in production, consider request-scoped contexts.
		_, err := db.Exec(
			context.Background(),
			`INSERT INTO users (id, display_name, role)
			 VALUES ($1, $2, 'contributor')
			 ON CONFLICT (id) DO NOTHING`,
			req.ID, nullableString(req.DisplayName),
		)
		if err != nil {
			// Database errors (connection failures, constraint violations, etc.) return 500.
			// In production, consider logging the full error for debugging while returning generic message to clients.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		// Return HTTP 201 Created with the new user record as JSON.
		// time.Now() generates the server-side timestamp; this differs from database-generated
		// created_at values, so in production, fetch the record from the database to ensure consistency.
		c.JSON(http.StatusCreated, models.User{
			ID:          req.ID,
			DisplayName: req.DisplayName,
			Role:        "contributor",
			CreatedAt:   time.Now(),
		})
	}
}

// GetUser handles GET /api/users/:id requests to retrieve a user record by their authentication ID.
// The :id path parameter is the user's unique identifier from the authentication provider.
//
// SQL Operation:
// Executes SELECT id, display_name, role, created_at FROM users WHERE id = $1
// to fetch the user record. The query uses indexed lookup (primary key) for efficient retrieval.
// Nullable columns (display_name) are scanned into pointer types; if NULL in the database,
// the pointer is nil and omitted from the response JSON.
//
// Response:
// - 200 OK: User found; returns complete User model with all fields
// - 404 Not Found: No user with the given ID exists in the database
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetUser(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the :id path parameter from the URL.
		// Gin's Param() method retrieves named parameters defined in route registration.
		id := c.Param("id")

		// Create a User model instance to hold the scanned database values.
		var user models.User
		// Declare a pointer to hold nullable database values (SQL NULL -> nil).
		// This pattern is necessary because Go strings cannot represent database NULL;
		// pointers can (nil = NULL, &value = actual value).
		var displayName *string

		// QueryRow fetches a single row matching the WHERE condition.
		// This is more efficient than Query() which returns a rows iterator.
		// The $1 placeholder is parameterized to prevent SQL injection.
		err := db.QueryRow(
			context.Background(),
			`SELECT id, display_name, role, created_at FROM users WHERE id = $1`,
			id,
		// Scan() unpacks the query result into destination variables in column order.
		// If the row doesn't exist, Scan() returns pgx.ErrNoRows, converted to 404 response below.
		).Scan(&user.ID, &displayName, &user.Role, &user.CreatedAt)
		if err != nil {
			// QueryRow not finding a match is returned as an error; 404 is the correct HTTP response.
			// In production, distinguish between ErrNoRows (404) and other errors (500) for proper debugging.
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Dereference the nullable displayName pointer and assign to the model.
		// If displayName is nil (NULL in database), the omitempty JSON tag excludes it from the response.
		// This ensures the JSON response accurately reflects which fields were stored in the database.
		if displayName != nil {
			user.DisplayName = *displayName
		}

		// Return HTTP 200 OK with the complete user record serialized to JSON.
		// Gin automatically calls json.Marshal() based on the second argument's type.
		c.JSON(http.StatusOK, user)
	}
}

// GetAllUsers handles GET /api/users requests to retrieve all user records from the database.
// This endpoint is useful for admin dashboards, user directories, or permission management interfaces.
// Includes all users regardless of role, ordered by creation timestamp to show newest users first.
//
// SQL Operation:
// Executes SELECT id, display_name, role, created_at FROM users
// to retrieve all user records. Results are ordered by created_at DESC to show newest users first.
// Nullable columns (display_name) are scanned into pointer types (*string) and omitted from JSON
// responses if NULL per the struct tag annotations.
//
// Response:
// - 200 OK: Query successful; returns array of User models (empty array if no users exist)
// - 500 Internal Server Error: Database error (connection, query failure, etc.)
func GetAllUsers(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute multi-row SELECT query to retrieve all users, ordered by creation timestamp descending.
		// Query() returns a Rows iterator for variable-length result sets.
		// The iterator must be explicitly closed to release database resources.
		rows, err := db.Query(
			context.Background(),
			`SELECT id, display_name, role, created_at FROM users ORDER BY created_at DESC`,
		)
		if err != nil {
			// Query errors indicate database connectivity or syntax issues (this should not happen in production).
			// Return 500 and log the error for operational troubleshooting.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query users"})
			return
		}
		// Defer rows.Close() to ensure the result iterator is properly cleaned up.
		// Failure to close rows can leak server-side cursor resources; in high-traffic systems,
		// this can eventually exhaust the connection pool and cause cascading failures.
		defer rows.Close()

		// Initialize a slice to hold all user records from the query results.
		// Using make with zero length and capacity allows dynamic growth via append().
		var users []models.User

		// Iterate over all result rows using rows.Next().
		// Next() returns false when all rows have been consumed or an error occurs.
		for rows.Next() {
			var user models.User
			var displayName *string

			// Scan the current row's values into Go variables.
			// Pointers like displayName allow NULL values to be represented as nil;
			// scalar fields receive zero values if NULL (Go's default behavior).
			err := rows.Scan(
				&user.ID,
				&displayName,
				&user.Role,
				&user.CreatedAt,
			)
			if err != nil {
				// Scan errors indicate corrupted data or type mismatches and trigger a 500 response.
				// In production, log this error with row information for debugging.
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan user row"})
				return
			}

			// Dereference the nullable displayName pointer and assign to the model.
			// This pattern distinguishes between unset fields (NULL -> nil) and default values.
			// Consistent with GetUser for API consistency.
			if displayName != nil {
				user.DisplayName = *displayName
			}

			// Append the populated user to the result slice.
			// Go's slice append operation automatically handles growth and reallocation.
			users = append(users, user)
		}

		// Check rows iterator error status after the loop completes.
		// This catches any errors that occurred during iteration but were not raised in Next().
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error iterating users"})
			return
		}

		// Ensure non-nil JSON output: initialize to empty slice if no users exist.
		// This provides a consistent API contract; empty results are always [] in JSON.
		if users == nil {
			users = []models.User{}
		}

		// Marshal the users slice to JSON and return HTTP 200 OK.
		// Gin's JSON() method handles encoding; Content-Type is automatically set to application/json.
		c.JSON(http.StatusOK, users)
	}
}

