package models

import "time"

// Album represents a music album or compilation in the system.
// Albums are the core entity around which all metadata (artists, artwork) is organized.
// Each album can be associated with one or more Artist records through a many-to-many
// relationship table (album_artists). Albums are uniquely identified by UUIDs and
// immutably linked to their creation timestamps.
type Album struct {
	// ID is a UUID-based unique identifier for the album record, serving as the primary key
	// in the albums table. Generated server-side using RFC 4122 UUID v4 upon album creation.
	ID string `json:"id"`

	// Title is the required name or title of the album. This field must not be empty.
	// It represents the official or commonly-used album name as released or cataloged.
	Title string `json:"title"`

	// ReleaseYear is an optional integer representing the year the album was released.
	// Stored as a 4-digit year (e.g., 2023). If not provided, the field may be omitted in responses.
	ReleaseYear int `json:"release_year,omitempty"`

	// SubmittedBy is an optional reference to the User ID of the person who submitted this album record.
	// Provides attribution and audit trail for community contributions to the database.
	SubmittedBy string `json:"submitted_by,omitempty"`

	// CreatedAt records the UTC timestamp when the album record was first created in the database.
	// This is a server-generated, immutable field set automatically upon record creation.
	CreatedAt time.Time `json:"created_at"`

	// Artists is a slice of Artist records associated with this album.
	// This field is populated via an INNER JOIN on the album_artists junction table
	// during album retrieval and represents all artists credited on the album.
	// The slice is omitted from JSON responses if empty.
	Artists []Artist `json:"artists,omitempty"`
}
