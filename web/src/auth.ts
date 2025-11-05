import { ensureBasePath, getBasePath, normalizeBasePath } from './utils/base_path.js';
import { notify } from './utils/notifications.js';

interface ErrorInfo {
  detail: string;
  payload: unknown;
  text: string;
  headers: Record<string, string>;
  code: string;
  type: string;
  status: string;
  details: unknown;
  retryAfter?: number;
}

interface ApiRequestOptions extends RequestInit {
  skipAuth?: boolean;
  apiKey?: string;
}

interface ApiError extends Error {
  status?: number;
  url?: string;
  headers?: Record<string, string>;
  code?: string;
  errorType?: string;
  errorStatus?: string;
  retryAfter?: number;
  errorDetails?: unknown;
  errorInfo?: ErrorInfo;
}

declare global {
  interface Window {
    mgmtApiKey?: string;
    MGMT_API_KEY?: string;
    __MGMT_API_KEY?: string;
    ui?: {
      showNotification?: (
        type: string,
        title?: string,
        message?: string,
        options?: unknown
      ) => void;
      withLoading?: <T>(executor: () => Promise<T>, messages?: Record<string, string>) => Promise<T>;
      confirm?: (
        title: string,
        message: string,
        options?: { type?: string; okText?: string; cancelText?: string }
      ) => Promise<boolean>;
      showConfirmation?: (options: {
        title: string;
        message: string;
        type?: string;
        confirmText?: string;
        confirmClass?: string;
      }) => Promise<boolean>;
      showProgressNotification?: (
        title: string,
        message: string,
        options?: {
          type?: string;
          showProgress?: boolean;
          showCancel?: boolean;
          onCancel?: () => void;
        }
      ) => { update?: (progress: number, nextMessage?: string) => void; close?: () => void } | undefined;
      showModal?: (title: string, content: string) => void;
    };
  }
}

export class AuthManager {
  private basePath: string;
  private apiBase: string;
  private managementApi: string;
  private cachedManagementKey: string | null;

  constructor() {
    const detected = ensureBasePath();
    this.basePath = normalizeBasePath(getBasePath() || detected);
    const origin =
      typeof window !== 'undefined' && window.location ? window.location.origin : '';
    this.apiBase = `${origin}${this.basePath}`;
    this.managementApi = `${this.apiBase}/routes/api/management`;
    this.cachedManagementKey = null;
  }

  isAuthenticated(): boolean {
    try {
      const hasCookie = document.cookie
        .split(';')
        .some((c) => c.trim().startsWith('mgmt_session='));
      const path =
        (typeof window !== 'undefined' &&
          window.location &&
          window.location.pathname) ||
        '';
      const onAdmin = /\/admin\/?$/.test(path);
      return hasCookie || onAdmin;
    } catch {
      return false;
    }
  }

  async ensureAuthenticated(): Promise<boolean> {
    if (this.isAuthenticated()) return true;
    try {
      window.location.replace(`${this.basePath || ''}/login`);
    } catch {
      /* ignore */
    }
    return false;
  }

  normalizeEndpoint(endpoint: string = ''): string {
    if (!endpoint) return '/';
    if (typeof endpoint !== 'string') endpoint = String(endpoint);
    if (endpoint.startsWith('http://') || endpoint.startsWith('https://')) return endpoint;
    if (endpoint.startsWith('/')) return endpoint;
    return `/${endpoint}`;
  }

  buildManagementUrl(endpoint: string = ''): string {
    const normalized = this.normalizeEndpoint(endpoint);
    if (normalized.startsWith('http://') || normalized.startsWith('https://')) {
      return normalized;
    }
    return `${this.managementApi}${normalized}`;
  }

  enhancedEndpoint(path: string = ''): string {
    return path.startsWith('/') ? path : `/${path}`;
  }

  buildEnhancedUrl(path: string = ''): string {
    return this.buildManagementUrl(this.enhancedEndpoint(path));
  }

  apiRequestEnhanced(path: string, options: ApiRequestOptions = {}): Promise<unknown> {
    return this.apiRequest(this.enhancedEndpoint(path), options);
  }

  buildManagementWsUrl(endpoint: string, query: string = ''): string {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const suffix = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
    let url = `${wsProtocol}//${window.location.host}${this.basePath}${suffix}`;
    if (query) url += query.startsWith('?') ? query : `?${query}`;
    return url;
  }

  async apiRequest(endpoint: string, options: ApiRequestOptions = {}): Promise<any> {
    const isFormData = options.body instanceof FormData;
    const headers = this.normalizeHeaders(options.headers);
    const hasContentType = Object.keys(headers).some(
      (key) => key.toLowerCase() === 'content-type'
    );
    if (!isFormData && !hasContentType && options.body) {
      headers['Content-Type'] = 'application/json';
    }

    const normalizedEndpoint = this.normalizeEndpoint(endpoint || '');
    const url = normalizedEndpoint.startsWith('http')
      ? normalizedEndpoint
      : this.buildManagementUrl(normalizedEndpoint);

    const { skipAuth, apiKey, ...fetchOptions } = options;
    const resolvedApiKey = this.resolveApiKey(apiKey);
    if (resolvedApiKey && !this.hasAuthHeader(headers)) {
      headers['x-goog-api-key'] = resolvedApiKey;
    }

    const response = await fetch(url, {
      ...fetchOptions,
      headers,
      credentials:
        ((fetchOptions.credentials as RequestCredentials | undefined) ??
          'same-origin') as RequestCredentials
    });

    if (response.status === 401 && !skipAuth) {
      try {
        window.location.replace(`${this.basePath || ''}/login`);
      } catch {
        /* ignore */
      }
      return response;
    }

    if (response.status === 403 && !skipAuth) {
      const errorInfo = await this.parseErrorResponse(response);
      const detail = errorInfo.detail || '访问被拒绝';
      if (
        detail.includes('remote management disabled') ||
        detail.includes('remote access')
      ) {
        this.show403Dialog(detail);
      } else {
        notify.error(`访问受限: ${this.friendlyErrorMessage(detail)}`);
      }

      const err: ApiError = new Error(`API Error (403): ${detail}`);
      err.status = 403;
      err.url = endpoint;
      err.headers = errorInfo.headers || {};
      err.errorInfo = errorInfo;
      if (errorInfo.code) err.code = errorInfo.code;
      if (errorInfo.type) err.errorType = errorInfo.type;
      if (errorInfo.status) err.errorStatus = errorInfo.status;
      if (errorInfo.retryAfter !== undefined) err.retryAfter = errorInfo.retryAfter;
      if (errorInfo.details) err.errorDetails = errorInfo.details;
      throw err;
    }

    if (!response.ok) {
      const errorInfo = await this.parseErrorResponse(response);
      const err: ApiError = new Error(
        `API Error (${response.status}): ${errorInfo.detail || 'unknown error'}`
      );
      err.status = response.status;
      err.url = endpoint;
      err.headers = errorInfo.headers || {};
      err.errorInfo = errorInfo;
      if (errorInfo.code) err.code = errorInfo.code;
      if (errorInfo.type) err.errorType = errorInfo.type;
      if (errorInfo.status) err.errorStatus = errorInfo.status;
      if (errorInfo.retryAfter !== undefined) err.retryAfter = errorInfo.retryAfter;
      if (errorInfo.details) err.errorDetails = errorInfo.details;
      throw err;
    }

    if (response.status === 204) return {};
    return response.json();
  }

  private async parseErrorResponse(resp: Response | null): Promise<ErrorInfo> {
    if (!resp) {
      return {
        detail: '',
        payload: null,
        text: '',
        headers: {},
        code: '',
        type: '',
        status: '',
        details: undefined
      };
    }
    let text = '';
    let payload: unknown = null;
    try {
      const cloned = resp.clone();
      text = await cloned.text();
      if (text) {
        try {
          payload = JSON.parse(text);
        } catch {
          payload = null;
        }
      }
    } catch {
      text = '';
      payload = null;
    }

    let detail = '';
    let code = '';
    let errType = '';
    let status = '';
    let extra: unknown = undefined;

    const extractError = (obj: unknown) => {
      if (!obj || typeof obj !== 'object') return;
      const record = obj as Record<string, unknown>;
      if (typeof record.message === 'string' && !detail) detail = record.message;
      if (typeof record.detail === 'string') detail = record.detail;
      if (record.code !== undefined && code === '') code = String(record.code);
      if (typeof record.type === 'string') errType = record.type;
      if (typeof record.status === 'string') status = record.status;
      if (record.details && typeof record.details === 'object') extra = record.details;
    };

    if (payload) {
      if (typeof payload === 'string') {
        detail = payload;
      } else if (typeof payload === 'object') {
        const obj = payload as Record<string, unknown>;
        if (obj.error) {
          if (typeof obj.error === 'string') {
            detail = obj.error;
          } else {
            extractError(obj.error);
          }
        }
        if (!detail && typeof obj.message === 'string') {
          detail = obj.message;
        }
        if (!detail && typeof obj.detail === 'string') {
          detail = obj.detail;
        }
      }
    }

    const headers: Record<string, string> = {};
    try {
      resp.headers?.forEach((v, k) => {
        headers[k] = v;
        headers[k.toLowerCase()] = v;
      });
    } catch {
      /* ignore */
    }

    let retryAfter: number | undefined = undefined;
    if (extra && typeof extra === 'object') {
      const ext = extra as Record<string, unknown>;
      const candidate = ext.retry_after ?? ext.retryAfter;
      if (typeof candidate === 'number' && Number.isFinite(candidate)) {
        retryAfter = candidate;
      }
    }
    if (retryAfter === undefined) {
      const raw = headers['retry-after'] ?? headers['Retry-After'];
      const parsed = raw !== undefined ? parseInt(raw, 10) : NaN;
      if (Number.isFinite(parsed) && parsed >= 0) {
        retryAfter = parsed;
      }
    }

    return {
      detail,
      payload,
      text,
      headers,
      code,
      type: errType,
      status,
      details: extra,
      retryAfter
    };
  }

  async showLoginDialog(): Promise<boolean> {
    try {
      window.location.replace(`${this.basePath || ''}/login`);
    } catch {
      /* ignore */
    }
    return false;
  }

  showAlert(type: string = 'info', message: unknown = ''): void {
    try {
      const m = String(message ?? '');
      if (
        typeof window !== 'undefined' &&
        window.ui &&
        typeof window.ui.showNotification === 'function'
      ) {
        const title =
          (
            {
              success: '成功',
              error: '错误',
              warning: '提示',
              info: '提示'
            } as Record<string, string>
          )[type] || '提示';
        window.ui.showNotification(type, title, m);
        return;
      }
      if (notify && typeof (notify as any)[type] === 'function') {
        (notify as any)[type](m);
        return;
      }
      window.alert?.(m);
    } catch {
      /* ignore */
    }
  }

  encodeHtml(value: unknown): string {
    return String(value ?? '').replace(/[&<>"']/g, (ch) => {
      const map: Record<string, string> = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
      };
      return map[ch] ?? ch;
    });
  }

  friendlyErrorMessage(detail: string): string {
    if (!detail) return '未知错误';
    const map: Record<string, string> = {
      'remote management disabled': '远程管理已禁用',
      'management key': '管理密钥问题',
      'invalid api key': '密钥无效',
      unauthorized: '未授权访问',
      forbidden: '访问被禁止',
      'not found': '资源不存在',
      'internal server error': '服务器内部错误',
      'bad gateway': '网关错误',
      'service unavailable': '服务不可用'
    };
    const lower = detail.toLowerCase();
    for (const k of Object.keys(map)) {
      if (lower.includes(k)) return map[k];
    }
    return detail.length > 50 ? `${detail.substring(0, 50)}...` : detail;
  }

  show403Dialog(_detail: string): void {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
      <div class="modal-content" role="dialog" aria-modal="true" style="max-width: 500px;">
        <button type="button" class="modal-close" aria-label="关闭">&times;</button>
        <div class="modal-header" style="text-align: center; margin-bottom: 24px;">
          <div style="font-size: 48px; margin-bottom: 12px;">🚫</div>
          <h2 style="margin: 0; color: #dc2626;">访问受限</h2>
        </div>
        <div class="modal-body" style="margin-bottom: 24px;">
          <p style="color: #6b7280; margin-bottom: 20px;">远程管理访问已被禁用，请尝试以下解决方案：</p>
          <div style="display: flex; flex-direction: column; gap: 12px;">
            <div style="padding: 16px; background: #f0f9ff; border-radius: 8px; border-left: 4px solid #3b82f6;">
              <strong style="color: #1e40af;">方案1：本地访问</strong>
              <p style="margin: 8px 0 0 0; color: #374151; font-size: 14px;">使用 <code style="background: #e5e7eb; padding: 2px 6px; border-radius: 4px;">localhost</code> 或 <code style="background: #e5e7eb; padding: 2px 6px; border-radius: 4px;">127.0.0.1</code> 访问管理界面</p>
            </div>
            <div style="padding: 16px; background: #f0fdf4; border-radius: 8px; border-left: 4px solid #16a34a;">
              <strong style="color: #15803d;">方案2：启用远程访问</strong>
              <p style="margin: 8px 0 0 0; color: #374151; font-size: 14px;">在配置文件中设置 <code style="background: #e5e7eb; padding: 2px 6px; border-radius: 4px;">management_allow_remote: true</code></p>
            </div>
            <div style="padding: 16px; background: #fffbeb; border-radius: 8px; border-left: 4px solid #f59e0b;">
              <strong style="color: #d97706;">方案3：联系管理员</strong>
              <p style="margin: 8px 0 0 0; color: #374151; font-size: 14px;">如果您是系统管理员，请检查服务器配置</p>
            </div>
          </div>
        </div>
        <div class="modal-footer" style="display: flex; justify-content: center;">
          <button type="button" class="btn btn-primary" onclick="this.closest('.modal').remove()"
                  style="padding: 10px 20px; background: #3b82f6; color: white; border: none; border-radius: 6px; cursor: pointer;">
            知道了
          </button>
        </div>
      </div>`;
    document.body.appendChild(modal);
    modal.querySelector('.modal-close')?.addEventListener('click', () => {
      modal.remove();
    });
    modal.addEventListener('click', (e) => {
      if (e.target === modal) modal.remove();
    });
  }

  setManagementKey(key: string): void {
    const normalized = this.normalizeKey(key);
    this.cachedManagementKey = normalized || null;
    if (!normalized) return;
    try {
      window.localStorage?.setItem('mgmt_api_key', normalized);
    } catch {
      /* ignore */
    }
    try {
      window.sessionStorage?.setItem('mgmt_api_key', normalized);
    } catch {
      /* ignore */
    }
  }

  getManagementKey(): string {
    return this.resolveApiKey();
  }

  private normalizeHeaders(input?: HeadersInit): Record<string, string> {
    const result: Record<string, string> = {};
    if (!input) return result;
    if (input instanceof Headers) {
      input.forEach((value, key) => {
        result[key] = value;
      });
      return result;
    }
    if (Array.isArray(input)) {
      for (const pair of input) {
        if (!Array.isArray(pair) || pair.length < 2) continue;
        const [key, value] = pair;
        if (!key) continue;
        result[String(key)] = String(value ?? '');
      }
      return result;
    }
    return { ...(input as Record<string, string>) };
  }

  private hasAuthHeader(headers: Record<string, string>): boolean {
    for (const key of Object.keys(headers)) {
      const lower = key.toLowerCase();
      if (
        lower === 'authorization' ||
        lower === 'x-goog-api-key' ||
        lower === 'x-api-key'
      ) {
        return true;
      }
    }
    return false;
  }

  private normalizeKey(value: unknown): string {
    if (typeof value === 'string') return value.trim();
    if (value === undefined || value === null) return '';
    try {
      return String(value).trim();
    } catch {
      return '';
    }
  }

  private resolveApiKey(explicit?: string): string {
    const candidates: Array<string | null | undefined> = [
      explicit,
      this.cachedManagementKey,
      this.readWindowKey(),
      this.readStorageKey('session'),
      this.readStorageKey('local'),
      this.readDatasetKey(),
      this.readMetaKey()
    ];
    for (const candidate of candidates) {
      const normalized = this.normalizeKey(candidate);
      if (normalized) {
        this.cachedManagementKey = normalized;
        return normalized;
      }
    }
    return '';
  }

  private readStorageKey(source: 'local' | 'session'): string {
    if (typeof window === 'undefined') return '';
    try {
      const storage =
        source === 'local' ? window.localStorage : window.sessionStorage;
      return storage?.getItem('mgmt_api_key') || '';
    } catch {
      return '';
    }
  }

  private readDatasetKey(): string {
    if (typeof document === 'undefined') return '';
    if (document.documentElement?.dataset?.mgmtApiKey) {
      return document.documentElement.dataset.mgmtApiKey;
    }
    if (document.body?.dataset?.mgmtApiKey) {
      return document.body.dataset.mgmtApiKey;
    }
    return '';
  }

  private readMetaKey(): string {
    if (typeof document === 'undefined') return '';
    const meta = document.querySelector('meta[name=\"mgmt-api-key\"]');
    return (meta && meta.getAttribute('content')) || '';
  }

  private readWindowKey(): string {
    if (typeof window === 'undefined') return '';
    const candidates = [
      (window as Record<string, any>).__MGMT_API_KEY,
      (window as Record<string, any>).MGMT_API_KEY,
      (window as Record<string, any>).mgmtApiKey
    ];
    for (const value of candidates) {
      if (typeof value === 'string' && value.trim() !== '') {
        return value;
      }
    }
    return '';
  }
}

export const auth = new AuthManager();
