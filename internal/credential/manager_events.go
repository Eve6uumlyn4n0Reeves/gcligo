package credential

import (
	"context"
	"time"

	"gcli2api-go/internal/events"
)

// CredentialSummary captures non-sensitive credential fields for event payloads.
type CredentialSummary struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Source        string    `json:"source,omitempty"`
	Email         string    `json:"email,omitempty"`
	ProjectID     string    `json:"project_id,omitempty"`
	Disabled      bool      `json:"disabled"`
	AutoBanned    bool      `json:"auto_banned"`
	BannedReason  string    `json:"banned_reason,omitempty"`
	SuccessCount  int64     `json:"success_count"`
	FailureCount  int       `json:"failure_count"`
	TotalRequests int64     `json:"total_requests"`
	HealthScore   float64   `json:"health_score"`
	LastSuccess   time.Time `json:"last_success,omitempty"`
	LastFailure   time.Time `json:"last_failure,omitempty"`
}

// CredentialEvent describes a single change to a credential.
type CredentialEvent struct {
	Action     string            `json:"action"`
	Timestamp  time.Time         `json:"timestamp"`
	Credential CredentialSummary `json:"credential"`
}

// CredentialSyncEvent contains a snapshot of all credentials after reload.
type CredentialSyncEvent struct {
	Timestamp   time.Time           `json:"timestamp"`
	Credentials []CredentialSummary `json:"credentials"`
}

func (m *Manager) getPublisher() events.Publisher {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.publisher
}

func (m *Manager) emitCredentialEvent(action string, cred *Credential) {
	publisher := m.getPublisher()
	if publisher == nil || cred == nil {
		return
	}

	summary := summarizeCredential(cred)
	publisher.Publish(
		context.Background(),
		events.TopicCredentialChanged,
		CredentialEvent{
			Action:     action,
			Timestamp:  time.Now().UTC(),
			Credential: summary,
		},
		map[string]string{"credential_id": summary.ID},
	)
}

func (m *Manager) emitCredentialSnapshot(creds []*Credential) {
	publisher := m.getPublisher()
	if publisher == nil {
		return
	}

	summaries := make([]CredentialSummary, 0, len(creds))
	for _, cred := range creds {
		if cred == nil {
			continue
		}
		summaries = append(summaries, summarizeCredential(cred))
	}

	publisher.Publish(
		context.Background(),
		events.TopicCredentialsSynced,
		CredentialSyncEvent{
			Timestamp:   time.Now().UTC(),
			Credentials: summaries,
		},
		nil,
	)
}

func summarizeCredential(cred *Credential) CredentialSummary {
	if cred == nil {
		return CredentialSummary{}
	}
	cred.mu.RLock()
	defer cred.mu.RUnlock()

	return CredentialSummary{
		ID:            cred.ID,
		Type:          cred.Type,
		Source:        cred.Source,
		Email:         cred.Email,
		ProjectID:     cred.ProjectID,
		Disabled:      cred.Disabled,
		AutoBanned:    cred.AutoBanned,
		BannedReason:  cred.BannedReason,
		SuccessCount:  cred.SuccessCount,
		FailureCount:  cred.FailureCount,
		TotalRequests: cred.TotalRequests,
		HealthScore:   cred.HealthScore,
		LastSuccess:   cred.LastSuccess,
		LastFailure:   cred.LastFailure,
	}
}
