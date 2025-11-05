package openai

import (
	"encoding/json"

	common "gcli2api-go/internal/handlers/common"
	"gcli2api-go/internal/models"
	tr "gcli2api-go/internal/translator"
	"github.com/gin-gonic/gin"
)

// Responses: 解析入参并分派到具体实现（fake/stream/final），功能保持不变
func (h *Handler) Responses(c *gin.Context) {
	// Use unified request parser
	req, err := common.ParseOpenAIRequest(c, "gemini-2.5-pro")
	if err != nil {
		common.AbortWithValidationError(c, err)
		return
	}

	// 翻译为 Gemini 请求
	reqJSON := tr.OpenAIResponsesToGeminiRequest(req.BaseModel, req.RawJSON, req.Stream)
	var gemReq map[string]any
	_ = json.Unmarshal(reqJSON, &gemReq)

	if req.Stream && h.cfg.FakeStreamingEnabled && models.IsFakeStreaming(req.Model) {
		h.responsesFakeStream(c, req.BaseModel, gemReq, req.Model)
		return
	}
	if req.Stream {
		h.responsesStream(c, req.BaseModel, gemReq, req.Model)
		return
	}
	h.responsesFinal(c, req.BaseModel, gemReq, req.Model)
}
