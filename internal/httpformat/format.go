package httpformat

import (
	"net/http"
	"strings"

	apperrors "gcli2api-go/internal/errors"
	"github.com/gin-gonic/gin"
)

// DetectFromContext determines the error format based on the gin context path.
func DetectFromContext(c *gin.Context) apperrors.ErrorFormat {
	if c == nil {
		return apperrors.FormatOpenAI
	}
	if path := c.FullPath(); path != "" {
		return DetectFromPath(path)
	}
	return DetectFromRequest(c.Request)
}

// DetectFromRequest determines the error format using an HTTP request.
func DetectFromRequest(r *http.Request) apperrors.ErrorFormat {
	if r == nil || r.URL == nil {
		return apperrors.FormatOpenAI
	}
	return DetectFromPath(r.URL.Path)
}

// DetectFromPath determines the error format based on a raw path string.
func DetectFromPath(path string) apperrors.ErrorFormat {
	path = strings.ToLower(path)
	if strings.Contains(path, "/v1beta/") ||
		strings.Contains(path, ":generatecontent") ||
		strings.Contains(path, ":streamgeneratecontent") ||
		strings.Contains(path, "/v1internal/") {
		return apperrors.FormatGemini
	}
	return apperrors.FormatOpenAI
}
