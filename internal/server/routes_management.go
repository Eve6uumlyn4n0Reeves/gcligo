package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gcli2api-go/internal/antitrunc"
	"gcli2api-go/internal/config"
	enhmgmt "gcli2api-go/internal/handlers/management"
	"gcli2api-go/internal/logging"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	netx "gcli2api-go/internal/netutil"
	oauth "gcli2api-go/internal/oauth"
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
)

// registerManagementRoutes mounts management API under /routes/api/management and alias /api/management.
func registerManagementRoutes2(root *gin.RouterGroup, cfg *config.Config, deps Dependencies, enhancedHandler *enhmgmt.EnhancedHandler) {
	if deps.CredentialManager == nil {
		return
	}
	mg := root.Group("/routes/api/management")
    mg.Use(managementRemoteGuard("/routes/api/management", cfg))
    if cfg.Security.ManagementReadOnly {
        mg.Use(managementReadOnlyGuard())
    }

	// Create management auth config for read/write separation
	authConfig := NewManagementAuthConfig(cfg)

	// Auth: cookie session or management key/hash or read-only key
	mAuth := mw.AuthConfig{
		AllowMultipleSources: false,
		AcceptCookieName:     "mgmt_session",
		CustomValidator: func(key string) bool {
			if enhancedHandler != nil && enhancedHandler.ValidateToken(key) {
				return true
			}
			// Check admin or read-only key
			return authConfig.ValidateToken(key) != AuthLevelNone
		},
	}
    mg.Use(func(c *gin.Context) {
        p := c.Request.URL.Path
        if c.Request.Method == http.MethodPost && (strings.HasSuffix(p, "/login") || strings.HasSuffix(p, "/logout")) {
            c.Next()
            return
        }
        mw.UnifiedAuth(mAuth)(c)

        // After auth, check if write operation requires admin privileges
        if isWriteOperation(c.Request.Method, c.Request.URL.Path, &cfg.Security) {
            token := ExtractToken(c)
            level := authConfig.ValidateToken(token)
            if level == AuthLevelReadOnly {
                c.JSON(http.StatusForbidden, gin.H{
                    "error": "Read-only access: write operations not permitted",
                })
                c.Abort()
                return
            }
        }
    })

	// Register core admin routes
	if enhancedHandler != nil {
		enhancedHandler.RegisterRoutes(mg)
	}

	// --- Extra management helpers (credential upload/validation, model variants, logs stream) ---
	mg.POST("/credentials", func(c *gin.Context) {
		var req struct {
			Filename    string         `json:"filename"`
			Credentials map[string]any `json:"credentials"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		fname := strings.TrimSpace(req.Filename)
		if fname == "" {
			fname = "credential-" + time.Now().Format("20060102-150405") + ".json"
		}
		if !strings.HasSuffix(strings.ToLower(fname), ".json") {
			fname += ".json"
		}
		if cfg.Security.AuthDir == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_dir not configured"})
			return
		}
		if err := os.MkdirAll(cfg.Security.AuthDir, 0o700); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		data, err := json.MarshalIndent(req.Credentials, "", "  ")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential payload"})
			return
		}
		if err := os.WriteFile(filepath.Join(cfg.Security.AuthDir, fname), data, 0o600); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := persistCredentialMap(c.Request.Context(), deps.Storage, fname, req.Credentials); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist credential to storage"})
			return
		}
		if err := deps.CredentialManager.LoadCredentials(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "uploaded", "filename": fname})
	})
	mg.POST("/credentials/validate", func(c *gin.Context) {
		var req map[string]any
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		// fast shape check
		typ, _ := req["Type"].(string)
		accessToken, _ := req["AccessToken"].(string)
		refreshToken, _ := req["RefreshToken"].(string)
		clientID, _ := req["client_id"].(string)
		clientSecret, _ := req["client_secret"].(string)
		tokenURI, _ := req["token_uri"].(string)
		problems := make([]string, 0)
		if typ == "" {
			problems = append(problems, "missing Type (oauth or api_key)")
		}
		if strings.EqualFold(typ, "oauth") {
			if accessToken == "" && refreshToken == "" {
				problems = append(problems, "oauth requires access_token or refresh_token")
			}
			if refreshToken != "" && (clientID == "" || clientSecret == "" || tokenURI == "") {
				problems = append(problems, "refresh_token provided but client_id/client_secret/token_uri missing")
			}
		}
		if len(problems) > 0 {
			c.JSON(http.StatusOK, gin.H{"valid": false, "problems": problems})
			return
		}
		if accessToken != "" {
			ctx := c.Request.Context()
			om := oauth.NewManager(cfg.OAuth.ClientID, cfg.OAuth.ClientSecret, cfg.OAuth.RedirectURL)
			if ok, err := om.ValidateToken(ctx, accessToken); err == nil {
				c.JSON(http.StatusOK, gin.H{"valid": ok, "problems": problems})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"valid": true, "problems": problems})
	})
	mg.POST("/credentials/validate-batch", func(c *gin.Context) {
		var req struct {
			Items []map[string]any `json:"items"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		results := make([]gin.H, 0, len(req.Items))
		for i, item := range req.Items {
			ok, problems := validateCredentialShape(item)
			results = append(results, gin.H{"index": i, "valid": ok, "problems": problems})
		}
		c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
	})
	mg.POST("/credentials/validate-zip", func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
			return
		}
		r, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer r.Close()
		data, err := io.ReadAll(r)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zip"})
			return
		}
		validateTokens := strings.EqualFold(strings.TrimSpace(c.Query("validate_tokens")), "true")
		tokenOK, tokenFail, tokenTotal := 0, 0, 0
		om := oauth.NewManager(cfg.OAuth.ClientID, cfg.OAuth.ClientSecret, cfg.OAuth.RedirectURL)
		results := make([]gin.H, 0)
		for _, zf := range zr.File {
			if zf.FileInfo().IsDir() {
				continue
			}
			name := strings.TrimSpace(zf.Name)
			if !strings.HasSuffix(strings.ToLower(name), ".json") {
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				results = append(results, gin.H{"file": name, "valid": false, "problems": []string{err.Error()}, "grade": "recoverable"})
				continue
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				results = append(results, gin.H{"file": name, "valid": false, "problems": []string{err.Error()}, "grade": "recoverable"})
				continue
			}
			var obj map[string]any
			if err := json.Unmarshal(content, &obj); err != nil {
				results = append(results, gin.H{"file": name, "valid": false, "problems": []string{"invalid json"}, "grade": "recoverable"})
				continue
			}
			ok, problems := validateCredentialShape(obj)
			if validateTokens {
				if at, _ := obj["AccessToken"].(string); at != "" {
					tokenTotal++
					if good, err := om.ValidateToken(c.Request.Context(), at); err == nil && good {
						tokenOK++
					} else {
						tokenFail++
					}
				}
			}
			grade := "recoverable"
			if !ok {
				grade = "permanent"
			}
			results = append(results, gin.H{"file": name, "valid": ok, "problems": problems, "grade": grade})
		}
		out := gin.H{"results": results, "files": len(results)}
		if validateTokens {
			out["token_checks"] = gin.H{"ok": tokenOK, "fail": tokenFail, "total": tokenTotal}
		}
		c.JSON(http.StatusOK, out)
	})
	mg.POST("/credentials/upload", func(c *gin.Context) {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
			return
		}
		fh, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer fh.Close()
		data, err := io.ReadAll(fh)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if cfg.Security.AuthDir == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_dir not configured"})
			return
		}
		if err := os.MkdirAll(cfg.Security.AuthDir, 0o700); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		lower := strings.ToLower(fileHeader.Filename)
		added, failed := make([]string, 0), make([]string, 0)
		if strings.HasSuffix(lower, ".zip") {
			zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zip"})
				return
			}
			for _, zf := range zr.File {
				if zf.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(zf.Name), ".json") {
					continue
				}
				rc, err := zf.Open()
				if err != nil {
					failed = append(failed, fmt.Sprintf("%s: %v", zf.Name, err))
					continue
				}
				content, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					failed = append(failed, fmt.Sprintf("%s: %v", zf.Name, err))
					continue
				}
				if !json.Valid(content) {
					failed = append(failed, fmt.Sprintf("%s: invalid json", zf.Name))
					continue
				}
				fname := sanitizeCredentialFilename(zf.Name)
				if err := writeCredentialFile(cfg.Security.AuthDir, fname, content); err != nil {
					failed = append(failed, fmt.Sprintf("%s: %v", fname, err))
					continue
				}
				if err := persistCredentialJSON(c.Request.Context(), deps.Storage, fname, content); err != nil {
					_ = os.Remove(filepath.Join(cfg.Security.AuthDir, fname))
					failed = append(failed, fmt.Sprintf("%s: %v", fname, err))
					continue
				}
				added = append(added, fname)
			}
		} else {
			if !json.Valid(data) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
				return
			}
			fname := sanitizeCredentialFilename(fileHeader.Filename)
			if err := writeCredentialFile(cfg.Security.AuthDir, fname, data); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if err := persistCredentialJSON(c.Request.Context(), deps.Storage, fname, data); err != nil {
				_ = os.Remove(filepath.Join(cfg.Security.AuthDir, fname))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist credential to storage"})
				return
			}
			added = append(added, fname)
		}
		if len(added) > 0 {
			_ = deps.CredentialManager.LoadCredentials()
		}
		c.JSON(http.StatusOK, gin.H{"added": added, "errors": failed})
	})

	// Model variant config helpers
	mg.GET("/models/variant-config", func(c *gin.Context) {
		config := models.DefaultVariantConfig()
		if deps.Storage != nil {
			if data, err := deps.Storage.GetConfig(c.Request.Context(), "model_variant_config"); err == nil {
				if configData, ok := data.(map[string]interface{}); ok {
					if jsonBytes, err := json.Marshal(configData); err == nil {
						var stored models.VariantConfig
						if json.Unmarshal(jsonBytes, &stored) == nil {
							config = &stored
						}
					}
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"config": config})
	})
	mg.PUT("/models/variant-config", func(c *gin.Context) {
		var config models.VariantConfig
		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if config.FakeStreamingPrefix == "" {
			config.FakeStreamingPrefix = "假流式/"
		}
		if config.AntiTruncationPrefix == "" {
			config.AntiTruncationPrefix = "流式抗截断/"
		}
		if config.SearchSuffix == "" {
			config.SearchSuffix = "-search"
		}
		if config.ThinkingSuffixes == nil {
			config.ThinkingSuffixes = models.DefaultVariantConfig().ThinkingSuffixes
		}
		if deps.Storage != nil {
			configMap := map[string]interface{}{
				"fake_streaming_prefix":  config.FakeStreamingPrefix,
				"anti_truncation_prefix": config.AntiTruncationPrefix,
				"thinking_suffixes":      config.ThinkingSuffixes,
				"search_suffix":          config.SearchSuffix,
				"custom_prefixes":        config.CustomPrefixes,
				"custom_suffixes":        config.CustomSuffixes,
			}
			if err := deps.Storage.SetConfig(c.Request.Context(), "model_variant_config", configMap); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save config"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "variant config updated", "config": config})
	})
	mg.GET("/models/generate-variants", func(c *gin.Context) {
		cfgv := models.DefaultVariantConfig()
		if deps.Storage != nil {
			if data, err := deps.Storage.GetConfig(c.Request.Context(), "model_variant_config"); err == nil {
				if configData, ok := data.(map[string]interface{}); ok {
					if b, err := json.Marshal(configData); err == nil {
						var stored models.VariantConfig
						if json.Unmarshal(b, &stored) == nil {
							cfgv = &stored
						}
					}
				}
			}
		}
		variants := models.AllVariantsWithConfig(cfgv)
		c.JSON(http.StatusOK, gin.H{"variants": variants, "count": len(variants), "config": cfgv})
	})
	mg.POST("/models/parse-features", func(c *gin.Context) {
		var req struct {
			Models []string `json:"models"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		cfgv := models.DefaultVariantConfig()
		if deps.Storage != nil {
			if data, err := deps.Storage.GetConfig(c.Request.Context(), "model_variant_config"); err == nil {
				if configData, ok := data.(map[string]interface{}); ok {
					if b, err := json.Marshal(configData); err == nil {
						var stored models.VariantConfig
						if json.Unmarshal(b, &stored) == nil {
							cfgv = &stored
						}
					}
				}
			}
		}
		results := make(map[string]models.ModelFeatures)
		for _, m := range req.Models {
			results[m] = models.ParseModelFeaturesWithConfig(m, cfgv)
		}
		c.JSON(http.StatusOK, gin.H{"results": results, "config": cfgv})
	})

	// Logs stream (WS)
	allowedEnv := strings.TrimSpace(os.Getenv("MANAGEMENT_ALLOWED_ORIGINS"))
	var allowed []string
	if allowedEnv != "" {
		for _, p := range strings.Split(allowedEnv, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				allowed = append(allowed, p)
			}
		}
	}
	upgrader := ws.Upgrader{CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := neturl.Parse(origin)
		if err != nil {
			return false
		}
		if strings.EqualFold(u.Host, r.Host) {
			return true
		}
		for _, a := range allowed {
			if strings.EqualFold(a, origin) {
				return true
			}
			if au, err2 := neturl.Parse(a); err2 == nil && au.Host != "" {
				if strings.EqualFold(au.Host, u.Host) {
					return true
				}
			} else if strings.EqualFold(a, u.Host) {
				return true
			}
		}
		return false
	}}
	mg.GET("/logs/stream", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		// Try to add client (may fail if max connections reached)
		if err := logging.GetWSLogger().AddClient(conn); err != nil {
			_ = conn.WriteJSON(map[string]string{
				"error": "Maximum connections reached",
			})
			conn.Close()
			c.Status(http.StatusServiceUnavailable)
			return
		}

		// Set read deadline and pong handler
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			return nil
		})

		// Start ping ticker
		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := conn.WriteControl(ws.PingMessage, []byte("ping"), time.Now().Add(10*time.Second)); err != nil {
						return
					}
				case <-done:
					return
				}
			}
		}()

		// Read loop (keeps connection alive)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				close(done)
				logging.GetWSLogger().RemoveClient(conn)
				break
			}
		}
	})

	// Alias: redirect /api/management/* -> /routes/api/management/* (preserve method via 307)
	alias := root.Group("/api/management")
	alias.Any("/*path", func(c *gin.Context) {
		target := "/routes/api/management" + c.Param("path")
		if q := c.Request.URL.RawQuery; q != "" {
			target = target + "?" + q
		}
		mw.RecordManagementAccess("/api/management", "redirect", netx.ClassifyClientSource(netx.ExtractIPFromRequest(c.Request)))
		c.Redirect(http.StatusTemporaryRedirect, target)
	})

	// Anti-truncation dry-run endpoint
	mg.POST("/antitrunc/dry-run", func(c *gin.Context) {
		var req antitrunc.DryRunRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		// Perform dry-run
		resp, err := antitrunc.DryRun(&req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check for debug header
		debugHeader := c.GetHeader("X-Debug-Antitrunc")
		if debugHeader != "" {
			// Include additional debug information
			c.JSON(http.StatusOK, gin.H{
				"result":       resp,
				"debug_header": debugHeader,
				"request_info": gin.H{
					"has_text":    req.Text != "",
					"has_payload": len(req.Payload) > 0,
					"rules_count": len(req.Rules),
				},
			})
			return
		}

		c.JSON(http.StatusOK, resp)
	})

	// Assembly endpoints under the same management group
	registerAssemblyRoutes(mg, cfg, deps)
}

// isWriteOperation 统一判定管理端“写操作”。
// 规则：
// 1) 非 GET/HEAD/OPTIONS 一律视为写；
// 2) 对只读方法，若命中 Blocklist 视为写；若同时命中 Allowlist 则视为读（Allowlist 优先）。
func isWriteOperation(method, path string, sec *config.SecurityConfig) bool {
    m := strings.ToUpper(strings.TrimSpace(method))
    // 明确的写方法
    switch m {
    case http.MethodGet, http.MethodHead, http.MethodOptions:
        // 继续走路径级判定
    default:
        return true
    }
    // 路径级判定（可选配置）
    p := strings.TrimSpace(strings.ToLower(path))
    if p == "" || sec == nil {
        return false
    }
    // Allowlist 优先
    if matchAny(p, sec.ManagementWritePathAllowlist) {
        return false
    }
    if matchAny(p, sec.ManagementWritePathBlocklist) {
        return true
    }
    return false
}

// matchAny 支持三种匹配：
// - 精确相等
// - 前缀匹配：以 "prefix*" 结尾
// - 后缀匹配：以 "*suffix" 开头
func matchAny(path string, patterns []string) bool {
    if len(patterns) == 0 {
        return false
    }
    for _, raw := range patterns {
        s := strings.ToLower(strings.TrimSpace(raw))
        if s == "" {
            continue
        }
        if s == path {
            return true
        }
        if strings.HasSuffix(s, "*") {
            // prefix*
            prefix := strings.TrimSuffix(s, "*")
            if strings.HasPrefix(path, prefix) {
                return true
            }
        } else if strings.HasPrefix(s, "*") {
            // *suffix
            suffix := strings.TrimPrefix(s, "*")
            if strings.HasSuffix(path, suffix) {
                return true
            }
        }
    }
    return false
}

func validateCredentialShape(req map[string]any) (bool, []string) {
	typ, _ := req["Type"].(string)
	accessToken, _ := req["AccessToken"].(string)
	refreshToken, _ := req["RefreshToken"].(string)
	clientID, _ := req["client_id"].(string)
	clientSecret, _ := req["client_secret"].(string)
	tokenURI, _ := req["token_uri"].(string)

	problems := make([]string, 0)
	if strings.TrimSpace(typ) == "" {
		problems = append(problems, "missing Type (oauth or api_key)")
	}
	if strings.EqualFold(strings.TrimSpace(typ), "oauth") {
		if strings.TrimSpace(accessToken) == "" && strings.TrimSpace(refreshToken) == "" {
			problems = append(problems, "oauth requires access_token or refresh_token")
		}
		if strings.TrimSpace(refreshToken) != "" && (strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" || strings.TrimSpace(tokenURI) == "") {
			problems = append(problems, "refresh_token provided but client_id/client_secret/token_uri missing")
		}
	}
	return len(problems) == 0, problems
}
