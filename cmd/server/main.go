package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/constants"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/events"
	"gcli2api-go/internal/logging"
	monenh "gcli2api-go/internal/monitoring"
	tracing "gcli2api-go/internal/monitoring/tracing"
	srv "gcli2api-go/internal/server"
	usagestats "gcli2api-go/internal/stats"
	store "gcli2api-go/internal/storage"
	"gcli2api-go/internal/translator"
	log "github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	cfg := config.LoadWithFile(*configPath)
	if cfg == nil {
		log.Fatal("Failed to load configuration")
	}
	if *debug {
		cfg.Security.Debug = true
		cfg.SyncFromDomains()
	}

	if err := cfg.ValidateAndExpandPaths(); err != nil {
		log.WithError(err).Fatal("invalid configuration paths")
	}
	if err := logging.Setup(cfg); err != nil {
		log.WithError(err).Fatal("failed to configure logging")
	}

	traceShutdown, err := tracing.Init(context.Background())
	if err != nil {
		log.WithError(err).Warn("failed to initialize tracing")
	}
	if traceShutdown != nil {
		defer func() {
			if err := traceShutdown(context.Background()); err != nil {
				log.WithError(err).Warn("failed to shutdown tracing")
			}
		}()
	}

	// Enforce single upstream provider: gemini (Code Assist)
	up := strings.ToLower(strings.TrimSpace(cfg.Upstream.UpstreamProvider))
	if up != "" && up != "gemini" && up != "code_assist" {
		// 严格限制唯一上游，但不崩溃进程；记录错误并继续以 gemini 模式运行
		log.Errorf("unsupported upstream_provider=%s; forcing 'gemini'", up)
		cfg.Upstream.UpstreamProvider = "gemini"
		cfg.SyncFromDomains()
	}
	log.Infof("Starting GCLI2API-Go (config: %s)", *configPath)

	if strings.TrimSpace(cfg.OAuth.ClientID) == "" || strings.TrimSpace(cfg.OAuth.ClientSecret) == "" {
		log.Warn("OAuth client credentials are not configured; OAuth onboarding features will be unavailable")
	}
	translator.ConfigureSanitizer(cfg.ResponseShaping.SanitizerEnabled, cfg.ResponseShaping.SanitizerPatterns)

	// This build targets Gemini CLI (Code Assist) upstream only.

	// Build credential sources
	var credSources []credential.CredentialSource

	// Add file source (always present for backward compatibility)
	if cfg.Security.AuthDir != "" {
		credSources = append(credSources, credential.NewFileSource(cfg.Security.AuthDir))
	}

	// Add environment variable source if enabled
	if cfg.Execution.AutoLoadEnvCreds {
		envSrc := credential.NewEnvSource()
		credSources = append(credSources, envSrc)
		log.Info("Environment variable credential support enabled (GCLI_CREDS_*)")
	}

	credOpts := credential.Options{
		AuthDir:                    cfg.Security.AuthDir,
		RotationThreshold:          int32(cfg.Execution.CallsPerRotation),
		MaxConcurrentPerCredential: cfg.Execution.MaxConcurrentPerCredential,
		Sources:                    credSources,
		RefreshAheadSeconds:        cfg.OAuth.RefreshAheadSeconds,
		AutoBan: credential.AutoBanConfig{
			Enabled:              cfg.AutoBan.Enabled,
			Threshold429:         cfg.AutoBan.Ban429Threshold,
			Threshold403:         cfg.AutoBan.Ban403Threshold,
			Threshold401:         cfg.AutoBan.Ban401Threshold,
			Threshold5xx:         cfg.AutoBan.Ban5xxThreshold,
			ConsecutiveFailLimit: cfg.AutoBan.ConsecutiveFails,
		},
		AutoRecoveryEnabled:  cfg.AutoBan.RecoveryEnabled,
		AutoRecoveryInterval: time.Duration(cfg.AutoBan.RecoveryIntervalMin) * time.Minute,
	}
	credMgr := credential.NewManager(credOpts)
	eventHub := events.NewHub()
	if cm := config.GetConfigManager(); cm != nil {
		cm.SetEventPublisher(eventHub)
	}
	credMgr.SetEventPublisher(eventHub)
	if cfg.Security.Debug {
		eventHub.Subscribe(events.TopicConfigUpdated, func(_ context.Context, evt events.Event) {
			log.WithField("topic", evt.Topic).Debugf("config event: %v", evt.Payload)
		})
		eventHub.Subscribe(events.TopicCredentialChanged, func(_ context.Context, evt events.Event) {
			log.WithField("topic", evt.Topic).Tracef("credential change: %v", evt.Payload)
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storageBackend, err := buildStorageBackend(ctx, cfg)
	if err != nil {
		// 存储后端初始化失败时降级为文件后端，避免服务无法启动
		log.WithError(err).Warn("Primary storage backend initialization failed; attempting fallback to file backend")

		// 保存原始配置用于日志
		originalBackend := cfg.Storage.Backend

		// 强制回退到本地文件
		cfg.Storage.Backend = "file"
		cfg.SyncFromDomains()
		storageBackend, err = buildStorageBackend(ctx, cfg)
		if err != nil {
			// 文件后端也失败，这是严重问题，但不中断启动
			// 服务可以在无持久化存储的情况下运行（仅使用内存中的凭证）
			log.WithError(err).Error("File backend fallback failed; service will run without persistent storage")
			log.Warn("Credentials will only be loaded from auth directory; storage-based features will be unavailable")
			storageBackend = nil
		} else {
			log.WithFields(log.Fields{
				"original_backend": originalBackend,
				"fallback_backend": "file",
			}).Info("Successfully fell back to file storage backend")
		}
	}
	defer func() {
		if storageBackend != nil {
			_ = storageBackend.Close()
		}
	}()

	// 镜像凭证从存储到本地文件系统
	// 这是一个优化操作，失败不应影响服务启动
	if mirrored, err := mirrorCredentialsFromStorage(ctx, storageBackend, cfg.Security.AuthDir); err != nil {
		// 镜像失败只记录警告，不影响服务启动
		// 凭证仍然可以从存储后端直接读取
		log.WithError(err).Warn("Failed to mirror credentials from storage to local filesystem; credentials will be loaded from storage backend directly")
	} else if mirrored {
		log.Info("Successfully mirrored credentials from storage to local filesystem")
	}

	if err := credMgr.LoadCredentials(); err != nil {
		log.Warnf("Load credentials: %v", err)
	}

	backendLabel := store.DetectBackendLabel(cfg, storageBackend)
	metrics := monenh.NewEnhancedMetrics()
	monenh.SetDefaultMetrics(metrics)
	if storageBackend != nil {
		storageBackend = store.WithInstrumentation(storageBackend, metrics, backendLabel)
	}

	credMgr.WatchAuthDirectory(ctx)

	if storageBackend != nil {
		go startStorageMirror(ctx, storageBackend, cfg.Security.AuthDir, credMgr)
	}

	go credMgr.StartPeriodicRefresh(ctx, constants.CredentialRefreshInterval)
	go credMgr.StartAutoRecovery(ctx)

	usageInterval := time.Duration(cfg.RateLimit.UsageResetIntervalHours) * time.Hour
	usage := usagestats.NewUsageStats(storageBackend, usageInterval, cfg.RateLimit.UsageResetTimezone, cfg.RateLimit.UsageResetHourLocal)

	deps := srv.Dependencies{
		CredentialManager: credMgr,
		UsageStats:        usage,
		Storage:           storageBackend,
		EnhancedMetrics:   metrics,
	}
	openaiEngine, geminiEngine, sharedRouter := srv.BuildEngines(cfg, deps)

	// Restore routing cooldown state from storage if enabled
	if cfg.Routing.PersistState && storageBackend != nil && sharedRouter != nil {
		go restoreRoutingState(ctx, storageBackend, sharedRouter)
		// Start periodic persistence
		go startRoutingStatePersistence(ctx, storageBackend, sharedRouter, time.Duration(cfg.Routing.PersistIntervalSec)*time.Second)
	}

	openaiSrv := &http.Server{Addr: ":" + cfg.Server.OpenAIPort, Handler: openaiEngine}
	// Gemini 原生端口仅在配置非空且不为 "0" 时启动
	var geminiSrv *http.Server
	if strings.TrimSpace(cfg.Server.GeminiPort) != "" && strings.TrimSpace(cfg.Server.GeminiPort) != "0" {
		geminiSrv = &http.Server{Addr: ":" + cfg.Server.GeminiPort, Handler: geminiEngine}
	}

	// Start OpenAI server
	go func() {
		log.Infof("OpenAI API listening on :%s", cfg.Server.OpenAIPort)
		if err := openaiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 关键端口异常不再直接退出；记录错误并等待关停信号
			log.Errorf("openai server: %v", err)
		}
	}()

	// Start Gemini server (optional)
	if geminiSrv != nil {
		go func() {
			log.Infof("Gemini API listening on :%s", cfg.Server.GeminiPort)
			if err := geminiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Errorf("gemini server: %v", err)
			}
		}()
	} else {
		log.Infof("Gemini API disabled (gemini_port unset or 0)")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("Shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), constants.ServerShutdownTimeout)
	defer cancelShutdown()

	// Shutdown both servers gracefully
	go func() { _ = openaiSrv.Shutdown(shutdownCtx) }()
	if geminiSrv != nil {
		go func() { _ = geminiSrv.Shutdown(shutdownCtx) }()
	}

	// Wait a bit for graceful shutdown
	time.Sleep(constants.ServerGracefulWait)
	log.Info("Servers stopped")
}

// Persist routing cooldowns to storage periodically.
