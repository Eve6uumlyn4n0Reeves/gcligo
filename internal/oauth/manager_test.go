package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type testOAuthServer struct {
	t      *testing.T
	server *httptest.Server
	client *http.Client

	mu             sync.Mutex
	refreshHandled int
}

func newTestOAuthServer(t *testing.T) *testOAuthServer {
	t.Helper()

	s := &testOAuthServer{t: t}
	mux := http.NewServeMux()

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = r.ParseForm()
		grant := r.Form.Get("grant_type")
		switch grant {
		case "refresh_token":
			s.mu.Lock()
			s.refreshHandled++
			s.mu.Unlock()
			resp := TokenResponse{
				AccessToken:  "refreshed-token",
				RefreshToken: "next-refresh-token",
				ExpiresIn:    3600,
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			resp := TokenResponse{
				AccessToken:  "access-" + r.Form.Get("code"),
				RefreshToken: "refresh-" + r.Form.Get("code"),
				ExpiresIn:    3600,
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"email":          "tester@example.com",
			"verified_email": true,
			"name":           "Test User",
		})
	})

	mux.HandleFunc("/tokeninfo", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	s.server = httptest.NewServer(mux)
	s.client = s.server.Client()
	return s
}

func (s *testOAuthServer) close() {
	s.server.Close()
}

type fakeProjectDetector struct {
	mu            sync.Mutex
	emailByToken  map[string]string
	projectCalled bool
	enableCalled  bool
}

func newFakeProjectDetector(emails map[string]string) *fakeProjectDetector {
	return &fakeProjectDetector{emailByToken: emails}
}

func (f *fakeProjectDetector) ListProjects(ctx context.Context, token string) ([]ProjectInfo, error) {
	f.mu.Lock()
	f.projectCalled = true
	f.mu.Unlock()
	return []ProjectInfo{
		{ProjectID: "p-1", Name: "Proj 1"},
	}, nil
}

func (f *fakeProjectDetector) GetUserEmail(ctx context.Context, token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if email, ok := f.emailByToken[token]; ok {
		return email, nil
	}
	return "", context.DeadlineExceeded
}

func (f *fakeProjectDetector) EnableRequiredAPIs(ctx context.Context, token, projectID string) error {
	f.mu.Lock()
	f.enableCalled = true
	f.mu.Unlock()
	return nil
}

func TestManagerAuthFlowAndCallback(t *testing.T) {
	oauthServer := newTestOAuthServer(t)
	defer oauthServer.close()

	detector := newFakeProjectDetector(map[string]string{"token-A": "user@example.com"})
	mgr := NewManager(
		"client-id",
		"client-secret",
		"http://localhost/callback",
		WithHTTPClient(oauthServer.client),
		WithProjectDetectorFactory(func() projectDetector { return detector }),
		WithOAuthEndpoint(oauth2.Endpoint{
			AuthURL:  oauthServer.server.URL + "/auth",
			TokenURL: oauthServer.server.URL + "/token",
		}),
		WithTokenURL(oauthServer.server.URL+"/token"),
		WithUserInfoEndpoint(oauthServer.server.URL+"/userinfo"),
		WithTokenInfoEndpoint(oauthServer.server.URL+"/tokeninfo"),
		WithNowFunc(func() time.Time { return time.Unix(1_700_000_000, 0) }),
	)

	authURL, state, err := mgr.StartAuthFlow("p-test")
	if err != nil {
		t.Fatalf("StartAuthFlow failed: %v", err)
	}
	if state == "" {
		t.Fatalf("expected state to be generated")
	}
	if _, err := url.Parse(authURL); err != nil {
		t.Fatalf("authURL not valid: %v", err)
	}

	creds, err := mgr.HandleCallback(context.Background(), "code-123", state)
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}

	if creds.AccessToken != "access-code-123" {
		t.Fatalf("unexpected access token %q", creds.AccessToken)
	}
	if creds.RefreshToken != "refresh-code-123" {
		t.Fatalf("unexpected refresh %q", creds.RefreshToken)
	}
	if got := creds.TokenURI; got != oauthServer.server.URL+"/token" {
		t.Fatalf("unexpected token URI %q", got)
	}
}

func TestManagerRefreshToken(t *testing.T) {
	oauthServer := newTestOAuthServer(t)
	defer oauthServer.close()

	mgr := NewManager(
		"a", "b", "",
		WithHTTPClient(oauthServer.client),
		WithTokenURL(oauthServer.server.URL+"/token"),
		WithOAuthEndpoint(oauth2.Endpoint{
			AuthURL:  oauthServer.server.URL + "/auth",
			TokenURL: oauthServer.server.URL + "/token",
		}),
	)

	creds := &Credentials{
		ClientID:     "a",
		ClientSecret: "b",
		RefreshToken: "initial-refresh",
		ProjectID:    "prj",
	}

	if err := mgr.RefreshToken(context.Background(), creds); err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if creds.AccessToken != "refreshed-token" {
		t.Fatalf("unexpected access token %q", creds.AccessToken)
	}
	if creds.RefreshToken != "next-refresh-token" {
		t.Fatalf("unexpected refresh token %q", creds.RefreshToken)
	}
	if creds.ExpiresAt.IsZero() {
		t.Fatalf("expected expiresAt to be set")
	}
}

func TestManagerBatchGetUserEmails(t *testing.T) {
	detector := newFakeProjectDetector(map[string]string{
		"token-1": "a@example.com",
		"token-2": "b@example.com",
	})
	mgr := NewManager("id", "secret", "",
		WithProjectDetectorFactory(func() projectDetector { return detector }),
	)

	ctx := context.Background()
	result, err := mgr.BatchGetUserEmails(ctx, []string{"token-1", "token-2", "token-3", ""})
	if err != nil {
		t.Fatalf("BatchGetUserEmails failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(result))
	}
	if result["token-1"] != "a@example.com" {
		t.Fatalf("unexpected email for token-1: %q", result["token-1"])
	}
}

func TestManagerUserProfileAndValidation(t *testing.T) {
	oauthServer := newTestOAuthServer(t)
	defer oauthServer.close()

	mgr := NewManager(
		"id", "secret", "",
		WithHTTPClient(oauthServer.client),
		WithUserInfoEndpoint(oauthServer.server.URL+"/userinfo"),
		WithTokenInfoEndpoint(oauthServer.server.URL+"/tokeninfo"),
	)

	profile, err := mgr.GetUserProfile(context.Background(), "token-A")
	if err != nil {
		t.Fatalf("GetUserProfile failed: %v", err)
	}
	if profile.Email != "tester@example.com" {
		t.Fatalf("unexpected email %q", profile.Email)
	}

	valid, err := mgr.ValidateToken(context.Background(), "token-A")
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if !valid {
		t.Fatalf("expected token to be valid")
	}
}

func TestManagerCleanupSessions(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	mgr := NewManager("id", "secret", "",
		WithNowFunc(func() time.Time { return now }),
	)

	mgr.sessions["active"] = &AuthSession{State: "active", CreatedAt: now}
	mgr.sessions["expired"] = &AuthSession{State: "expired", CreatedAt: now.Add(-11 * time.Minute)}

	mgr.CleanupExpiredSessions()

	if _, ok := mgr.sessions["expired"]; ok {
		t.Fatalf("expected expired session to be removed")
	}
	if _, ok := mgr.sessions["active"]; !ok {
		t.Fatalf("expected active session to remain")
	}
}
