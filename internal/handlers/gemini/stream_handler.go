package gemini

import (
	"github.com/gin-gonic/gin"
)

// StreamGenerateContent bridges Gemini streaming responses to SSE.
func (h *Handler) StreamGenerateContent(c *gin.Context) {
	session, handled := newStreamSession(h, c)
	if handled {
		return
	}
	session.execute()
}
