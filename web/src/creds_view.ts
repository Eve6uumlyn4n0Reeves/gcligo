import type { Credential } from './creds/types.js';
import type { CredentialsManager } from './creds.js';

export type CredentialHelpers = Pick<
  CredentialsManager,
  'formatTimestamp' | 'normalizeTimestamp' | 'renderHealthBar'
>;

const ERROR_CODE_DESCRIPTIONS: Record<string, string> = {
  '429': '请求过多',
  '403': '权限不足',
  '401': '认证失败',
  '500': '服务器错误',
  '502': '网关错误',
  '503': '服务不可用'
};

function escapeAttr(value: string): string {
  return value.replace(/"/g, '&quot;');
}

function friendlyErrorCodes(codes: Array<string | number>): string {
  return codes
    .map((code) => {
      const key = String(code);
      const desc = ERROR_CODE_DESCRIPTIONS[key];
      return desc ? `${desc}(${key})` : key;
    })
    .join('、');
}

export function renderActionButtons(cred: Credential): string {
  const id = cred.filename ?? cred.id ?? '';
  if (cred.auto_banned) {
    return `
      <button class="btn btn-success btn-sm" aria-label="恢复凭证 ${id}" onclick="window.credsManager.recoverCredential('${id}')">恢复</button>
      <button class="btn btn-danger btn-sm" aria-label="删除凭证 ${id}" onclick="window.credsManager.deleteCredential('${id}')">删除</button>`;
  }
  if (cred.disabled) {
    return `
      <button class="btn btn-success btn-sm" aria-label="启用凭证 ${id}" onclick="window.credsManager.enableCredential('${id}')">启用</button>
      <button class="btn btn-danger btn-sm" aria-label="删除凭证 ${id}" onclick="window.credsManager.deleteCredential('${id}')">删除</button>`;
  }
  return `
    <button class="btn btn-warning btn-sm" aria-label="禁用凭证 ${id}" onclick="window.credsManager.disableCredential('${id}')">禁用</button>
    <button class="btn btn-danger btn-sm" aria-label="删除凭证 ${id}" onclick="window.credsManager.deleteCredential('${id}')">删除</button>`;
}

export function renderCredentialCard(cred: Credential, helpers: CredentialHelpers): string {
  const lastSuccess = helpers.formatTimestamp(
    helpers.normalizeTimestamp(cred.last_success_ts ?? cred.last_success)
  );
  const lastFailure = helpers.formatTimestamp(
    helpers.normalizeTimestamp(cred.last_failure_ts ?? cred.last_failure),
    '无记录'
  );
  const credKey = cred.filename ?? cred.id ?? cred.email ?? cred.project_id ?? '';
  const safeKey = escapeAttr(String(credKey));

  const statusClass = cred.auto_banned || cred.disabled ? 'status-disabled' : 'status-active';
  const statusLabel = cred.auto_banned
    ? `已封禁: ${cred.banned_reason || '未知原因'}`
    : cred.disabled
    ? '已禁用'
    : '活跃';

  const errorSection =
    cred.error_codes && cred.error_codes.length > 0
      ? `<div class="credential-info" style="color:#ef4444;">最近错误: ${friendlyErrorCodes(cred.error_codes)}</div>`
      : '';

  return `
    <div class="credential-card ${cred.disabled ? 'disabled' : ''}" data-cred-id="${safeKey}">
      <div class="credential-header">
        <div class="credential-name">${cred.filename ?? safeKey}</div>
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
      <div class="credential-info">最后成功: ${lastSuccess}</div>
      <div class="credential-info">最后失败: ${lastFailure}</div>
      ${errorSection}
      <div class="credential-actions">${renderActionButtons(cred)}</div>
    </div>`;
}

export function renderStatusBadge(cred: Credential): string {
  if (cred.auto_banned) {
    return `<span class="status-badge status-disabled">已封禁: ${cred.banned_reason ?? '未知原因'}</span>`;
  }
  if (cred.disabled) {
    return `<span class="status-badge status-disabled">已禁用</span>`;
  }
  return `<span class="status-badge status-active">活跃</span>`;
}

export function renderCredentialRow(cred: Credential, helpers: CredentialsManager): string {
  const healthScore = (cred.health_score ?? 0) * 100;
  const successRate = (cred.success_rate ?? 0) * 100;
  return `
    <tr>
      <td>${cred.id ?? cred.filename ?? ''}</td>
      <td>${cred.project_id ?? '-'}</td>
      <td>${cred.email ?? '-'}</td>
      <td>${renderStatusBadge(cred)}</td>
      <td>${helpers.renderHealthBar(healthScore)}</td>
      <td>${cred.total_requests ?? cred.total_calls ?? 0}</td>
      <td>${successRate.toFixed(1)}%</td>
      <td>${helpers.formatTimestamp(helpers.normalizeTimestamp(cred.last_success_ts ?? cred.last_success))}</td>
      <td>${helpers.formatTimestamp(helpers.normalizeTimestamp(cred.last_failure_ts ?? cred.last_failure), '无记录')}</td>
      <td>${renderActionButtons(cred)}</td>
    </tr>`;
}

export function renderPager(page: number, pages: number, size: number, total: number): string {
  return `
    <div style="display:flex; gap:8px; align-items:center; margin: 8px 0;">
      <button class="btn" aria-label="上一页" onclick="window.credsManager.setPage(${Math.max(1, page - 1)})">上一页</button>
      <span style="color:#666;">第 ${page}/${pages} 页（共 ${total} 条）</span>
      <button class="btn" aria-label="下一页" onclick="window.credsManager.setPage(${Math.min(pages, page + 1)})">下一页</button>
      <label style="margin-left:8px; color:#666;">每页
        <select onchange="window.credsManager.setPageSize(this.value)">
          <option ${size === 10 ? 'selected' : ''}>10</option>
          <option ${size === 20 ? 'selected' : ''}>20</option>
          <option ${size === 50 ? 'selected' : ''}>50</option>
        </select>
      </label>
    </div>`;
}

export function renderCredentialsList(manager: CredentialsManager): string {
  const total = manager.getFilteredCredentials().length;
  const pager = renderPager(manager.page, Math.max(1, Math.ceil(total / (manager.pageSize || 20))), manager.pageSize, total);
  return `${pager}<div class="credentials-grid" aria-live="polite"></div>${pager}`;
}

export function populateCredentialGrid(
  gridEl: Element | null,
  items: Credential[],
  manager: CredentialsManager
): void {
  if (!gridEl) return;
  const arr = Array.isArray(items) ? items.slice() : [];
  gridEl.innerHTML = '';
  if (arr.length === 0) return;
  const chunkSize = 12;
  let index = 0;
  const renderChunk = () => {
    const frag = document.createDocumentFragment();
    for (let i = 0; i < chunkSize && index < arr.length; i += 1, index += 1) {
      const wrapper = document.createElement('div');
      wrapper.innerHTML = renderCredentialCard(arr[index], manager);
      const card = wrapper.firstElementChild;
      if (card) frag.appendChild(card);
    }
    gridEl.appendChild(frag);
    if (index < arr.length) requestAnimationFrame(renderChunk);
  };
  requestAnimationFrame(renderChunk);
}

export function renderCredentialsTable(manager: CredentialsManager): string {
  const credentials: Credential[] = manager.getCredentials?.() ?? [];
  if (credentials.length === 0) return '<p>暂无凭证</p>';
  return `
    <table class="credential-table">
      <thead>
        <tr><th>ID</th><th>项目ID</th><th>邮箱</th><th>状态</th><th>健康评分</th><th>请求数</th><th>成功率</th><th>最后成功</th><th>最后失败</th><th>操作</th></tr>
      </thead>
      <tbody>
        ${credentials.map((cred) => renderCredentialRow(cred, manager)).join('')}
      </tbody>
    </table>`;
}

declare global {
  interface Window {
    credsViewRenderCredentialsList?: (manager: CredentialsManager) => string;
    credsViewPopulateCredentialGrid?: (
      grid: Element | null,
      items: Credential[],
      manager: CredentialsManager
    ) => void;
    credsViewRenderPager?: (page: number, pages: number, size: number, total: number) => string;
    credsViewRenderCredentialsTable?: (manager: CredentialsManager) => string;
    credsViewRenderCredentialRow?: (cred: Credential, manager: CredentialsManager) => string;
  }
}

window.credsViewRenderCredentialsList = renderCredentialsList;
window.credsViewPopulateCredentialGrid = populateCredentialGrid;
window.credsViewRenderPager = renderPager;
window.credsViewRenderCredentialsTable = renderCredentialsTable;
window.credsViewRenderCredentialRow = renderCredentialRow;

export default {
  renderActionButtons,
  renderCredentialCard,
  renderCredentialsList,
  populateCredentialGrid,
  renderCredentialsTable,
  renderCredentialRow,
  renderStatusBadge,
  renderPager
};
