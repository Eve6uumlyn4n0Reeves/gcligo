package common

import (
	"encoding/json"
	"net/http"
	"strings"

	apperrors "gcli2api-go/internal/errors"
	"gcli2api-go/internal/httpformat"
	"github.com/gin-gonic/gin"
)

// AbortWithAPIError serializes the provided APIError using the detected error format and aborts the request.
func AbortWithAPIError(c *gin.Context, err *apperrors.APIError) {
	if err == nil {
		err = apperrors.New(http.StatusInternalServerError, "server_error", "server_error", "unknown error")
	}

	format := httpformat.DetectFromContext(c)
	payload, marshalErr := err.ToJSON(format)
	if marshalErr != nil {
		// Fallback: use a minimal OpenAI-compatible envelope.
		fallback := gin.H{
			"error": gin.H{
				"message": err.Message,
				"type":    err.Type,
				"code":    err.Code,
			},
		}
		c.JSON(safeStatus(err.HTTPStatus), fallback)
		c.Abort()
		return
	}

	c.Data(safeStatus(err.HTTPStatus), "application/json", payload)
	c.Abort()
}

// AbortWithError constructs an APIError from the provided fields and aborts the request.
func AbortWithError(c *gin.Context, status int, typ, message string) {
	typ = normalizeType(typ)
	err := apperrors.New(safeStatus(status), typ, typ, firstNonEmpty(message, "internal error"))
	AbortWithAPIError(c, err)
}

// AbortWithUpstreamError attaches upstream payload details (if any) before aborting with the new error system.
func AbortWithUpstreamError(c *gin.Context, status int, typ, message string, upstream []byte) {
	typ = normalizeType(typ)
	err := apperrors.New(safeStatus(status), typ, typ, firstNonEmpty(message, "upstream error"))
	if len(upstream) > 0 {
		if err.Details == nil {
			err.Details = make(map[string]interface{})
		}
		var decoded any
		if json.Unmarshal(upstream, &decoded) == nil {
			err.Details["upstream"] = decoded
		} else {
			err.Details["upstream_raw"] = string(upstream)
		}
	}
	AbortWithAPIError(c, err)
}

// DetectFromContext infers the error response format based on the request path.
func DetectFromContext(c *gin.Context) apperrors.ErrorFormat {
	path := c.FullPath()
	if path == "" && c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	path = strings.ToLower(path)

	switch {
	case strings.Contains(path, "/v1beta/"),
		strings.Contains(path, ":generatecontent"),
		strings.Contains(path, ":streamgeneratecontent"),
		strings.Contains(path, "/v1internal/"):
		return apperrors.FormatGemini
	default:
		return apperrors.FormatOpenAI
	}
}

func normalizeType(typ string) string {
	if strings.TrimSpace(typ) == "" {
		return "server_error"
	}
	return typ
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func safeStatus(status int) int {
	if status >= 400 && status <= 599 {
		return status
	}
	return http.StatusInternalServerError
}
