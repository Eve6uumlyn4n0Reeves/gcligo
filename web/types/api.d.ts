/**
 * API type definitions for GCLI2API-Go
 * 
 * This file contains type definitions for:
 * - API request/response types
 * - Gemini API types
 * - OpenAI API types
 * - Internal API types
 */

// ============================================================================
// Base API Types
// ============================================================================

/**
 * Base API response
 */
interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: APIError;
  message?: string;
}

/**
 * API error
 */
interface APIError {
  code: string;
  message: string;
  details?: any;
  stack?: string;
}

/**
 * Pagination parameters
 */
interface PaginationParams {
  page?: number;
  pageSize?: number;
  offset?: number;
  limit?: number;
}

/**
 * Paginated response
 */
interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

// ============================================================================
// Request Context Types
// ============================================================================

/**
 * Request context for API calls
 */
interface RequestContext {
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: any;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
  signal?: AbortSignal;
}

/**
 * Retry configuration
 */
interface RetryConfig {
  maxRetries: number;
  retryDelay: number;
  retryOn?: number[];
  shouldRetry?: (error: Error, attempt: number) => boolean;
}

/**
 * Circuit breaker state
 */
interface CircuitBreakerState {
  state: 'closed' | 'open' | 'half-open';
  failures: number;
  successes: number;
  lastFailureTime?: number;
  nextAttemptTime?: number;
}

// ============================================================================
// Gemini API Types
// ============================================================================

/**
 * Gemini chat completion request
 */
interface GeminiChatRequest {
  model: string;
  messages: GeminiMessage[];
  temperature?: number;
  topP?: number;
  topK?: number;
  maxOutputTokens?: number;
  stopSequences?: string[];
  safetySettings?: GeminiSafetySetting[];
  stream?: boolean;
}

/**
 * Gemini message
 */
interface GeminiMessage {
  role: 'user' | 'model';
  parts: GeminiPart[];
}

/**
 * Gemini message part
 */
interface GeminiPart {
  text?: string;
  inlineData?: {
    mimeType: string;
    data: string;
  };
  fileData?: {
    mimeType: string;
    fileUri: string;
  };
}

/**
 * Gemini safety setting
 */
interface GeminiSafetySetting {
  category: string;
  threshold: string;
}

/**
 * Gemini chat completion response
 */
interface GeminiChatResponse {
  candidates: GeminiCandidate[];
  promptFeedback?: GeminiPromptFeedback;
  usageMetadata?: GeminiUsageMetadata;
}

/**
 * Gemini candidate
 */
interface GeminiCandidate {
  content: GeminiMessage;
  finishReason?: string;
  safetyRatings?: GeminiSafetyRating[];
  citationMetadata?: GeminiCitationMetadata;
}

/**
 * Gemini safety rating
 */
interface GeminiSafetyRating {
  category: string;
  probability: string;
}

/**
 * Gemini citation metadata
 */
interface GeminiCitationMetadata {
  citations: GeminiCitation[];
}

/**
 * Gemini citation
 */
interface GeminiCitation {
  startIndex: number;
  endIndex: number;
  uri: string;
  title?: string;
  license?: string;
  publicationDate?: string;
}

/**
 * Gemini prompt feedback
 */
interface GeminiPromptFeedback {
  blockReason?: string;
  safetyRatings?: GeminiSafetyRating[];
}

/**
 * Gemini usage metadata
 */
interface GeminiUsageMetadata {
  promptTokenCount: number;
  candidatesTokenCount: number;
  totalTokenCount: number;
}

// ============================================================================
// OpenAI API Types
// ============================================================================

/**
 * OpenAI chat completion request
 */
interface OpenAIChatRequest {
  model: string;
  messages: OpenAIMessage[];
  temperature?: number;
  top_p?: number;
  n?: number;
  stream?: boolean;
  stop?: string | string[];
  max_tokens?: number;
  presence_penalty?: number;
  frequency_penalty?: number;
  logit_bias?: Record<string, number>;
  user?: string;
}

/**
 * OpenAI message
 */
interface OpenAIMessage {
  role: 'system' | 'user' | 'assistant' | 'function';
  content: string | OpenAIMessageContent[];
  name?: string;
  function_call?: OpenAIFunctionCall;
}

/**
 * OpenAI message content (for multimodal)
 */
interface OpenAIMessageContent {
  type: 'text' | 'image_url';
  text?: string;
  image_url?: {
    url: string;
    detail?: 'auto' | 'low' | 'high';
  };
}

/**
 * OpenAI function call
 */
interface OpenAIFunctionCall {
  name: string;
  arguments: string;
}

/**
 * OpenAI chat completion response
 */
interface OpenAIChatResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: OpenAIChoice[];
  usage?: OpenAIUsage;
}

/**
 * OpenAI choice
 */
interface OpenAIChoice {
  index: number;
  message: OpenAIMessage;
  finish_reason: string;
}

/**
 * OpenAI usage
 */
interface OpenAIUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

/**
 * OpenAI streaming chunk
 */
interface OpenAIStreamChunk {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: OpenAIStreamChoice[];
}

/**
 * OpenAI streaming choice
 */
interface OpenAIStreamChoice {
  index: number;
  delta: {
    role?: string;
    content?: string;
  };
  finish_reason?: string;
}

// ============================================================================
// Internal API Types
// ============================================================================

/**
 * Credential info - matches backend admin_creds.go response format
 */
interface CredentialInfo {
  // Core identification
  id: string;
  filename: string;
  type: string;
  email?: string;
  project_id?: string;

  // Status flags (use these instead of 'status' enum)
  disabled: boolean;
  auto_banned: boolean;
  banned_reason?: string;
  ban_until?: string;

  // Health metrics
  healthy: boolean;
  score: number;
  health_score: number;
  failure_weight: number;

  // Request statistics
  total_requests: number;
  success_count: number;
  failure_count: number;
  consecutive_fails: number;
  success_rate: number;

  // Error tracking
  last_error_code?: string;

  // Timestamps
  last_success?: string;
  last_failure?: string;

  // Legacy/optional fields for backward compatibility
  /** @deprecated Use disabled and auto_banned instead */
  status?: 'active' | 'inactive' | 'banned' | 'expired';
  /** @deprecated Backend doesn't provide this field */
  name?: string;
  /** @deprecated Backend doesn't provide this field */
  createdAt?: string;
  /** @deprecated Backend doesn't provide this field */
  updatedAt?: string;
  /** @deprecated Use last_success instead */
  lastUsedAt?: string;
  /** @deprecated Use total_requests instead */
  usageCount?: number;
  /** @deprecated Use failure_count instead */
  errorCount?: number;
  /** @deprecated Use banned_reason instead */
  banReason?: string;
  /** @deprecated Use ban_until instead */
  banUntil?: string;
  /** @deprecated Backend provides failure_reason in GetCredential only */
  failure_reason?: string;
}

/**
 * Batch operation request
 */
export interface BatchCredentialRequest {
  ids: string[];
  concurrency?: number;
}

/**
 * Batch progress event
 */
export interface BatchProgressEvent {
  completed: number;
  success_count: number;
  failure_count: number;
  timestamp: string;
}

/**
 * Batch operation result item
 */
export interface BatchOperationResultItem {
  id: string;
  success: boolean;
  error?: string;
}

/**
 * Batch operation response
 */
export interface BatchOperationResponse {
  operation: 'enable' | 'disable' | 'delete' | 'recover';
  results: BatchOperationResultItem[];
  total: number;
  success_count: number;
  failure_count: number;
  concurrency: number;
  duration_ms: number;
  progress: BatchProgressEvent[];
  warning?: string;
}

/**
 * Batch task summary information
 */
export interface BatchTaskSummary {
  id: string;
  operation: 'enable' | 'disable' | 'delete' | 'recover';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  total: number;
  completed: number;
  success: number;
  failure: number;
  progress: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
}

export interface BatchTaskDetail extends BatchTaskSummary {
  results?: BatchOperationResultItem[];
}

export interface BatchTaskListResponse {
  tasks: BatchTaskSummary[];
  total: number;
}

export interface BatchTaskProgressEvent {
  status: BatchTaskSummary['status'];
  progress: number;
  completed: number;
  success: number;
  failure: number;
  error?: string;
}

/**
 * Server configuration
 */
interface ServerConfig {
  server: {
    host: string;
    port: number;
    basePath: string;
  };
  storage: {
    type: 'file' | 'redis' | 'mongodb' | 'postgres';
    config: Record<string, any>;
  };
  credentials: {
    autoRotate: boolean;
    rotateInterval: number;
    maxRetries: number;
  };
  logging: {
    level: string;
    format: string;
    output: string;
  };
}

/**
 * Server stats
 */
interface ServerStats {
  uptime: number;
  requests: {
    total: number;
    success: number;
    error: number;
  };
  credentials: {
    total: number;
    active: number;
    banned: number;
  };
  cache: {
    hits: number;
    misses: number;
    size: number;
  };
  memory: {
    used: number;
    total: number;
    percent: number;
  };
}

/**
 * Log entry
 */
interface LogEntry {
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  context?: Record<string, any>;
  error?: {
    message: string;
    stack?: string;
  };
}

/**
 * Metrics data
 */
interface MetricsData {
  timestamp: number;
  requests: number;
  errors: number;
  latency: number;
  throughput: number;
}

// ============================================================================
// Batch Request Types
// ============================================================================

/**
 * Batch request
 */
interface BatchRequest {
  id: string;
  requests: BatchRequestItem[];
}

/**
 * Batch request item
 */
interface BatchRequestItem {
  id: string;
  method: string;
  url: string;
  headers?: Record<string, string>;
  body?: any;
}

/**
 * Batch response
 */
interface BatchResponse {
  id: string;
  responses: BatchResponseItem[];
}

/**
 * Batch response item
 */
interface BatchResponseItem {
  id: string;
  status: number;
  headers?: Record<string, string>;
  body?: any;
  error?: APIError;
}

// ============================================================================
// Export
// ============================================================================

export {};
