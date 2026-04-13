package models

import "time"

// Artist represents a music artist or performer in the system.
// Artists are the primary contributors to Albums and can have associated metadata such as
// MusicBrainz identifiers, profile images, and submission provenance. Multiple artists can
// be associated with a single Album through a many-to-many relationship.
type Artist struct {
	// ID is a UUID-based unique identifier for the artist record. This is the primary key
	// used in all database queries and foreign key relationships. It is generated server-side
	// using RFC 4122 UUID v4 format upon creation.
	ID string `json:"id"`

	// Name is the required display name of the artist or group. This is typically the stage name
	// or official name used in the music industry. The field must not be empty and should be
	// unique where possible, though duplicates are permitted to support homonyms and variant spellings.
	Name string `json:"name"`

	// ArtistBio is an optional external identifier from MusicBrainz, a community-maintained
	// music database. This identifier enables integration with MusicBrainz APIs and helps link
	// artist records across the SwaRupa system and external music metadata services.
	// See https://musicbrainz.org/ for more information.
	ArtistBio string `json:"artist_bio,omitempty"`

	// ImageURL is an optional HTTP(S) URL pointing to a profile or promotional image of the artist.
	// The field may reference images hosted on Content Delivery Networks (CDNs) or dedicated cloud storage.
	// Clients should treat this as a reference; no validation of URL validity is performed server-side.
	ImageURL string `json:"image_url,omitempty"`

	// SubmittedBy is an optional reference to the User ID of the person who submitted this artist record.
	// This field provides provenance tracking for community submissions and audit logging.
	SubmittedBy string `json:"submitted_by,omitempty"`

	// CreatedAt records the UTC timestamp when the artist record was first created in the database.
	// This is a server-generated field that is immutable and set automatically upon record creation.
	CreatedAt time.Time `json:"created_at"`
}
