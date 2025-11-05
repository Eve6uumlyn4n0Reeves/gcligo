package adapter

import (
	"context"
	"time"
)

// CredentialState 表示凭证的当前状态
type CredentialState struct {
	ID              string                 `json:"id" yaml:"id"`
	Disabled        bool                   `json:"disabled" yaml:"disabled"`
	LastUsed        *time.Time             `json:"last_used,omitempty" yaml:"last_used,omitempty"`
	LastSuccess     *time.Time             `json:"last_success,omitempty" yaml:"last_success,omitempty"`
	LastFailure     *time.Time             `json:"last_failure,omitempty" yaml:"last_failure,omitempty"`
	FailureCount    int                    `json:"failure_count" yaml:"failure_count"`
	SuccessCount    int                    `json:"success_count" yaml:"success_count"`
	FailureReason   string                 `json:"failure_reason,omitempty" yaml:"failure_reason,omitempty"`
	HealthScore     float64                `json:"health_score" yaml:"health_score"`
	UsageStats      map[string]interface{} `json:"usage_stats,omitempty" yaml:"usage_stats,omitempty"`
	ErrorRate       float64                `json:"error_rate" yaml:"error_rate"`
	AvgResponseTime time.Duration          `json:"avg_response_time" yaml:"avg_response_time"`
	CreatedAt       time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" yaml:"updated_at"`
}

// Credential 表示凭证信息
type Credential struct {
	ID           string                 `json:"id" yaml:"id"`
	Name         string                 `json:"name" yaml:"name"`
	Type         string                 `json:"type" yaml:"type"` // oauth, api_key, service_account
	Token        string                 `json:"-" yaml:"-"`       // 不序列化到文件
	RefreshToken string                 `json:"-" yaml:"-"`       // 不序列化到文件
	AccessToken  string                 `json:"access_token,omitempty" yaml:"access_token,omitempty"`
	APIKey       string                 `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	ClientID     string                 `json:"client_id,omitempty" yaml:"client_id,omitempty"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	FilePath     string                 `json:"file_path,omitempty" yaml:"file_path,omitempty"`
	State        *CredentialState       `json:"state,omitempty" yaml:"state,omitempty"`
}

// StorageAdapter 统一的存储适配器接口
type StorageAdapter interface {
	// 基础 CRUD 操作
	StoreCredential(ctx context.Context, cred *Credential) error
	LoadCredential(ctx context.Context, id string) (*Credential, error)
	DeleteCredential(ctx context.Context, id string) error
	UpdateCredential(ctx context.Context, cred *Credential) error

	// 批量操作
	GetAllCredentials(ctx context.Context) ([]*Credential, error)
	GetAllCredentialStates(ctx context.Context) (map[string]*CredentialState, error)
	UpdateCredentialStates(ctx context.Context, states map[string]*CredentialState) error

	// 凭证发现和管理
	DiscoverCredentials(ctx context.Context) ([]*Credential, error)
	RefreshCredential(ctx context.Context, credID string) (*Credential, error)

	// 使用统计和分析
	UpdateUsageStats(ctx context.Context, credID string, stats map[string]interface{}) error
	GetUsageStats(ctx context.Context, credID string) (map[string]interface{}, error)
	GetUsageStatsSummary(ctx context.Context) (map[string]interface{}, error)

	// 批量状态管理
	EnableCredentials(ctx context.Context, credIDs []string) error
	DisableCredentials(ctx context.Context, credIDs []string) error
	DeleteCredentials(ctx context.Context, credIDs []string) error

	// 监控和健康检查
	GetHealthyCredentials(ctx context.Context) ([]*Credential, error)
	GetUnhealthyCredentials(ctx context.Context) ([]*Credential, error)
	ValidateCredential(ctx context.Context, cred *Credential) error

	// 存储后端特定操作
	Ping(ctx context.Context) error
	Close() error

	// 配置和元数据
	GetConfig() map[string]interface{}
	SetConfig(ctx context.Context, config map[string]interface{}) error
}

// CredentialFilter 凭证过滤器
type CredentialFilter struct {
	Disabled  *bool      `json:"disabled,omitempty"`
	Type      string     `json:"type,omitempty"`
	MinHealth *float64   `json:"min_health,omitempty"`
	MaxHealth *float64   `json:"max_health,omitempty"`
	MaxError  *float64   `json:"max_error,omitempty"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

// ApplyFilter 应用过滤器
func ApplyFilter(credentials []*Credential, filter *CredentialFilter) []*Credential {
	if filter == nil {
		return credentials
	}

	var result []*Credential
	for _, cred := range credentials {
		if filter.Disabled != nil && cred.State.Disabled != *filter.Disabled {
			continue
		}
		if filter.Type != "" && cred.Type != filter.Type {
			continue
		}
		if filter.MinHealth != nil && cred.State.HealthScore < *filter.MinHealth {
			continue
		}
		if filter.MaxHealth != nil && cred.State != nil && cred.State.HealthScore > *filter.MaxHealth {
			continue
		}
		if filter.MaxError != nil && cred.State.ErrorRate > *filter.MaxError {
			continue
		}
		if filter.LastUsed != nil && (cred.State.LastUsed == nil || cred.State.LastUsed.Before(*filter.LastUsed)) {
			continue
		}
		result = append(result, cred)
	}
	return result
}

// CredentialStats 凭证统计信息
type CredentialStats struct {
	TotalCredentials    int                    `json:"total_credentials"`
	ActiveCredentials   int                    `json:"active_credentials"`
	DisabledCredentials int                    `json:"disabled_credentials"`
	HealthScore         float64                `json:"avg_health_score"`
	ErrorRate           float64                `json:"avg_error_rate"`
	TotalUsage          int64                  `json:"total_usage"`
	TotalSuccess        int64                  `json:"total_success"`
	TotalFailure        int64                  `json:"total_failure"`
	TypeDistribution    map[string]int         `json:"type_distribution"`
	LastUpdated         time.Time              `json:"last_updated"`
	DetailStats         map[string]interface{} `json:"detail_stats,omitempty"`
}
