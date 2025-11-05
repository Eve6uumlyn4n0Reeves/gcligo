package common

import (
	"encoding/json"
	"net/http"
)

// SSEWriteEvent writes an SSE event with the given name and JSON payload.
func SSEWriteEvent(w http.ResponseWriter, flusher http.Flusher, event string, payload any) error {
	if event != "" {
		if _, err := w.Write([]byte("event: " + event + "\n")); err != nil {
			return err
		}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// SSEWriteData writes a generic SSE data line with JSON payload (no event name).
func SSEWriteData(w http.ResponseWriter, flusher http.Flusher, payload any) error {
	return SSEWriteEvent(w, flusher, "", payload)
}

// SSEWriteDone writes the [DONE] marker commonly used for SSE endings.
func SSEWriteDone(w http.ResponseWriter, flusher http.Flusher) error {
	if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}
