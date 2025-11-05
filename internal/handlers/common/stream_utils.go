package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"gcli2api-go/internal/constants"
	"github.com/gin-gonic/gin"
)

// SSEEvent represents a parsed SSE payload.
type SSEEvent struct {
	Raw  []byte
	Data map[string]any
}

// SSEScanner iterates over SSE events from an upstream stream.
type SSEScanner struct {
	scanner *bufio.Scanner
}

// NewSSEScanner creates a scanner with standard buffer settings.
func NewSSEScanner(r io.Reader) *SSEScanner {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, constants.SSEScannerInitialBufferSize)
	scanner.Buffer(buf, constants.SSEScannerMaxBufferSize)
	return &SSEScanner{scanner: scanner}
}

// PrepareSSE sets standard headers for SSE and returns writer/ flusher pair.
func PrepareSSE(c *gin.Context) (gin.ResponseWriter, http.Flusher) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	w := c.Writer
	fl, _ := w.(http.Flusher)
	return w, fl
}

// Next returns the next SSE event. When done is true, the stream finished.
func (s *SSEScanner) Next() (*SSEEvent, bool, error) {
	for s.scanner.Scan() {
		line := s.scanner.Bytes()
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimSpace(line[len("data: "):])
		if bytes.EqualFold(data, []byte("[DONE]")) {
			return nil, true, nil
		}
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			continue
		}
		return &SSEEvent{Raw: append([]byte(nil), data...), Data: obj}, false, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}
