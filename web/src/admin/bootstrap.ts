import { modulePath } from '../core/module_paths';
import { AdminApp, setAdminDependencies } from './app';

export interface AdminBootstrapContext {
  basePath: string;
  assetVersion: string;
  metaPayload: any;
  metaError: string;
  assetMismatch: { expected: string; server: string } | null;
}

export async function bootstrapAdmin(): Promise<void> {
  const [
    authMod,
    apiMod,
    uiMod,
    oauthMod,
    configMod,
    dashboardMod,
    quickSwitcherMod,
    upstreamMod,
    layoutMod,
    keyboardMod,
    a11yMod
  ] = await Promise.all([
    import(modulePath('auth', '/dist/auth.js')),
    import(modulePath('api', '/dist/api.js')),
    import(modulePath('ui', '/dist/ui.js')),
    import(modulePath('oauth', '/dist/oauth.js')),
    import(modulePath('config', '/dist/config.js')),
    import(modulePath('dashboard', '/dist/dashboard.js')),
    import(modulePath('quickswitcher', '/dist/quick_switcher.js')),
    import(modulePath('upstream', '/dist/upstream.js')),
    import(modulePath('layout', '/dist/layout.js')),
    import(modulePath('keyboard', '/dist/utils/keyboard.js')),
    import(modulePath('a11y', '/dist/utils/a11y.js'))
  ]);

  setAdminDependencies({
    auth: authMod.auth,
    api: apiMod.api,
    ui: uiMod.ui,
    oauthManager: oauthMod.oauthManager,
    configManager: configMod.configManager,
    dashboard: dashboardMod.dashboard,
    renderQuickSwitcher: quickSwitcherMod.renderQuickSwitcher,
    upstream: upstreamMod.upstream,
    layoutInitHashRouter: layoutMod.initHashRouter,
    layoutSetHashForTab: layoutMod.setHashForTab,
    layoutBindSidebar: layoutMod.bindSidebar,
    layoutToggleSidebar: layoutMod.toggleSidebar,
    layoutIsMobile: layoutMod.isMobile,
    isFormInput: keyboardMod.isFormInput,
    addSkipLinks: a11yMod.addSkipLinks,
    announce: a11yMod.announce,
    manageFocus: a11yMod.manageFocus,
    enhanceButton: a11yMod.enhanceButton,
  });

  const admin = new AdminApp();
  (window as any).admin = admin;

  const start = () => {
    admin.initialize().catch((err) => {
      console.error('Admin initialize failed:', err);
    });
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', start, { once: true });
  } else {
    start();
  }

  window.addEventListener('beforeunload', () => {
    admin.destroy();
  });
}

bootstrapAdmin().catch((err) => {
  console.error('Admin bootstrap failed:', err);
});
