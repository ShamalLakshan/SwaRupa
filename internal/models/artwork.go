package models

import "time"

// ArtworkSource represents a location or platform where an artwork image is found.
// Multiple sources can represent the same artwork image from different providers,
// enabling deduplication and comparison (e.g., Apple Music, Amazon, Bandcamp).
type ArtworkSource struct {
	// ID is a UUID-based unique identifier for the artwork source record.
	ID string `json:"id"`

	// ArtworkID is a foreign key reference to the Artwork record with which this source is associated.
	ArtworkID string `json:"artwork_id"`

	// SourceName is the name of the platform or service where the image is hosted.
	// Examples: "apple_music", "amazon", "bandcamp", "spotify", "youtube"
	SourceName string `json:"source_name"`

	// SourcePage is an optional URL or identifier pointing to the page/product on the source platform.
	// Useful for tracking provenance and attribution.
	SourcePage string `json:"source_page,omitempty"`

	// ImageURL is the HTTP(S) URL to the artwork image as hosted by this source.
	// Different sources may host the same image at different URLs with different quality/formats.
	ImageURL string `json:"image_url"`

	// SourceType categorizes the source: "storefront" (official store like Apple/Amazon),
	// "community" (user-uploaded or forum), "official" (direct from artist/label).
	// Defaults to "storefront" and used for ranking/filtering sources.
	SourceType string `json:"source_type"`

	// ConfidenceScore is a float (0.0-1.0) indicating confidence that this image is correct.
	// System-assigned based on source_type; moderators can override.
	ConfidenceScore float64 `json:"confidence_score"`

	// QualityScore is a float (0.0-1.0) rating the visual quality/resolution of the image.
	// Used for moderators to rank competing sources and pick the best.
	QualityScore float64 `json:"quality_score"`

	// IsPrimary is a boolean flag indicating if this is the chosen/canonical source for the artwork.
	// Only one source per artwork should have is_primary=true; typically set during moderation.
	IsPrimary bool `json:"is_primary"`

	// DiscoveredBy indicates how this source was found: "system" (auto-scraped) or user_id (manually added).
	DiscoveredBy string `json:"discovered_by,omitempty"`

	// CreatedAt records the UTC timestamp when this source record was created.
	CreatedAt time.Time `json:"created_at"`
}

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

	// Sources is a slice of ArtworkSource records representing where this artwork image can be found.
	// Populated from the artwork_sources table when fetching artwork details.
	// Multiple sources enable deduplication and lets moderators compare versions (quality, availability).
	// Nested in response as "sources" array for API clients.
	Sources []ArtworkSource `json:"sources,omitempty"`
}
