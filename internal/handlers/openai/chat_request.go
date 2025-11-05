package openai

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/models"
	tr "gcli2api-go/internal/translator"
	upstream "gcli2api-go/internal/upstream"
	"github.com/gin-gonic/gin"
)

type chatRequestContext struct {
	raw       map[string]any
	gemReq    map[string]any
	rawJSON   []byte
	model     string
	baseModel string
	stream    bool
}

func (ctx *chatRequestContext) upstreamPayload(project string) []byte {
	payload := map[string]any{
		"model":   ctx.baseModel,
		"project": project,
		"request": ctx.gemReq,
	}
	b, _ := json.Marshal(payload)
	return b
}

func (ctx *chatRequestContext) cloneForContinuation() map[string]any {
	return cloneMap(ctx.gemReq)
}

func (ctx *chatRequestContext) isStreaming() bool {
	return ctx.stream
}

func (ctx *chatRequestContext) modelID() string {
	return ctx.model
}

func buildChatRequest(h *Handler, c *gin.Context) (*chatRequestContext, *chatError) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		return nil, newChatError(http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err), "invalid_request_error")
	}
	if normalized, status, msg := validateAndNormalizeOpenAI(raw, true); status != 0 {
		return nil, newChatError(status, msg, "invalid_request_error")
	} else {
		raw = normalized
	}

	model, _ := raw["model"].(string)
	if model == "" {
		model = "gemini-2.5-pro"
	}
	stream, _ := raw["stream"].(bool)
	baseModel := models.BaseFromFeature(model)

	c.Set("model", model)
	c.Set("base_model", baseModel)

	rawJSON, _ := json.Marshal(raw)
	reqJSON := tr.OpenAIToGeminiRequest(baseModel, rawJSON, stream)

	var gemReq map[string]any
	_ = json.Unmarshal(reqJSON, &gemReq)

	if models.IsSearch(model) {
		injectSearchTool(gemReq)
	}
	mergeToolResponses(raw, gemReq)

	return &chatRequestContext{
		raw:       raw,
		gemReq:    gemReq,
		rawJSON:   rawJSON,
		model:     model,
		baseModel: baseModel,
		stream:    stream,
	}, nil
}

func (h *Handler) resolveChatClient(c *gin.Context) (geminiClient, *credential.Credential) {
	client := h.baseClient
	var usedCred *credential.Credential
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
	return client, usedCred
}
