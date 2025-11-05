package credential

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// BatchOperationResult captures the outcome of a batch credential mutation.
type BatchOperationResult struct {
	ID      string
	Success bool
	Err     error
}

// ErrorMessage returns the string form of the stored error (if any).
func (r BatchOperationResult) ErrorMessage() string {
	if r.Err == nil {
		return ""
	}
	return r.Err.Error()
}

// BatchEnableCredentials flips Disabled=false for every provided credential id.
func (m *Manager) BatchEnableCredentials(ctx context.Context, credIDs []string) []BatchOperationResult {
	results := make([]BatchOperationResult, len(credIDs))
	targets := make([]*Credential, 0, len(credIDs))

	m.mu.Lock()
	for i, id := range credIDs {
		cred := m.findCredentialLocked(id)
		if cred == nil {
			results[i] = BatchOperationResult{ID: id, Err: fmt.Errorf("credential %s not found", id)}
			continue
		}
		cred.Disabled = false
		cred.FailureCount = 0
		targets = append(targets, cred)
		results[i] = BatchOperationResult{ID: id, Success: true}
	}
	m.mu.Unlock()

	for _, cred := range targets {
		m.persistCredentialState(cred, true)
		m.emitCredentialEvent("enabled", cred.Clone())
	}

	log.Infof("Batch enable completed: total=%d success=%d failure=%d", len(credIDs), countBatchSuccess(results), countBatchFailures(results))
	return results
}

// BatchDisableCredentials flips Disabled=true for every provided credential id.
func (m *Manager) BatchDisableCredentials(ctx context.Context, credIDs []string) []BatchOperationResult {
	results := make([]BatchOperationResult, len(credIDs))
	targets := make([]*Credential, 0, len(credIDs))

	m.mu.Lock()
	for i, id := range credIDs {
		cred := m.findCredentialLocked(id)
		if cred == nil {
			results[i] = BatchOperationResult{ID: id, Err: fmt.Errorf("credential %s not found", id)}
			continue
		}
		cred.Disabled = true
		targets = append(targets, cred)
		results[i] = BatchOperationResult{ID: id, Success: true}
	}
	m.mu.Unlock()

	for _, cred := range targets {
		m.persistCredentialState(cred, true)
		m.emitCredentialEvent("disabled", cred.Clone())
	}

	log.Infof("Batch disable completed: total=%d success=%d failure=%d", len(credIDs), countBatchSuccess(results), countBatchFailures(results))
	return results
}

// BatchDeleteCredentials removes credentials and backing files for every id.
func (m *Manager) BatchDeleteCredentials(ctx context.Context, credIDs []string) []BatchOperationResult {
	results := make([]BatchOperationResult, len(credIDs))
	for i, id := range credIDs {
		err := m.DeleteCredential(id)
		results[i] = BatchOperationResult{
			ID:      id,
			Success: err == nil,
			Err:     err,
		}
	}
	log.Infof("Batch delete completed: total=%d success=%d failure=%d", len(credIDs), countBatchSuccess(results), countBatchFailures(results))
	return results
}

// BatchRecoverCredentials clears ban/disable state for the provided ids.
func (m *Manager) BatchRecoverCredentials(ctx context.Context, credIDs []string) []BatchOperationResult {
	results := make([]BatchOperationResult, len(credIDs))
	for i, id := range credIDs {
		if ctx != nil && ctx.Err() != nil {
			results[i] = BatchOperationResult{
				ID:  id,
				Err: ctx.Err(),
			}
			continue
		}
		err := m.recoverCredential(ctx, id)
		results[i] = BatchOperationResult{
			ID:      id,
			Success: err == nil,
			Err:     err,
		}
	}
	log.Infof("Batch recover completed: total=%d success=%d failure=%d", len(credIDs), countBatchSuccess(results), countBatchFailures(results))
	return results
}

func (m *Manager) findCredentialLocked(id string) *Credential {
	for _, cred := range m.credentials {
		if cred != nil && cred.ID == id {
			return cred
		}
	}
	return nil
}

func countBatchSuccess(results []BatchOperationResult) int {
	count := 0
	for _, r := range results {
		if r.Success {
			count++
		}
	}
	return count
}

func countBatchFailures(results []BatchOperationResult) int {
	return len(results) - countBatchSuccess(results)
}
