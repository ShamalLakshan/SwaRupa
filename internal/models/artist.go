package models

import "time"

type Artist struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	MusicBrainzID string    `json:"musicbrainz_id,omitempty"`
	ImageURL      string    `json:"image_url,omitempty"`
	SubmittedBy   string    `json:"submitted_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
