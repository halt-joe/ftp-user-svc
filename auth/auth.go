package auth

import (
	"net/http"
)

// Custom Errors
const (
	ErrUnauthorized = "Unauthorized (Failed Authentication)"
)

// APIKey - Authentication used by the service
var APIKey string = ""

// Authenticate - perform the authentication check
func Authenticate(r *http.Request) bool {
	result := false
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == APIKey {
		result = true
	}
	return result
}
