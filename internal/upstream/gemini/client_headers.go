package gemini

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// generateGeminiCLIUserAgent creates a User-Agent string that mimics Gemini CLI client
func generateGeminiCLIUserAgent() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	goVersion := runtime.Version()
	return fmt.Sprintf("gemini-code-assist-cli/1.0.0 (%s; %s) %s", osName, arch, goVersion)
}

// applyDefaultHeaders centralizes default/override header logic
func (c *Client) applyDefaultHeaders(ctx context.Context, req *http.Request, bearer string) {
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	// Force gemini-cli fingerprint for all upstream requests
	req.Header.Set("User-Agent", generateGeminiCLIUserAgent())
	gv := runtime.Version()
	if strings.HasPrefix(gv, "go") {
		gv = gv[2:]
	}
	if gv == "" {
		gv = "unknown"
	}
	req.Header.Set("X-Goog-Api-Client", "gl-go/"+gv)
	req.Header.Set("Client-Metadata", "ideType=IDE_UNSPECIFIED,platform=PLATFORM_UNSPECIFIED,pluginType=GEMINI")

	// Apply header passthrough with whitelist filtering
	if c.cfg.Security.HeaderPassthroughConfig.Enabled {
		if hdr := getHeaderOverrides(ctx); hdr != nil {
			// Create header filter
			filter := NewHeaderFilter(c.cfg.Security.HeaderPassthroughConfig)

			// Filter and apply allowed headers
			filtered := filter.FilterHeaders(hdr)
			for key, values := range filtered {
				// Only set if not already present
				if req.Header.Get(key) == "" {
					for _, v := range values {
						req.Header.Add(key, v)
					}
				}
			}

			// Special handling for X-Request-ID -> X-Client-Request-ID
			if rid := filter.GetAllowedHeader(hdr, "X-Request-ID"); rid != "" && req.Header.Get("X-Request-ID") == "" {
				req.Header.Set("X-Client-Request-ID", rid)
			}
		}
	}

	if req.Header.Get("X-Goog-User-Project") == "" {
		if c.credentials != nil && strings.TrimSpace(c.credentials.ProjectID) != "" {
			req.Header.Set("X-Goog-User-Project", strings.TrimSpace(c.credentials.ProjectID))
		} else if strings.TrimSpace(c.cfg.GoogleProjID) != "" {
			req.Header.Set("X-Goog-User-Project", strings.TrimSpace(c.cfg.GoogleProjID))
		}
	}
}
