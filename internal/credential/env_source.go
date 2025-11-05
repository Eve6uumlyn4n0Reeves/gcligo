package credential

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// EnvSource loads credentials from environment variables matching GCLI_CREDS_* pattern
type EnvSource struct {
	prefix string
}

// NewEnvSource creates a new environment variable credential source
func NewEnvSource() *EnvSource {
	return &EnvSource{
		prefix: "GCLI_CREDS_",
	}
}

// Name returns the source identifier
func (s *EnvSource) Name() string {
	return "env"
}

// Load retrieves all credentials from environment variables
// Supports both:
// - GCLI_CREDS_1, GCLI_CREDS_2, ... (numbered)
// - GCLI_CREDS_projectname, GCLI_CREDS_myproject, ... (named)
// Values can be either:
// - Direct JSON string
// - Base64-encoded JSON (auto-detected)
func (s *EnvSource) Load(ctx context.Context) ([]*Credential, error) {
	creds := make([]*Credential, 0)
	seen := make(map[string]struct{})

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]

		// Skip if not matching our prefix
		if !strings.HasPrefix(key, s.prefix) {
			continue
		}

		// Extract credential identifier (e.g., "1" from "GCLI_CREDS_1" or "project1" from "GCLI_CREDS_project1")
		id := strings.TrimPrefix(key, s.prefix)
		if id == "" {
			continue
		}

		// Skip duplicates
		if _, exists := seen[id]; exists {
			log.Warnf("Duplicate credential environment variable: %s", key)
			continue
		}

		// Attempt to parse credential
		cred, err := s.parseCredential(id, val)
		if err != nil {
			log.WithError(err).Warnf("Failed to parse credential from %s", key)
			continue
		}

		if cred == nil {
			continue
		}

		creds = append(creds, cred)
		seen[id] = struct{}{}
	}

	if len(creds) > 0 {
		log.Infof("Loaded %d credential(s) from environment variables", len(creds))
	}

	return creds, nil
}

// parseCredential attempts to parse a credential from raw value
// Handles both plain JSON and base64-encoded JSON
func (s *EnvSource) parseCredential(id, rawValue string) (*Credential, error) {
	if rawValue == "" {
		return nil, fmt.Errorf("empty value")
	}

	var credData map[string]interface{}

	// Try direct JSON parse first
	err := json.Unmarshal([]byte(rawValue), &credData)
	if err != nil {
		// Try base64 decode + JSON parse
		decoded, decodeErr := base64.StdEncoding.DecodeString(rawValue)
		if decodeErr != nil {
			return nil, fmt.Errorf("not valid JSON or base64: %w", err)
		}
		if err := json.Unmarshal(decoded, &credData); err != nil {
			return nil, fmt.Errorf("base64 content not valid JSON: %w", err)
		}
	}

	// Build credential from parsed data
	cred := &Credential{
		Type:   "oauth", // Environment creds are assumed to be OAuth
		Source: fmt.Sprintf("env:%s", id),
	}

	// Extract fields from parsed JSON
	if clientID, ok := credData["client_id"].(string); ok {
		cred.ClientID = clientID
	}
	if clientSecret, ok := credData["client_secret"].(string); ok {
		cred.ClientSecret = clientSecret
	}
	if refreshToken, ok := credData["refresh_token"].(string); ok {
		cred.RefreshToken = refreshToken
	}
	if accessToken, ok := credData["access_token"].(string); ok {
		cred.AccessToken = accessToken
	}
	if projectID, ok := credData["project_id"].(string); ok {
		cred.ProjectID = projectID
	}
	if email, ok := credData["email"].(string); ok {
		cred.Email = email
	}

	// Use project_id as ID if available, otherwise use the env variable suffix
	if cred.ProjectID != "" {
		cred.ID = fmt.Sprintf("env-%s.json", cred.ProjectID)
	} else {
		cred.ID = fmt.Sprintf("env-%s.json", id)
	}

	// Validate minimum required fields
	if cred.ClientID == "" || cred.ClientSecret == "" {
		return nil, fmt.Errorf("missing required fields (client_id, client_secret)")
	}

	if cred.RefreshToken == "" && cred.AccessToken == "" {
		return nil, fmt.Errorf("missing refresh_token or access_token")
	}

	return cred, nil
}

// EnvSource is read-only, so Save is not supported
func (s *EnvSource) Save(ctx context.Context, cred *Credential) error {
	return fmt.Errorf("env source is read-only")
}

// EnvSource is read-only, so Delete is not supported
func (s *EnvSource) Delete(ctx context.Context, credID string) error {
	return fmt.Errorf("env source is read-only")
}

// Load all env credentials at startup
// Returns true if any env credentials were found
func LoadEnvCredentials() bool {
	count := 0
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GCLI_CREDS_") {
			count++
		}
	}
	return count > 0
}

// AutoLoadEnvCredsEnabled checks if AUTO_LOAD_ENV_CREDS is enabled
func AutoLoadEnvCredsEnabled() bool {
	val := strings.ToLower(os.Getenv("AUTO_LOAD_ENV_CREDS"))
	return val == "true" || val == "1" || val == "yes" || val == "on"
}
