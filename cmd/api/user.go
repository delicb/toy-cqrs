package main

// UserModel represents what clients of this API see from user.
type UserModel struct {
	ID      string `json:"id,omitempty"`
	Email   string `json:"email,omitempty"`
	Enabled bool   `json:"enabled"`
}
