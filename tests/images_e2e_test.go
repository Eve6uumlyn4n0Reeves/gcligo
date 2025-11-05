package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	oh "gcli2api-go/internal/handlers/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// e2e: /v1/images/generations should yield equivalent responses
// for alias model (nano-banana*) and target model (gemini-2.5-flash-image-preview).
func TestImagesGenerations_AliasEquivalence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Stub upstream Code Assist endpoint
	upstream := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with a minimal Gemini-style image response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io := map[string]any{
			"response": map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{
									"inlineData": map[string]any{
										"mimeType": "image/png",
										"data":     "QUJD", // base64("ABC")
									},
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(io)
	}))
	defer upstream.Close()

	cfg := &config.Config{CodeAssist: upstream.URL}
	h := oh.New(cfg, nil, nil, nil, nil)

	r := gin.New()
	r.POST("/v1/images/generations", h.ImagesGenerations)

	// Request using alias model
	bodyAlias := map[string]any{
		"model":  "nano-banana",
		"prompt": "test",
		"n":      1,
		"size":   "1024x1024",
	}
	b1, _ := json.Marshal(bodyAlias)
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/v1/images/generations", bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Request using target model
	bodyTarget := map[string]any{
		"model":  "gemini-2.5-flash-image-preview",
		"prompt": "test",
		"n":      1,
		"size":   "1024x1024",
	}
	b2, _ := json.Marshal(bodyTarget)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/v1/images/generations", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var out1, out2 map[string]any
	_ = json.Unmarshal(w1.Body.Bytes(), &out1)
	_ = json.Unmarshal(w2.Body.Bytes(), &out2)
	assert.Equal(t, out2, out1, "alias and target model responses should be identical")
}
