/**
 * 凭证详情渲染模块
 * 提供详情视图相关的渲染函数
 */

import type { Credential } from './types.js';

export interface DetailRenderOptions {
    healthScore?: number;
    healthLevel?: string;
    healthColor?: string;
    showActions?: boolean;
}

/**
 * 渲染凭证详情页面
 */
export function renderCredentialDetail(
    cred: Credential,
    options: DetailRenderOptions = {}
): string {
    const {
        healthScore = 0,
        healthLevel = 'unknown',
        healthColor = '#999',
        showActions = true,
    } = options;

    const isAutoBanned = Boolean(cred.auto_banned);
    const isDisabled = Boolean(cred.disabled);
    const statusClass = isAutoBanned ? 'banned' : isDisabled ? 'disabled' : 'active';
    const statusText = isAutoBanned ? '已封禁' : isDisabled ? '已禁用' : '正常';

    return `
        <div class="credential-detail">
            <div class="detail-header">
                <h2>凭证详情</h2>
                <button class="btn btn-secondary" onclick="credsManager.closeDetail()">
                    返回列表
                </button>
            </div>

            <div class="detail-body">
                ${renderBasicInfo(cred, statusClass, statusText)}
                ${renderHealthInfo(cred, healthScore, healthLevel, healthColor)}
                ${renderQuotaInfo(cred)}
                ${renderUsageStats(cred)}
                ${renderTimestamps(cred)}
                ${isAutoBanned ? renderBanInfo(cred) : ''}
            </div>

            ${showActions ? renderDetailActions(cred, isDisabled, isAutoBanned) : ''}
        </div>
    `;
}

/**
 * 渲染基本信息
 */
function renderBasicInfo(cred: Credential, statusClass: string, statusText: string): string {
    return `
        <div class="detail-section">
            <h3>基本信息</h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">邮箱:</span>
                    <span class="detail-value">${cred.email || '-'}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">项目ID:</span>
                    <span class="detail-value">${cred.project_id || '-'}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">文件名:</span>
                    <span class="detail-value">${cred.filename || '-'}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">状态:</span>
                    <span class="detail-value">
                        <span class="status-badge status-${statusClass}">${statusText}</span>
                    </span>
                </div>
            </div>
        </div>
    `;
}

/**
 * 渲染健康信息
 */
function renderHealthInfo(
    cred: Credential,
    healthScore: number,
    healthLevel: string,
    healthColor: string
): string {
    const successCount = Number(cred.success_count) || 0;
    const failureCount = Number(cred.failure_count) || 0;

    return `
        <div class="detail-section">
            <h3>健康状态</h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">健康评分:</span>
                    <span class="detail-value" style="color: ${healthColor}">
                        ${Math.round(healthScore * 100)}% (${healthLevel})
                    </span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">成功次数:</span>
                    <span class="detail-value">${successCount}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">失败次数:</span>
                    <span class="detail-value">${failureCount}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">成功率:</span>
                    <span class="detail-value">
                        ${calculateSuccessRate(successCount, failureCount)}%
                    </span>
                </div>
            </div>
        </div>
    `;
}

/**
 * 渲染配额信息
 */
function renderQuotaInfo(cred: Credential): string {
    const quotaUsed = Number(cred.quota_used) || 0;
    const quotaLimit = Number(cred.quota_limit) || 0;
    const quotaPercent = quotaLimit > 0 ? Math.round((quotaUsed / quotaLimit) * 100) : 0;

    return `
        <div class="detail-section">
            <h3>配额信息</h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">已用配额:</span>
                    <span class="detail-value">${quotaUsed}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">配额限制:</span>
                    <span class="detail-value">${quotaLimit || '无限制'}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">使用率:</span>
                    <span class="detail-value">${quotaPercent}%</span>
                </div>
                <div class="detail-item full-width">
                    <div class="quota-bar">
                        <div class="quota-fill" style="width: ${quotaPercent}%"></div>
                    </div>
                </div>
            </div>
        </div>
    `;
}

/**
 * 渲染使用统计
 */
function renderUsageStats(cred: Credential): string {
    const successCount = Number(cred.success_count) || 0;
    const failureCount = Number(cred.failure_count) || 0;

    return `
        <div class="detail-section">
            <h3>使用统计</h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">总请求数:</span>
                    <span class="detail-value">${successCount + failureCount}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">平均响应时间:</span>
                    <span class="detail-value">${cred.avg_response_time || '-'} ms</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">最后使用:</span>
                    <span class="detail-value">${formatTimestamp(cred.last_used_at)}</span>
                </div>
            </div>
        </div>
    `;
}

/**
 * 渲染时间戳信息
 */
function renderTimestamps(cred: Credential): string {
    return `
        <div class="detail-section">
            <h3>时间信息</h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">创建时间:</span>
                    <span class="detail-value">${formatTimestamp(cred.created_at)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">更新时间:</span>
                    <span class="detail-value">${formatTimestamp(cred.updated_at)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">最后成功:</span>
                    <span class="detail-value">${formatTimestamp(cred.last_success_ts || cred.last_success)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">最后失败:</span>
                    <span class="detail-value">${formatTimestamp(cred.last_failure_ts || cred.last_failure)}</span>
                </div>
            </div>
        </div>
    `;
}

/**
 * 渲染封禁信息
 */
function renderBanInfo(cred: Credential): string {
    return `
        <div class="detail-section detail-section-warning">
            <h3>⚠️ 封禁信息</h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">封禁原因:</span>
                    <span class="detail-value">${cred.banned_reason || '未知'}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">封禁时间:</span>
                    <span class="detail-value">${formatTimestamp(cred.banned_at)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">封禁次数:</span>
                    <span class="detail-value">${cred.ban_count || 0}</span>
                </div>
            </div>
        </div>
    `;
}

/**
 * 渲染操作按钮
 */
function renderDetailActions(cred: Credential, isDisabled: boolean, isAutoBanned: boolean): string {
    const credKey = cred.filename || cred.id || cred.email || cred.project_id || '';

    return `
        <div class="detail-actions">
            ${!isDisabled ? `
                <button class="btn btn-warning" onclick="credsManager.disableCredential('${credKey}')">
                    禁用凭证
                </button>
            ` : `
                <button class="btn btn-success" onclick="credsManager.enableCredential('${credKey}')">
                    启用凭证
                </button>
            `}
            ${isAutoBanned ? `
                <button class="btn btn-info" onclick="credsManager.unbanCredential('${credKey}')">
                    解除封禁
                </button>
            ` : ''}
            <button class="btn btn-primary" onclick="credsManager.refreshCredential('${credKey}')">
                刷新凭证
            </button>
            <button class="btn btn-secondary" onclick="credsManager.testCredential('${credKey}')">
                测试连接
            </button>
            <button class="btn btn-danger" onclick="credsManager.deleteCredential('${credKey}')">
                删除凭证
            </button>
        </div>
    `;
}

/**
 * 计算成功率
 */
function calculateSuccessRate(successCount: number, failureCount: number): number {
    const total = successCount + failureCount;
    if (total === 0) return 0;
    return Math.round((successCount / total) * 100);
}

/**
 * 格式化时间戳
 */
function formatTimestamp(timestamp: any): string {
    if (!timestamp) return '-';
    
    try {
        const date = new Date(timestamp);
        if (isNaN(date.getTime())) return '-';
        
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
        });
    } catch {
        return '-';
    }
}

