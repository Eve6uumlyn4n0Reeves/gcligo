import { modulePath } from '../core/module_paths';
import { ui } from '../ui';

const t = (key: string, fallback: string) =>
  (ui && typeof ui.t === 'function' && ui.t(key)) || fallback;

export const logsSkeleton = `
  <div class="card" style="min-height:320px;">
    <h3 style="margin-bottom:12px;">${t('logs_loading_title', '日志面板加载中')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${t('status_loading_page', '正在加载页面...')}</p>
    <div class="skeleton-grid" style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;">
      ${(ui.renderSkeleton ? ui.renderSkeleton(4) : '<div class="skeleton-box"></div>'.repeat(4))}
    </div>
  </div>
`;

const logsError = (message: string) => `
  <div class="card" style="padding:32px;text-align:center;">
    <div style="font-size:44px;margin-bottom:12px;">⚠️</div>
    <h3 style="margin-bottom:8px;">${t('logs_load_failed', '日志面板加载失败')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${message || t('status_retry_hint', '请稍后重试')}</p>
    <button class="btn btn-primary" onclick="window.admin && window.admin.switchTab('logs')">
      ${t('action_retry', '重试')}
    </button>
    <button class="btn btn-secondary" style="margin-left:8px;" onclick="window.admin && window.admin.switchTab('dashboard')">
      ${t('action_back_dashboard', '返回仪表盘')}
    </button>
  </div>
`;

type LogsModule = {
  renderLogsPage: () => string;
  initialize?: () => void;
  refresh?: () => Promise<void>;
  exportLogs?: () => void;
};

export async function loadLogsTab(): Promise<LogsModule> {
  try {
    const mod = await import(modulePath('logs', '/js/logs.js'));
    const manager = mod.logsManager ?? mod.default ?? mod;
    if (manager && typeof manager.initialize === 'function') {
      manager.initialize();
    }
    return manager as LogsModule;
  } catch (error) {
    console.error('[admin] failed to load logs module', error);
    const message =
      error instanceof Error ? error.message : t('status_retry_hint', '请稍后重试');
    return {
      renderLogsPage: () => logsError(message),
      exportLogs: () => {},
    };
  }
}
