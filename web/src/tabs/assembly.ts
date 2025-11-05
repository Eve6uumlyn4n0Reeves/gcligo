import { modulePath } from '../core/module_paths';
import { ui } from '../ui';

const t = (key: string, fallback: string) =>
  (ui && typeof ui.t === 'function' && ui.t(key)) || fallback;

export const assemblySkeleton = `
  <div class="card" style="min-height:340px;">
    <h3 style="margin-bottom:12px;">${t('assembly_loading_title', '装配台加载中')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${t('status_loading_page', '正在加载页面...')}</p>
    <div class="skeleton-grid" style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;">
      ${(ui.renderSkeleton ? ui.renderSkeleton(6) : '<div class="skeleton-box"></div>'.repeat(6))}
    </div>
  </div>
`;

const assemblyError = (message: string) => `
  <div class="card" style="padding:32px;text-align:center;">
    <div style="font-size:48px;margin-bottom:12px;">⚠️</div>
    <h3 style="margin-bottom:8px;">${t('assembly_load_failed', '装配台加载失败')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${message || t('status_retry_hint', '请稍后重试')}</p>
    <button class="btn btn-primary" onclick="window.admin && window.admin.switchTab('assembly')">
      ${t('action_retry', '重试')}
    </button>
    <button class="btn btn-secondary" style="margin-left:8px;" onclick="window.admin && window.admin.switchTab('dashboard')">
      ${t('action_back_dashboard', '返回仪表盘')}
    </button>
  </div>
`;

type AssemblyModule = {
  renderPage: () => string | Promise<string>;
  update?: () => Promise<void>;
  startAutoRefresh?: (interval: number) => void;
  stopAutoRefresh?: () => void;
};

export async function loadAssemblyTab(): Promise<AssemblyModule> {
  try {
    const mod = await import(modulePath('assemblyPage', '/js/pages/assembly_page.js'));
    return (mod.default ?? mod) as AssemblyModule;
  } catch (error) {
    console.error('[admin] failed to load assembly module', error);
    const message =
      error instanceof Error ? error.message : t('status_retry_hint', '请稍后重试');
    return {
      renderPage: () => assemblyError(message),
      update: async () => {},
      stopAutoRefresh: () => {},
    };
  }
}
