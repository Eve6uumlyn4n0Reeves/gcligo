package management

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// GetModelRegistry returns the current model registry (or empty list if none)
func (h *AdminAPIHandler) GetModelRegistry(c *gin.Context) {
	h.GetModelRegistryByChannel(c)
}

func (h *AdminAPIHandler) GetModelRegistryByChannel(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusOK, gin.H{"models": []any{}})
		return
	}
	key := channelKey(c.Param("channel"))
	v, err := h.storage.GetConfig(c.Request.Context(), key)
	if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": []any{}})
		return
	}
	if v == nil {
		c.JSON(http.StatusOK, gin.H{"models": []any{}})
		return
	}
	// 尝试叠加禁用原因（来自 auto-probe），便于前端卡片展示
	// 仅在返回结构可被解析为[]RegistryEntry时进行
	b, _ := json.Marshal(v)
	var entries []models.RegistryEntry
	if err := json.Unmarshal(b, &entries); err == nil {
		// load reasons map
		reasons := map[string]string{}
		if raw, rerr := h.storage.GetConfig(c.Request.Context(), "disabled_model_reasons"); rerr == nil && raw != nil {
			rb, _ := json.Marshal(raw)
			_ = json.Unmarshal(rb, &reasons)
		}
		dm := map[string]struct{}{}
		for _, d := range h.cfg.DisabledModels {
			dm[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
		}
		for i := range entries {
			base := strings.ToLower(strings.TrimSpace(entries[i].Base))
			if _, off := dm[base]; off {
				if entries[i].DisabledReason == "" {
					entries[i].DisabledReason = reasons[base]
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"models": entries})
		return
	}
	// fallback: 原样透传
	c.JSON(http.StatusOK, gin.H{"models": v})
}

// ReplaceModelRegistry replaces the entire registry with provided list
func (h *AdminAPIHandler) ReplaceModelRegistry(c *gin.Context) {
	h.ReplaceModelRegistryByChannel(withChannel(c, "openai"))
}

func (h *AdminAPIHandler) ReplaceModelRegistryByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	var req struct {
		Models []models.RegistryEntry `json:"models"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	// sanitize and compute IDs if missing; enforce upstream
	for i := range req.Models {
		if req.Models[i].Upstream == "" {
			req.Models[i].Upstream = "code_assist"
		}
		if req.Models[i].Upstream != "code_assist" {
			respondError(c, http.StatusBadRequest, "unsupported upstream; only code_assist allowed")
			return
		}
		if strings.TrimSpace(req.Models[i].ID) == "" {
			req.Models[i].ID = models.BuildVariantID(req.Models[i].Base, req.Models[i].FakeStreaming, req.Models[i].AntiTrunc, req.Models[i].Thinking, req.Models[i].Search)
		}
		if strings.TrimSpace(req.Models[i].Base) == "" {
			// derive base from ID
			req.Models[i].Base = models.BaseFromFeature(req.Models[i].ID)
		}
	}
	// de-duplicate by final ID (first wins)
	deduped := make([]models.RegistryEntry, 0, len(req.Models))
	seen := make(map[string]struct{}, len(req.Models))
	for _, e := range req.Models {
		id := strings.TrimSpace(e.ID)
		if id == "" {
			id = models.BuildVariantID(e.Base, e.FakeStreaming, e.AntiTrunc, e.Thinking, e.Search)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		e.ID = id
		deduped = append(deduped, e)
	}
	removed := len(req.Models) - len(deduped)
	if err := h.storage.SetConfig(c.Request.Context(), channelKey(c.Param("channel")), deduped); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(c, "model.replace", log.Fields{"count": len(deduped), "removed": removed})
	c.JSON(http.StatusOK, gin.H{"message": "registry updated", "count": len(deduped), "removed": removed})
}

// AddModelRegistry appends a model entry
func (h *AdminAPIHandler) AddModelRegistry(c *gin.Context) {
	h.AddModelRegistryByChannel(withChannel(c, "openai"))
}

func (h *AdminAPIHandler) AddModelRegistryByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	var entry models.RegistryEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	if entry.Upstream == "" {
		entry.Upstream = "code_assist"
	}
	if entry.Upstream != "code_assist" {
		respondError(c, http.StatusBadRequest, "unsupported upstream; only code_assist allowed")
		return
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = models.BuildVariantID(entry.Base, entry.FakeStreaming, entry.AntiTrunc, entry.Thinking, entry.Search)
	}
	// load existing
	var existing []models.RegistryEntry
	if v, err := h.storage.GetConfig(c.Request.Context(), channelKey(c.Param("channel"))); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &existing)
	} else if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
	}
	// upsert by final ID (replace existing if same id present)
	id := strings.TrimSpace(entry.ID)
	if id == "" {
		id = models.BuildVariantID(entry.Base, entry.FakeStreaming, entry.AntiTrunc, entry.Thinking, entry.Search)
		entry.ID = id
	}
	replaced := false
	for i := range existing {
		if strings.TrimSpace(existing[i].ID) == id {
			existing[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		existing = append(existing, entry)
	}
	if err := h.storage.SetConfig(c.Request.Context(), channelKey(c.Param("channel")), existing); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(c, "model.add", log.Fields{"id": entry.ID, "base": entry.Base})
	c.JSON(http.StatusOK, gin.H{"message": "model added", "id": entry.ID})
}

// DeleteModelRegistry removes an entry by id
func (h *AdminAPIHandler) DeleteModelRegistry(c *gin.Context) {
	h.DeleteModelRegistryByChannel(withChannel(c, "openai"))
}

func (h *AdminAPIHandler) DeleteModelRegistryByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	target := c.Param("id")
	var existing []models.RegistryEntry
	if v, err := h.storage.GetConfig(c.Request.Context(), channelKey(c.Param("channel"))); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &existing)
	} else if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
	}
	out := make([]models.RegistryEntry, 0, len(existing))
	removed := false
	for _, e := range existing {
		if e.ID == target {
			removed = true
			continue
		}
		out = append(out, e)
	}
	if !removed {
		respondError(c, http.StatusNotFound, "model not found")
		return
	}
	if err := h.storage.SetConfig(c.Request.Context(), channelKey(c.Param("channel")), out); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(c, "model.delete", log.Fields{"id": target})
	c.JSON(http.StatusOK, gin.H{"message": "model removed", "id": target})
}

// SeedDefaultRegistry writes a curated default registry to storage
func (h *AdminAPIHandler) SeedDefaultRegistry(c *gin.Context) {
	h.SeedDefaultRegistryByChannel(withChannel(c, "openai"))
}

func (h *AdminAPIHandler) SeedDefaultRegistryByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	defs := models.DefaultRegistry()
	for i := range defs {
		if strings.TrimSpace(defs[i].ID) == "" {
			defs[i].ID = models.BuildVariantID(defs[i].Base, defs[i].FakeStreaming, defs[i].AntiTrunc, defs[i].Thinking, defs[i].Search)
		}
	}
	if err := h.storage.SetConfig(c.Request.Context(), channelKey(c.Param("channel")), defs); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "seeded", "count": len(defs)})
}

// ImportModelRegistryByChannel supports importing models via JSON body (append/replace) or multipart upload of multiple JSON files.
func (h *AdminAPIHandler) ImportModelRegistryByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	key := channelKey(c.Param("channel"))
	mode := c.DefaultQuery("mode", "append") // append or replace
	ct := c.ContentType()
	var incoming []models.RegistryEntry
	if strings.HasPrefix(ct, "application/json") {
		var wrapper struct {
			Models []models.RegistryEntry `json:"models"`
		}
		if err := c.ShouldBindJSON(&wrapper); err != nil {
			respondError(c, http.StatusBadRequest, "invalid json")
			return
		}
		incoming = wrapper.Models
	} else if strings.HasPrefix(ct, "multipart/") {
		form, err := c.MultipartForm()
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid multipart")
			return
		}
		files := form.File["files"]
		for _, fh := range files {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(f)
			f.Close()
			var entry models.RegistryEntry
			if json.Unmarshal(data, &entry) == nil {
				incoming = append(incoming, entry)
			}
		}
		// also support single file field "file"
		if len(incoming) == 0 {
			if fh, err := c.FormFile("file"); err == nil {
				if f, err2 := fh.Open(); err2 == nil {
					data, _ := io.ReadAll(f)
					f.Close()
					var entry models.RegistryEntry
					if json.Unmarshal(data, &entry) == nil {
						incoming = append(incoming, entry)
					}
				}
			}
		}
	} else {
		respondError(c, http.StatusBadRequest, "unsupported content-type")
		return
	}
	// sanitize
	for i := range incoming {
		if incoming[i].Upstream == "" {
			incoming[i].Upstream = "code_assist"
		}
		if strings.TrimSpace(incoming[i].ID) == "" {
			incoming[i].ID = models.BuildVariantID(incoming[i].Base, incoming[i].FakeStreaming, incoming[i].AntiTrunc, incoming[i].Thinking, incoming[i].Search)
		}
		if strings.TrimSpace(incoming[i].Base) == "" {
			incoming[i].Base = models.BaseFromFeature(incoming[i].ID)
		}
	}
	var current []models.RegistryEntry
	if v, err := h.storage.GetConfig(c.Request.Context(), key); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &current)
	} else if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
	}
	var final []models.RegistryEntry
	if strings.ToLower(mode) == "replace" {
		final = incoming
	} else {
		final = append(current, incoming...)
	}
	// de-duplicate by computed final id (first wins, preserving earlier entries)
	deduped := make([]models.RegistryEntry, 0, len(final))
	seen := make(map[string]struct{}, len(final))
	for _, e := range final {
		id := strings.TrimSpace(e.ID)
		if id == "" {
			id = models.BuildVariantID(e.Base, e.FakeStreaming, e.AntiTrunc, e.Thinking, e.Search)
			e.ID = id
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, e)
	}
	if err := h.storage.SetConfig(c.Request.Context(), key, deduped); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "imported", "count": len(incoming), "removed": len(final) - len(deduped)})
}

// ExportModelRegistryByChannel exports the registry as JSON
func (h *AdminAPIHandler) ExportModelRegistryByChannel(c *gin.Context) {
	key := channelKey(c.Param("channel"))
	if h.storage == nil {
		c.JSON(http.StatusOK, gin.H{"models": []any{}})
		return
	}
	v, err := h.storage.GetConfig(c.Request.Context(), key)
	if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": []any{}})
		return
	}
	if v == nil {
		c.JSON(http.StatusOK, gin.H{"models": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": v})
}
