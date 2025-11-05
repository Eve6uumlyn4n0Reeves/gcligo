package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// ProjectInfo represents Google Cloud project information
type ProjectInfo struct {
	ProjectID     string `json:"projectId"`
	ProjectNumber string `json:"projectNumber"`
	Name          string `json:"name"`
	State         string `json:"lifecycleState"`
}

// ProjectDetector detects and manages Google Cloud projects
type ProjectDetector struct {
	client *http.Client
}

// NewProjectDetector creates a new project detector
func NewProjectDetector() *ProjectDetector {
	return &ProjectDetector{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListProjects lists all projects accessible by the access token
func (pd *ProjectDetector) ListProjects(ctx context.Context, accessToken string) ([]ProjectInfo, error) {
	url := "https://cloudresourcemanager.googleapis.com/v1/projects"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := pd.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("failed to list projects: 403 forbidden (check IAM permissions for resourcemanager.projects.list)")
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("failed to list projects: 401 unauthorized (access token invalid or expired)")
		}
		return nil, fmt.Errorf("failed to list projects: status %d", resp.StatusCode)
	}

	var result struct {
		Projects []ProjectInfo `json:"projects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %w", err)
	}

	return result.Projects, nil
}

// EnableAPI enables a Google Cloud API for the project
func (pd *ProjectDetector) EnableAPI(ctx context.Context, accessToken, projectID, serviceName string) error {
	url := fmt.Sprintf("https://serviceusage.googleapis.com/v1/projects/%s/services/%s:enable", projectID, serviceName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// retry with simple backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := pd.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(1+attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			log.Infof("API %s enabled for project %s", serviceName, projectID)
			return nil
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
		time.Sleep(time.Duration(1+attempt) * time.Second)
	}
	if lastErr != nil {
		return fmt.Errorf("failed to enable API after retries: %w", lastErr)
	}
	return fmt.Errorf("failed to enable API: unknown error")
}

// EnableRequiredAPIs enables all required APIs for Gemini CLI
func (pd *ProjectDetector) EnableRequiredAPIs(ctx context.Context, accessToken, projectID string) error {
	requiredAPIs := []string{
		"generativelanguage.googleapis.com",
		"aiplatform.googleapis.com",
		"cloudresourcemanager.googleapis.com",
		// Required for Code Assist (Cloud AI Companion)
		"cloudaicompanion.googleapis.com",
	}

	for _, api := range requiredAPIs {
		if err := pd.EnableAPI(ctx, accessToken, projectID, api); err != nil {
			log.Warnf("Failed to enable API %s: %v", api, err)
			// Continue with other APIs
		}
	}

	return nil
}

// GetUserEmail retrieves the user's email address
func (pd *ProjectDetector) GetUserEmail(ctx context.Context, accessToken string) (string, error) {
	url := "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := pd.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo struct {
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo.Email, nil
}
