package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImagesGenerations_Success(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		GoogleProjID:            "proj-123",
		OpenAIImagesIncludeMIME: true,
		RoutingDebugHeaders:     true,
		AutoImagePlaceholder:    true,
	}

	stub := &stubGeminiClient{
		generateFunc: func(ctx context.Context, payload []byte) (*http.Response, error) {
			var req map[string]any
			require.NoError(t, json.Unmarshal(payload, &req))

			t.Logf("images payload: %#v", req)
			require.Equal(t, "gemini-2.5-flash-image-preview", req["model"])
			require.Equal(t, "proj-123", req["project"])

			request, ok := req["request"].(map[string]any)
			require.True(t, ok)

			genCfg, ok := request["generationConfig"].(map[string]any)
			require.True(t, ok)
			require.EqualValues(t, 2, genCfg["candidateCount"])
			respMods, ok := genCfg["responseModalities"].([]any)
			require.True(t, ok)
			require.Contains(t, respMods, "Image")

			if imageCfg, hasImageCfg := genCfg["imageConfig"].(map[string]any); hasImageCfg {
				require.Equal(t, "16:9", imageCfg["aspectRatio"])
			}

			contents, ok := request["contents"].([]any)
			require.True(t, ok)
			require.NotEmpty(t, contents)

			var promptFound bool
			for _, cc := range contents {
				if partWrap, ok := cc.(map[string]any); ok {
					if parts, ok := partWrap["parts"].([]any); ok {
						for _, pp := range parts {
							if part, ok := pp.(map[string]any); ok {
								if _, ok := part["inlineData"]; ok {
									promptFound = true
								}
								if txt, ok := part["text"].(string); ok && strings.Contains(txt, "draw cat") {
									promptFound = true
								}
							}
						}
					}
				}
			}
			require.True(t, promptFound, "expected prompt or inline data part in contents")

			respObj := map[string]any{
				"response": map[string]any{
					"candidates": []any{
						map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{"inlineData": map[string]any{"mimeType": "image/jpeg", "data": "BASE64JPEG"}},
								},
							},
						},
					},
				},
			}
			raw, _ := json.Marshal(respObj)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(raw)),
				Header:     make(http.Header),
			}, nil
		},
	}

	handler := &Handler{
		cfg:         cfg,
		baseClient:  stub,
		clientCache: make(map[string]geminiClient),
	}

	router := gin.New()
	router.POST("/v1/images/generations", handler.ImagesGenerations)

	body := map[string]any{
		"prompt": "draw cat",
		"size":   "1280x720",
		"n":      2,
		"model":  "gemini-2.5-flash-image-preview",
	}
	w := postJSON(t, router, "/v1/images/generations", body)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	first := data[0].(map[string]any)
	require.Equal(t, "BASE64JPEG", first["b64_json"])
	require.Equal(t, "image/jpeg", first["mime_type"])
}

func TestImagesGenerations_UnsupportedModel(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	handler := &Handler{
		cfg:         &config.Config{},
		baseClient:  &stubGeminiClient{},
		clientCache: make(map[string]geminiClient),
	}
	router := gin.New()
	router.POST("/v1/images/generations", handler.ImagesGenerations)

	body := map[string]any{
		"prompt": "draw cat",
		"model":  "dall-e-3",
	}
	w := postJSON(t, router, "/v1/images/generations", body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "unsupported image model")
}

func TestImagesGenerations_InvalidJSON(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	handler := &Handler{
		cfg:         &config.Config{},
		baseClient:  &stubGeminiClient{},
		clientCache: make(map[string]geminiClient),
	}
	router := gin.New()
	router.POST("/v1/images/generations", handler.ImagesGenerations)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid json")
}
