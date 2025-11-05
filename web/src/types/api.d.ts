/**
 * API 相关类型定义
 * 定义所有 API 请求、响应、错误处理相关的类型
 */

/**
 * API 响应基础类型
 */
export interface ApiResponse<T = any> {
  data?: T;
  error?: ApiError;
  status: number;
  headers?: Record<string, string>;
}

/**
 * API 错误类型
 */
export interface ApiError {
  code: string;
  message: string;
  type?: string;
  status?: number | string;
  details?: any;
  retryAfter?: number;
  url?: string;
  path?: string;
  headers?: Record<string, string>;
  text?: string;
  payload?: any;
}

/**
 * API 请求选项
 */
export interface ApiRequestOptions extends RequestInit {
  skipAuth?: boolean;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
  onProgress?: (progress: number) => void;
}

/**
 * 请求上下文
 */
export interface RequestContext {
  path: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  headers?: Record<string, string>;
  body?: any;
  attempt: number;
  maxRetries: number;
  timeout?: number;
  startTime?: number;
}

/**
 * 批量请求管理器选项
 */
export interface BatchRequestOptions {
  maxConcurrent?: number;
  timeout?: number;
  retries?: number;
}

/**
 * 凭证数据类型
 * 注意：这个接口应该与 src/creds/types.ts 中的 Credential 接口兼容
 */
export interface Credential {
  filename?: string;
  email?: string;
  project_id?: string;
  display_name?: string;
  id?: string;
  disabled?: boolean;
  auto_banned?: boolean;
  banned_reason?: string;
  last_success_time?: string;
  last_success_ts?: number;
  last_success?: number | string;
  last_failure_time?: string;
  last_failure_ts?: number;
  last_failure?: number | string;
  error_history?: Record<string, number>;
  total_calls?: number;
  total_requests?: number;
  gemini_2_5_pro_calls?: number;
  success_rate?: number;
  failure_weight?: number;
  health_score?: number;
  error_codes?: string[];
  token?: string;
  refresh_token?: string;
  created_at?: string;
  updated_at?: string;
  last_used?: string;
  usage_count?: number;
  error_count?: number;
  expires_at?: string;
  metadata?: Record<string, any>;
  [key: string]: unknown;
}

/**
 * 凭证列表响应
 */
export interface CredentialsResponse {
  credentials: Credential[];
  total: number;
  active: number;
  banned: number;
  expired: number;
}

/**
 * 配置数据类型
 */
export interface ConfigData {
  openai_port?: string;
  gemini_port?: string;
  base_path?: string;
  management_key?: string;
  storage_backend?: string;
  retry_enabled?: boolean;
  retry_max?: number;
  rate_limit_enabled?: boolean;
  rate_limit_rps?: number;
  calls_per_rotation?: number;
  auto_ban_enabled?: boolean;
  auto_recovery_enabled?: boolean;
  [key: string]: any;
}

/**
 * 使用统计数据类型
 */
export interface UsageStats {
  total_requests?: number;
  successful_requests?: number;
  failed_requests?: number;
  total_tokens?: number;
  input_tokens?: number;
  output_tokens?: number;
  by_model?: Record<string, ModelStats>;
  by_credential?: Record<string, CredentialStats>;
  time_range?: {
    start: string;
    end: string;
  };
}

/**
 * 模型统计数据
 */
export interface ModelStats {
  requests: number;
  tokens: number;
  errors: number;
  avg_latency?: number;
}

/**
 * 凭证统计数据
 */
export interface CredentialStats {
  requests: number;
  tokens: number;
  errors: number;
  last_used?: string;
}

/**
 * 模型注册表项
 */
export interface ModelRegistryEntry {
  id: string;
  name: string;
  display_name?: string;
  provider: string;
  enabled: boolean;
  variants?: string[];
  capabilities?: string[];
  max_tokens?: number;
  context_window?: number;
  metadata?: Record<string, any>;
}

/**
 * 路由装配台配置
 */
export interface AssemblyConfig {
  id: string;
  name: string;
  description?: string;
  routes: RouteConfig[];
  created_at?: string;
  updated_at?: string;
  active?: boolean;
}

export interface ErrorDetail {
  message: string;
  type: string;
  code?: string;
  status?: string;
  http_code: number;
  details?: any;
}

export interface ErrorResponse {
  error: ErrorDetail;
}

export type ChatCompletionRole = 'system' | 'user' | 'assistant' | 'tool';

export interface ChatCompletionMessage {
  role: ChatCompletionRole;
  content: string | Array<{ type: 'text' | 'image_url'; text?: string; image_url?: { url: string } }>;
  name?: string;
  tool_call_id?: string;
}

export interface ChatCompletionRequest {
  model: string;
  messages: ChatCompletionMessage[];
  temperature?: number;
  top_p?: number;
  max_tokens?: number;
  stream?: boolean;
  tools?: Array<{
    type: 'function';
    function: {
      name: string;
      description?: string;
      parameters?: Record<string, any>;
    };
  }>;
  response_format?: { type: 'json_object' | 'text'; schema?: any };
}

export interface ChatCompletionChoice {
  index: number;
  finish_reason: 'stop' | 'length' | 'tool_calls' | 'content_filter';
  message: ChatCompletionMessage & {
    reasoning_content?: string;
    tool_calls?: Array<{
      id: string;
      type: 'function';
      function: { name: string; arguments: string };
    }>;
  };
}

export interface ChatCompletionUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface ChatCompletionResponse {
  id: string;
  object: 'chat.completion';
  created: number;
  model: string;
  choices: ChatCompletionChoice[];
  usage?: ChatCompletionUsage;
  reasoning?: {
    content?: string;
  };
}

/**
 * 路由配置
 */
export interface RouteConfig {
  path: string;
  method: string;
  upstream: string;
  model_mapping?: Record<string, string>;
  middleware?: string[];
  enabled: boolean;
}

/**
 * OAuth 配置
 */
export interface OAuthConfig {
  client_id?: string;
  redirect_url?: string;
  scopes?: string[];
  enabled?: boolean;
}

/**
 * OAuth 令牌响应
 */
export interface OAuthTokenResponse {
  access_token: string;
  refresh_token?: string;
  token_type: string;
  expires_in: number;
  scope?: string;
}

/**
 * 日志条目
 */
export interface LogEntry {
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error' | 'fatal';
  message: string;
  fields?: Record<string, any>;
  source?: string;
  trace_id?: string;
}

/**
 * 日志查询选项
 */
export interface LogQueryOptions {
  level?: string;
  source?: string;
  start_time?: string;
  end_time?: string;
  limit?: number;
  offset?: number;
  search?: string;
}

/**
 * 流式洞察数据
 */
export interface StreamingInsights {
  sse_lines_emitted?: number;
  disconnect_reasons?: Record<string, number>;
  tool_call_events?: number;
  anti_truncation_attempts?: number;
  model_fallbacks?: number;
  thinking_removed?: number;
  last_updated?: string;
}

/**
 * 系统信息
 */
export interface SystemInfo {
  version?: string;
  go_version?: string;
  openai_port?: string;
  gemini_port?: string;
  admin_version?: string;
  uptime?: string;
  memory_usage?: string;
  goroutines?: number;
  cpu_usage?: number;
}

/**
 * 健康检查响应
 */
export interface HealthCheckResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  checks: Record<string, HealthCheck>;
  timestamp: string;
}

/**
 * 健康检查项
 */
export interface HealthCheck {
  status: 'pass' | 'fail' | 'warn';
  message?: string;
  duration_ms?: number;
}

/**
 * 导出数据选项
 */
export interface ExportOptions {
  format: 'json' | 'csv' | 'yaml';
  include?: string[];
  exclude?: string[];
  compress?: boolean;
}

/**
 * 导入数据选项
 */
export interface ImportOptions {
  format: 'json' | 'csv' | 'yaml';
  merge?: boolean;
  overwrite?: boolean;
  validate?: boolean;
}

/**
 * 分页选项
 */
export interface PaginationOptions {
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

/**
 * 分页响应
 */
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}
