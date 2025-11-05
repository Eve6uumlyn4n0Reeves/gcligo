package upstream

import (
	"context"
	"net/http"

	"gcli2api-go/internal/credential"
)

// RequestContext 封装一次上游请求的上下文。
type RequestContext struct {
	// Ctx 是请求级别的 context。
	Ctx context.Context
	// Credential 表示当前使用的凭证，可为 nil。
	Credential *credential.Credential
	// BaseModel 是调用所针对的基础模型。
	BaseModel string
	// ProjectID 指定请求使用的 Google 项目，可为空。
	ProjectID string
	// Body 是已经序列化好的请求体。
	Body []byte
	// HeaderOverrides 允许 provider 应用客户端传入的 HTTP 头。
	HeaderOverrides http.Header
}

// ProviderResponse 表示一次调用的结果。
type ProviderResponse struct {
	Resp       *http.Response
	UsedModel  string
	Err        error
	Credential *credential.Credential
}

// ProviderListResponse 用于列出上游模型。
type ProviderListResponse struct {
	Models     []string
	Err        error
	Credential *credential.Credential
}
