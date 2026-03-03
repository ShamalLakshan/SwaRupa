package models

type Album struct {
	ID          string `json:"id"`
	ArtistID    string `json:"artist_id"`
	Title       string `json:"title"`
	ReleaseYear int    `json:"release_year,omitempty"`
}
