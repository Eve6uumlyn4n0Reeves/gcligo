/**
 * 全局类型定义
 * 定义 Window 接口扩展和全局变量类型
 */

/**
 * 管理控制台引导上下文
 */
interface AdminBootstrapContext {
  basePath: string;
  assetVersion: string;
  metaPayload: MetaPayload | null;
  metaError: string;
  assetMismatch: {
    expected: string;
    server: string;
  } | null;
}

/**
 * 元数据负载
 */
interface MetaPayload {
  base_path?: string;
  asset_version?: string;
  [key: string]: any;
}

/**
 * 错误详情
 */
interface ErrorDetails {
  status: number | string;
  code: string;
  type: string;
  detail: string;
  headers: Record<string, any>;
  text: string;
  payload?: any;
  url: string;
  path: string;
  details?: any;
  retryAfter?: number;
}

/**
 * 错误操作
 */
interface ErrorAction {
  id: string;
  text: string;
  handler: () => void | Promise<void>;
}

/**
 * 通知类型
 */
type NotificationType = 'success' | 'error' | 'warning' | 'info';

/**
 * 对话框选项
 */
interface DialogOptions {
  title?: string;
  message?: string;
  confirmText?: string;
  cancelText?: string;
  type?: 'confirm' | 'alert' | 'prompt';
  defaultValue?: string;
}

/**
 * API 请求上下文
 */
interface ApiRequestContext {
  path?: string;
  attempt?: number;
  maxRetries?: number;
  [key: string]: any;
}

/**
 * 缓存管理器接口
 */
interface CacheManager {
  get(key: string): any;
  set(key: string, value: any, ttl?: number): void;
  delete(key: string): void;
  clear(): void;
  has(key: string): boolean;
  size(): number;
  keys(): string[];
  getOrSet(key: string, factory: () => any, ttl?: number): any;
  getStats(): { hits: number; misses: number; hitRate: number };
}

/**
 * 刷新管理器接口
 */
interface RefreshManager {
  start(interval: number, callback: () => void | Promise<void>): void;
  stop(): void;
  isRunning(): boolean;
}

/**
 * 事件总线接口
 */
interface EventBus {
  on(event: string, handler: (...args: any[]) => void): void;
  off(event: string, handler: (...args: any[]) => void): void;
  emit(event: string, ...args: any[]): void;
  once(event: string, handler: (...args: any[]) => void): void;
}

/**
 * UI 辅助工具接口
 */
interface UIHelper {
  // 主题管理
  themeKey: string;
  currentTheme: string;
  setTheme(theme: string): void;
  getTheme(): string;

  // 语言管理
  langKey: string;
  currentLang: string;
  dict: Record<string, Record<string, string>>;
  t(key: string, fallback?: string): string;
  setLang(lang: string): void;
  getLang(): string;

  // 加载状态
  globalLoading: HTMLElement | null;
  showLoading(message?: string): void;
  hideLoading(): void;
  withLoading<T>(executor: () => Promise<T>, messages?: Record<string, string>): Promise<T>;

  // 通知系统
  notificationCenter: import('./components').NotificationCenter;
  showNotification(type: NotificationType, title?: string, message?: string, options?: any): void;
  showProgressNotification(title: string, message: string, options?: any): any;

  // 对话框
  dialogs: import('./components').DialogManager;
  showModal(title: string, content: string): void;
  confirm(title: string, message: string, options?: any): Promise<boolean>;
  showConfirmation(options: any): Promise<boolean>;

  // 错误处理
  showErrorDetails(details: ErrorDetails): void;
  handleError(error: any, context?: string): void;

  // DOM 操作
  createElement<K extends keyof HTMLElementTagNameMap>(
    tag: K,
    attrs?: Record<string, string>,
    children?: (HTMLElement | string)[]
  ): HTMLElementTagNameMap[K];
  escapeHTML(value: string | null | undefined): string;

  // 事件处理
  on(element: HTMLElement, event: string, handler: EventListener): void;
  off(element: HTMLElement, event: string, handler: EventListener): void;

  // 工具函数
  formatDate(date: string | Date, format?: string): string;
  formatNumber(num: number, decimals?: number): string;
  formatBytes(bytes: number): string;
  formatDuration(ms: number): string;

  // 缓存和刷新
  cache: CacheManager;
  refreshManager: RefreshManager;
  eventBus: EventBus;
}

/**
 * 管理应用接口
 */
interface AdminApp {
  // 初始化
  init(): Promise<void>;
  initialized: boolean;

  // 标签页管理
  currentTab: string;
  tabs: string[];
  switchTab(tabId: string): void;
  registerTab(tabId: string, handler: any): void;

  // 数据刷新
  refreshData(): Promise<void>;
  autoRefresh: any;

  // 状态管理
  upstreamDetail: any;
  modules: Record<string, any>;
  moduleManager: any;
  eventBus: any;
}

/**
 * API 客户端接口
 */
interface ApiClient {
  request<T = any>(path: string, options?: any): Promise<T>;
  get<T = any>(path: string, options?: any): Promise<T>;
  post<T = any>(path: string, data?: any, options?: any): Promise<T>;
  put<T = any>(path: string, data?: any, options?: any): Promise<T>;
  delete<T = any>(path: string, options?: any): Promise<T>;
  patch<T = any>(path: string, data?: any, options?: any): Promise<T>;
}

/**
 * Window 接口扩展
 */
declare global {
  interface Window {
    /**
     * 管理控制台引导上下文
     */
    __ADMIN_BOOTSTRAP_CTX__?: AdminBootstrapContext;

    /**
     * 资源版本
     */
    __ASSET_VERSION__?: string;

    /**
     * 基础路径
     */
    __BASE_PATH__?: string;

    /**
     * 延迟加载标志
     */
    __ADMIN_DEFER_LOAD__?: boolean;

    /**
     * 加载管理模块函数
     */
    loadAdminModule?: () => void;

    /**
     * 关闭模态框
     */
    closeModal?: () => void;

    /**
     * 打开模态框
     */
    openModal?: (title: string, content: string) => void;

    /**
     * UI 辅助工具实例
     */
    ui?: UIHelper;

    /**
     * 认证实例
     */
    auth?: import('./services').AuthManager;

    /**
     * API 实例
     */
    api?: ApiClient;

    /**
     * 管理应用实例
     */
    admin?: AdminApp;

    /**
     * 配置管理器
     */
    configManager?: import('./services').ConfigManager;

    /**
     * 日志管理器
     */
    logsManager?: import('./services').LogsManager;

    /**
     * 指标管理器
     */
    metricsManager?: import('./services').MetricsManager;

    /**
     * 凭证管理器
     */
    credsManager?: import('./services').CredentialsManager;

    /**
     * OAuth 管理器
     */
    oauthManager?: import('./services').OAuthManager;

    /**
     * 模型注册表管理器
     */
    registryManager?: import('./services').RegistryManager;

    /**
     * 流式洞察管理器
     */
    streamingManager?: import('./services').StreamingManager;

    /**
     * 装配台管理器
     */
    assemblyManager?: import('./services').AssemblyManager;

    /**
     * 仪表板服务
     */
    dashboard?: import('./services').DashboardService;

    /**
     * 上游服务
     */
    upstream?: import('./services').UpstreamService;
  }
}

/**
 * 模块声明
 */
declare module '*.css' {
  const content: string;
  export default content;
}

declare module '*.json' {
  const value: any;
  export default value;
}

export {};
