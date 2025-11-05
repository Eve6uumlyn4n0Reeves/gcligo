package tests

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"gcli2api-go/internal/config"
	enh "gcli2api-go/internal/handlers/management"
	"gcli2api-go/internal/models"
)

func TestRegistryImport_MultipartAppendAndReplace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := newTempFileBackend(t)
	cfg := &config.Config{}
	h := enh.NewAdminAPIHandler(cfg, nil, nil, nil, st)
	r := gin.New()
	grp := r.Group("/routes/api/management")
	h.RegisterRoutes(grp)

	// Build multipart with two files, same model to test dedup
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	entry := models.RegistryEntry{Base: "gemini-2.5-pro", Enabled: true, Upstream: "code_assist"}
	raw1, _ := json.Marshal(entry)
	w1, _ := mw.CreateFormFile("files", "a.json")
	_, _ = w1.Write(raw1)
	// duplicate
	raw2, _ := json.Marshal(entry)
	w2, _ := mw.CreateFormFile("files", "b.json")
	_, _ = w2.Write(raw2)
	_ = mw.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/routes/api/management/models/openai/registry/import?mode=append", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// verify registry has exactly one after dedup
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/routes/api/management/models/openai/registry", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	arr, _ := got["models"].([]any)
	require.Equal(t, 1, len(arr))

	// Now replace with a different entry using single "file"
	var buf2 bytes.Buffer
	mw2 := multipart.NewWriter(&buf2)
	entry2 := models.RegistryEntry{Base: "gemini-2.5-flash", Enabled: true, Upstream: "code_assist"}
	raw3, _ := json.Marshal(entry2)
	fw, _ := mw2.CreateFormFile("file", "one.json")
	_, _ = fw.Write(raw3)
	_ = mw2.Close()

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/routes/api/management/models/openai/registry/import?mode=replace", &buf2)
	req.Header.Set("Content-Type", mw2.FormDataContentType())
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// verify replaced
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/routes/api/management/models/openai/registry", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	arr, _ = got["models"].([]any)
	require.Equal(t, 1, len(arr))
	m0, _ := arr[0].(map[string]any)
	// export path returns the stored object; verify base is flash
	require.Equal(t, "gemini-2.5-flash", m0["base"])
}

// 注：newTempFileBackend 已在 registry_test.go 中定义为同包函数，这里复用。
