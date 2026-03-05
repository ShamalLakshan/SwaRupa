package models

import "time"

type Album struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	ReleaseYear int       `json:"release_year,omitempty"`
	SubmittedBy string    `json:"submitted_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Artists     []Artist  `json:"artists,omitempty"`
}
