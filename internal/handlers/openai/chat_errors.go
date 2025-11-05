package openai

import (
	common "gcli2api-go/internal/handlers/common"
	"github.com/gin-gonic/gin"
)

type chatError struct {
	status  int
	message string
	code    string
	body    []byte
}

func (e *chatError) write(c *gin.Context) {
	if e == nil {
		return
	}
	if len(e.body) > 0 {
		common.AbortWithUpstreamError(c, e.status, e.code, e.message, e.body)
		return
	}
	common.AbortWithError(c, e.status, e.code, e.message)
}

func newChatError(status int, message, code string) *chatError {
	return &chatError{status: status, message: message, code: code}
}

func newChatErrorWithBody(status int, message, code string, body []byte) *chatError {
	return &chatError{status: status, message: message, code: code, body: body}
}
