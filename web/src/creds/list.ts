/**
 * 凭证列表渲染模块
 * 提供列表视图相关的渲染函数
 */

import type { Credential } from './types.js';

export interface ListRenderOptions {
    page: number;
    pages: number;
    pageSize: number;
    total: number;
    virtualMode?: boolean;
}

/**
 * 渲染凭证卡片
 */
export function renderCredentialCard(
    cred: Credential,
    options: {
        healthScore?: number;
        healthLevel?: string;
        healthColor?: string;
        onAction?: (action: string, credId: string) => void;
    } = {}
): string {
    const {
        healthScore = 0,
        healthLevel = 'unknown',
        healthColor = '#999',
    } = options;

    const credKey = cred.filename || cred.id || cred.email || cred.project_id || '';
    const isAutoBanned = Boolean(cred.auto_banned);
    const isDisabled = Boolean(cred.disabled);
    const statusClass = isAutoBanned ? 'banned' : isDisabled ? 'disabled' : 'active';
    const statusText = isAutoBanned ? '已封禁' : isDisabled ? '已禁用' : '正常';

    return `
        <div class="credential-card ${statusClass}" data-filename="${credKey}" data-cred-id="${credKey}">
            <div class="credential-header">
                <div class="credential-title">
                    <span class="credential-icon">🔑</span>
                    <span class="credential-name">${cred.email || cred.project_id || credKey}</span>
                </div>
                <div class="credential-status">
                    <span class="status-badge status-${statusClass}">${statusText}</span>
                </div>
            </div>
            <div class="credential-body">
                <div class="credential-info">
                    <div class="info-item">
                        <span class="info-label">项目ID:</span>
                        <span class="info-value">${cred.project_id || '-'}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">健康度:</span>
                        <span class="info-value" style="color: ${healthColor}">
                            ${Math.round(healthScore * 100)}% (${healthLevel})
                        </span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">配额:</span>
                        <span class="info-value">${cred.quota_used || 0} / ${cred.quota_limit || '∞'}</span>
                    </div>
                </div>
            </div>
            <div class="credential-footer">
                <button class="btn btn-sm btn-primary" onclick="credsManager.viewCredentialDetail('${credKey}')">
                    查看详情
                </button>
                ${!isDisabled ? `
                    <button class="btn btn-sm btn-warning" onclick="credsManager.disableCredential('${credKey}')">
                        禁用
                    </button>
                ` : `
                    <button class="btn btn-sm btn-success" onclick="credsManager.enableCredential('${credKey}')">
                        启用
                    </button>
                `}
                <button class="btn btn-sm btn-danger" onclick="credsManager.deleteCredential('${credKey}')">
                    删除
                </button>
            </div>
        </div>
    `;
}

/**
 * 渲染凭证列表
 */
export function renderCredentialsList(
    credentials: Credential[],
    options: ListRenderOptions
): string {
    if (credentials.length === 0) {
        return renderEmptyState();
    }

    const cards = credentials.map(cred => renderCredentialCard(cred)).join('');
    const pager = renderPager(options);

    return `
        <div class="credentials-list">
            <div class="credentials-grid">
                ${cards}
            </div>
            ${pager}
        </div>
    `;
}

/**
 * 渲染空状态
 */
export function renderEmptyState(): string {
    return `
        <div class="empty">
            <div class="empty-icon">🔐</div>
            <div class="empty-title">暂无凭证</div>
            <div class="empty-hint">建议先通过 OAuth 或上传 JSON 添加凭证</div>
            <div style="margin-top:8px;display:flex;gap:8px;justify-content:center;">
                <button class="btn btn-primary" onclick="window.admin.switchTab('oauth')">
                    ➕ 添加凭证
                </button>
            </div>
        </div>
    `;
}

/**
 * 渲染分页器
 */
export function renderPager(options: ListRenderOptions): string {
    const { page, pages, pageSize, total } = options;

    if (pages <= 1) {
        return '';
    }

    const prevDisabled = page <= 1;
    const nextDisabled = page >= pages;

    return `
        <div class="pager">
            <div class="pager-info">
                显示 ${(page - 1) * pageSize + 1}-${Math.min(page * pageSize, total)} / 共 ${total} 项
            </div>
            <div class="pager-controls">
                <button 
                    class="btn btn-sm btn-secondary" 
                    ${prevDisabled ? 'disabled' : ''}
                    onclick="credsManager.goToPage(${page - 1})"
                >
                    上一页
                </button>
                <span class="pager-current">第 ${page} / ${pages} 页</span>
                <button 
                    class="btn btn-sm btn-secondary" 
                    ${nextDisabled ? 'disabled' : ''}
                    onclick="credsManager.goToPage(${page + 1})"
                >
                    下一页
                </button>
            </div>
        </div>
    `;
}

/**
 * 渲染凭证表格（备用视图）
 */
export function renderCredentialsTable(credentials: Credential[]): string {
    if (credentials.length === 0) {
        return renderEmptyState();
    }

    const rows = credentials.map(cred => renderCredentialRow(cred)).join('');

    return `
        <div class="credentials-table-wrapper">
            <table class="credentials-table">
                <thead>
                    <tr>
                        <th>邮箱/项目</th>
                        <th>项目ID</th>
                        <th>状态</th>
                        <th>健康度</th>
                        <th>配额</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody>
                    ${rows}
                </tbody>
            </table>
        </div>
    `;
}

/**
 * 渲染凭证表格行
 */
export function renderCredentialRow(cred: Credential): string {
    const credKey = cred.filename || cred.id || cred.email || cred.project_id || '';
    const isAutoBanned = Boolean(cred.auto_banned);
    const isDisabled = Boolean(cred.disabled);
    const statusClass = isAutoBanned ? 'banned' : isDisabled ? 'disabled' : 'active';
    const statusText = isAutoBanned ? '已封禁' : isDisabled ? '已禁用' : '正常';

    return `
        <tr class="credential-row ${statusClass}" data-filename="${credKey}">
            <td>${cred.email || '-'}</td>
            <td>${cred.project_id || '-'}</td>
            <td><span class="status-badge status-${statusClass}">${statusText}</span></td>
            <td>${Math.round((cred.health_score || 0) * 100)}%</td>
            <td>${cred.quota_used || 0} / ${cred.quota_limit || '∞'}</td>
            <td>
                <button class="btn btn-sm btn-link" onclick="credsManager.viewCredentialDetail('${credKey}')">
                    详情
                </button>
            </td>
        </tr>
    `;
}

/**
 * 渲染筛选器
 */
export function renderFilters(options: {
    projects: string[];
    currentFilters: {
        search: string;
        status: string;
        health: string;
        project: string;
    };
}): string {
    const { projects, currentFilters } = options;

    return `
        <div class="filters">
            <div class="filter-group">
                <input 
                    type="text" 
                    class="form-control" 
                    placeholder="搜索凭证..." 
                    value="${currentFilters.search}"
                    oninput="credsManager.updateFilter('search', this.value)"
                />
            </div>
            <div class="filter-group">
                <select 
                    class="form-control" 
                    onchange="credsManager.updateFilter('status', this.value)"
                >
                    <option value="all" ${currentFilters.status === 'all' ? 'selected' : ''}>全部状态</option>
                    <option value="active" ${currentFilters.status === 'active' ? 'selected' : ''}>正常</option>
                    <option value="disabled" ${currentFilters.status === 'disabled' ? 'selected' : ''}>已禁用</option>
                    <option value="banned" ${currentFilters.status === 'banned' ? 'selected' : ''}>已封禁</option>
                </select>
            </div>
            <div class="filter-group">
                <select 
                    class="form-control" 
                    onchange="credsManager.updateFilter('health', this.value)"
                >
                    <option value="all" ${currentFilters.health === 'all' ? 'selected' : ''}>全部健康度</option>
                    <option value="excellent" ${currentFilters.health === 'excellent' ? 'selected' : ''}>优秀 (≥90%)</option>
                    <option value="good" ${currentFilters.health === 'good' ? 'selected' : ''}>良好 (≥70%)</option>
                    <option value="fair" ${currentFilters.health === 'fair' ? 'selected' : ''}>一般 (≥50%)</option>
                    <option value="poor" ${currentFilters.health === 'poor' ? 'selected' : ''}>较差 (<50%)</option>
                </select>
            </div>
            <div class="filter-group">
                <select 
                    class="form-control" 
                    onchange="credsManager.updateFilter('project', this.value)"
                >
                    <option value="all" ${currentFilters.project === 'all' ? 'selected' : ''}>全部项目</option>
                    ${projects.map(project => `
                        <option value="${project}" ${currentFilters.project === project ? 'selected' : ''}>
                            ${project}
                        </option>
                    `).join('')}
                </select>
            </div>
        </div>
    `;
}

