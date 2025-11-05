package management

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"gcli2api-go/internal/discovery"
	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// UpstreamSuggest aggregates candidate base models from config and known list
func (h *AdminAPIHandler) UpstreamSuggest(c *gin.Context) {
	existing := models.ActiveEntriesByChannel(h.cfg, h.storage, "openai")
	existingBase := map[string]struct{}{}
	for _, e := range existing {
		b := strings.TrimSpace(strings.ToLower(models.ParseModelName(e.ID).BaseName))
		if b == "" {
			b = strings.TrimSpace(strings.ToLower(e.Base))
		}
		if b != "" {
			existingBase[b] = struct{}{}
		}
	}
	// preferred bases from config
	pref := make([]string, 0, len(h.cfg.PreferredBaseModels))
	for _, p := range h.cfg.PreferredBaseModels {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			pref = append(pref, p)
		}
	}
	union := map[string]struct{}{}
	addBase := func(base string) {
		trim := strings.TrimSpace(base)
		if trim == "" {
			return
		}
		baseName := strings.ToLower(models.ParseModelName(trim).BaseName)
		if baseName != "" {
			union[baseName] = struct{}{}
		}
	}
	for _, p := range pref {
		addBase(p)
	}

	// upstream-discovered bases
	if h.modelFinder != nil {
		if upstreamBases, err := h.modelFinder.GetBases(c.Request.Context()); err == nil {
			log.WithFields(log.Fields{"component": "upstream_suggest", "upstream_bases": len(upstreamBases)}).Debug("merged upstream bases into suggestion set")
			for _, b := range upstreamBases {
				addBase(b)
			}
		} else if err != nil {
			log.WithError(err).WithField("component", "upstream_suggest").Warn("upstream discovery unavailable, using fallbacks")
		}
	}
	// fallback defaults
	for _, b := range models.DefaultBaseModels() {
		addBase(b)
	}

	bases := make([]string, 0, len(union))
	missing := make([]string, 0)
	for b := range union {
		bases = append(bases, b)
		if _, ok := existingBase[b]; !ok {
			missing = append(missing, b)
		}
	}
	existingList := make([]string, 0, len(existingBase))
	for b := range existingBase {
		existingList = append(existingList, b)
	}
	sort.Strings(bases)
	sort.Strings(missing)
	sort.Strings(existingList)
	meta := make(map[string]models.BaseDescriptor, len(bases))
	for _, b := range bases {
		meta[b] = models.DescribeBase(b)
	}
	c.JSON(200, gin.H{"bases": bases, "missing": missing, "existing_bases": existingList, "preferred": pref, "meta": meta})
}

// RefreshUpstreamModels manually triggers upstream model discovery and returns results
func (h *AdminAPIHandler) RefreshUpstreamModels(c *gin.Context) {
	if h.modelFinder == nil {
		respondError(c, http.StatusServiceUnavailable, "upstream model discovery not available")
		return
	}
	var req struct {
		Force   bool `json:"force"`
		Timeout int  `json:"timeout"`
	}
	_ = c.ShouldBindJSON(&req)
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	startTime := time.Now()
	if req.Force {
		h.modelFinder = discovery.NewUpstreamModelDiscovery(h.cfg, h.credMgr)
	}
	upstreamBases, err := h.modelFinder.GetBases(ctx)
	duration := time.Since(startTime)
	if err != nil {
		respondError(c, http.StatusServiceUnavailable, "failed to discover upstream models: "+err.Error(), gin.H{
			"duration": duration.String(),
			"forced":   req.Force,
		})
		return
	}
	cached := duration < time.Second
	if len(upstreamBases) == 0 {
		respondError(c, http.StatusServiceUnavailable, "no upstream models discovered", gin.H{
			"cached":   cached,
			"duration": duration.String(),
		})
		return
	}
	existing := models.ActiveEntriesByChannel(h.cfg, h.storage, "openai")
	existingBase := map[string]struct{}{}
	for _, e := range existing {
		b := strings.TrimSpace(strings.ToLower(models.ParseModelName(e.ID).BaseName))
		if b == "" {
			b = strings.TrimSpace(strings.ToLower(e.Base))
		}
		if b != "" {
			existingBase[b] = struct{}{}
		}
	}
	newModels := make([]string, 0)
	for _, base := range upstreamBases {
		baseName := strings.ToLower(models.ParseModelName(base).BaseName)
		if _, exists := existingBase[baseName]; !exists {
			newModels = append(newModels, base)
		}
	}
	sort.Strings(upstreamBases)
	sort.Strings(newModels)
	suggestions := make([]models.RegistryEntry, 0, len(newModels))
	for _, base := range newModels {
		entry := models.RegistryEntry{Base: base, Upstream: "code_assist", Enabled: true, Stream: true, FakeStreaming: false, AntiTrunc: false, Thinking: "auto", Search: false, Group: "discovered"}
		entry.ID = models.BuildVariantID(entry.Base, entry.FakeStreaming, entry.AntiTrunc, entry.Thinking, entry.Search)
		suggestions = append(suggestions, entry)
	}
	log.WithFields(log.Fields{"component": "upstream_refresh", "total_discovered": len(upstreamBases), "new_models": len(newModels), "cached": cached, "duration": duration.String(), "forced": req.Force}).Info("upstream model refresh completed")
	c.JSON(200, gin.H{"success": true, "discovered_models": upstreamBases, "new_models": newModels, "existing_count": len(existingBase), "suggestions": suggestions, "cached": cached, "duration": duration.String(), "forced": req.Force, "timestamp": time.Now().UTC()})
}
