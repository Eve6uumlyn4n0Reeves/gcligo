package upstream

import (
	"context"
	"io"
	"net/http"
)

type ctxKey int

const (
	ctxHeaders ctxKey = iota
)

// WithHeaderOverrides 将请求中的 Header 附着到 context 中，供上游实现选择性透传。
func WithHeaderOverrides(ctx context.Context, hdr http.Header) context.Context {
	if hdr == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxHeaders, hdr)
}

// HeaderOverrides 从 context 中读取 Header 附加信息。
func HeaderOverrides(ctx context.Context) http.Header {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(ctxHeaders); v != nil {
		if h, ok := v.(http.Header); ok {
			return h
		}
	}
	return nil
}

// ReadAll 读取并返回响应体，读取完成后自动关闭。
func ReadAll(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
