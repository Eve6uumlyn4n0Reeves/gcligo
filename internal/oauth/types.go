package oauth

import (
	"time"
)

// Credentials represents OAuth credentials
type Credentials struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token"`
	TokenURI     string    `json:"token_uri"`
	ProjectID    string    `json:"project_id"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scopes       []string  `json:"scopes,omitempty"`
}

// IsExpired checks if the access token is expired
func (c *Credentials) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return true
	}
	// Consider expired 3 minutes before actual expiration
	return time.Now().Add(3 * time.Minute).After(c.ExpiresAt)
}

// AuthSession represents an OAuth authentication session
type AuthSession struct {
	State        string
	CodeVerifier string
	ProjectID    string
	CreatedAt    time.Time
}

// TokenResponse represents OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserProfile represents user profile information from Google
type UserProfile struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}
