package handlers

import (
	"context"
	"net/http"

	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
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
// Service Operation:
// The handler calls UserService.CreateUser() which executes INSERT INTO users (id, display_name, role)
// VALUES ($1, $2, 'contributor') with ON CONFLICT (id) DO NOTHING to ensure idempotency.
// If a user with the given ID already exists, the operation is idempotent and returns the existing user.
// The role is hardcoded to 'contributor' for new users, indicating basic submission permissions.
//
// Response:
// - 201 Created: User successfully created or already exists; returns User model with timestamp
// - 400 Bad Request: Missing required 'id' field in JSON body
// - 500 Internal Server Error: Database error (connection, constraints, etc.)
func CreateUser(svc *services.UserService) gin.HandlerFunc {
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

		// Call the UserService to create or retrieve the user.
		// The service handles database operations and returns a complete User model with server-generated timestamp.
		// This ensures consistency with database-generated timestamps rather than client-side values.
		user, err := svc.CreateUser(context.Background(), req.ID, req.DisplayName)
		if err != nil {
			// Database errors (connection failures, constraint violations, etc.) return 500.
			// In production, consider logging the full error for debugging while returning generic message to clients.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		// Return HTTP 201 Created with the new user record as JSON.
		// The User model is returned from the service with all fields properly populated including timestamps.
		c.JSON(http.StatusCreated, user)
	}
}

// GetUser handles GET /api/users/:id requests to retrieve a user record by their authentication ID.
// The :id path parameter is the user's unique identifier from the authentication provider.
//
// Service Operation:
// Calls UserService.GetUserByID() which executes SELECT id, display_name, role, created_at FROM users WHERE id = $1
// to fetch the user record. The query uses indexed lookup (primary key) for efficient retrieval.
//
// Response:
// - 200 OK: User found; returns complete User model with all fields
// - 404 Not Found: No user with the given ID exists in the database
// - 500 Internal Server Error: Database or service error (connection, query failure, etc.)
func GetUser(svc *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the :id path parameter from the URL.
		// Gin's Param() method retrieves named parameters defined in route registration.
		id := c.Param("id")

		// Call the UserService to retrieve the user by ID.
		// The service handles database queries and returns the User model or an error.
		user, err := svc.GetUserByID(context.Background(), id)
		if err != nil {
			// QueryRow not finding a match is returned as an error; 404 is the correct HTTP response.
			// In production, distinguish between ErrNoRows (404) and other errors (500) for proper debugging.
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
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
// Service Operation:
// Calls UserService.GetAllUsers() which executes SELECT id, display_name, role, created_at FROM users
// to retrieve all user records. Results are ordered by created_at DESC to show newest users first.
//
// Response:
// - 200 OK: Query successful; returns array of User models (empty array if no users exist)
// - 500 Internal Server Error: Database or service error (connection, query failure, etc.)
func GetAllUsers(svc *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call the UserService to retrieve all users.
		// The service handles database queries and returns a slice of User models or an error.
		users, err := svc.GetAllUsers(context.Background())
		if err != nil {
			// Query errors indicate database connectivity or syntax issues (this should not happen in production).
			// Return 500 and log the error for operational troubleshooting.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query users"})
			return
		}

		// Marshal the users slice to JSON and return HTTP 200 OK.
		// Gin's JSON() method handles encoding; Content-Type is automatically set to application/json.
		c.JSON(http.StatusOK, users)
	}
}
