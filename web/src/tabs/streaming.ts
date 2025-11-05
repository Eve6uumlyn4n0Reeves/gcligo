import { modulePath } from '../core/module_paths';
import { ui } from '../ui';

const t = (key: string, fallback: string) =>
  (ui && typeof ui.t === 'function' && ui.t(key)) || fallback;

export const streamingSkeleton = `
  <div class="card" style="min-height:280px;">
    <h3 style="margin-bottom:12px;">${t('streaming_loading_title', '流式监控加载中')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${t('status_loading_page', '正在加载页面...')}</p>
    <div class="skeleton-grid" style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;">
      ${(ui.renderSkeleton ? ui.renderSkeleton(4) : '<div class="skeleton-box"></div>'.repeat(4))}
    </div>
  </div>
`;

const streamingError = (message: string) => `
  <div class="card" style="padding:32px;text-align:center;">
    <div style="font-size:44px;margin-bottom:12px;">⚠️</div>
    <h3 style="margin-bottom:8px;">${t('streaming_load_failed', '流式监控加载失败')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${message || t('status_retry_hint', '请稍后重试')}</p>
    <button class="btn btn-primary" onclick="window.admin && window.admin.switchTab('streaming')">
      ${t('action_retry', '重试')}
    </button>
    <button class="btn btn-secondary" style="margin-left:8px;" onclick="window.admin && window.admin.switchTab('dashboard')">
      ${t('action_back_dashboard', '返回仪表盘')}
    </button>
  </div>
`;

type StreamingModule = {
  renderPage: () => string | Promise<string>;
  refresh?: () => Promise<void>;
  startAutoRefresh?: (interval: number) => void;
  stopAutoRefresh?: () => void;
};

export async function loadStreamingTab(): Promise<StreamingModule> {
  try {
    const mod = await import(modulePath('streaming', '/js/streaming.js'));
    return (mod.streamingManager ?? mod.default ?? mod) as StreamingModule;
  } catch (error) {
    console.error('[admin] failed to load streaming module', error);
    const message =
      error instanceof Error ? error.message : t('status_retry_hint', '请稍后重试');
    return {
      renderPage: () => streamingError(message),
      refresh: async () => {},
      stopAutoRefresh: () => {},
    };
  }
}
