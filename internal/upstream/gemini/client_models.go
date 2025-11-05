package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"gcli2api-go/internal/upstream"
)

var (
	geminiModelPattern = regexp.MustCompile(`(?i)gemini-[a-z0-9][a-z0-9\._-]*`)
	trailingTrimChars  = " ,.;:\"'()[]{}<>"
)

// ListModels tries multiple upstream endpoints to discover available base models.
func (c *Client) ListModels(ctx context.Context, projectID string) ([]string, error) {
	attempts := []func(context.Context, string) ([]string, error){
		c.listModelsCatalog,
		c.listModelsViaLoadCodeAssist,
	}
	var lastErr error
	for _, attempt := range attempts {
		ids, err := attempt(ctx, projectID)
		if len(ids) > 0 {
			return ids, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no models discovered")
}

func (c *Client) listModelsCatalog(ctx context.Context, projectID string) ([]string, error) {
	bearer := strings.TrimSpace(c.getToken())
	if bearer == "" {
		return nil, errors.New("missing access token")
	}

	base := strings.TrimSuffix(c.cfg.CodeAssist, "/")
	if base == "" {
		base = "https://cloudcode-pa.googleapis.com"
	}

	endpoints := []string{base + "/v1/models?pageSize=200&view=FULL"}
	if projectID != "" {
		escaped := url.PathEscape(projectID)
		endpoints = append([]string{
			fmt.Sprintf("%s/v1/projects/%s/locations/global/models?pageSize=200&view=FULL", base, escaped),
			fmt.Sprintf("%s/v1/projects/%s/locations/-/models?pageSize=200&view=FULL", base, escaped),
		}, endpoints...)
	}

	var lastErr error
	for _, endpoint := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		if projectID != "" {
			req.Header.Set("X-Goog-User-Project", projectID)
		}
		c.applyDefaultHeaders(ctx, req, bearer)

		resp, err := c.cli.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("catalog status %d", resp.StatusCode)
			continue
		}
		ids := extractModelIDsFromJSON(body)
		if len(ids) > 0 {
			return ids, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("catalog returned no models")
}

func (c *Client) listModelsViaLoadCodeAssist(ctx context.Context, projectID string) ([]string, error) {
	payload := map[string]any{
		"metadata": map[string]any{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	if projectID != "" {
		payload["cloudaicompanionProject"] = projectID
	}
	body, _ := json.Marshal(payload)

	resp, err := c.Action(ctx, "loadCodeAssist", body)
	if err != nil {
		return nil, err
	}
	data, readErr := upstream.ReadAll(resp)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("loadCodeAssist status %d", resp.StatusCode)
	}
	ids := extractModelIDsFromJSON(data)
	if len(ids) == 0 {
		return nil, fmt.Errorf("loadCodeAssist returned no models")
	}
	return ids, nil
}

func extractModelIDsFromJSON(data []byte) []string {
	dest := make(map[string]struct{})
	var payload any
	if err := json.Unmarshal(data, &payload); err == nil {
		collectModelIDs(payload, dest)
	}
	if len(dest) == 0 {
		addModelMatches(string(data), dest)
	}
	if len(dest) == 0 {
		return nil
	}
	out := make([]string, 0, len(dest))
	for id := range dest {
		out = append(out, id)
	}
	return out
}

func collectModelIDs(value any, dest map[string]struct{}) {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if strings.Contains(strings.ToLower(key), "model") {
				if s, ok := item.(string); ok {
					addModelMatches(s, dest)
				}
			}
			collectModelIDs(item, dest)
		}
	case []any:
		for _, item := range v {
			collectModelIDs(item, dest)
		}
	case string:
		addModelMatches(v, dest)
	}
}

func addModelMatches(input string, dest map[string]struct{}) {
	if input == "" {
		return
	}
	matches := geminiModelPattern.FindAllString(input, -1)
	if len(matches) == 0 {
		return
	}
	for _, match := range matches {
		cleaned := strings.Trim(strings.ToLower(match), trailingTrimChars)
		if cleaned == "" {
			continue
		}
		dest[cleaned] = struct{}{}
	}
}
