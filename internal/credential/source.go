package credential

import (
	"context"
)

// CredentialSource 定义凭证的统一读取接口，便于替换不同来源（文件、远端存储等）。
type CredentialSource interface {
	Name() string
	Load(ctx context.Context) ([]*Credential, error)
}

// WritableCredentialSource 支持将凭证回写至来源。
type WritableCredentialSource interface {
	CredentialSource
	Save(ctx context.Context, cred *Credential) error
	Delete(ctx context.Context, id string) error
}

// StatefulCredentialSource 负责持久化运行时状态，例如失败次数、禁用标记等。
type StatefulCredentialSource interface {
	CredentialSource
	RestoreState(ctx context.Context, cred *Credential) error
	PersistState(ctx context.Context, cred *Credential, state *CredentialState) error
	DeleteState(ctx context.Context, id string) error
}
