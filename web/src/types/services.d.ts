/**
 * 服务相关类型定义
 * 定义所有服务层的接口和类型
 */

import type { Credential, ConfigData, UsageStats, LogEntry, LogQueryOptions } from './api';

/**
 * 认证管理器接口
 */
export interface AuthManager {
  /**
   * 检查是否已认证
   */
  isAuthenticated(): boolean;

  /**
   * 确保已认证
   */
  ensureAuthenticated(): Promise<boolean>;

  /**
   * 登录
   */
  login(key: string): Promise<boolean>;

  /**
   * 登出
   */
  logout(): Promise<void>;

  /**
   * API 请求
   */
  apiRequest<T = any>(path: string, options?: RequestInit): Promise<T>;

  /**
   * 获取基础路径
   */
  getBasePath(): string;

  /**
   * 获取 API 基础 URL
   */
  getApiBase(): string;
}

/**
 * 配置管理器接口
 */
export interface ConfigManager {
  /**
   * 获取配置
   */
  get(key?: string): Promise<any>;

  /**
   * 设置配置
   */
  set(key: string, value: any): Promise<void>;

  /**
   * 批量更新配置
   */
  update(config: Partial<ConfigData>): Promise<void>;

  /**
   * 刷新配置
   */
  refresh(): Promise<void>;

  /**
   * 重置配置
   */
  reset(): Promise<void>;

  /**
   * 导出配置
   */
  export(format?: 'json' | 'yaml'): Promise<string>;

  /**
   * 导入配置
   */
  import(data: string, format?: 'json' | 'yaml'): Promise<void>;
}

/**
 * 凭证管理器接口
 * 注意：这是一个宽松的接口定义，实际实现可能包含更多方法
 */
export interface CredentialsManager {
  // 公共属性
  filters: any;
  page: number;
  pageSize: number;

  // 核心方法
  loadCredentials(): Promise<Credential[]>;
  getCredentials(): Credential[];
  getActiveCredentialsCount(): number;
  getFilteredCredentials(): Credential[];
  getProjectList(): string[];

  // 凭证操作
  enableCredential(filename: string): Promise<void>;
  disableCredential(filename: string): Promise<void>;
  deleteCredential(filename: string): Promise<void>;
  refreshCredentials(): Promise<void>;
  reloadCredentials(): Promise<void>;
  recoverCredential(filename: string): Promise<void>;
  recoverAllCredentials(): Promise<void>;

  // 健康检查
  calculateHealthScore(credential?: Credential | null): number;
  getHealthLevel(score: number): any;
  getHealthColor(level: any): string;

  // 渲染方法
  renderCredentialsList(): string;
  renderCredentialCard(cred: Credential): string;
  renderCredentialsPage(): string;
  renderCredentialsTable(): string;
  renderCredentialRow(cred: Credential): string;
  renderActionButtons(cred: Credential): string;
  renderStatusBadge(cred: Credential): string;
  renderHealthBar(score: number): string;
  renderPager(page: number, pages: number, size: number, total: number): string;

  // DOM 操作
  populateCredentialGrid(gridEl: Element | null): void;
  attachFilters(): void;
  bindDomRefresh(): void;
  setPage(page: number): void;
  setPageSize(size: number): void;
  highlightCredential(identifier: string): void;

  // 批量操作
  toggleBatchMode(): void;
  showBatchMode(): void;
  hideBatchMode(): void;
  performBatchAction(action: string): Promise<void>;
  selectAllCredentials(): void;
  clearSelection(): void;
  handleCheckboxChange(checkbox: HTMLInputElement): void;
  cancelBatchOperation(): void;
  showBatchResults(results: any): void;

  // 其他方法
  initialize(): Promise<void>;
  onRefresh(callback: (credentials: Credential[]) => void): void;
  probeFastLiveness(): Promise<void>;
  formatTimestamp(ts?: number, emptyText?: string): string;
  normalizeTimestamp(value?: number | string | null): number;
  mountVirtual(containerId?: string): void;
  getVirtualPref(): boolean;
  toggleVirtual(v: boolean): void;

  // 允许索引签名以支持动态属性
  [key: string]: any;
}

/**
 * OAuth 管理器接口
 */
export interface OAuthManager {
  /**
   * 开始 OAuth 流程
   */
  startFlow(): Promise<void>;

  /**
   * 处理 OAuth 回调
   */
  handleCallback(code: string): Promise<void>;

  /**
   * 获取授权 URL
   */
  getAuthUrl(): string;

  /**
   * 刷新令牌
   */
  refreshToken(): Promise<void>;

  /**
   * 撤销令牌
   */
  revokeToken(): Promise<void>;
}

/**
 * 日志管理器接口
 */
export interface LogsManager {
  /**
   * 查询日志
   */
  query(options?: LogQueryOptions): Promise<LogEntry[]>;

  /**
   * 获取实时日志流
   */
  stream(callback: (entry: LogEntry) => void): () => void;

  /**
   * 清除日志
   */
  clear(): Promise<void>;

  /**
   * 导出日志
   */
  export(options?: LogQueryOptions): Promise<string>;

  /**
   * 设置日志级别
   */
  setLevel(level: string): Promise<void>;
}

/**
 * 指标管理器接口
 */
export interface MetricsManager {
  /**
   * 获取使用统计
   */
  getUsageStats(timeRange?: { start: string; end: string }): Promise<UsageStats>;

  /**
   * 获取实时指标
   */
  getRealTimeMetrics(): Promise<Record<string, any>>;

  /**
   * 订阅指标更新
   */
  subscribe(callback: (metrics: Record<string, any>) => void): () => void;

  /**
   * 导出指标
   */
  export(format?: 'json' | 'csv'): Promise<string>;
}

/**
 * 流式洞察管理器接口
 */
export interface StreamingManager {
  /**
   * 获取流式洞察数据
   */
  getInsights(): Promise<any>;

  /**
   * 刷新洞察数据
   */
  refresh(): Promise<void>;

  /**
   * 订阅洞察更新
   */
  subscribe(callback: (insights: any) => void): () => void;
}

/**
 * 模型注册表管理器接口
 */
export interface RegistryManager {
  /**
   * 获取所有模型
   */
  listModels(): Promise<any[]>;

  /**
   * 获取单个模型
   */
  getModel(id: string): Promise<any>;

  /**
   * 添加模型
   */
  addModel(model: any): Promise<any>;

  /**
   * 更新模型
   */
  updateModel(id: string, updates: any): Promise<any>;

  /**
   * 删除模型
   */
  deleteModel(id: string): Promise<void>;

  /**
   * 启用模型
   */
  enableModel(id: string): Promise<void>;

  /**
   * 禁用模型
   */
  disableModel(id: string): Promise<void>;

  /**
   * 刷新注册表
   */
  refresh(): Promise<void>;
}

/**
 * 装配台管理器接口
 */
export interface AssemblyManager {
  /**
   * 获取所有装配配置
   */
  listConfigs(): Promise<any[]>;

  /**
   * 获取单个装配配置
   */
  getConfig(id: string): Promise<any>;

  /**
   * 创建装配配置
   */
  createConfig(config: any): Promise<any>;

  /**
   * 更新装配配置
   */
  updateConfig(id: string, updates: any): Promise<any>;

  /**
   * 删除装配配置
   */
  deleteConfig(id: string): Promise<void>;

  /**
   * 激活装配配置
   */
  activateConfig(id: string): Promise<void>;

  /**
   * 停用装配配置
   */
  deactivateConfig(id: string): Promise<void>;

  /**
   * 预览装配配置
   */
  previewConfig(config: any): Promise<any>;

  /**
   * 应用装配配置
   */
  applyConfig(id: string): Promise<void>;

  /**
   * 回滚装配配置
   */
  rollbackConfig(): Promise<void>;
}

/**
 * 仪表板服务接口
 */
export interface DashboardService {
  /**
   * 获取仪表板数据
   */
  getData(): Promise<any>;

  /**
   * 刷新仪表板
   */
  refresh(): Promise<void>;

  /**
   * 获取系统信息
   */
  getSystemInfo(): Promise<any>;

  /**
   * 获取健康状态
   */
  getHealthStatus(): Promise<any>;
}

/**
 * 上游服务接口
 */
export interface UpstreamService {
  /**
   * 获取上游详情
   */
  getDetails(): Promise<any>;

  /**
   * 测试上游连接
   */
  testConnection(): Promise<boolean>;

  /**
   * 获取上游状态
   */
  getStatus(): Promise<any>;
}

/**
 * 存储服务接口
 */
export interface StorageService {
  /**
   * 获取存储信息
   */
  getInfo(): Promise<any>;

  /**
   * 清除缓存
   */
  clearCache(): Promise<void>;

  /**
   * 导出数据
   */
  export(options?: any): Promise<string>;

  /**
   * 导入数据
   */
  import(data: string, options?: any): Promise<void>;

  /**
   * 备份数据
   */
  backup(): Promise<string>;

  /**
   * 恢复数据
   */
  restore(backup: string): Promise<void>;
}

