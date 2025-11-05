import { modulePath } from '../core/module_paths';
import { ui } from '../ui';

const t = (key: string, fallback: string) =>
  (ui && typeof ui.t === 'function' && ui.t(key)) || fallback;

export const registrySkeleton = `
  <div class="card" style="min-height:320px;">
    <h3 style="margin-bottom:12px;">${t('registry_loading_title', '模型注册中心加载中')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${t('status_loading_page', '正在加载页面...')}</p>
    <div class="skeleton-grid" style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;">
      ${(ui.renderSkeleton ? ui.renderSkeleton(6) : '<div class="skeleton-box"></div>'.repeat(6))}
    </div>
  </div>
`;

const registryError = (message: string) => `
  <div class="card" style="padding:32px;text-align:center;">
    <div style="font-size:44px;margin-bottom:12px;">⚠️</div>
    <h3 style="margin-bottom:8px;">${t('registry_load_failed', '模型注册中心加载失败')}</h3>
    <p class="text-muted" style="margin-bottom:20px;">${message || t('status_retry_hint', '请稍后重试')}</p>
    <button class="btn btn-primary" onclick="window.admin && window.admin.switchTab('models')">
      ${t('action_retry', '重试')}
    </button>
    <button class="btn btn-secondary" style="margin-left:8px;" onclick="window.admin && window.admin.switchTab('dashboard')">
      ${t('action_back_dashboard', '返回仪表盘')}
    </button>
  </div>
`;

type RegistryModule = {
  renderRegistryPage: () => string;
  refreshGroups?: () => Promise<void>;
  refreshRegistry?: () => Promise<void>;
  applyDescriptorToForm?: (baseID: string) => void;
  ensureBaseOptions?: () => Promise<void>;
  loadTemplate?: () => Promise<void>;
  describeOptions?: (model: any) => any;
  computeDisplayId?: (model: any) => string;
  models?: any[];
};

export async function loadRegistryTab(): Promise<RegistryModule> {
  try {
    const mod = await import(modulePath('registry', '/js/registry.js'));
    const manager = mod.registryManager ?? mod.default ?? mod;
    if (manager && typeof manager.ensureBaseOptions === 'function') {
      await manager.ensureBaseOptions();
    }
    if (manager && typeof manager.loadTemplate === 'function') {
      await manager.loadTemplate();
    }
    return manager as RegistryModule;
  } catch (error) {
    console.error('[admin] failed to load registry module', error);
    const message =
      error instanceof Error ? error.message : t('status_retry_hint', '请稍后重试');
    return {
      renderRegistryPage: () => registryError(message),
      refreshGroups: async () => {},
      refreshRegistry: async () => {},
      applyDescriptorToForm: () => {},
      ensureBaseOptions: async () => {},
      loadTemplate: async () => {},
      describeOptions: () => [],
      computeDisplayId: () => '',
      models: [],
    };
  }
}
