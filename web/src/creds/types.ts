export type CredentialStatusFilter = 'all' | 'active' | 'disabled' | 'banned';
export type CredentialHealthFilter = 'all' | 'excellent' | 'good' | 'poor';
export type HealthLevel = 'excellent' | 'good' | 'poor' | 'critical';

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
  [key: string]: unknown;
}

export interface CredentialFilters {
  search: string;
  status: CredentialStatusFilter;
  health: CredentialHealthFilter;
  project: string;
}

export interface PaginatedResult<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  pages: number;
}

export type CredentialBatchAction = 'enable' | 'disable' | 'delete' | 'health-check';

export interface BatchActionItem {
  id: string;
  action: CredentialBatchAction;
}

export interface BatchOperationResult {
  success: string[];
  failed: string[];
  errors: string[];
}
