package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gcli2api-go/internal/models"
	tracing "gcli2api-go/internal/monitoring/tracing"
	store "gcli2api-go/internal/storage"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ApplyPlan applies a saved plan. If transaction is supported, performs atomically.
func (s *AssemblyService) ApplyPlan(ctx context.Context, name string) error {
	if s.st == nil {
		return &store.ErrNotSupported{Operation: "apply plan"}
	}
	key := "assembly_plan:" + sanitizePlanName(name)
	v, err := s.st.GetConfig(ctx, key)
	if err != nil || v == nil {
		return err
	}
	plan, _ := v.(map[string]any)

	curOA := models.ActiveEntriesByChannel(s.cfg, s.st, "openai")
	curGM := models.ActiveEntriesByChannel(s.cfg, s.st, "gemini")

	curVar := map[string]any{}
	if s.st != nil {
		if v, err := s.st.GetConfig(ctx, "model_variant_config"); err == nil {
			if vm, ok := v.(map[string]any); ok {
				curVar = vm
			}
		}
	}

	_ = s.st.SetConfig(ctx, "assembly_plan_backup:"+sanitizePlanName(name), map[string]any{
		"ts": time.Now().Unix(), "models_openai": curOA, "models_gemini": curGM, "variant_config": curVar,
	})

	idKeyApply := fmt.Sprintf("%s:apply:%s", s.backendLabel, sanitizePlanName(name))
	if tx, err := s.st.BeginTransaction(ctx); err == nil && tx != nil {
		s.recordTxAttempt()
		if err := s.applyWithSetter(ctx, plan, tx, "apply", idKeyApply); err != nil {
			_ = tx.Rollback(ctx)
			s.recordTxFailure()
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			s.recordTxFailure()
			return err
		}
		s.recordTxCommit()
		return nil
	}
	return s.applyWithSetter(ctx, plan, s.st, "apply", idKeyApply)
}

func (s *AssemblyService) applyWithSetter(ctx context.Context, plan map[string]any, setter interface {
	SetConfig(context.Context, string, interface{}) error
}, stage string, idKey string) error {
	updates, err := updatesFromPlan(plan)
	if err != nil {
		return err
	}
	return s.applyUpdates(ctx, setter, updates, stage, idKey)
}

func (s *AssemblyService) RollbackPlan(ctx context.Context, name string) error {
	if s.st == nil {
		return &store.ErrNotSupported{Operation: "rollback plan"}
	}
	key := "assembly_plan_backup:" + sanitizePlanName(name)
	v, err := s.st.GetConfig(ctx, key)
	if err != nil || v == nil {
		return err
	}
	plan, _ := v.(map[string]any)

	idKeyRollback := fmt.Sprintf("%s:rollback:%s", s.backendLabel, sanitizePlanName(name))
	if tx, err := s.st.BeginTransaction(ctx); err == nil && tx != nil {
		s.recordTxAttempt()
		if err := s.applyBackupWithSetter(ctx, plan, tx, "rollback", idKeyRollback); err != nil {
			_ = tx.Rollback(ctx)
			s.recordTxFailure()
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			s.recordTxFailure()
			return err
		}
		s.recordTxCommit()
		return nil
	}
	return s.applyBackupWithSetter(ctx, plan, s.st, "rollback", idKeyRollback)
}

func (s *AssemblyService) applyBackupWithSetter(ctx context.Context, plan map[string]any, setter interface {
	SetConfig(context.Context, string, interface{}) error
}, stage string, idKey string) error {
	updates, err := updatesFromBackup(plan)
	if err != nil {
		return err
	}
	return s.applyUpdates(ctx, setter, updates, stage, idKey)
}

func (s *AssemblyService) applyUpdates(ctx context.Context, setter interface {
	SetConfig(context.Context, string, interface{}) error
}, updates []configUpdate, stage string, idKey string) error {
	if backend, ok := setter.(store.Backend); ok {
		return s.applyUpdatesWithBackend(ctx, backend, updates, stage, idKey)
	}
	for _, upd := range updates {
		if err := setter.SetConfig(ctx, upd.key, upd.value); err != nil {
			return err
		}
	}
	return nil
}

func (s *AssemblyService) applyUpdatesWithBackend(ctx context.Context, backend store.Backend, updates []configUpdate, stage string, idKey string) error {
	history := make([]priorSnapshot, 0, len(updates))

	for _, upd := range updates {
		prev, err := backend.GetConfig(ctx, upd.key)
		exists := true
		if err != nil {
			if _, ok := err.(*store.ErrNotFound); ok {
				exists = false
			} else {
				return err
			}
		}
		history = append(history, priorSnapshot{key: upd.key, value: prev, exists: exists})
	}

	stageLabel := strings.TrimSpace(stage)
	if stageLabel == "" {
		stageLabel = "apply"
	}

	if applier, ok := backend.(store.ConfigBatchApplier); ok && strings.TrimSpace(idKey) != "" {
		mutations := make([]store.ConfigMutation, 0, len(updates))
		for _, upd := range updates {
			mutations = append(mutations, store.ConfigMutation{
				Key:    upd.key,
				Value:  upd.value,
				Delete: false,
			})
		}
		opts := store.BatchApplyOptions{
			IdempotencyKey: idKey,
			TTL:            2 * time.Minute,
			Stage:          stageLabel,
		}

		start := time.Now()
		ctxSpan, span := tracing.StartSpan(ctx, "assembly", "PlanApply", trace.WithSpanKind(trace.SpanKindInternal))
		span.SetAttributes(
			attribute.String("assembly.backend", s.backendLabel),
			attribute.String("assembly.stage", stageLabel),
			attribute.String("assembly.idempotency_key", idKey),
			attribute.Int("assembly.mutation_count", len(mutations)),
		)

		statusLabel := "success"
		err := applier.ApplyConfigBatch(ctxSpan, mutations, opts)
		duration := time.Since(start)
		if err != nil {
			statusLabel = "error"
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
		s.recordPlanApply(stageLabel, statusLabel, duration)

		if err != nil {
			return err
		}
		return nil
	}

	for idx, upd := range updates {
		if err := backend.SetConfig(ctx, upd.key, upd.value); err != nil {
			s.restoreConfigs(ctx, backend, history[:idx])
			return err
		}
	}
	return nil
}

func (s *AssemblyService) restoreConfigs(ctx context.Context, backend store.Backend, history []priorSnapshot) {
	for i := len(history) - 1; i >= 0; i-- {
		entry := history[i]
		if !entry.exists {
			_ = backend.DeleteConfig(ctx, entry.key)
			continue
		}
		_ = backend.SetConfig(ctx, entry.key, entry.value)
	}
}
