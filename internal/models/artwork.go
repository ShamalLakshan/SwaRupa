package models

type Artwork struct {
	ID             string `json:"id"`
	AlbumID        string `json:"album_id"`
	SourceID       string `json:"source_id"`
	ImageURL       string `json:"image_url"`
	ThumbnailURL   string `json:"thumbnail_url,omitempty"`
	IsOfficial     bool   `json:"is_official"`
	SubmittedBy    string `json:"submitted_by,omitempty"`
	ApprovalStatus string `json:"approval_status"`
	PriorityScore  int    `json:"priority_score"`
}
