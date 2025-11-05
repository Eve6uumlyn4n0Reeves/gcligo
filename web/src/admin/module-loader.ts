/**
 * 管理模块加载器
 * 处理管理控制台主模块的加载
 */

import { loadResourceWithRetry } from './resource-loader';
import { renderAdminLoadError } from './error-handler';

/**
 * 加载管理模块
 */
export function loadAdminModule(): void {
  const version =
    (typeof window !== 'undefined' && (window as any).__ASSET_VERSION__) || '';
  // 强制加入 t= 时间戳，避免同一版本号下的缓存/会话短时缓存干扰热修复
  const suffix = version
    ? `?v=${encodeURIComponent(version)}&t=${Date.now()}`
    : `?t=${Date.now()}`;
  const base =
    ((window as any).__ADMIN_BOOTSTRAP_CTX__ &&
      (window as any).__ADMIN_BOOTSTRAP_CTX__.basePath) ||
    '';
  const src = (base ? `${base}/admin.js` : `/admin.js`) + suffix;

  loadResourceWithRetry(src, 'script')
    .then(() => {
      // 二次保险：脚本加载成功后若未自动初始化，这里手动触发
      try {
        if (
          (window as any).admin &&
          typeof (window as any).admin.initialize === 'function'
        ) {
          (window as any).admin.initialize();
        }
      } catch (e) {
        // no-op
      }
    })
    .catch((err) => {
      renderAdminLoadError(err, (window as any).__ADMIN_BOOTSTRAP_CTX__ || {});
    });
}

/**
 * 初始化模块加载
 */
export function initializeModuleLoading(): void {
  (window as any).renderAdminLoadError = renderAdminLoadError;
  (window as any).loadAdminModule = loadAdminModule;

  if ((window as any).__ADMIN_DEFER_LOAD__) {
    delete (window as any).__ADMIN_DEFER_LOAD__;
    loadAdminModule();
  }
}

