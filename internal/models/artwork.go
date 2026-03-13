package models

import "time"

// Artwork represents album cover artwork or promotional images associated with a specific Album.
// Users can submit multiple artwork candidates for each album, each with approval status,
// priority scoring, and source attribution. Official artwork takes precedence in display logic.
type Artwork struct {
	// ID is a UUID-based unique identifier for the artwork record. Generated server-side
	// using RFC 4122 UUID v4 upon creation and serves as the primary key in the artworks table.
	ID string `json:"id"`

	// AlbumID is a foreign key reference to the Album record with which this artwork is associated.
	// This field is required and immutable. Multiple artwork records can reference the same album.
	AlbumID string `json:"album_id"`

	// SourceID is an optional external identifier indicating the origin of the artwork image.
	// For example, it might reference an identifier from a third-party image API, CDN, or content provider.
	// This enables tracking and attribution of image sources across the system.
	SourceID string `json:"source_id,omitempty"`

	// ImageURL is a required HTTP(S) URL pointing to the full-resolution artwork image.
	// This field must not be empty and should reference a publicly accessible, persistent URI.
	// Clients render this URL to display album artwork in the user interface.
	ImageURL string `json:"image_url"`

	// ThumbnailURL is an optional HTTP(S) URL pointing to a reduced-resolution or downsampled version
	// of the artwork image. Used for performance optimization in gallery views or list displays.
	// If not provided, clients should request a thumbnail from the image provider or use a resize service.
	ThumbnailURL string `json:"thumbnail_url,omitempty"`

	// IsOfficial is a boolean flag indicating whether this artwork is the official or canonical
	// image from the music publisher or label. Official artwork typically takes precedence
	// in display priorities and approval workflows. Set to true only for officially licensed images.
	IsOfficial bool `json:"is_official"`

	// SubmittedBy is an optional reference to the User ID of the person who submitted this artwork record.
	// Provides attribution and audit trail for community contributions to the artwork database.
	SubmittedBy string `json:"submitted_by,omitempty"`

	// ApprovalStatus indicates the moderation state of this artwork record.
	// Valid values are: "pending" (awaiting review), "approved" (verified and published),
	// "rejected" (failed verification or quality checks). New artworks default to "pending" upon creation.
	ApprovalStatus string `json:"approval_status"`

	// PriorityScore is a non-negative integer ranking this artwork relative to others for the same album.
	// Higher scores indicate higher display priority. This field is used in sorting and selection logic
	// to determine which artwork is shown first to end users. Moderators adjust scores during approval.
	PriorityScore int `json:"priority_score"`

	// CreatedAt records the UTC timestamp when the artwork record was first created in the database.
	// This is a server-generated, immutable field set automatically upon record creation.
	CreatedAt time.Time `json:"created_at"`
}
