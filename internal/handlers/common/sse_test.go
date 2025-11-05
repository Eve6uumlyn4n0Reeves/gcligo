package common

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type flushRecorder struct {
	http.ResponseWriter
	flushed bool
}

func (f *flushRecorder) Flush() { f.flushed = true }

func TestSSEWriteEventAndDone(t *testing.T) {
	rr := httptest.NewRecorder()
	fr := &flushRecorder{ResponseWriter: rr}
	payload := map[string]any{"hello": "world"}
	if err := SSEWriteEvent(fr, fr, "greeting", payload); err != nil {
		t.Fatalf("SSEWriteEvent: %v", err)
	}
	if !fr.flushed {
		t.Fatalf("expected flush after event")
	}
	body := rr.Body.Bytes()
	if !bytes.Contains(body, []byte("event: greeting\n")) || !bytes.Contains(body, []byte("data: {")) {
		t.Fatalf("unexpected body: %s", string(body))
	}
	// done
	if err := SSEWriteDone(fr, fr); err != nil {
		t.Fatalf("SSEWriteDone: %v", err)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("data: [DONE]\n\n")) {
		t.Fatalf("missing DONE marker: %s", rr.Body.String())
	}
}
