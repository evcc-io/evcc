package auth

import "time"

// API is the external auth API
type API interface {
	RemoveAdminPassword()
	SetAdminPassword(string) error
	IsAdminPasswordValid(string) bool
	GenerateJwtToken(time.Duration) (string, error)
	ValidateJwtToken(string) (bool, error)
	IsAdminPasswordConfigured() bool
}
