/**
 * 管理控制台错误处理
 * 处理资源加载失败、版本不匹配等错误
 */

import type { AdminBootstrapContext } from './bootstrap';

/**
 * HTML 转义函数
 */
function escapeHtml(value: string | null | undefined): string {
  return String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

/**
 * 渲染管理控制台加载错误
 */
export function renderAdminLoadError(
  event: Error | Event,
  context?: AdminBootstrapContext
): void {
  try {
    console.error('管理控制台静态资源加载失败', event);
  } catch (_err) {
    // ignore
  }

  const container = document.getElementById('app-container');
  if (!container) return;

  const ctx = context || (window as any).__ADMIN_BOOTSTRAP_CTX__ || {};
  const path = window.location.pathname || '';
  const base =
    ctx.basePath !== undefined
      ? ctx.basePath
      : (window as any).__BASE_PATH__ || '';
  const assetVersionHtml = ((window as any).__ASSET_VERSION__ || '').toString();
  const assetVersionServer =
    ctx.metaPayload && typeof ctx.metaPayload.asset_version === 'string'
      ? ctx.metaPayload.asset_version
      : '';
  const assetMismatch =
    ctx.assetMismatch ||
    (assetVersionHtml &&
      assetVersionServer &&
      assetVersionHtml !== assetVersionServer);
  const metaError = ctx.metaError || '';
  const asset = (base ? base + '/' : '/') + 'admin.js';
  const host = window.location.host;
  const protocol = window.location.protocol;

  const assetMismatchHtml = assetMismatch
    ? `
      <div style="margin: 16px 0; padding: 14px 16px; border-left: 4px solid #f59e0b; background:#fffbeb; border-radius: 8px;">
        <strong>资源版本不一致：</strong>
        <div style="font-size:13px; color:#92400e; margin-top:6px;">
          页面静态资源版本为 <code>${escapeHtml(assetVersionHtml)}</code>，
          但服务器报告为 <code>${escapeHtml(assetVersionServer)}</code>。请尝试强制刷新或清理 CDN 缓存后重试。
        </div>
      </div>
    `
    : '';

  const metaErrorHtml = metaError
    ? `
      <div style="margin: 16px 0; padding: 14px 16px; border-left: 4px solid #2563eb; background:#eff6ff; border-radius: 8px;">
        <strong>元信息获取失败：</strong>
        <div style="font-size:13px; color:#1e3a8a; margin-top:6px;">
          调用 <code>${escapeHtml((base ? base + '/' : '/') + 'meta/base-path')}</code> 失败：${escapeHtml(metaError)}
        </div>
      </div>
    `
    : '';

  container.innerHTML = `
    <div class="card" style="padding:48px 24px; line-height:1.6; border:1px solid #fca5a5; border-radius:12px; background:#fff5f5;">
      <h2 style="color:#dc2626; margin-bottom:12px;">🚨 管理控制台加载失败</h2>
      <p style="color:#444;">未能加载管理资源 <code>${escapeHtml(asset)}</code></p>
      
      <div style="margin: 20px 0; padding: 16px; background: #f8f9fa; border-radius: 8px; border-left: 4px solid #dc2626;">
        <h4 style="color:#dc2626; margin: 0 0 12px 0;">诊断信息</h4>
        <ul style="color:#666; font-size:13px; margin:0; padding-left: 20px;">
          <li>当前访问路径：<code>${escapeHtml(path)}</code></li>
          <li>检测到的Base Path：<code>${escapeHtml(base || '(空)')}</code></li>
          <li>尝试加载的资源：<code>${escapeHtml(asset)}</code></li>
          <li>HTML Asset 版本：<code>${escapeHtml(assetVersionHtml || '(未设置)')}</code></li>
          <li>服务端 Asset 版本：<code>${escapeHtml(assetVersionServer || '(未知)')}</code></li>
          <li>当前主机：<code>${escapeHtml(host)}</code></li>
        </ul>
      </div>
      ${assetMismatchHtml}
      ${metaErrorHtml}

      <div style="margin: 20px 0;">
        <h4 style="color:#374151; margin: 0 0 12px 0;">🔧 解决方案</h4>
        <div style="display: grid; gap: 12px;">
          <div style="padding: 12px; background: #f0f9ff; border-radius: 6px; border-left: 3px solid #0ea5e9;">
            <strong>方案 1: 强制刷新</strong>
            <p style="margin: 6px 0 0 0; font-size: 13px; color: #666;">按 <kbd>Ctrl+Shift+R</kbd> (或 <kbd>Cmd+Shift+R</kbd>) 强制刷新页面</p>
          </div>
          <div style="padding: 12px; background: #f0f9ff; border-radius: 6px; border-left: 3px solid #0ea5e9;">
            <strong>方案 2: 检查部署配置</strong>
            <p style="margin: 6px 0 0 0; font-size: 13px; color: #666;">确保服务器的 <code>BASE_PATH</code> 配置与访问路径一致</p>
          </div>
          <div style="padding: 12px; background: #f0f9ff; border-radius: 6px; border-left: 3px solid #0ea5e9;">
            <strong>方案 3: 直接访问</strong>
            <p style="margin: 6px 0 0 0; font-size: 13px; color: #666;">尝试通过根路径访问：<a href="${protocol}//${host}/admin" style="color: #0ea5e9;">${protocol}//${host}/admin</a></p>
          </div>
        </div>
      </div>

      <div style="display: flex; gap: 12px; justify-content: center; margin-top: 24px;">
        <button onclick="window.location.reload()" style="padding: 10px 20px; background: #0ea5e9; color: white; border: none; border-radius: 6px; cursor: pointer;">
          🔄 重新加载
        </button>
        <button onclick="window.location.href='${protocol}//${host}/admin'" style="padding: 10px 20px; background: #16a34a; color: white; border: none; border-radius: 6px; cursor: pointer;">
          🏠 根路径访问
        </button>
        <button onclick="window.open('${protocol}//${host}/routes', '_blank')" style="padding: 10px 20px; background: #6366f1; color: white; border: none; border-radius: 6px; cursor: pointer;">
          📋 查看路由信息
        </button>
      </div>
    </div>
  `;
}

