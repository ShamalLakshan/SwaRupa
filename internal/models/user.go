// Package models defines the core data structures for the SwaRupa music metadata API.
// These types represent domain entities persisted in PostgreSQL and transmitted via JSON in HTTP responses.
package models

import "time"

// User represents an authenticated user or contributor in the system.
// Each user has an authentication ID (typically from Firebase or Supabase Auth),
// an optional display name, and a role that determines their permissions in the system.
// Users can submit metadata about artists, albums, and artwork for community-driven curation.
type User struct {
	// ID is the unique identifier for the user, typically assigned by the authentication provider
	// (Firebase UID, Supabase Auth UUID, etc.). This field is required and used as the primary key.
	ID string `json:"id"`

	// DisplayName is the optional human-readable name of the user. This field may be displayed
	// in the user interface or API responses to identify who submitted a particular contribution.
	// If not provided, the client may choose to display the ID or a generic identifier.
	DisplayName string `json:"display_name,omitempty"`

	// Role represents the authorization level of the user within the system.
	// Common roles include "contributor" (can submit metadata), "moderator" (can review submissions),
	// and "admin" (has full system access). This field is required and defaults to "contributor" upon creation.
	Role string `json:"role"`

	// CreatedAt records the exact timestamp when the user account was first created in the system.
	// This is a server-generated field set automatically upon user creation and is immutable.
	CreatedAt time.Time `json:"created_at"`
}
