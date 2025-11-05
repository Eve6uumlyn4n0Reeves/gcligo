package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// Google OAuth endpoints
	AuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenURL = "https://oauth2.googleapis.com/token"

	DefaultRedirectURI       = "http://localhost:8085/oauth2callback"
	DefaultUserInfoEndpoint  = "https://www.googleapis.com/oauth2/v2/userinfo"
	DefaultTokenInfoEndpoint = "https://www.googleapis.com/oauth2/v1/tokeninfo"
)

var (
	// Google Cloud scopes
	DefaultScopes = []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}
)

type projectDetector interface {
	ListProjects(ctx context.Context, accessToken string) ([]ProjectInfo, error)
	GetUserEmail(ctx context.Context, accessToken string) (string, error)
	EnableRequiredAPIs(ctx context.Context, accessToken, projectID string) error
}

// ManagerOption customizes Manager creation.
type ManagerOption func(*Manager)

// Manager handles OAuth authentication flows
type Manager struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	sessions     map[string]*AuthSession
	sessionMu    sync.RWMutex
	httpClient   *http.Client

	detectorFactory   func() projectDetector
	oauthEndpoint     oauth2.Endpoint
	tokenURL          string
	userInfoEndpoint  string
	tokenInfoEndpoint string
	now               func() time.Time
}

// NewManager creates a new OAuth manager
func NewManager(clientID, clientSecret, redirectURI string, opts ...ManagerOption) *Manager {
	m := &Manager{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  firstNonEmpty(redirectURI, DefaultRedirectURI),
		scopes:       append([]string(nil), DefaultScopes...),
		sessions:     make(map[string]*AuthSession),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		detectorFactory: func() projectDetector {
			return NewProjectDetector()
		},
		oauthEndpoint:     google.Endpoint,
		tokenURL:          TokenURL,
		userInfoEndpoint:  DefaultUserInfoEndpoint,
		tokenInfoEndpoint: DefaultTokenInfoEndpoint,
		now:               time.Now,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m
}

// WithHTTPClient overrides the HTTP client used for outbound calls.
func WithHTTPClient(client *http.Client) ManagerOption {
	return func(m *Manager) {
		if client != nil {
			m.httpClient = client
		}
	}
}

// WithProjectDetectorFactory overrides the project detector factory.
func WithProjectDetectorFactory(factory func() projectDetector) ManagerOption {
	return func(m *Manager) {
		if factory != nil {
			m.detectorFactory = factory
		}
	}
}

// WithOAuthEndpoint overrides the OAuth endpoints (auth/token) used in flows.
func WithOAuthEndpoint(endpoint oauth2.Endpoint) ManagerOption {
	return func(m *Manager) {
		if endpoint.AuthURL != "" && endpoint.TokenURL != "" {
			m.oauthEndpoint = endpoint
		}
	}
}

// WithTokenURL overrides the token refresh endpoint.
func WithTokenURL(tokenURL string) ManagerOption {
	return func(m *Manager) {
		if tokenURL != "" {
			m.tokenURL = tokenURL
		}
	}
}

// WithUserInfoEndpoint overrides the user info endpoint.
func WithUserInfoEndpoint(endpoint string) ManagerOption {
	return func(m *Manager) {
		if endpoint != "" {
			m.userInfoEndpoint = endpoint
		}
	}
}

// WithTokenInfoEndpoint overrides the token validation endpoint.
func WithTokenInfoEndpoint(endpoint string) ManagerOption {
	return func(m *Manager) {
		if endpoint != "" {
			m.tokenInfoEndpoint = endpoint
		}
	}
}

// WithNowFunc overrides the clock used for time calculations (testing).
func WithNowFunc(now func() time.Time) ManagerOption {
	return func(m *Manager) {
		if now != nil {
			m.now = now
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (m *Manager) ensureClientCredentials() error {
	if strings.TrimSpace(m.clientID) == "" || strings.TrimSpace(m.clientSecret) == "" {
		return fmt.Errorf("oauth client credentials not configured")
	}
	return nil
}

// StartAuthFlow initiates OAuth authentication flow
func (m *Manager) StartAuthFlow(projectID string) (authURL, state string, err error) {
	if err := m.ensureClientCredentials(); err != nil {
		return "", "", err
	}

	// Generate state
	state = uuid.New().String()

	// Generate PKCE code verifier
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate code verifier: %w", err)
	}

	// Store session
	m.sessionMu.Lock()
	m.sessions[state] = &AuthSession{
		State:        state,
		CodeVerifier: codeVerifier,
		ProjectID:    projectID,
		CreatedAt:    m.now(),
	}
	m.sessionMu.Unlock()

	// Build auth URL
	config := m.getOAuthConfig()
	authURL = config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("code_challenge", generateCodeChallenge(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	if projectID != "" {
		authURL += "&project=" + url.QueryEscape(projectID)
	}

	log.Infof("OAuth flow started for project: %s, state: %s", projectID, state)
	return authURL, state, nil
}

// HandleCallback handles OAuth callback
func (m *Manager) HandleCallback(ctx context.Context, code, state string) (*Credentials, error) {
	// Get session
	m.sessionMu.RLock()
	session, exists := m.sessions[state]
	m.sessionMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid state or session expired")
	}

	if err := m.ensureClientCredentials(); err != nil {
		return nil, err
	}

	// Exchange code for token
	config := m.getOAuthConfig()
	httpClientCtx := ctx
	if m.httpClient != nil {
		httpClientCtx = context.WithValue(ctx, oauth2.HTTPClient, m.httpClient)
	}

	token, err := config.Exchange(httpClientCtx, code,
		oauth2.SetAuthURLParam("code_verifier", session.CodeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Create credentials
	creds := &Credentials{
		ClientID:     m.clientID,
		ClientSecret: m.clientSecret,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenURI:     m.tokenURL,
		ProjectID:    session.ProjectID,
		ExpiresAt:    token.Expiry,
		Scopes:       m.scopes,
	}

	// Clean up session
	m.sessionMu.Lock()
	delete(m.sessions, state)
	m.sessionMu.Unlock()

	log.Infof("OAuth callback successful for project: %s", session.ProjectID)
	return creds, nil
}

// RefreshToken refreshes an access token
func (m *Manager) RefreshToken(ctx context.Context, creds *Credentials) error {
	if creds.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}
	if err := m.ensureClientCredentials(); err != nil {
		return err
	}

	data := url.Values{
		"client_id":     {m.clientID},
		"client_secret": {m.clientSecret},
		"refresh_token": {creds.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	// Update credentials
	creds.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		creds.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		creds.ExpiresAt = m.now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	log.Infof("Token refreshed successfully for project: %s", creds.ProjectID)
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (m *Manager) CleanupExpiredSessions() {
	m.sessionMu.Lock()
	defer m.sessionMu.Unlock()

	expiry := m.now().Add(-10 * time.Minute)
	for state, session := range m.sessions {
		if session.CreatedAt.Before(expiry) {
			delete(m.sessions, state)
		}
	}
}

func (m *Manager) getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     m.clientID,
		ClientSecret: m.clientSecret,
		RedirectURL:  m.redirectURI,
		Scopes:       m.scopes,
		Endpoint:     m.oauthEndpoint,
	}
}

// Helper functions for PKCE
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	// S256: BASE64URL-ENCODE(SHA256(verifier))
	sha := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sha[:])
}

// GetUserProjects lists projects accessible by the given access token
func (m *Manager) GetUserProjects(ctx context.Context, accessToken string) ([]ProjectInfo, error) {
	return m.detectorFactory().ListProjects(ctx, accessToken)
}

// GetUserEmail retrieves the user's email using the access token
func (m *Manager) GetUserEmail(ctx context.Context, accessToken string) (string, error) {
	return m.detectorFactory().GetUserEmail(ctx, accessToken)
}

// EnableAPIs enables required Google APIs for a project
func (m *Manager) EnableAPIs(ctx context.Context, accessToken, projectID string) error {
	return m.detectorFactory().EnableRequiredAPIs(ctx, accessToken, projectID)
}

// BatchGetUserEmails retrieves user emails for multiple access tokens
func (m *Manager) BatchGetUserEmails(ctx context.Context, tokens []string) (map[string]string, error) {
	results := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent requests
	semaphore := make(chan struct{}, 5)

	for _, token := range tokens {
		if strings.TrimSpace(token) == "" {
			continue
		}

		wg.Add(1)
		go func(accessToken string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			if email, err := m.GetUserEmail(ctx, accessToken); err == nil {
				mu.Lock()
				results[accessToken] = email
				mu.Unlock()
			} else {
				log.WithError(err).Debugf("Failed to get email for token %s...", accessToken[:min(8, len(accessToken))])
			}
		}(token)
	}

	wg.Wait()
	return results, nil
}

// GetUserProfile retrieves detailed user profile information
func (m *Manager) GetUserProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", m.userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user profile: %d %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &profile, nil
}

// ValidateToken checks if an access token is still valid
func (m *Manager) ValidateToken(ctx context.Context, accessToken string) (bool, error) {
	if accessToken == "" {
		return false, fmt.Errorf("access token is required")
	}

	u, err := url.Parse(m.tokenInfoEndpoint)
	if err != nil {
		return false, fmt.Errorf("failed to parse token info endpoint: %w", err)
	}
	query := u.Query()
	query.Set("access_token", accessToken)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
