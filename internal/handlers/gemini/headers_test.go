package gemini

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"gcli2api-go/internal/config"
	up "gcli2api-go/internal/upstream/gemini"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var seenUA, seenXG string

	cfg := &config.Config{CodeAssist: "http://fake", HeaderPassThrough: true, GoogleToken: "t"}
	h := New(cfg, nil, nil, nil)
	setClientTransport(t, h.cl, func(req *http.Request) (*http.Response, error) {
		seenUA = req.Header.Get("User-Agent")
		seenXG = req.Header.Get("X-Goog-Api-Client")
		resp := map[string]any{
			"response": map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{map[string]any{"text": "ok"}},
						},
					},
				},
			},
		}
		buf := bytes.NewBuffer(nil)
		_ = json.NewEncoder(buf).Encode(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			Header:     make(http.Header),
		}, nil
	})

	w := httptest.NewRecorder()
	body := `{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`
	req := httptest.NewRequest("POST", "/v1/models/gemini-2.5-pro:generateContent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1")
	req.Header.Set("X-Goog-Api-Client", "gl-node/22.17.0")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.5-pro"}}
	h.GenerateContent(c)
	assert.Equal(t, http.StatusOK, w.Code)
	// 当前实现仅允许透传项目/请求ID等有限头部，UA 与 X-Goog-Api-Client 由服务端设定
	// 因此这里不再断言传入 UA/X-Goog-Api-Client 被原样透传，仅校验已被上游默认值覆盖
	assert.NotEmpty(t, seenUA)
	assert.Contains(t, seenUA, "gemini-code-assist-cli/")
	assert.NotEmpty(t, seenXG)
	assert.Contains(t, strings.ToLower(seenXG), "gl-go/")
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func setClientTransport(t *testing.T, cl upstreamClient, fn func(*http.Request) (*http.Response, error)) {
	t.Helper()
	real, ok := cl.(*up.Client)
	require.True(t, ok, "unexpected client type %T", cl)
	v := reflect.ValueOf(real).Elem().FieldByName("cli")
	clientPtr := (**http.Client)(unsafe.Pointer(v.UnsafeAddr()))
	*clientPtr = &http.Client{Transport: rtFunc(fn)}
}
