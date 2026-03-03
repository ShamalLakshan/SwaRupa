package models

type Artist struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MusicBrainzID string `json:"musicbrainz_id,omitempty"`
}
