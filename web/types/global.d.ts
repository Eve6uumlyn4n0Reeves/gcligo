/**
 * Global type definitions for GCLI2API-Go Web Application
 * 
 * This file contains type definitions for:
 * - Window interface extensions
 * - Global utility types
 * - DOM extensions
 * - Third-party library augmentations
 */

// ============================================================================
// Window Interface Extensions
// ============================================================================

interface Window {
  /**
   * Asset version for cache busting
   */
  __ASSET_VERSION__?: string;

  /**
   * Admin bootstrap context
   */
  __ADMIN_BOOTSTRAP_CTX__?: {
    basePath: string;
    version?: string;
    env?: string;
  };

  /**
   * Admin application instance
   */
  admin?: AdminApp;

  /**
   * UI helper instance
   */
  ui?: UIHelper;

  /**
   * Configuration manager
   */
  configManager?: ConfigManager;

  /**
   * Logs manager
   */
  logsManager?: LogsManager;

  /**
   * Metrics view
   */
  metricsView?: MetricsView;

  /**
   * Credentials manager
   */
  credsManager?: CredentialsManager;

  /**
   * Global modal functions
   */
  openModal?: (title: string, content: string, options?: ModalOptions) => void;
  closeModal?: () => void;

  /**
   * Global notification function
   */
  showNotification?: (message: string, type?: NotificationType) => void;

  /**
   * Global loading indicator
   */
  showLoading?: (message?: string) => void;
  hideLoading?: () => void;

  /**
   * Tab management
   */
  tabManager?: TabManager;

  /**
   * API client
   */
  apiClient?: APIClient;
}

// ============================================================================
// Admin Application Types
// ============================================================================

interface AdminApp {
  init(): Promise<void>;
  destroy(): void;
  refresh(): Promise<void>;
  getState(): AdminAppState;
}

interface AdminAppState {
  initialized: boolean;
  loading: boolean;
  error?: Error;
  currentTab?: string;
}

// ============================================================================
// UI Helper Types
// ============================================================================

interface UIHelper {
  // Language and i18n
  dict: Record<string, Record<string, string>>;
  currentLang: string;
  t(key: string): string;
  setLanguage?(lang: string): void;
  loadLang?(): string | null;
  saveLang?(l: string): void;

  // Loading indicators
  globalLoading: HTMLElement | null;
  showGlobalLoading?(message?: string): void;
  hideGlobalLoading?(): void;
  createGlobalLoading?(): void;

  // Notifications
  notificationCenter?: any;
  showNotification?(type?: string, title?: string, message?: string, options?: unknown): void;
  showProgressNotification?(title: string, message?: string, options?: unknown): string;
  removeNotification?(id: string): void;
  getNotificationIcon?(type: string): string;
  clearNotifications?(): void;

  // Modals and dialogs
  dialogs?: any;
  openModal?(title: string, content: string, options?: ModalOptions): void;
  closeModal?(id?: string): void;
  showModal?(title: string, contentHtml: string): HTMLElement | null;
  confirm?(title: string, message: string, options?: any): Promise<boolean>;
  confirmDelete?(itemName?: string): Promise<boolean>;
  confirmWarning?(title: string, message: string, options?: any): Promise<boolean>;
  getConfirmIcon?(type: string): string;
  showConfirmation?(options?: any): void;
  hideConfirmation?(): void;

  // Tooltips
  initTooltips?(): void;
  updateTooltip?(element: HTMLElement, text: string): void;

  // Theme
  currentTheme: string;
  themeKey?: string;
  setTheme?(theme: string): void;
  toggleTheme?(): void;
  applyTheme?(name: string): void;
  loadTheme?(): string | null;
  saveTheme?(t: string): void;

  // Utilities
  escapeHTML?(str: string | null | undefined): string;
  debounce?(fn: Function, wait?: number): Function;
  getHashParams?(): { path: string; params: Record<string, string> };
  setHashParams?(patch?: Record<string, any>, options?: { path?: string }): void;

  // Components
  renderSkeleton?(lines?: number): string;
  renderEmpty?(title?: string, hint?: string): string;
  renderErrorCard?(msg?: string, detail?: string): string;

  // Banner
  banner?(id: string, type: string, text: string): void;
  hideBanner?(id: string): void;

  // Error handling
  showErrorDetails?(info?: any): void;

  // Async operations
  withLoading?<T>(executor: () => Promise<T>, messages?: Record<string, any>): Promise<T>;
  setButtonLoading?(button: HTMLButtonElement | null, loading: boolean): void;

  // Legacy
  showAlert?(type?: string, title?: string, message?: string): void;
}

interface Dialog {
  id: string;
  element: HTMLElement;
  title: string;
  content: string;
  options?: ModalOptions;
}

interface ModalOptions {
  width?: string;
  height?: string;
  closable?: boolean;
  backdrop?: boolean;
  onClose?: () => void;
  buttons?: ModalButton[];
}

interface ModalButton {
  text: string;
  type?: 'primary' | 'secondary' | 'danger';
  onClick?: () => void;
}

type NotificationType = 'success' | 'error' | 'warning' | 'info';

// ============================================================================
// Manager Types
// ============================================================================

interface ConfigManager {
  load(): Promise<void>;
  save(config: any): Promise<void>;
  get(key: string): any;
  set(key: string, value: any): void;
  reset(): void;
}

interface LogsManager {
  load(): Promise<void>;
  refresh(): Promise<void>;
  clear(): Promise<void>;
  download(): void;
  filter(query: string): void;
}

interface MetricsView {
  load(): Promise<void>;
  refresh(): Promise<void>;
  updateChart(data: any): void;
}

interface CredentialsManager {
  load(): Promise<void>;
  add(credential: Credential): Promise<void>;
  remove(id: string): Promise<void>;
  update(id: string, credential: Partial<Credential>): Promise<void>;
  refresh(): Promise<void>;
}

interface Credential {
  id: string;
  name: string;
  type: string;
  token?: string;
  apiKey?: string;
  createdAt: string;
  updatedAt: string;
  status: 'active' | 'inactive' | 'expired';
}

// ============================================================================
// Tab Manager Types
// ============================================================================

interface TabManager {
  currentTab: string;
  tabs: Map<string, Tab>;
  switchTab(tabId: string): void;
  registerTab(tab: Tab): void;
  unregisterTab(tabId: string): void;
}

interface Tab {
  id: string;
  name: string;
  element: HTMLElement;
  onActivate?: () => void;
  onDeactivate?: () => void;
}

// ============================================================================
// API Client Types
// ============================================================================

interface APIClient {
  get<T = any>(url: string, options?: RequestOptions): Promise<T>;
  post<T = any>(url: string, data?: any, options?: RequestOptions): Promise<T>;
  put<T = any>(url: string, data?: any, options?: RequestOptions): Promise<T>;
  delete<T = any>(url: string, options?: RequestOptions): Promise<T>;
  patch<T = any>(url: string, data?: any, options?: RequestOptions): Promise<T>;
}

interface RequestOptions {
  headers?: Record<string, string>;
  timeout?: number;
  retries?: number;
  signal?: AbortSignal;
}

// ============================================================================
// Utility Types
// ============================================================================

/**
 * Make all properties of T optional recursively
 */
type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

/**
 * Make all properties of T required recursively
 */
type DeepRequired<T> = {
  [P in keyof T]-?: T[P] extends object ? DeepRequired<T[P]> : T[P];
};

/**
 * Extract promise type
 */
type Awaited<T> = T extends Promise<infer U> ? U : T;

/**
 * Function that returns a promise
 */
type AsyncFunction<T = any> = (...args: any[]) => Promise<T>;

/**
 * Event handler type
 */
type EventHandler<T = Event> = (event: T) => void;

/**
 * Callback type
 */
type Callback<T = void> = (result: T) => void;

/**
 * Error callback type
 */
type ErrorCallback = (error: Error) => void;

// ============================================================================
// DOM Utility Types
// ============================================================================

/**
 * HTML element with specific tag name
 */
type HTMLElementTagNameMap = {
  div: HTMLDivElement;
  span: HTMLSpanElement;
  button: HTMLButtonElement;
  input: HTMLInputElement;
  select: HTMLSelectElement;
  textarea: HTMLTextAreaElement;
  form: HTMLFormElement;
  a: HTMLAnchorElement;
  img: HTMLImageElement;
  table: HTMLTableElement;
  tr: HTMLTableRowElement;
  td: HTMLTableCellElement;
  th: HTMLTableCellElement;
  ul: HTMLUListElement;
  ol: HTMLOListElement;
  li: HTMLLIElement;
};

/**
 * Query selector result type
 */
type QueryResult<T extends keyof HTMLElementTagNameMap> = HTMLElementTagNameMap[T] | null;

// ============================================================================
// Module Path Types
// ============================================================================

/**
 * Module path resolver function
 */
type ModulePathResolver = (name: string, path: string) => string;

/**
 * Module loader function
 */
type ModuleLoader<T = any> = (path: string) => Promise<T>;

// ============================================================================
// Export for module augmentation
// ============================================================================

export {};

