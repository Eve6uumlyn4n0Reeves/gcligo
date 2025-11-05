/**
 * 统一 API 包装层：在 auth.apiRequest 之上提供窄薄封装
 * 包含统一错误处理、429退避重试、网络状态管理
 */
import { auth } from '../auth.js';
import { ui } from '../ui.js';

const RETRYABLE_STATUS = new Set([408, 429, 500, 502, 503, 504]);
const RETRYABLE_CODES = new Set(['timeout', 'connection_error', 'network_error', 'dns_error', 'rate_limit_exceeded', 'server_error', 'bad_gateway', 'service_unavailable']);
type FriendlyErrorGuide = {
  id: string;
  title: string;
  message: string;
  actions?: string[];
  match: (details: any) => boolean;
};

function shouldRetryRequest(status: any, code: any): boolean {
  if (status === 'network') return true;
  const numericStatus = typeof status === 'number' ? status : parseInt(status, 10);
  if (Number.isFinite(numericStatus) && RETRYABLE_STATUS.has(numericStatus)) {
    return true;
  }
  if (typeof numericStatus === 'number' && numericStatus >= 500 && numericStatus < 600) {
    return true;
  }
  if (code) {
    const normalized = String(code).toLowerCase();
    if (RETRYABLE_CODES.has(normalized)) {
      return true;
    }
  }
  return false;
}

function resolveRetryAfterMs(details: any): number | undefined {
  if (!details) return undefined;
  const candidates = [];
  if (details.retryAfter !== undefined) candidates.push(details.retryAfter);
  if (details.headers) {
    const hdr = details.headers['retry-after'] ?? details.headers['Retry-After'];
    if (hdr !== undefined) candidates.push(parseInt(hdr, 10));
  }
  if (details.details && typeof details.details === 'object') {
    const val = details.details.retry_after ?? details.details.retryAfter;
    if (val !== undefined) candidates.push(val);
  }
  for (const raw of candidates) {
    if (typeof raw === 'number' && Number.isFinite(raw) && raw > 0) return raw * 1000;
    const parsed = parseInt(raw, 10);
    if (Number.isFinite(parsed) && parsed > 0) return parsed * 1000;
  }
  return undefined;
}

function buildErrorActions(details: any): any[] {
  return [
    { id:'detail', text:'查看详情', handler: ()=> ui.showErrorDetails(details) },
    { id:'copy', text:'复制错误', handler: async ()=> {
      try {
        const raw = details?.text || (details?.payload ? JSON.stringify(details.payload, null, 2) : '');
        if (raw) {
          await navigator.clipboard.writeText(raw);
          ui.showNotification('success', '已复制', '错误原文已复制到剪贴板');
        }
      } catch (_) {}
    } }
  ];
}

function formatErrorTitle(base: string, details: any): string {
  if (details?.code) {
    return `${base} (${details.code})`;
  }
  return base;
}

const normalizeText = (value: unknown): string =>
  typeof value === 'string' ? value.toLowerCase() : '';

const FRIENDLY_ERROR_GUIDES: FriendlyErrorGuide[] = [
  {
    id: 'remote_management_disabled',
    title: '远程管理未启用',
    message: '当前服务器禁用了远程管理访问，仅允许本地或内网控制台操作。',
    actions: [
      '如需远程访问，请在配置中设置 management_allow_remote: true 并重新加载服务',
      '若无法修改配置，请通过 SSH/隧道连接到部署主机后再访问 /admin'
    ],
    match: (details: any) => {
      const detail = normalizeText(details?.detail || details?.message);
      const code = normalizeText(details?.code);
      const status = normalizeText(details?.status);
      return (
        detail.includes('remote management disabled') ||
        detail.includes('remote access') ||
        code === 'remote_management_disabled' ||
        status === 'remote_management_disabled'
      );
    }
  },
  {
    id: 'invalid_api_key',
    title: 'API 密钥无效',
    message: '提供的管理 API 密钥无效、缺失或已过期。',
    actions: [
      '确认浏览器保存的密钥仍有效，必要时在管理控制台重新生成',
      '通过命令 `localStorage.setItem(\"mgmt_api_key\", \"<your-key>\")` 更新浏览器存储'
    ],
    match: (details: any) => {
      const code = normalizeText(details?.code);
      const detail = normalizeText(details?.detail);
      return (
        code === 'invalid_api_key' ||
        (detail.includes('api key') && (detail.includes('invalid') || detail.includes('missing')))
      );
    }
  },
  {
    id: 'admin_required',
    title: '需要管理员权限',
    message: '此操作仅对具备管理员角色的凭证开放。',
    actions: [
      '使用包含 `is_admin: true` 权限的凭证重新登录',
      '联系平台管理员授予当前账号相应权限'
    ],
    match: (details: any) => {
      const detail = normalizeText(details?.detail);
      const status = normalizeText(details?.status);
      return detail.includes('admin required') || status === 'forbidden';
    }
  }
];

function resolveFriendlyGuide(details: any): FriendlyErrorGuide | null {
  if (!details) return null;
  for (const guide of FRIENDLY_ERROR_GUIDES) {
    try {
      if (guide.match(details)) {
        return guide;
      }
    } catch {
      /* ignore individual matcher errors */
    }
  }
  return null;
}

function composeGuideMessage(
  guide: FriendlyErrorGuide | null,
  fallback: string
): string {
  if (!guide) {
    return fallback;
  }
  const lines = [guide.message];
  if (guide.actions?.length) {
    lines.push(...guide.actions.map((action) => `· ${action}`));
  }
  return lines.join('\n');
}

function shouldShowNotification(context: any, retryable: boolean): boolean {
  if (!retryable) return true;
  if (!context) return true;
  const max = context.maxRetries ?? 3;
  return context.attempt >= max;
}

// 构造错误详情，便于“查看原文”与复制
function buildErrorDetails(error: any, context: any = {}): any {
  const info = error && (error.errorInfo || {});
  const headers = { ...(info?.headers || {}), ...(error?.headers || {}) };
  return {
    status: (() => {
      const raw = error?.status ?? error?.response?.status ?? info?.status;
      const parsed = typeof raw === 'number' ? raw : parseInt(raw, 10);
      return Number.isFinite(parsed) ? parsed : (raw || 'unknown');
    })(),
    code: info?.code || error?.code || '',
    type: info?.type || error?.errorType || '',
    detail: info?.detail || info?.message || error?.message || '',
    headers,
    text: info?.text || '',
    payload: info?.payload,
    url: error?.url || '',
    path: context?.path || '',
    details: info?.details || error?.errorDetails,
    retryAfter: info?.retryAfter ?? error?.retryAfter
  };
}

const createServerErrorHandler = (status: number, title: string, fallback: string) => (error: any, context: any = {}) => {
  const details = buildErrorDetails(error, context);
  const retryable = shouldRetryRequest(status, details.code);
  const friendly = resolveFriendlyGuide(details);
  const fallbackTitle = formatErrorTitle(title, details);
  const notificationTitle = friendly?.title ? friendly.title : fallbackTitle;
  const notificationMessage = friendly
    ? composeGuideMessage(friendly, fallback)
    : details.detail || fallback;
  if (shouldShowNotification(context, retryable)) {
    ui.showNotification('error', notificationTitle, notificationMessage, {
      actions: buildErrorActions(details)
    });
  }
  const retryAfter = resolveRetryAfterMs(details);
  return { handled: false, retry: retryable, retryAfter, exponentialBackoff: retryable && !retryAfter };
};

// 错误分类和处理策略（最小加工：仅提示+详情，不吞掉错误/不改写语义）
const ERROR_HANDLERS: any = {
  401: () => ({ handled: true, retry: false }), // 由 auth.js 负责跳转
  403: () => ({ handled: true, retry: false }), // 由 auth.js 负责提示
  429: (error: any, context: any = {}) => {
    const details = buildErrorDetails(error, context);
    const wait = Number.isFinite(details.retryAfter) ? Math.min(details.retryAfter, 300) : 60;
    ui.showNotification('warning', formatErrorTitle('请求过多(429)', details), `请等待 ${wait} 秒后重试`, {
      duration: wait * 1000,
      actions: buildErrorActions(details)
    });
    return { handled: false, retry: false };
  },
  500: createServerErrorHandler(500, '服务器错误', '服务器内部错误，请稍后重试'),
  502: createServerErrorHandler(502, '网关错误', '上游服务暂不可用，请稍后重试'),
  503: createServerErrorHandler(503, '服务暂停', '服务临时维护中，请稍后重试'),
  504: createServerErrorHandler(504, '请求超时', '请求超时，请稍后重试'),
  408: createServerErrorHandler(408, '请求超时', '请求超时，请稍后重试'),
  network: (error: any, context: any = {}) => {
    if (!navigator.onLine) {
      ui.banner('netBanner', 'warning', ui.t('network_offline_banner'));
    } else if (shouldShowNotification(context, true)) {
      ui.showNotification('error', '网络错误', '网络连接异常，请检查网络设置', {
        actions: buildErrorActions(buildErrorDetails(error, context))
      });
    }
    return { handled: true, retry: true, retryAfter: 5000, exponentialBackoff: false };
  }
};

ERROR_HANDLERS.default = (error: any, context: any = {}) => {
  const details = buildErrorDetails(error, context);
  const status = details.status;
  const retryable = shouldRetryRequest(status, details.code);
  const friendly = resolveFriendlyGuide(details);
  const fallbackTitle = formatErrorTitle('请求失败', details);
  const fallbackMessage = details.detail || '请求失败，请稍后重试';
  const title = friendly?.title ? friendly.title : fallbackTitle;
  const message = friendly ? composeGuideMessage(friendly, fallbackMessage) : fallbackMessage;
  if (shouldShowNotification(context, retryable)) {
    ui.showNotification('error', title, message, {
      actions: buildErrorActions(details)
    });
  }
  const retryAfter = resolveRetryAfterMs(details);
  return { handled: false, retry: retryable, retryAfter, exponentialBackoff: retryable && !retryAfter };
};

// 网络状态监控
let isOnline = navigator.onLine;
window.addEventListener('online', () => {
  if (!isOnline) {
    isOnline = true;
    ui.showNotification('success', '网络已恢复', ui.t('notify_network_online'));
    // 移除离线横幅
    const banner = document.getElementById('netBanner');
    if (banner) banner.remove();
  }
});

window.addEventListener('offline', () => {
  isOnline = false;
  ui.showNotification('warning', '网络已断开', ui.t('notify_network_offline'));
  ui.banner('netBanner', 'warning', ui.t('network_offline_banner'));
});

// 请求缓存管理
export const requestCache = new Map();
const CACHE_TTL = 30 * 1000; // 30秒缓存

// 带重试和缓存的API请求包装器
export async function apiRequestWithRetry(requestFn: () => Promise<any>, context: any = {}): Promise<any> {
  const maxRetries = context.maxRetries ?? 3;
  let attempt = 0;
  let lastError: any = null;

  // 检查缓存（仅对GET请求）
  const cacheKey = context.cacheKey;
  if (cacheKey && context.method !== 'POST' && context.method !== 'PUT' && context.method !== 'DELETE') {
    const cached = requestCache.get(cacheKey);
    if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
      return cached.data;
    }
  }

  while (attempt <= maxRetries) {
    try {
      const result = await requestFn();

      // 缓存成功的GET请求结果
      if (cacheKey && context.method !== 'POST' && context.method !== 'PUT' && context.method !== 'DELETE') {
        requestCache.set(cacheKey, {
          data: result,
          timestamp: Date.now()
        });
      }

      return result;
    } catch (error: unknown) {
      lastError = error;
      attempt++;

      // 解析错误状态
      const err = error as any;
      const status = err.status ?? err.response?.status ?? (err.name === 'TypeError' ? 'network' : 'unknown');
      const handler = ERROR_HANDLERS[status] || ERROR_HANDLERS.default;

      if (handler) {
        const result = handler(error, { ...context, attempt, maxRetries });

        if (!result.retry || attempt > maxRetries) {
          if (!result.handled) {
            throw error;
          }
          break;
        }

        // 计算重试延迟
        const delay = result.retryAfter || (result.exponentialBackoff ?
          Math.min(1000 * Math.pow(2, attempt - 1), 30000) : 1000);

        // 等待重试
        await new Promise(resolve => setTimeout(resolve, delay));
        continue;
      }

      // 未处理的错误，直接抛出
      if (attempt > maxRetries) {
        throw error;
      }

      // 默认重试延迟
      await new Promise(resolve => setTimeout(resolve, 1000 * attempt));
    }
  }

  if (lastError) {
    throw lastError;
  }
}

// 批量请求管理器
export class BatchRequestManager {
  queue: any[];
  processing: boolean;
  batchSize: number;
  batchDelay: number;

  constructor() {
    this.queue = [];
    this.processing = false;
    this.batchSize = 10;
    this.batchDelay = 100; // 100ms批量间隔
  }

  async add(requestFn: () => Promise<any>, context: any = {}): Promise<any> {
    return new Promise((resolve, reject) => {
      this.queue.push({ requestFn, context, resolve, reject });
      this.process();
    });
  }

  async process(): Promise<void> {
    if (this.processing || this.queue.length === 0) return;

    this.processing = true;

    while (this.queue.length > 0) {
      const batch = this.queue.splice(0, this.batchSize);

      // 并行执行批量请求
      const promises = batch.map(async ({ requestFn, context, resolve, reject }: any) => {
        try {
          const result = await apiRequestWithRetry(requestFn, context);
          resolve(result);
        } catch (error) {
          reject(error);
        }
      });

      await Promise.allSettled(promises);

      // 批量间隔
      if (this.queue.length > 0) {
        await new Promise(resolve => setTimeout(resolve, this.batchDelay));
      }
    }

    this.processing = false;
  }
}

export const batchManager = new BatchRequestManager();

// 增强的API请求函数
export const mg = async (path: string, options: any = {}): Promise<any> => {
  const context = {
    context: 'management',
    path,
    method: options.method || 'GET',
    cacheKey: options.cache !== false ? `mg:${path}` : null
  };

  if (options.batch) {
    return batchManager.add(() => auth.apiRequest(path, options), context);
  }

  return apiRequestWithRetry(
    () => auth.apiRequest(path, options),
    context
  );
};

export const enhanced = async (path: string, options: any = {}): Promise<any> => {
  const context = {
    context: 'enhanced',
    path,
    method: options.method || 'GET',
    cacheKey: options.cache !== false ? `enh:${path}` : null
  };

  if (options.batch) {
    return batchManager.add(() => auth.apiRequestEnhanced(path, options), context);
  }

  return apiRequestWithRetry(
    () => auth.apiRequestEnhanced(path, options),
    context
  );
};

export function withQuery(path: string, params: any = {}): string {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return;
    search.append(key, String(value));
  });
  const query = search.toString();
  if (!query) return path;
  return `${path}${path.includes('?') ? '&' : '?'}${query}`;
}

export function encodeSegment(value: any): string {
  return encodeURIComponent(value ?? '');
}
