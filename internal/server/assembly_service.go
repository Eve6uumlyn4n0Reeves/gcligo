package server

import (
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/monitoring"
	store "gcli2api-go/internal/storage"
	route "gcli2api-go/internal/upstream/strategy"
)

type AssemblyService struct {
	cfg          *config.Config
	st           store.Backend
	metrics      *monitoring.EnhancedMetrics
	backendLabel string
	strategy     *route.Strategy
}

type AssemblyAudit struct {
	ActorLabel string
	ActorID    string
	Reason     string
}

func (a AssemblyAudit) metricsActor() string {
	if a.ActorLabel == "" {
		return "unknown"
	}
	return a.ActorLabel
}

func NewAssemblyService(cfg *config.Config, st store.Backend, metrics *monitoring.EnhancedMetrics, strategy *route.Strategy) *AssemblyService {
	return &AssemblyService{
		cfg:          cfg,
		st:           st,
		metrics:      metrics,
		backendLabel: store.DetectBackendLabel(cfg, st),
		strategy:     strategy,
	}
}

func (s *AssemblyService) recordTxAttempt() {
	if s.metrics != nil {
		s.metrics.RecordTransactionAttempt(s.backendLabel)
	}
}

func (s *AssemblyService) recordTxCommit() {
	if s.metrics != nil {
		s.metrics.RecordTransactionCommit(s.backendLabel)
	}
}

func (s *AssemblyService) recordTxFailure() {
	if s.metrics != nil {
		s.metrics.RecordTransactionFailure(s.backendLabel)
	}
}

func (s *AssemblyService) recordPlanApply(stage, status string, duration time.Duration) {
	if s.metrics != nil {
		s.metrics.RecordPlanApply(s.backendLabel, stage, status, duration)
	}
}

func (s *AssemblyService) RecordOperation(action, status string, audit AssemblyAudit) {
	actor := audit.metricsActor()
	monitoring.AssemblyOperationsTotal.WithLabelValues(action, status, actor).Inc()
}
