package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	feat "gcli2api-go/internal/features"
	common "gcli2api-go/internal/handlers/common"
	mw "gcli2api-go/internal/middleware"
	upstream "gcli2api-go/internal/upstream"
)

func (s *streamSession) prepareSSEHeaders() {
	s.ginCtx.Status(http.StatusOK)
	s.ginCtx.Header("Content-Type", "text/event-stream")
	s.ginCtx.Header("Cache-Control", "no-cache")
	s.ginCtx.Header("Connection", "keep-alive")
}

type streamStats struct {
	sseCount  int
	toolCount int
}

func (s *streamSession) wrapResponseBody(body io.Reader) io.Reader {
	if !s.useAnti {
		return body
	}

	handler := feat.NewStreamHandler(feat.AntiTruncationConfig{MaxAttempts: s.handler.cfg.AntiTruncationMax, Enabled: true})
	contFn := func(cctx context.Context) (io.Reader, error) {
		payload := map[string]any{"model": s.baseModel, "project": s.effProject, "request": s.decoratedReq}
		b, _ := json.Marshal(payload)
		resp, err := s.client.Generate(cctx, b)
		if err != nil {
			return nil, err
		}
		by, err := upstream.ReadAll(resp)
		if err != nil {
			return nil, err
		}
		var obj map[string]any
		if json.Unmarshal(by, &obj) == nil {
			if r, ok := obj["response"]; ok {
				by, _ = json.Marshal(r)
			}
		}
		var buf bytes.Buffer
		buf.WriteString("data: ")
		buf.Write(by)
		buf.WriteString("\n\n")
		buf.WriteString("data: [DONE]\n\n")
		mw.RecordAntiTruncAttempt("gemini", s.path, 1)
		return bytes.NewReader(buf.Bytes()), nil
	}
	if wrapped, err := handler.WrapStream(s.ctx, body, contFn); err == nil {
		return wrapped
	}
	return body
}

func (s *streamSession) pumpStream(reader io.Reader) streamStats {
	writer := s.ginCtx.Writer
	flusher, _ := writer.(http.Flusher)

	stats := streamStats{}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimSpace(line[len("data: "):])
		if bytes.EqualFold(data, []byte("[DONE]")) {
			_ = common.SSEWriteDone(writer, flusher)
			stats.sseCount++
			mw.RecordSSEClose("gemini", s.path, "done")
			break
		}
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err == nil {
			if r, ok := obj["response"]; ok {
				if b, err := json.Marshal(r); err == nil {
					writer.Write([]byte("data: "))
					writer.Write(b)
					writer.Write([]byte("\n\n"))
					if flusher != nil {
						flusher.Flush()
					}
					stats.sseCount++
					if rr, ok := r.(map[string]any); ok {
						stats.toolCount += countFunctionCalls(rr)
					}
					continue
				}
			}
		}
		writer.Write([]byte("data: "))
		writer.Write(data)
		writer.Write([]byte("\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		stats.sseCount++

		var direct map[string]any
		if json.Unmarshal(data, &direct) == nil {
			stats.toolCount += countFunctionCalls(direct)
		}
	}

	return stats
}
