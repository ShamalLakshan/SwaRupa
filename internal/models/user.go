package models

import "time"

type User struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name,omitempty"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}
