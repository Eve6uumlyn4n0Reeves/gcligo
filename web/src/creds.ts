import { auth } from './auth.js';
import { ui } from './ui.js';
import { credsService } from './creds_service.js';
import { createVirtualizer } from './utils/virtualize.js';
import { applyTableAria } from './utils/a11y.js';

import {
  type Credential,
  type CredentialFilters,
  type CredentialBatchAction
} from './creds/types.js';
import {
  type HealthThresholds,
  DEFAULT_THRESHOLDS,
  calculateHealthScore as calcHealthScore,
  getHealthLevel as resolveHealthLevel,
  getHealthColor as resolveHealthColor
} from './creds/health.js';
import { filterCredentials, collectProjects } from './creds/filter.js';
import { paginate } from './creds/pagination.js';

declare global {
  interface Window {
    // admin 和 credsManager 已在 global.d.ts 中定义
    credsViewRenderCredentialsList?: (manager: CredentialsManager) => string;
    credsViewPopulateCredentialGrid?: (
      grid: Element | null,
      items: Credential[],
      manager: CredentialsManager
    ) => void;
    credsViewRenderPager?: (
      page: number,
      pages: number,
      pageSize: number,
      total: number
    ) => string;
    credsViewRenderCredentialCard?: (credential: Credential, manager: CredentialsManager) => string;
    credsViewRenderCredentialsTable?: (manager: CredentialsManager) => string;
    credsViewRenderCredentialRow?: (credential: Credential, manager: CredentialsManager) => string;
    credsViewRenderCredentialGrid?: (credentials: Credential[], manager: CredentialsManager) => string;
  }
}

type RefreshCallback = (credentials: Credential[]) => void;

const DEFAULT_FILTERS: CredentialFilters = {
  search: '',
  status: 'all',
  health: 'all',
  project: 'all'
};

const FILTER_CACHE_TTL = 1000;
const MAX_FILTER_CACHE = 10;

export class CredentialsManager {
  private credentials: Credential[] = [];
  private refreshCallbacks: RefreshCallback[] = [];
  public filters: CredentialFilters = { ...DEFAULT_FILTERS };

  public page = 1;
  public pageSize = 20;
  private pendingCredentialView: Credential[] = [];
  private domCallbackRegistered = false;
  private virtualizer: any = null;

  private filterCache: Map<string, Credential[]> = new Map();
  private lastFilterTime = 0;
  private readonly filterDebounceMs = 300;

  private readonly healthThresholds: HealthThresholds = { ...DEFAULT_THRESHOLDS };

  private batchMode = false;
  private selectedItems: Set<string> = new Set();
  private batchProgress: any = null;

  constructor() {}

  async loadCredentials(): Promise<Credential[]> {
    try {
      this.credentials = await credsService.list();
      return this.credentials;
    } catch (error) {
      console.error('Failed to load credentials:', error);
      auth.showAlert('error', '加载凭证失败: ' + (error as Error).message);
      return [];
    }
  }

  getCredentials(): Credential[] {
    return this.credentials;
  }

  getActiveCredentialsCount(): number {
    return this.credentials.filter((cred) => !cred.disabled).length;
  }

  calculateHealthScore(credential?: Credential | null): number {
    if (!credential) return 0;
    return calcHealthScore(credential);
  }

  getHealthLevel(score: number): ReturnType<typeof resolveHealthLevel> {
    return resolveHealthLevel(score, this.healthThresholds);
  }

  getHealthColor(level: ReturnType<typeof resolveHealthLevel>): string {
    return resolveHealthColor(level);
  }

  getFilteredCredentials(): Credential[] {
    const cacheKey = `${this.filters.search}|${this.filters.status}|${this.filters.health}|${this.filters.project}`;
    const now = Date.now();
    if (this.filterCache.has(cacheKey) && now - this.lastFilterTime < FILTER_CACHE_TTL) {
      return this.filterCache.get(cacheKey)!;
    }

    const filtered = filterCredentials(this.credentials, this.filters, {
      thresholds: this.healthThresholds,
      now
    });

    this.filterCache.set(cacheKey, filtered);
    this.lastFilterTime = now;
    if (this.filterCache.size > MAX_FILTER_CACHE) {
      const oldestKey = this.filterCache.keys().next().value;
      if (oldestKey) {
        this.filterCache.delete(oldestKey);
      }
    }

    return filtered;
  }

  getProjectList(): string[] {
    return collectProjects(this.credentials);
  }

  async enableCredential(filename: string): Promise<void> {
    try {
      await credsService.enable(filename);
      auth.showAlert('success', '凭证已启用');
      await this.refreshCredentials();
    } catch (error) {
      auth.showAlert('error', '启用失败: ' + (error as Error).message);
    }
  }

  async disableCredential(filename: string): Promise<void> {
    try {
      await credsService.disable(filename);
      auth.showAlert('success', '凭证已禁用');
      await this.refreshCredentials();
    } catch (error) {
      auth.showAlert('error', '禁用失败: ' + (error as Error).message);
    }
  }

  async deleteCredential(filename: string): Promise<void> {
    const confirmed = await this.confirmDelete(filename);
    if (!confirmed) return;

    try {
      if (window.ui?.withLoading) {
        await window.ui.withLoading(
          () => credsService.delete(filename),
          {
            loadingText: '正在删除凭证...',
            successMessage: '凭证删除成功',
            errorMessage: '删除凭证失败'
          }
        );
      } else {
        await credsService.delete(filename);
        auth.showAlert('success', '凭证已删除');
      }
      await this.refreshCredentials();
    } catch (error) {
      console.error('Failed to delete credential:', error);
      if (!window.ui?.withLoading) {
        auth.showAlert('error', '删除失败: ' + (error as Error).message);
      }
    }
  }

  private async confirmDelete(filename: string): Promise<boolean> {
    if (window.ui?.showConfirmation) {
      return window.ui.showConfirmation({
        title: '删除凭证',
        message: `确定要删除凭证 "${filename}" 吗？此操作无法撤销。`,
        type: 'danger',
        confirmText: '删除',
        confirmClass: 'btn-danger'
      });
    }
    if (window.ui?.confirm) {
      return window.ui.confirm('删除凭证', `确定要删除凭证 "${filename}" 吗？此操作不可恢复。`, {
        okText: '删除',
        cancelText: '取消'
      });
    }
    return window.confirm(`确定要删除凭证 "${filename}" 吗？此操作不可恢复。`);
  }

  async reloadCredentials(): Promise<void> {
    try {
      await credsService.reload();
      auth.showAlert('success', '凭证已重载');
      await this.refreshCredentials();
    } catch (error) {
      auth.showAlert('error', '重载失败: ' + (error as Error).message);
    }
  }

  async recoverAllCredentials(): Promise<void> {
    try {
      const execute = () => credsService.recoverAll();
      const result = window.ui?.withLoading
        ? await window.ui.withLoading(execute, {
            loadingText: '正在恢复所有凭证...',
            successMessage: '批量恢复已完成',
            errorMessage: '批量恢复失败'
          })
        : await execute();
      const recovered =
        result && typeof result === 'object' && 'count' in result
          ? Number((result as Record<string, unknown>).count ?? 0)
          : 0;
      if (!window.ui?.withLoading) {
        auth.showAlert('success', `批量恢复完成，已恢复 ${recovered} 个凭证`);
      } else {
        auth.showAlert('info', `已恢复 ${recovered} 个凭证`);
      }
      await this.refreshCredentials();
    } catch (error) {
      console.error('Failed to recover all credentials:', error);
      auth.showAlert('error', '批量恢复失败: ' + (error as Error).message);
    }
  }

  async recoverCredential(filename: string): Promise<void> {
    if (!filename) return;
    try {
      const execute = () => credsService.recover(filename);
      if (window.ui?.withLoading) {
        await window.ui.withLoading(execute, {
          loadingText: `正在恢复凭证 "${filename}"...`,
          successMessage: `凭证 "${filename}" 已恢复`,
          errorMessage: `恢复凭证 "${filename}" 失败`
        });
      } else {
        await execute();
        auth.showAlert('success', `凭证 "${filename}" 已恢复`);
      }
      await this.refreshCredentials();
      this.highlightCredential(filename);
    } catch (error) {
      console.error(`Failed to recover credential ${filename}:`, error);
      auth.showAlert('error', `恢复失败: ${(error as Error).message}`);
    }
  }

  async refreshCredentials(): Promise<void> {
    await this.loadCredentials();
    this.notifyRefreshCallbacks();
  }

  highlightCredential(identifier: string): void {
    if (!identifier) return;
    const list = document.getElementById('credentialsList');
    if (!list) return;
    const esc =
      window.CSS?.escape?.(identifier) ??
      identifier.replace(/"/g, '\\"');
    const card = list.querySelector<HTMLElement>(`[data-cred-id="${esc}"]`);
    if (!card) return;
    card.classList.add('row-highlight');
    card.scrollIntoView({ behavior: 'smooth', block: 'center' });
    setTimeout(() => card.classList.remove('row-highlight'), 2000);
  }

  onRefresh(callback: RefreshCallback): void {
    this.refreshCallbacks.push(callback);
  }

  private notifyRefreshCallbacks(): void {
    this.refreshCallbacks.forEach((callback) => callback(this.credentials));
  }

  bindDomRefresh(): void {
    if (this.domCallbackRegistered) return;
    this.domCallbackRegistered = true;
    this.onRefresh(() => {
      const list = document.getElementById('credentialsList');
      if (list) {
        list.innerHTML = this.renderCredentialsList();
        if (this.getVirtualPref()) {
          this.mountVirtual();
        } else {
          const grid = list.querySelector<HTMLElement>('.credentials-grid');
          if (window.credsViewPopulateCredentialGrid) {
            window.credsViewPopulateCredentialGrid(grid, this.pendingCredentialView, this);
          } else {
            this.populateCredentialGrid(grid);
          }
        }
      }
      this.createBatchToolbar();
      this.createBatchModeToggleButton();
    });
  }

  normalizeTimestamp(value?: number | string | null): number {
    if (!value) return 0;
    if (typeof value === 'number') return value;
    const parsed = Date.parse(value);
    return Number.isNaN(parsed) ? 0 : Math.floor(parsed / 1000);
  }

  formatTimestamp(ts?: number, emptyText = '从未'): string {
    if (!ts) return emptyText;
    return new Date(ts * 1000).toLocaleString();
  }

  renderCredentialsList(): string {
    const filtered = this.getFilteredCredentials();
    const { items, pages, page } = paginate(filtered, this.page, this.pageSize);
    this.page = page;
    this.pendingCredentialView = items;
    if (items.length === 0) {
      return `<div class="empty"><div class="empty-icon">🔐</div><div class="empty-title">暂无凭证</div><div class="empty-hint">建议先通过 OAuth 或上传 JSON 添加凭证</div><div style="margin-top:8px;display:flex;gap:8px;justify-content:center;"><button class="btn btn-primary" onclick="window.admin.switchTab('oauth')">➕ 添加凭证</button></div></div>`;
    }
    return window.credsViewRenderCredentialsList
      ? window.credsViewRenderCredentialsList(this)
      : this.renderInternalCredentialList(items, page, pages);
  }

  private renderInternalCredentialList(
    credentials: Credential[],
    page: number,
    pages: number
  ): string {
    const rows = credentials
      .map((cred) => window.credsViewRenderCredentialCard
        ? window.credsViewRenderCredentialCard(cred, this)
        : this.renderCredentialCard(cred))
      .join('');
    const pager = window.credsViewRenderPager
      ? window.credsViewRenderPager(page, pages, this.pageSize, this.getFilteredCredentials().length)
      : '';
    return `
      <div class="credentials-grid">
        ${rows}
      </div>
      ${pager}
    `;
  }

  renderCredentialCard(cred: Credential): string {
    const lastSuccessTs = this.normalizeTimestamp(cred.last_success_ts ?? cred.last_success);
    const lastFailureTs = this.normalizeTimestamp(cred.last_failure_ts ?? cred.last_failure);
    const isAutoBanned = Boolean(cred.auto_banned);
    const isDisabled = Boolean(cred.disabled);
    const credKey =
      cred.filename ?? cred.id ?? cred.email ?? cred.project_id ?? '';
    const safeKey = String(credKey).replace(/"/g, '&quot;');
    const statusLabel = isAutoBanned
      ? `已封禁: ${cred.banned_reason || '未知原因'}`
      : isDisabled
      ? '已禁用'
      : '活跃';
    const statusClass = isAutoBanned || isDisabled ? 'status-disabled' : 'status-active';

    return `
      <div class="credential-card ${isDisabled ? 'disabled' : ''}" data-cred-id="${safeKey}">
        <div class="credential-header">
          <div class="credential-name">${cred.filename ?? cred.id ?? '未命名凭证'}</div>
          <div class="credential-status ${statusClass}">
            ${statusLabel}
          </div>
        </div>
        <div class="credential-info">项目: ${cred.project_id ?? 'N/A'}</div>
        <div class="credential-info">邮箱: ${cred.email ?? 'N/A'}</div>
        <div class="credential-info">总调用: ${cred.total_calls ?? cred.total_requests ?? 0}</div>
        <div class="credential-info">Gemini 2.5 Pro: ${cred.gemini_2_5_pro_calls ?? 0}</div>
        <div class="credential-info">成功率: ${(((cred.success_rate ?? 0) * 100).toFixed(1))}%</div>
        <div class="credential-info">失败权重: ${(cred.failure_weight ?? 0).toFixed(2)}</div>
        <div class="credential-info">健康评分: ${(((cred.health_score ?? 0) * 100).toFixed(0))}%</div>
        <div class="credential-info">最后成功: ${this.formatTimestamp(lastSuccessTs)}</div>
        <div class="credential-info">最后失败: ${this.formatTimestamp(lastFailureTs, '无记录')}</div>
        ${cred.error_codes?.length
          ? `<div class="credential-info" style="color: #ef4444;">错误码: ${cred.error_codes.join(', ')}</div>`
          : ''
        }
        <div class="credential-actions">
          ${this.renderActionButtons(cred)}
        </div>
      </div>
    `;
  }

  renderActionButtons(cred: Credential): string {
    const id = cred.filename ?? cred.id ?? '';
    if (cred.auto_banned) {
      return `
        <button class="btn btn-success btn-sm" aria-label="恢复凭证 ${id}" onclick="window.credsManager.recoverCredential('${id}')">恢复</button>
        <button class="btn btn-danger btn-sm" aria-label="删除凭证 ${id}" onclick="window.credsManager.deleteCredential('${id}')">删除</button>
      `;
    }
    if (cred.disabled) {
      return `
        <button class="btn btn-success btn-sm" aria-label="启用凭证 ${id}" onclick="window.credsManager.enableCredential('${id}')">启用</button>
        <button class="btn btn-danger btn-sm" aria-label="删除凭证 ${id}" onclick="window.credsManager.deleteCredential('${id}')">删除</button>
      `;
    }
    return `
      <button class="btn btn-warning btn-sm" aria-label="禁用凭证 ${id}" onclick="window.credsManager.disableCredential('${id}')">禁用</button>
      <button class="btn btn-danger btn-sm" aria-label="删除凭证 ${id}" onclick="window.credsManager.deleteCredential('${id}')">删除</button>
    `;
  }

  renderPager(page: number, pages: number, size: number, total: number): string {
    if (window.credsViewRenderPager) {
      return window.credsViewRenderPager(page, pages, size, total);
    }
    return '';
  }

  populateCredentialGrid(gridEl: Element | null): void {
    if (!gridEl) return;
    const fragment = document.createDocumentFragment();
    for (const cred of this.pendingCredentialView) {
      const wrapper = document.createElement('div');
      wrapper.innerHTML = this.renderCredentialCard(cred);
      const card = wrapper.firstElementChild;
      if (card) fragment.appendChild(card);
    }
    gridEl.innerHTML = '';
    gridEl.appendChild(fragment);
  }

  mountVirtual(containerId = 'credVirtual'): void {
    const host = document.getElementById(containerId);
    if (!host) return;
    if (this.virtualizer?.destroy) {
      this.virtualizer.destroy();
      this.virtualizer = null;
    }

    const filtered = this.getFilteredCredentials();
    const indices = filtered.map((_, idx) => idx);

    applyTableAria(host, '凭证卡片（虚拟滚动）');

    const render = (index: number) => {
      const cred = filtered[indices[index]];
      const row = document.createElement('div');
      row.className = 'metrics-virtual-row';
      row.setAttribute('role', 'row');
      const wrapper = document.createElement('div');
      wrapper.innerHTML = this.renderCredentialCard(cred);
      const card = wrapper.firstElementChild;
      if (card) row.appendChild(card);
      return row;
    };

    this.virtualizer = createVirtualizer(host, {
      itemCount: indices.length,
      rowHeight: 220,
      overscan: 6,
      render
    });
  }

  getVirtualPref(): boolean {
    try {
      const v = localStorage.getItem('ui:credsVirtual');
      if (v === '1' || v === '0') return v === '1';
    } catch {
      // ignore
    }
    try {
      const total = Array.isArray(this.credentials) ? this.credentials.length : 0;
      if (total > 80) return true;
    } catch {
      // ignore
    }
    return false;
  }

  toggleVirtual(v: boolean): void {
    try {
      localStorage.setItem('ui:credsVirtual', v ? '1' : '0');
    } catch {
      // ignore
    }
    const list = document.getElementById('credentialsList');
    if (!list) return;
    const html = window.credsViewRenderCredentialsList
      ? window.credsViewRenderCredentialsList(this)
      : this.renderCredentialsList();
    list.innerHTML = html;
    if (this.getVirtualPref()) {
      this.mountVirtual();
    } else {
      const grid = list.querySelector<HTMLElement>('.credentials-grid');
      if (window.credsViewPopulateCredentialGrid) {
        window.credsViewPopulateCredentialGrid(grid, this.pendingCredentialView, this);
      } else {
        this.populateCredentialGrid(grid);
      }
    }
  }

  renderCredentialsPage(): string {
    return `
      <div class="card">
        <h2>凭证管理</h2>
        <div style="margin: 16px 0; display:flex; gap:10px; flex-wrap:wrap; align-items:center;">
          <button class="btn btn-primary" onclick="window.credsManager.reloadCredentials()">🔄 重载凭证</button>
          <button class="btn" onclick="window.credsManager.probeFastLiveness()">🩺 测活（flash）</button>
          <button class="btn btn-info" onclick="window.admin.switchTab('oauth')">➕ 添加凭证</button>
          <input id="credSearch" type="text" placeholder="搜索 文件名/邮箱/项目ID" style="padding: 8px; border-radius: 6px; border: 1px solid #e5e7eb; min-width: 240px;">
          <select id="credStatus" class="form-control" style="padding: 8px; border-radius: 6px; border: 1px solid #e5e7eb;">
            <option value="all">全部</option>
            <option value="active">活跃</option>
            <option value="disabled">已禁用</option>
            <option value="banned">已封禁</option>
          </select>
        </div>
        <div id="credentialsList">
          <div class="loading">
            <div class="spinner"></div>
            <p>加载凭证中...</p>
          </div>
        </div>
      </div>
    `;
  }

  async probeFastLiveness(): Promise<void> {
    try {
      const res = await credsService.probeFlash('gemini-2.5-flash', 10) as any;
      const ok = (res?.results ?? []).filter((r: { ok: boolean }) => r.ok).length;
      const total = (res?.results ?? []).length;
      auth.showAlert('info', `测活完成：${ok}/${total} 正常`);
      await this.refreshCredentials();
      window.dispatchEvent(new CustomEvent('probe-history-updated'));
    } catch (error) {
      auth.showAlert('error', '测活失败: ' + (error as Error).message);
    }
  }

  attachFilters(): void {
    const searchEl = document.getElementById('credSearch') as HTMLInputElement | null;
    const statusEl = document.getElementById('credStatus') as HTMLSelectElement | null;
    let timer: number | null = null;

    if (searchEl) {
      searchEl.value = this.filters.search;
      searchEl.addEventListener('input', () => {
        if (timer) window.clearTimeout(timer);
        timer = window.setTimeout(() => {
          this.page = 1;
          this.filters.search = searchEl.value;
          this.refreshListDom();
        }, this.filterDebounceMs);
      });
    }

    if (statusEl) {
      statusEl.value = this.filters.status;
      statusEl.addEventListener('change', () => {
        this.page = 1;
        this.filters.status = statusEl.value as CredentialFilters['status'];
        this.refreshListDom();
      });
    }
  }

  setPage(page: number): void {
    this.page = Math.max(1, Number.parseInt(String(page), 10) || 1);
    this.refreshListDom();
  }

  setPageSize(size: number): void {
    this.pageSize = Number.parseInt(String(size), 10) || 20;
    this.page = 1;
    this.refreshListDom();
  }

  private refreshListDom(): void {
    const list = document.getElementById('credentialsList');
    if (!list) return;
    const html = window.credsViewRenderCredentialsList
      ? window.credsViewRenderCredentialsList(this)
      : this.renderCredentialsList();
    list.innerHTML = html;
    if (this.getVirtualPref()) {
      this.mountVirtual();
    } else {
      const grid = list.querySelector<HTMLElement>('.credentials-grid');
      if (window.credsViewPopulateCredentialGrid) {
        window.credsViewPopulateCredentialGrid(grid, this.pendingCredentialView, this);
      } else {
        this.populateCredentialGrid(grid);
      }
    }
  }

  renderCredentialsTable(): string {
    if (window.credsViewRenderCredentialsTable) {
      return window.credsViewRenderCredentialsTable(this);
    }
    return '<p>暂无凭证</p>';
  }

  renderCredentialRow(cred: Credential): string {
    if (window.credsViewRenderCredentialRow) {
      return window.credsViewRenderCredentialRow(cred, this);
    }
    const healthScore = (cred.health_score ?? 0) * 100;
    const successRate = (cred.success_rate ?? 0) * 100;

    return `
      <tr>
        <td>${cred.id ?? cred.filename ?? ''}</td>
        <td>${cred.project_id ?? '-'}</td>
        <td>${cred.email ?? '-'}</td>
        <td>${this.renderStatusBadge(cred)}</td>
        <td>${this.renderHealthBar(healthScore)}</td>
        <td>${cred.total_requests ?? cred.total_calls ?? 0}</td>
        <td>${successRate.toFixed(1)}%</td>
        <td>${this.formatTimestamp(this.normalizeTimestamp(cred.last_success_ts ?? cred.last_success))}</td>
        <td>${this.formatTimestamp(this.normalizeTimestamp(cred.last_failure_ts ?? cred.last_failure), '无记录')}</td>
        <td>${this.renderActionButtons(cred)}</td>
      </tr>
    `;
  }

  renderStatusBadge(cred: Credential): string {
    if (cred.auto_banned) {
      return `<span class="status-badge status-disabled">已封禁: ${cred.banned_reason || '未知原因'}</span>`;
    }
    if (cred.disabled) {
      return `<span class="status-badge status-disabled">已禁用</span>`;
    }
    return `<span class="status-badge status-active">活跃</span>`;
  }

  renderHealthBar(score: number): string {
    let className = 'low';
    if (score >= 70) className = 'high';
    else if (score >= 40) className = 'medium';
    return `
      <div class="health-bar">
        <div class="health-bar-fill ${className}" style="width: ${score}%"></div>
      </div>
      <span style="font-size: 12px; color: #666;">${score.toFixed(0)}%</span>
    `;
  }

  async initialize(): Promise<void> {
    window.addEventListener('credentialsChanged', () => {
      this.refreshCredentials();
    });
    this.initBatchOperations();
    await this.loadCredentials();
  }

  private initBatchOperations(): void {
    this.selectedItems = new Set();
    this.createBatchToolbar();
  }

  async performBatchOperation(
    operation: string,
    items: string[],
    options: {
      confirmTitle?: string;
      confirmMessage?: string;
      successMessage?: string;
      operationName?: string;
      concurrency?: number;
    } = {}
  ): Promise<void> {
    if (items.length === 0) {
      ui.showNotification?.('warning', '没有选择项目', '请先选择要操作的凭证');
      return;
    }

    const {
      confirmTitle = '确认批量操作',
      confirmMessage = `确定要对 ${items.length} 个凭证执行 ${operation} 操作吗？`,
      successMessage = '批量操作完成',
      operationName = operation,
      concurrency = 3
    } = options;

    const confirmed = await ui.confirm?.(confirmTitle, confirmMessage, {
      type: 'warning',
      okText: '确认',
      cancelText: '取消'
    });
    if (!confirmed) return;

    const progress: any = ui.showProgressNotification?.(
      '批量操作进行中',
      `正在${operationName} 0/${items.length} 个凭证`,
      {
        type: 'info',
        showProgress: true,
        showCancel: true,
        onCancel: () => this.cancelBatchOperation()
      }
    );

    const results = {
      success: [] as string[],
      failed: [] as string[],
      errors: [] as string[]
    };
    let completed = 0;
    this.batchProgress = progress;

    try {
      for (let i = 0; i < items.length; i += concurrency) {
        const chunk = items.slice(i, i + concurrency);
        await Promise.allSettled(
          chunk.map(async (item) => {
            try {
              await this.executeSingleOperation(operation as CredentialBatchAction, item);
              results.success.push(item);
            } catch (error) {
              results.failed.push(item);
              results.errors.push(`${item}: ${(error as Error).message}`);
            } finally {
              completed++;
              const progressPercent = Math.round((completed / items.length) * 100);
              progress?.update?.(
                progressPercent,
                `正在${operationName} ${completed}/${items.length} 个凭证`
              );
            }
          })
        );
      }

      if (results.failed.length === 0) {
        ui.showNotification?.('success', successMessage, `成功${operationName} ${results.success.length} 个凭证`);
      } else {
        ui.showNotification?.(
          'warning',
          '部分操作失败',
          `成功: ${results.success.length}, 失败: ${results.failed.length}`,
          {
            actions: [
              {
                id: 'details',
                text: '查看详情',
                handler: () => this.showBatchResults(results)
              }
            ]
          }
        );
      }
    } catch (error) {
      ui.showNotification?.('error', '批量操作失败', (error as Error).message);
    } finally {
      progress?.close?.();
      this.batchProgress = null;
      await this.refreshCredentials();
      this.clearSelection();
    }
  }

  private async executeSingleOperation(
    operation: CredentialBatchAction,
    filename: string
  ): Promise<void> {
    switch (operation) {
      case 'enable':
        await credsService.enable(filename);
        break;
      case 'disable':
        await credsService.disable(filename);
        break;
      case 'delete':
        await credsService.delete(filename);
        break;
      case 'health-check':
        await credsService.probeFlash();
        break;
      default:
        throw new Error(`Unknown operation: ${operation}`);
    }
  }

  cancelBatchOperation(): void {
    this.batchProgress?.close?.();
    this.batchProgress = null;
    ui.showNotification?.('info', '操作已取消', '批量操作已被用户取消');
  }

  showBatchResults(results: { success: string[]; failed: string[]; errors: string[] }): void {
    const content = `
      <div class="batch-results">
        <div class="results-summary">
          <div class="result-item success">
            <span class="result-icon">✅</span>
            <span class="result-text">成功: ${results.success.length}</span>
          </div>
          <div class="result-item failed">
            <span class="result-icon">❌</span>
            <span class="result-text">失败: ${results.failed.length}</span>
          </div>
        </div>
        ${
          results.errors.length > 0
            ? `<div class="error-details">
                <h4>错误详情:</h4>
                <ul>${results.errors.map((error) => `<li>${error}</li>`).join('')}</ul>
              </div>`
            : ''
        }
      </div>
    `;
    ui.showModal?.('批量操作结果', content);
  }

  private createBatchToolbar(): void {
    if (document.querySelector('.batch-toolbar')) return;

    const toolbar = document.createElement('div');
    toolbar.className = 'batch-toolbar';
    toolbar.id = 'batch-toolbar';
    toolbar.style.display = 'none';
    toolbar.innerHTML = `
      <div class="batch-toolbar-content">
        <div class="batch-selection-info">
          <span class="batch-count">已选择 0 项</span>
          <button type="button" class="btn btn-link btn-sm" onclick="credsManager.selectAllCredentials()">全选</button>
          <button type="button" class="btn btn-link btn-sm" onclick="credsManager.clearSelection()">清除选择</button>
        </div>
        <div class="batch-actions">
          <div class="btn-group">
            <button type="button" class="btn btn-success btn-sm" onclick="credsManager.performBatchAction('enable')">
              <i class="icon">✓</i> 启用
            </button>
            <button type="button" class="btn btn-warning btn-sm" onclick="credsManager.performBatchAction('disable')">
              <i class="icon">⏸</i> 禁用
            </button>
            <button type="button" class="btn btn-danger btn-sm" onclick="credsManager.performBatchAction('delete')">
              <i class="icon">🗑</i> 删除
            </button>
          </div>
          <div class="btn-group">
            <button type="button" class="btn btn-info btn-sm" onclick="credsManager.performBatchAction('health-check')">
              <i class="icon">🔍</i> 快速测活
            </button>
            <button type="button" class="btn btn-secondary btn-sm" onclick="credsManager.performBatchAction('export')">
              <i class="icon">📥</i> 导出数据
            </button>
          </div>
        </div>
        <button type="button" class="batch-close" onclick="credsManager.hideBatchMode()" aria-label="关闭批量模式">×</button>
      </div>
      <div class="batch-progress" style="display: none;">
        <div class="progress-bar">
          <div class="progress-fill"></div>
        </div>
        <div class="progress-text">处理中...</div>
      </div>
    `;

    const mainContent = document.querySelector('.main-content');
    if (mainContent) {
      mainContent.insertBefore(toolbar, mainContent.firstChild);
    }
  }

  private createBatchModeToggleButton(): void {
    if (document.getElementById('batch-mode-toggle-btn')) return;
    const button = document.createElement('button');
    button.id = 'batch-mode-toggle-btn';
    button.className = 'batch-mode-toggle';
    button.title = '批量操作';
    button.setAttribute('aria-label', '切换批量操作模式');
    button.innerHTML = '☑';
    button.onclick = () => this.toggleBatchMode();
    document.body.appendChild(button);
  }

  toggleBatchMode(): void {
    const button = document.getElementById('batch-mode-toggle-btn');
    if (this.batchMode) {
      this.hideBatchMode();
      button?.classList.remove('active');
      if (button) {
        button.innerHTML = '☑';
        button.title = '批量操作';
      }
    } else {
      this.showBatchMode();
      button?.classList.add('active');
      if (button) {
        button.innerHTML = '×';
        button.title = '退出批量模式';
      }
    }
    this.batchMode = !this.batchMode;
  }

  showBatchMode(): void {
    const toolbar = document.getElementById('batch-toolbar');
    if (toolbar) {
      toolbar.style.display = 'block';
      toolbar.classList.add('active');
    }
    this.addBatchCheckboxes();
    document.body.classList.add('batch-mode');
  }

  hideBatchMode(): void {
    const toolbar = document.getElementById('batch-toolbar');
    if (toolbar) {
      toolbar.style.display = 'none';
      toolbar.classList.remove('active');
    }
    this.removeBatchCheckboxes();
    this.clearSelection();
    document.body.classList.remove('batch-mode');
  }

  private addBatchCheckboxes(): void {
    document.querySelectorAll<HTMLElement>('.credential-card').forEach((card) => {
      if (card.querySelector('.batch-checkbox')) return;
      const filename = card.dataset.filename ?? card.getAttribute('data-filename') ?? card.dataset.credId ?? '';
      if (!filename) return;
      const overlay = document.createElement('div');
      overlay.className = 'batch-checkbox-overlay';
      overlay.innerHTML = `
        <input type="checkbox" class="batch-checkbox" data-item-id="${filename}"
          onchange="credsManager.handleCheckboxChange(this)">
      `;
      card.style.position = 'relative';
      card.appendChild(overlay);
    });
  }

  private removeBatchCheckboxes(): void {
    document.querySelectorAll('.batch-checkbox-overlay').forEach((overlay) => overlay.remove());
  }

  handleCheckboxChange(checkbox: HTMLInputElement): void {
    const itemId = checkbox.dataset.itemId ?? '';
    if (checkbox.checked) {
      this.selectedItems.add(itemId);
    } else {
      this.selectedItems.delete(itemId);
    }
    this.updateBatchUI();
  }

  selectAllCredentials(): void {
    document.querySelectorAll<HTMLInputElement>('.batch-checkbox:not([disabled])').forEach((checkbox) => {
      checkbox.checked = true;
      if (checkbox.dataset.itemId) {
        this.selectedItems.add(checkbox.dataset.itemId);
      }
    });
    this.updateBatchUI();
  }

  clearSelection(): void {
    this.selectedItems.clear();
    document.querySelectorAll<HTMLInputElement>('.batch-checkbox').forEach((checkbox) => {
      checkbox.checked = false;
    });
    this.updateBatchUI();
  }

  private updateBatchUI(): void {
    const count = this.selectedItems.size;
    const countElement = document.querySelector('.batch-count');
    if (countElement) {
      countElement.textContent = `已选择 ${count} 项`;
    }
    document.querySelectorAll<HTMLButtonElement>('.batch-actions .btn').forEach((btn) => {
      btn.disabled = count === 0;
    });
  }

  async performBatchAction(action: CredentialBatchAction | 'export'): Promise<void> {
    const selectedItems = Array.from(this.selectedItems);
    if (selectedItems.length === 0) {
      if (window.ui?.showNotification) {
        window.ui.showNotification('请先选择要操作的项目', 'warning');
      } else {
        window.alert('请先选择要操作的项目');
      }
      return;
    }

    const actionName = this.getBatchActionName(action);
    const confirmed = await this.confirmBatchAction(action, actionName, selectedItems.length);
    if (!confirmed) return;

    this.showBatchProgress();
    try {
      await this.executeBatchAction(action, selectedItems);
      if (window.ui?.showNotification) {
        window.ui.showNotification(`批量${actionName}完成`, 'success');
      } else {
        window.alert(`批量${actionName}完成`);
      }
      this.clearSelection();
      await this.refreshCredentials();
    } catch (error) {
      console.error('Batch operation failed:', error);
      if (window.ui?.showNotification) {
        window.ui.showNotification(`批量${actionName}失败: ${(error as Error).message}`, 'error');
      } else {
        window.alert(`批量${actionName}失败: ${(error as Error).message}`);
      }
    } finally {
      this.hideBatchProgress();
    }
  }

  private async confirmBatchAction(action: string, actionName: string, count: number): Promise<boolean> {
    if (window.ui?.showConfirmation) {
      return window.ui.showConfirmation({
        title: `批量${actionName}`,
        message: `确定要${actionName} ${count} 个凭证吗？此操作无法撤销。`,
        type: action === 'delete' ? 'danger' : 'warning',
        confirmText: `${actionName} ${count} 个凭证`,
        confirmClass: action === 'delete' ? 'btn-danger' : 'btn-warning'
      });
    }
    return window.confirm(`确定要${actionName} ${count} 个凭证吗？`);
  }

  private async executeBatchAction(action: CredentialBatchAction | 'export', items: string[]): Promise<void> {
    const total = items.length;
    let completed = 0;
    for (const itemId of items) {
      try {
        switch (action) {
          case 'enable':
            await this.enableCredential(itemId);
            break;
          case 'disable':
            await this.disableCredential(itemId);
            break;
          case 'delete':
            await credsService.delete(itemId);
            break;
          case 'health-check':
            await credsService.probeFlash();
            break;
          case 'export':
            console.warn('导出操作尚未实现。');
            break;
          default:
            throw new Error(`Unknown action: ${action}`);
        }
        completed++;
        this.updateBatchProgress(completed, total);
      } catch (error) {
        console.error(`Failed to ${action} item ${itemId}:`, error);
      }
    }
    if (completed < total) {
      throw new Error(`${total - completed} 个项目操作失败`);
    }
  }

  private getBatchActionName(action: string): string {
    const names: Record<string, string> = {
      enable: '启用',
      disable: '禁用',
      delete: '删除',
      'health-check': '健康检查',
      export: '导出'
    };
    return names[action] ?? action;
  }

  private showBatchProgress(): void {
    const progressElement = document.querySelector<HTMLElement>('.batch-progress');
    if (progressElement) progressElement.style.display = 'block';
  }

  private hideBatchProgress(): void {
    const progressElement = document.querySelector<HTMLElement>('.batch-progress');
    if (progressElement) progressElement.style.display = 'none';
  }

  private updateBatchProgress(completed: number, total: number): void {
    const progressFill = document.querySelector<HTMLElement>('.progress-fill');
    const progressText = document.querySelector<HTMLElement>('.progress-text');
    const percentage = total === 0 ? 0 : (completed / total) * 100;
    if (progressFill) {
      progressFill.style.width = `${percentage}%`;
    }
    if (progressText) {
      progressText.textContent = `处理中... ${completed}/${total}`;
    }
  }
}

export const credsManager = new CredentialsManager();
window.credsManager = credsManager;
