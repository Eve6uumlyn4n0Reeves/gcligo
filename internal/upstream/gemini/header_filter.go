package gemini

import (
	"net/http"
	"strings"

	"gcli2api-go/internal/config"
	log "github.com/sirupsen/logrus"
)

// HeaderFilter filters headers based on allow/deny lists
type HeaderFilter struct {
	enabled   bool
	allowList map[string]bool
	denyList  map[string]bool
	auditLog  bool
}

// NewHeaderFilter creates a new header filter from configuration
func NewHeaderFilter(cfg config.HeaderPassthroughConfig) *HeaderFilter {
	if !cfg.Enabled {
		return &HeaderFilter{enabled: false}
	}

	filter := &HeaderFilter{
		enabled:   true,
		allowList: make(map[string]bool),
		denyList:  make(map[string]bool),
		auditLog:  cfg.AuditLog,
	}

	// Normalize and populate allow list
	for _, h := range cfg.AllowList {
		normalized := strings.ToLower(strings.TrimSpace(h))
		if normalized != "" {
			filter.allowList[normalized] = true
		}
	}

	// Normalize and populate deny list
	for _, h := range cfg.DenyList {
		normalized := strings.ToLower(strings.TrimSpace(h))
		if normalized != "" {
			filter.denyList[normalized] = true
		}
	}

	// Default allow list if none specified
	if len(filter.allowList) == 0 && len(filter.denyList) == 0 {
		filter.allowList["x-request-id"] = true
		filter.allowList["x-goog-user-project"] = true
	}

	return filter
}

// FilterHeaders filters headers based on allow/deny lists
// Returns a new http.Header with only allowed headers
func (f *HeaderFilter) FilterHeaders(source http.Header) http.Header {
	if !f.enabled || source == nil {
		return nil
	}

	filtered := make(http.Header)
	var allowed, denied []string

	for key, values := range source {
		normalized := strings.ToLower(key)

		// Check deny list first (deny takes precedence)
		if f.denyList[normalized] {
			denied = append(denied, key)
			continue
		}

		// If allow list is specified, only allow listed headers
		if len(f.allowList) > 0 {
			if !f.allowList[normalized] {
				continue
			}
		}

		// Copy allowed header
		for _, v := range values {
			filtered.Add(key, v)
		}
		allowed = append(allowed, key)
	}

	// Audit log if enabled
	if f.auditLog && (len(allowed) > 0 || len(denied) > 0) {
		log.WithFields(log.Fields{
			"allowed": allowed,
			"denied":  denied,
		}).Debug("Header passthrough filter applied")
	}

	return filtered
}

// IsAllowed checks if a specific header is allowed
func (f *HeaderFilter) IsAllowed(headerName string) bool {
	if !f.enabled {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(headerName))

	// Check deny list first
	if f.denyList[normalized] {
		return false
	}

	// If allow list is specified, check it
	if len(f.allowList) > 0 {
		return f.allowList[normalized]
	}

	// If no allow list and not in deny list, allow
	return true
}

// GetAllowedHeader safely gets a header value if it's allowed
func (f *HeaderFilter) GetAllowedHeader(source http.Header, headerName string) string {
	if !f.IsAllowed(headerName) {
		return ""
	}
	return source.Get(headerName)
}

