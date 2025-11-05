package openai

import "github.com/gin-gonic/gin"

// ChatCompletions handles POST /v1/chat/completions by translating the request to Gemini.
func (h *Handler) ChatCompletions(c *gin.Context) {
	var modelRecorded string
	defer func() {
		h.recordUsage(c, modelRecorded, c.Writer.Status() < 400, nil, 0, 0)
	}()

	reqCtx, errResp := buildChatRequest(h, c)
	if errResp != nil {
		errResp.write(c)
		return
	}

	modelRecorded = reqCtx.modelID()

	client, usedCred := h.resolveChatClient(c)
	if reqCtx.isStreaming() {
		if err := h.streamChatCompletions(c, reqCtx, client, &usedCred); err != nil {
			err.write(c)
		}
		return
	}

	if err := h.completeChat(c, reqCtx, &usedCred); err != nil {
		err.write(c)
	}
}
