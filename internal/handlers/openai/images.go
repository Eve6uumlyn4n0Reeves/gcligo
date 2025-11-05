package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gcli2api-go/internal/credential"
	common "gcli2api-go/internal/handlers/common"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
	util "gcli2api-go/internal/utils"
	"github.com/gin-gonic/gin"
)

type imagesRequest struct {
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
	Model  string `json:"model"`
}

// POST /v1/images/generations -> Gemini image models (no external providers in default build)
func (h *Handler) ImagesGenerations(c *gin.Context) {
	var req imagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request_error", "invalid json")
		return
	}
	// Default model if missing
	if req.Model == "" {
		req.Model = "gemini-2.5-flash-image"
	}
	// 兼容别名：nano-banana* -> gemini-2.5-flash-image-preview
	if mapped, ok := models.ResolveAlias(req.Model); ok {
		req.Model = mapped
	}
	// 为日志上下文提供模型/基座
	if req.Model != "" {
		c.Set("model", req.Model)
		c.Set("base_model", models.BaseFromFeature(req.Model))
	}
	// Route to Gemini image models (e.g., gemini-2.5-flash-image[-preview])
	if isGeminiImageModel(req.Model) {
		// Prefer strategy pick to allow routing debug headers
		var usedCred *credential.Credential
		client := h.baseClient
		if h.router != nil {
			ctxWith := upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header)
			if cred, info := h.router.PickWithInfo(ctxWith, upstream.HeaderOverrides(ctxWith)); cred != nil {
				usedCred = cred
				if h.cfg.RoutingDebugHeaders {
					if info != nil {
						c.Writer.Header().Set("X-Routing-Credential", info.CredID)
						c.Writer.Header().Set("X-Routing-Reason", info.Reason)
						if info.StickySource != "" {
							c.Writer.Header().Set("X-Routing-Sticky-Source", info.StickySource)
						}
					} else {
						c.Writer.Header().Set("X-Routing-Credential", cred.ID)
					}
				}
				client = h.getClientFor(cred)
			}
		}
		if usedCred == nil {
			client, usedCred = h.getUpstreamClient(c.Request.Context())
		}
		n := req.N
		if n <= 0 {
			n = 1
		}
		aspect := aspectFromSize(req.Size)
		gemReq := map[string]any{
			"contents":         []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": req.Prompt}}}},
			"generationConfig": map[string]any{"candidateCount": n, "responseModalities": []any{"Image"}},
		}
		if aspect != "" {
			gemReq["generationConfig"].(map[string]any)["imageConfig"] = map[string]any{"aspectRatio": aspect}
		}
		effProject := h.cfg.GoogleProjID
		if usedCred != nil && usedCred.ProjectID != "" {
			effProject = usedCred.ProjectID
		}
		base := models.BaseFromFeature(req.Model)
		// Inject placeholder for flash-image-preview when only aspect ratio provided
		_ = util.ApplyFlashImagePreviewPlaceholder(gemReq, base, h.cfg.AutoImagePlaceholder)
		payload := map[string]any{"model": base, "project": effProject, "request": gemReq}
		b, _ := json.Marshal(payload)
		ctx, cancel := context.WithTimeout(upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header), 120*time.Second)
		defer cancel()
		resp, err := client.Generate(ctx, b)
		if err != nil {
			common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
			return
		}
		by, err := upstream.ReadAll(resp)
		if err != nil {
			common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
			return
		}
		if resp != nil && resp.StatusCode >= 400 {
			if usedCred != nil {
				h.credMgr.MarkFailure(usedCred.ID, "upstream_error", resp.StatusCode)
			}
			common.AbortWithUpstreamError(c, http.StatusBadGateway, "upstream_error", "upstream error", by)
			return
		}
		var obj map[string]any
		_ = json.Unmarshal(by, &obj)
		var images []map[string]any
		if r, ok := obj["response"].(map[string]any); ok {
			if cands, ok := r["candidates"].([]any); ok {
				for _, cc := range cands {
					if cand, ok := cc.(map[string]any); ok {
						if content, ok := cand["content"].(map[string]any); ok {
							if parts, ok := content["parts"].([]any); ok {
								for _, pp := range parts {
									if p0, ok := pp.(map[string]any); ok {
										if in, ok := p0["inlineData"].(map[string]any); ok {
											mime := "image/png"
											if v, ok := in["mimeType"].(string); ok && v != "" {
												mime = v
											}
											dataB64, _ := in["data"].(string)
											if dataB64 != "" {
												// OpenAI images API prefers base64 payload; mime can be optionally included
												entry := map[string]any{"b64_json": dataB64}
												if h.cfg.OpenAIImagesIncludeMIME {
													entry["mime_type"] = mime
												}
												images = append(images, entry)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		if usedCred != nil {
			h.credMgr.MarkSuccess(usedCred.ID)
			if h.router != nil {
				h.router.OnResult(usedCred.ID, 200)
			}
		}
		c.JSON(http.StatusOK, gin.H{"created": time.Now().Unix(), "data": images})
		return
	}

	common.AbortWithError(c, http.StatusBadRequest, "invalid_request_error", "unsupported image model")
}

func isGeminiImageModel(model string) bool {
	m := models.BaseFromFeature(model)
	if strings.Contains(m, "flash-image") {
		return true
	}
	return strings.HasPrefix(m, "gemini-2.5-flash-image")
}

func aspectFromSize(size string) string {
	size = strings.TrimSpace(size)
	if size == "" {
		return "1:1"
	}
	// size like 1024x1024 or 1280x720
	var w, h int
	n, _ := fmt.Sscanf(size, "%dx%d", &w, &h)
	if n != 2 || w <= 0 || h <= 0 {
		return "1:1"
	}
	g := gcd(w, h)
	return fmt.Sprintf("%d:%d", w/g, h/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}
