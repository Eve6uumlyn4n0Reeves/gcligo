/**
 * GCLI2API-Go 管理控制台主入口脚本
 * 负责页面初始化、路由切换和模块协调
 */

import { modulePath } from '../core/module_paths';
import { ModuleManager } from '../core/module_manager';
import { assemblySkeleton, loadAssemblyTab } from '../tabs/assembly';
import { loadStreamingTab, streamingSkeleton } from '../tabs/streaming';
import { loadLogsTab, logsSkeleton } from '../tabs/logs';
import { loadRegistryTab, registrySkeleton } from '../tabs/registry';
import { bindAdminShortcuts } from './shortcuts';
import { createAutoRefreshManager } from './refresh';

let auth: any;
let api: any;
let ui: any;
let oauthManager: any;
let dashboard: any;
let configManager: any;
let renderQuickSwitcher: (...args: any[]) => any;
let upstream: any;
let layoutInitHashRouter: any;
let layoutSetHashForTab: any;
let layoutBindSidebar: any;
let layoutToggleSidebar: any;
let layoutIsMobile: any;
let isFormInput: (target: any) => boolean;
let addSkipLinks: () => void;
let announce: (...args: any[]) => void;
let manageFocus: (...args: any[]) => void;
let enhanceButton: (...args: any[]) => void;
let credsManager: any;
let logsManager: any;
let registryManager: any;
let metricsManager: any;
let streamingManager: any;
let assemblyPageModule: any;

export type AdminDependencies = {
	auth: any;
	api: any;
	ui: any;
	oauthManager: any;
	configManager: any;
	dashboard: any;
	renderQuickSwitcher: (...args: any[]) => any;
	upstream: any;
	layoutInitHashRouter: any;
	layoutSetHashForTab: any;
	layoutBindSidebar: any;
	layoutToggleSidebar: any;
	layoutIsMobile: any;
	isFormInput: (target: any) => boolean;
	addSkipLinks: () => void;
	announce: (...args: any[]) => void;
	manageFocus: (...args: any[]) => void;
	enhanceButton: (...args: any[]) => void;
};

export function setAdminDependencies(deps: AdminDependencies) {
	auth = deps.auth;
	api = deps.api;
	ui = deps.ui;
	oauthManager = deps.oauthManager;
	configManager = deps.configManager;
	dashboard = deps.dashboard;
	renderQuickSwitcher = deps.renderQuickSwitcher;
	upstream = deps.upstream;
	layoutInitHashRouter = deps.layoutInitHashRouter;
	layoutSetHashForTab = deps.layoutSetHashForTab;
	layoutBindSidebar = deps.layoutBindSidebar;
	layoutToggleSidebar = deps.layoutToggleSidebar;
	layoutIsMobile = deps.layoutIsMobile;
	isFormInput = deps.isFormInput;
	addSkipLinks = deps.addSkipLinks;
	announce = deps.announce;
	manageFocus = deps.manageFocus;
	enhanceButton = deps.enhanceButton;
}

export class AdminApp {
    private currentTab: string;
    private tabs: string[];
    private initialized: boolean;
    private upstreamDetail: any;
	private moduleManager: ModuleManager;
	private modules: Record<string, any>;
	private eventBus: any;
	private _eventsBound?: boolean;
	private autoRefresh: ReturnType<typeof createAutoRefreshManager>;
	private detachShortcuts?: () => void;

    constructor() {
        this.currentTab = 'dashboard';
        // 将装配台集成到管理后台作为一级标签
        this.tabs = ['dashboard','assembly','credentials','oauth','stats','streaming','logs','models','config'];
        this.initialized = false;
        this.upstreamDetail = null;
        
        const coreModules = {
            auth,
            ui,
            dashboard,
            api,
            oauth: oauthManager,
            config: configManager
        };
        this.moduleManager = new ModuleManager(coreModules, this.createModuleLoaders());
        this.modules = this.moduleManager.cache();
        Object.assign(this.modules, {
            credentials: null,
            logs: null,
            metrics: null,
            streaming: null,
            registry: null,
            assembly: null
        });
		this.eventBus = ui.constructor.eventBus;
		this.autoRefresh = createAutoRefreshManager({
			getCurrentTab: () => this.currentTab,
			updateDashboard: () => this.updateDashboard(),
			getModules: () => this.modules,
			getMetricsManager: () => metricsManager
		});
		this.detachShortcuts = undefined;
    }

    async loadModule(moduleName: string): Promise<any> {
        return this.moduleManager.load(moduleName);
    }

    private createModuleLoaders(): Record<string, () => Promise<any>> {
        return {
            metrics: async () => {
                if (!metricsManager) {
                    const mod = await import(modulePath('metrics', '/js/metrics.js'));
                    metricsManager = mod.metricsManager;
                }
                return metricsManager;
            },
            assembly: async () => {
                if (!assemblyPageModule) {
                    assemblyPageModule = await loadAssemblyTab();
                }
                return assemblyPageModule;
            },
            streaming: async () => {
                if (!streamingManager) {
                    streamingManager = await loadStreamingTab();
                }
                return streamingManager;
            },
            credentials: async () => {
                if (!credsManager) {
                    const credsMod = await import(modulePath('creds', '/js/creds.js'));
                    credsManager = credsMod.credsManager;
                }
                const creds = credsManager;
                await creds.refreshCredentials();
                creds.bindDomRefresh();
                return creds;
            },
            logs: async () => {
                if (!logsManager) {
                    logsManager = await loadLogsTab();
                    if (typeof window !== 'undefined') {
                        (window as any).logsManager = logsManager;
                    }
                }
                return logsManager;
            },
            registry: async () => {
                if (!registryManager) {
                    registryManager = await loadRegistryTab();
                }
                return registryManager;
            },
        };
    }
    
    /**
     * 获取标签页对应的模块名
     */
    getTabModule(tabName: string): string | null {
        const tabModuleMap: Record<string, string | null> = {
            'credentials': 'credentials',
            'logs': 'logs',
            'models': 'registry',
            'stats': 'metrics',
            'streaming': 'streaming',
            'assembly': 'assembly',
            // 其他标签页使用预加载的模块，不需要懒加载
            'dashboard': null,
            'oauth': null,
            'config': null
        };

        return tabModuleMap[tabName];
    }

    /**
     * 动态导入模块
     */
    async importModule(path: string): Promise<any> {
        try {
            return await import(path);
        } catch (error) {
            console.error(`Failed to load module ${path}:`, error);
            throw error;
        }
    }

    /**
     * 初始化应用
     */
    async initialize(): Promise<void> {
        try {
            const t = (key: string) => ui.t(key);
            // 确保认证
            const isAuthenticated = await auth.ensureAuthenticated();
            if (!isAuthenticated) {
                // 等待登录事件后再继续初始化（避免卡在加载界面）
                const onLogin = async () => {
                    window.removeEventListener('auth:login', onLogin);
                    try {
                        await this._bootAfterAuth();
                    } catch (err: unknown) {
                        console.error(err);
                        const errorMsg = err instanceof Error ? err.message : String(err);
                        this.showErrorMessage(`${ui.t('error_init_failed')}: ${errorMsg}`);
                    }
                };
                window.addEventListener('auth:login', onLogin, { once: true });
                // 提示用户登录即可继续
                this.showErrorMessage(t('error_auth_required'));
                return;
            }
            await this._bootAfterAuth();
            // 初始网络状态横幅
            if (!navigator.onLine) {
                ui.banner('netBanner', 'warning', t('network_offline_banner'));
            }
            // 浏览器环境，不使用 Node.js 的 process
            console.log('GCLI2API-Go admin console ready');

        } catch (error: unknown) {
            console.error('Admin initialization failed:', error);
            const errorMsg = error instanceof Error ? error.message : String(error);
            this.showErrorMessage(`${ui.t('error_init_failed')}: ${errorMsg}`);
        }
    }

    async _bootAfterAuth(): Promise<void> {
        if (this.initialized) return;
        // 预取上游模型建议，供配置/注册等界面共用
        upstream.fetch().then((detail: any) => { this.upstreamDetail = detail; }).catch(() => { this.upstreamDetail = upstream.getCached(); });
        // 初始化模块
        await this.initializeModules();
        // 事件监听（仅绑定一次）
        if (!this._eventsBound) { this.setupEventListeners(); this._eventsBound = true; }
        // 无障碍
        this.initializeAccessibility();
        // 渲染
        await this.renderLayout();
        // 加载数据
        await this.loadInitialData();
		// 自动刷新
		this.autoRefresh.startDashboardRefresh();
        this.initialized = true;

        // 首次访问引导
        this.showFirstTimeGuide();
    }
    
    /**
     * 首次访问引导
     */
    showFirstTimeGuide() {
        const GUIDE_KEY = 'gcli2api_first_visit_guide_shown';
        const hasShown = localStorage.getItem(GUIDE_KEY);
        
        if (hasShown) return;
        
        // 延迟显示，确保界面已完全加载
        setTimeout(() => {
            const content = `
                <div style="text-align: center; padding: 20px;">
                    <div style="font-size: 48px; margin-bottom: 20px;">👋</div>
                    <h3 style="margin-bottom: 16px;">欢迎使用 GCLI2API-Go 管理控制台</h3>
                    <p style="color: #666; margin-bottom: 24px; line-height: 1.6;">
                        这是您首次使用本系统。以下快捷键可以帮助您更高效地管理服务。
                    </p>
                    
                    <div style="background: #f9fafb; border-radius: 8px; padding: 20px; text-align: left; max-width: 400px; margin: 0 auto 24px;">
                        <h4 style="margin: 0 0 12px 0; color: #374151;">⌨️ 常用快捷键</h4>
                        <div style="display: grid; gap: 8px;">
                            <div style="display: flex; justify-content: space-between; align-items: center;">
                                <span style="color: #6b7280;">快速切换</span>
                                <div style="display: flex; gap: 4px;">
                                    <kbd style="background: white; border: 1px solid #d1d5db; border-radius: 4px; padding: 2px 6px; font-size: 12px;">Ctrl</kbd>
                                    <kbd style="background: white; border: 1px solid #d1d5db; border-radius: 4px; padding: 2px 6px; font-size: 12px;">K</kbd>
                                </div>
                            </div>
                            <div style="display: flex; justify-content: space-between; align-items: center;">
                                <span style="color: #6b7280;">刷新当前页</span>
                                <div style="display: flex; gap: 4px;">
                                    <kbd style="background: white; border: 1px solid #d1d5db; border-radius: 4px; padding: 2px 6px; font-size: 12px;">Ctrl</kbd>
                                    <kbd style="background: white; border: 1px solid #d1d5db; border-radius: 4px; padding: 2px 6px; font-size: 12px;">R</kbd>
                                </div>
                            </div>
                            <div style="display: flex; justify-content: space-between; align-items: center;">
                                <span style="color: #6b7280;">快捷键帮助</span>
                                <div style="display: flex; gap: 4px;">
                                    <kbd style="background: white; border: 1px solid #d1d5db; border-radius: 4px; padding: 2px 6px; font-size: 12px;">Shift</kbd>
                                    <kbd style="background: white; border: 1px solid #d1d5db; border-radius: 4px; padding: 2px 6px; font-size: 12px;">?</kbd>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <p style="color: #9ca3af; font-size: 13px; margin-bottom: 16px;">
                        提示：随时按 <kbd style="background: #e5e7eb; padding: 2px 6px; border-radius: 4px; font-size: 12px;">Shift</kbd> + <kbd style="background: #e5e7eb; padding: 2px 6px; border-radius: 4px; font-size: 12px;">?</kbd> 查看完整快捷键列表
                    </p>
                    
                    <button class="btn btn-primary" onclick="document.getElementById('modal').classList.remove('active')" style="min-width: 120px;">
                        开始使用
                    </button>
                </div>
            `;
            
            ui.showModal('', content);
            
            // 标记已显示
            localStorage.setItem(GUIDE_KEY, 'true');
        }, 1000);
    }

    /**
     * 初始化各个模块
     */
    async initializeModules() {
        // 简化：非懒加载模块已在构造函数中可用；仅注册数据刷新管理器
        try { this.setupRefreshManager(); } catch (e) { console.warn('setupRefreshManager failed', e); }
    }

    /**
     * 设置数据刷新管理器
     */
    setupRefreshManager() {
        const refreshManager = ui.constructor.refreshManager;

        // 注册凭证数据源
        refreshManager.register('credentials', async () => {
            const credsModule = await this.getModule('credentials');
            return credsModule.getCredentials();
        }, {
            throttle: 2000,
            cache: true
        });

        // 注册指标数据源
        refreshManager.register('metrics', async () => {
            const metricsModule = await this.getModule('metrics');
            return metricsModule.getStats();
        }, {
            throttle: 1000,
            cache: true
        });

        // 监听数据更新事件
        this.eventBus.on('data:updated:credentials', (data: any) => {
            if (this.currentTab === 'credentials') {
                this.updateCredentialsView(data);
            }
        });

        this.eventBus.on('data:updated:metrics', (data: any) => {
            if (['dashboard', 'stats'].includes(this.currentTab)) {
                this.updateMetricsView(data);
            }
        });
    }

    /**
     * 更新凭证视图
     */
    updateCredentialsView(data: any): void {
        const module = this.modules && this.modules.credentials;
        if (module && typeof module.updateView === 'function') {
            module.updateView(data);
        }
    }

    /**
     * 更新指标视图
     */
    updateMetricsView(data: any): void {
        const module = this.modules && this.modules.metrics;
        if (module && typeof module.updateView === 'function') {
            module.updateView(data);
        }
    }

    /**
     * 初始化无障碍功能
     */
    initializeAccessibility() {
        // 添加跳转链接
        addSkipLinks();

        // 增强键盘导航
        this.enhanceKeyboardNavigation();

        // 设置页面级别的 ARIA 属性
        document.documentElement.setAttribute('lang', 'zh-CN');
        
        // 为主要区域添加地标
        setTimeout(() => {
            const mainContent = document.querySelector('.main-content');
            if (mainContent) {
                mainContent.setAttribute('id', 'main-content');
                mainContent.setAttribute('role', 'main');
            }

            const sidebar = document.querySelector('.sidebar');
            if (sidebar) {
                sidebar.setAttribute('id', 'sidebar');
                sidebar.setAttribute('role', 'navigation');
                sidebar.setAttribute('aria-label', '主导航');
            }
        }, 100);
    }

    /**
     * 增强键盘导航
     */
    enhanceKeyboardNavigation(): void {
        // 为所有按钮添加适当的 ARIA 属性
        document.addEventListener('click', (e: MouseEvent) => {
            const target = e.target as HTMLElement;
            if (target && target.tagName === 'BUTTON') {
                const button = target as HTMLButtonElement;

                // 增强常见按钮类型
                if (button.classList.contains('tab-button')) {
                    enhanceButton(button, {
                        controls: button.getAttribute('aria-controls'),
                        expanded: button.getAttribute('aria-selected') === 'true'
                    });
                }
            }
        });

        // 改进模态框焦点管理
        const originalShowLoginDialog = auth.showLoginDialog;
        auth.showLoginDialog = function(options: any) {
            const result = originalShowLoginDialog.call(this, options);

            // 在对话框显示后应用焦点管理
            setTimeout(() => {
                const modal = document.querySelector('.modal.active');
                if (modal) {
                    manageFocus(modal, { trap: true, autoFocus: true, restoreOnEscape: true });
                }
            }, 100);

            return result;
        };
    }

    /**
     * 设置事件监听
     */
    setupEventListeners(): void {
        // 使用解耦后的路由监听
        layoutInitHashRouter(() => this.getTabFromHash(), (tab: string) => {
            if (tab && tab !== this.currentTab) this.switchTab(tab);
        });

        // 监听凭证变更
        window.addEventListener('credentialsChanged', () => {
            this.updateDashboard();
        });

        // 监听指标变更
        window.addEventListener('metricsChanged', () => {
            this.updateDashboard();
        });

		// 监听键盘快捷键
		if (!this.detachShortcuts) {
			this.detachShortcuts = bindAdminShortcuts(this, isFormInput);
		}

        // 监听在线/离线状态
        window.addEventListener('online', () => {
            auth.showAlert('success', ui.t('notify_network_online'));
            ui.hideBanner('netBanner');
        });

        window.addEventListener('offline', () => {
            auth.showAlert('warning', ui.t('notify_network_offline'));
            ui.banner('netBanner', 'warning', ui.t('network_offline_banner'));
        });

        window.addEventListener('ui:lang-change', () => {
            this.renderLayout();
        });

        window.addEventListener('upstream:suggestions', (event: Event) => {
            const customEvent = event as CustomEvent;
            this.upstreamDetail = customEvent.detail;
            if (this.currentTab === 'config') {
                configManager.handleUpstreamSuggestions(this.upstreamDetail);
            }
        });

        window.addEventListener('probe-history-updated', () => {
            if (this.currentTab === 'config') {
                configManager.refreshProbeHistory(true);
            }
        });
    }


    /**
     * 导出当前标签页数据
     */
    exportCurrentTabData(): void {
        switch (this.currentTab) {
            case 'config':
                if ((window as any).configManager) {
                    (window as any).configManager.exportConfig();
                }
                break;
            case 'credentials':
                this.exportCredentialsData();
                break;
            case 'stats':
                this.exportStatsData();
                break;
            case 'logs':
                this.exportLogsData();
                break;
            default:
                if (window.ui && window.ui.showNotification) {
                    window.ui.showNotification('info', '当前页面不支持导出');
                }
        }
    }

    /**
     * 导出凭证数据
     */
    exportCredentialsData(): void {
        const credentials = credsManager.getCredentials();
        const data = {
            exported_at: new Date().toISOString(),
            credentials: credentials.map((cred: any) => ({
                filename: cred.filename,
                email: cred.email,
                project_id: cred.project_id,
                disabled: cred.disabled,
                health_score: cred.health_score,
                total_requests: cred.total_requests,
                success_rate: cred.success_rate
            }))
        };

        const jsonStr = JSON.stringify(data, null, 2);
        const blob = new Blob([jsonStr], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = `credentials-export-${new Date().toISOString().slice(0, 10)}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        if (window.ui && window.ui.showNotification) {
            window.ui.showNotification('凭证数据导出成功', 'success');
        }
    }

    /**
     * 导出统计数据
     */
    exportStatsData() {
        const stats = metricsManager.stats;
        if (!stats) {
            if (window.ui && window.ui.showNotification) {
                window.ui.showNotification('暂无统计数据可导出', 'warning');
            }
            return;
        }

        const data = {
            exported_at: new Date().toISOString(),
            stats: stats
        };

        const jsonStr = JSON.stringify(data, null, 2);
        const blob = new Blob([jsonStr], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = `stats-export-${new Date().toISOString().slice(0, 10)}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        if (window.ui && window.ui.showNotification) {
            window.ui.showNotification('success', '统计数据导出成功');
        }
    }

    /**
     * 导出日志数据
     */
    exportLogsData(): void {
        const logsManager = (window as any).logsManager;
        if (logsManager && typeof logsManager.exportLogs === 'function') {
            logsManager.exportLogs();
        } else {
            if (window.ui && window.ui.showNotification) {
                window.ui.showNotification('warning', '日志导出功能暂不可用');
            }
        }
    }

    /**
     * 标签页导航
     */
    navigateTab(direction: number): void {
        const currentIndex = this.tabs.indexOf(this.currentTab);
        if (currentIndex === -1) return;

        const newIndex = (currentIndex + direction + this.tabs.length) % this.tabs.length;
        const newTab = this.tabs[newIndex];
        if (newTab) {
            this.switchTab(newTab);
        }
    }

    /**
     * 显示跳转菜单
     */
    showGotoMenu() {
        const menuItems = [
            { key: 'd', name: '仪表板', tab: 'dashboard' },
            { key: 'c', name: '凭证管理', tab: 'credentials' },
            { key: 'o', name: 'OAuth', tab: 'oauth' },
            { key: 's', name: '统计', tab: 'stats' },
            { key: 'l', name: '日志', tab: 'logs' },
            { key: 'm', name: '模型注册', tab: 'models' },
            { key: 'g', name: '配置', tab: 'config' }
        ];

        const menuHTML = `
            <div class="goto-menu">
                <h4>快速跳转</h4>
                <div class="goto-items">
                    ${menuItems.map(item => `
                        <button class="goto-item" onclick="admin.switchTab('${item.tab}'); window.closeModal();">
                            <kbd>${item.key}</kbd>
                            <span>${item.name}</span>
                        </button>
                    `).join('')}
                </div>
                <p class="goto-hint">按相应字母键快速跳转</p>
            </div>
        `;

        // 移除主题选择器（固定极简主题，不暴露切换）
        try {
            const themeSelect = document.getElementById('themeSelect');
            if (themeSelect) {
                const label = themeSelect.closest('label');
                if (label && label.parentElement) label.remove(); else themeSelect.remove();
            }
        } catch {}

        const win = window as any;
        if (win.openModal) {
            win.openModal('快速跳转', menuHTML);
        }

        // 监听字母键
        const handleGotoKey = (e: KeyboardEvent) => {
            const item = menuItems.find(item => item.key === e.key.toLowerCase());
            if (item) {
                e.preventDefault();
                this.switchTab(item.tab);
                if (win.closeModal) win.closeModal();
                document.removeEventListener('keydown', handleGotoKey);
            }
            if (e.key === 'Escape') {
                if (win.closeModal) win.closeModal();
                document.removeEventListener('keydown', handleGotoKey);
            }
        };

        document.addEventListener('keydown', handleGotoKey);
    }

    /**
     * 渲染布局
     */
    async renderLayout(): Promise<void> {
        const container = document.getElementById('app-container');
        if (!container) {
            return;
        }

        const t = (key: string) => ui.t(key);
        const win1 = window as any;
        const basePath = (typeof auth.getBasePath === 'function' ? auth.getBasePath() : (win1.__BASE_PATH__ || '')) || '';
        const routesHref = basePath ? `${basePath}/routes` : '/routes';
        const initialTab = this.getTabFromHash() || this.currentTab || 'dashboard';
        this.currentTab = initialTab;

        container.innerHTML = `
            <div class="layout">
                <aside class="sidebar" id="sidebar">
                    <div class="sidebar-header">
                        <div class="brand">GCLI2API</div>
                    </div>
            <nav class="nav" role="tablist" aria-label="${t('aria_main_nav')}">
                <button id="tab-dashboard" aria-selected="${initialTab==='dashboard'}" role="tab" aria-controls="panel-dashboard" class="nav-btn tab-button ${initialTab==='dashboard' ? 'active' : ''}" data-tab="dashboard" onclick="window.admin.switchTab('dashboard')">${this.getTabIcon('dashboard')} ${t('tab_dashboard')}</button>
                <button id="tab-assembly" aria-selected="${initialTab==='assembly'}" role="tab" aria-controls="panel-assembly" class="nav-btn tab-button ${initialTab==='assembly' ? 'active' : ''}" data-tab="assembly" onclick="window.admin.switchTab('assembly')">${this.getTabIcon('assembly')} ${t('tab_assembly') || '路由装配台'}</button>
                <button id="tab-credentials" aria-selected="${initialTab==='credentials'}" role="tab" aria-controls="panel-credentials" class="nav-btn tab-button ${initialTab==='credentials' ? 'active' : ''}" data-tab="credentials" onclick="window.admin.switchTab('credentials')">${this.getTabIcon('credentials')} ${t('tab_credentials')}</button>
                <button id="tab-oauth" aria-selected="${initialTab==='oauth'}" role="tab" aria-controls="panel-oauth" class="nav-btn tab-button ${initialTab==='oauth' ? 'active' : ''}" data-tab="oauth" onclick="window.admin.switchTab('oauth')">${this.getTabIcon('oauth')} ${t('tab_oauth')}</button>
                <button id="tab-stats" aria-selected="${initialTab==='stats'}" role="tab" aria-controls="panel-stats" class="nav-btn tab-button ${initialTab==='stats' ? 'active' : ''}" data-tab="stats" onclick="window.admin.switchTab('stats')">${this.getTabIcon('stats')} ${t('tab_stats')}</button>
                <button id="tab-streaming" aria-selected="${initialTab==='streaming'}" role="tab" aria-controls="panel-streaming" class="nav-btn tab-button ${initialTab==='streaming' ? 'active' : ''}" data-tab="streaming" onclick="window.admin.switchTab('streaming')">${this.getTabIcon('streaming')} ${t('tab_streaming')}</button>
                <button id="tab-logs" aria-selected="${initialTab==='logs'}" role="tab" aria-controls="panel-logs" class="nav-btn tab-button ${initialTab==='logs' ? 'active' : ''}" data-tab="logs" onclick="window.admin.switchTab('logs')">${this.getTabIcon('logs')} ${t('tab_logs')}</button>
                <button id="tab-models" aria-selected="${initialTab==='models'}" role="tab" aria-controls="panel-models" class="nav-btn tab-button ${initialTab==='models' ? 'active' : ''}" data-tab="models" onclick="window.admin.switchTab('models')">${this.getTabIcon('models')} ${t('tab_models')}</button>
                <button id="tab-config" aria-selected="${initialTab==='config'}" role="tab" aria-controls="panel-config" class="nav-btn tab-button ${initialTab==='config' ? 'active' : ''}" data-tab="config" onclick="window.admin.switchTab('config')">${this.getTabIcon('config')} ${t('tab_config')}</button>
                <a class="nav-btn external" href="${routesHref}" title="查看对外开放的API端点">${t('nav_routes')}</a>
                <!-- assembly 已作为内置标签提供，无需外链 -->
            </nav>
                    <div class="sidebar-footer">v2.0.0 · geminicli 专属</div>
                </aside>
                <main class="main-content">
                    <div class="header">
                        <h1>${t('header_title')}</h1>
                        <p>${t('header_subtitle')}</p>
                        <div class="status-badges">
                            <button id="sidebarToggle" class="btn btn-secondary btn-sm" title="${t('tooltip_toggle_nav')}" aria-controls="sidebar" aria-expanded="false">☰ ${t('btn_toggle_nav')}</button>
                            <span class="badge badge-success" id="systemStatus">${t('badge_system_running')}</span>
                            <span class="badge badge-info" id="credentialCount">${t('badge_credentials')}: ${t('status_loading')}</span>
                            <span class="badge badge-warning" id="requestCount">${t('badge_requests')}: 0</span>
                            <span class="badge badge-info" id="userInfo">${t('badge_user')}: ${t('status_loading')}</span>
                            <span style="margin-left:auto"></span>
                            <!-- 主题选择器已隐藏，固定使用极简风格 -->
                            <label style="display:flex; align-items:center; gap:8px;">
                                <span style="color:#666;">${t('label_auto_refresh')}</span>
                                <input type="checkbox" id="autoRefreshToggle" />
                            </label>
                            <select id="autoRefreshInterval" class="form-control" style="padding:6px; border-radius:6px; border:1px solid #e5e7eb;">
                                <option value="15000">15s</option>
                                <option value="30000" selected>30s</option>
                                <option value="60000">60s</option>
                            </select>
                        </div>
                    </div>
                    <div id="tabContent" role="tabpanel" aria-labelledby="tab-${initialTab}" aria-live="polite">
                        ${this.renderTabContent(initialTab)}
                    </div>
                </main>
            </div>
        `;

        // 更新用户信息
        this.updateUserInfo();

        // 自动刷新、侧边栏（主题固定为极简风）
        ui.applyTheme('minimal');
        ui.initLangSelect(document.getElementById('langSelect') as HTMLSelectElement);
        this.autoRefresh.initControls();
        this.initSidebarControls();

        // 根据当前tab加载数据
        const win2 = window as any;
        const base = (win2.__ADMIN_BOOTSTRAP_CTX__ && win2.__ADMIN_BOOTSTRAP_CTX__.basePath) || '';
        if (initialTab === 'dashboard' || initialTab === 'stats') {
            try {
                if (!win2.metricsView) {
                    const v = win2.__ASSET_VERSION__ || '20251026';
                    await import(`${base ? base : ''}/js/metrics_view.js?v=${encodeURIComponent(v)}&t=${Date.now()}`);
                }
            } catch (_) { /* ignore, dashboard.update 会兜底 */ }
            metricsManager.refreshAllData().then(() => {
                const enabled = this.autoRefresh.isEnabled();
                if (enabled) metricsManager.startAutoRefresh(this.autoRefresh.getInterval());
            });
        } else if (initialTab === 'streaming') {
            const streamingModule = await this.loadModule('streaming');
            if (streamingModule && typeof streamingModule.refresh === 'function') {
                streamingModule.refresh().then(() => {
                    if (this.autoRefresh.isEnabled() && typeof streamingModule.startAutoRefresh === 'function') {
                        streamingModule.startAutoRefresh(this.autoRefresh.getInterval());
                    }
                });
            }
        } else if (initialTab === 'credentials') {
            try {
                if (!win2.credsViewRenderCredentialsList) {
                    const v = win2.__ASSET_VERSION__ || '20251026';
                    await import(`${base ? base : ''}/js/creds_view.js?v=${encodeURIComponent(v)}&t=${Date.now()}`);
                }
            } catch (_) { /* optional enhancement; fallback to built-in renderer */ }
            credsManager.bindDomRefresh();
            credsManager.refreshCredentials().then(() => {
                const list = document.getElementById('credentialsList');
                if (list) {
                    list.innerHTML = credsManager.renderCredentialsList();
                    if (credsManager.getVirtualPref()) credsManager.mountVirtual();
                    else if (win2.credsViewPopulateCredentialGrid) win2.credsViewPopulateCredentialGrid(list.querySelector('.credentials-grid'), credsManager.pendingCredentialView, credsManager);
                    else credsManager.populateCredentialGrid(list.querySelector('.credentials-grid'));
                }
                credsManager.attachFilters();
            });
        } else if (initialTab === 'models') {
            registryManager.refreshGroups().finally(() => {
                registryManager.refreshRegistry();
            });
        }
    }

    /**
     * 渲染标签页内容
     */
    renderTabContent(tabName: string): string {
        switch (tabName) {
            case 'dashboard':
                return dashboard.renderPage();
            case 'assembly':
                return (this.modules.assembly && this.modules.assembly.renderPage) ? this.modules.assembly.renderPage() : assemblySkeleton;
            case 'credentials':
                return credsManager.renderCredentialsPage();
            case 'oauth':
                return this.renderOAuthPage();
            case 'stats':
                return metricsManager.renderStatsPage();
            case 'streaming':
                return (this.modules.streaming && this.modules.streaming.renderPage)
                    ? this.modules.streaming.renderPage()
                    : streamingSkeleton;
            case 'logs':
                return (logsManager && typeof logsManager.renderLogsPage === 'function') ? logsManager.renderLogsPage() : logsSkeleton;
            case 'models':
                if (registryManager && typeof registryManager.renderRegistryPage === 'function') {
                    return registryManager.renderRegistryPage();
                }
                return registrySkeleton;
            case 'config':
                return configManager.renderConfigPage();
            default:
                return `<div class="loading">${ui.t('page_loading')}</div>`;
        }
    }

    /**
     * 渲染仪表盘
     */
    // renderDashboard 已移至 dashboard 模块

    /**
     * 渲染OAuth页面
     */
    renderOAuthPage() {
        return `
            <div class="card">
                <h2>${ui.t('oauth_heading')}</h2>
                ${oauthManager.getOAuthHTML()}
            </div>
        `;
    }

    /**
     * 渲染配置页面
     */
    renderConfigPage(): string { return configManager.renderConfigPage(); }

    /**
     * 获取模块实例（兼容旧代码）
     */
    async getModule(moduleName: string): Promise<any> {
        return this.modules[moduleName] || await this.loadModule(moduleName);
    }

    /**
     * 切换标签页
     */
    async switchTab(tabName: string): Promise<void> {
        if (this.currentTab === tabName) {
            return;
        }

        // 显示加载状态
        const tabContent = document.getElementById('tabContent');
        if (!tabContent) return;

        try {
            // 更新URL hash，支持深链接与前进/后退
            this.setHashForTab(tabName);

            // 更新标签按钮状态
            document.querySelectorAll('.tab-button').forEach(btn => {
                btn.classList.remove('active');
                btn.setAttribute('aria-selected', 'false');
            });
            const activeBtn = document.querySelector(`[data-tab="${tabName}"]`);
            if (activeBtn) {
                activeBtn.classList.add('active');
                activeBtn.setAttribute('aria-selected', 'true');
                (activeBtn as HTMLElement).focus();
            }

            let placeholder = `
                <div class="loading-container" style="display: flex; align-items: center; justify-content: center; min-height: 200px;">
                    <div class="spinner"></div>
                    <span style="margin-left: 12px;">${ui.t('status_loading_page')}</span>
                </div>
            `;
            if (tabName === 'assembly') {
                placeholder = assemblySkeleton;
            } else if (tabName === 'streaming') {
                placeholder = streamingSkeleton;
            } else if (tabName === 'logs') {
                placeholder = logsSkeleton;
            } else if (tabName === 'models') {
                placeholder = registrySkeleton;
            }
            tabContent.innerHTML = placeholder;
            tabContent.setAttribute('aria-labelledby', `tab-${tabName}`);

            // 停止之前标签页的自动刷新
            if (this.modules.metrics && this.modules.metrics.stopAutoRefresh) {
                this.modules.metrics.stopAutoRefresh();
            }
            if (this.modules.streaming && this.modules.streaming.stopAutoRefresh) {
                this.modules.streaming.stopAutoRefresh();
            }

            // 加载标签页所需的模块
            const requiredModule = this.getTabModule(tabName);
            let module = null;
            
            if (requiredModule) {
                module = await this.loadModule(requiredModule);
            }

            // 渲染标签页内容
            tabContent.innerHTML = this.renderTabContent(tabName);

            // 执行标签页特定的初始化
            await this.initializeTabContent(tabName, module);

            this.currentTab = tabName;

            // 无障碍通知
            const tabLabel = ui.t(`tab_${tabName}`);
            announce(`已切换到 ${tabLabel} 页面`);

            // 移动端切换后收起侧边栏
            if (this.isMobile()) {
                this.toggleSidebar(false);
            }

        } catch (error) {
            console.error(`Failed to switch to tab ${tabName}:`, error);
            
            // 恢复原内容并显示错误
            tabContent.innerHTML = `
                <div class="error-container" style="text-align: center; padding: 40px;">
                    <div style="font-size: 48px; margin-bottom: 16px;">⚠️</div>
                    <h3 style="color: var(--danger); margin-bottom: 12px;">加载失败</h3>
                    <p style="color: var(--muted); margin-bottom: 20px;">无法加载 ${ui.t(`tab_${tabName}`)} 页面</p>
                    <button class="btn btn-primary" onclick="app.switchTab('${tabName}')">重试</button>
                    <button class="btn btn-secondary" onclick="app.switchTab('dashboard')" style="margin-left: 8px;">返回仪表盘</button>
                </div>
            `;
            
            ui.showNotification('error', '加载失败', `无法加载 ${ui.t(`tab_${tabName}`)} 页面，请重试`);
        }
    }

    /**
     * 初始化标签页内容
     */
    async initializeTabContent(tabName: string, module: any): Promise<void> {
        switch (tabName) {
            case 'assembly':
                if (module && typeof module.update === 'function') {
                    await module.update();
                }
                break;
            case 'credentials':
                if (module && module.refreshCredentials) {
                    try {
                        const win = window as any;
                        if (!win.credsViewRenderCredentialsList) {
                            const base = (win.__ADMIN_BOOTSTRAP_CTX__ && win.__ADMIN_BOOTSTRAP_CTX__.basePath) || '';
                            await import(`${base ? base : ''}/js/creds_view.js`);
                        }
                    } catch (_) { /* optional enhancement */ }
                    const list = document.getElementById('credentialsList');
                    if (list) {
                        list.innerHTML = module.renderCredentialsList();
                        if (module.getVirtualPref()) {
                            module.mountVirtual();
                        } else {
                            module.populateCredentialGrid(list.querySelector('.credentials-grid'));
                        }
                    }
                    module.attachFilters();
                }
                break;

            case 'models':
                if (module && module.applyDescriptorToForm) {
                    const baseSelect = document.getElementById('regBase') as HTMLSelectElement;
                    if (baseSelect) {
                        module.applyDescriptorToForm(baseSelect.value);
                    }
                }
                break;

            case 'dashboard':
            case 'stats':
                const metricsModule = this.modules.metrics;
                if (metricsModule && metricsModule.startAutoRefresh && this.autoRefresh.isEnabled()) {
                    metricsModule.startAutoRefresh(this.autoRefresh.getInterval());
                }
                break;
            case 'streaming':
                if (module && typeof module.startAutoRefresh === 'function' && this.autoRefresh.isEnabled()) {
                    module.startAutoRefresh(this.autoRefresh.getInterval());
                }
                break;
        }
    }

    /**
     * 加载初始数据
     */
    async loadInitialData(): Promise<void> {
        // 确保统计视图模块已就绪，避免 dashboard.update() 期间访问 window.metricsView 为空
        try {
            const win = window as any;
            if (!win.metricsView) {
                const base = (win.__ADMIN_BOOTSTRAP_CTX__ && win.__ADMIN_BOOTSTRAP_CTX__.basePath) || '';
                const v = win.__ASSET_VERSION__ || '20251026';
                await import(`${base ? base : ''}/js/metrics_view.js?v=${encodeURIComponent(v)}&t=${Date.now()}`);
            }
        } catch (_) { /* 忽略，dashboard.update 内部仍有占位渲染 */ }

        await Promise.all([
            dashboard.update(),
            dashboard.updateStatusBadges()
        ]);
    }

    /**
     * 更新仪表盘
     */
    async updateDashboard(): Promise<void> {
        // 更新统计卡片
        await dashboard.update();
    }

    /**
     * 更新状态徽章
     */
    async updateStatusBadges(): Promise<void> {
        await dashboard.updateStatusBadges();
    }

    /**
     * 更新用户信息
     */
    updateUserInfo(): void {
        const userInfo = document.getElementById('userInfo');
        if (!userInfo) return;
        const t = (key: string) => ui.t(key);
        if (auth.isAuthenticated()) {
            userInfo.textContent = t('user_status_authenticated');
            userInfo.className = 'badge badge-success';
        } else {
            userInfo.textContent = t('user_status_none');
            userInfo.className = 'badge badge-warning';
        }
    }

    /**
     * 加载配置
     */
    async loadConfig(): Promise<any> { return configManager.loadAndRender(this.upstreamDetail); }

    /**
     * 渲染配置表单
     */
    renderConfigForm(config: any, suggestionDetail: any = null): string { return configManager.renderForm(config, suggestionDetail); }

    /**
     * 保存配置
     */
    async saveConfig(): Promise<any> {
        return configManager.saveConfig();
    }

    bindProbeHistoryControls(): void {
        return configManager.bindProbeHistoryControls();
    }

    async refreshProbeHistory(force: boolean = false): Promise<any> {
        return configManager.refreshProbeHistory(force);
    }

    renderProbeHistoryList(history: any): string { return configManager.renderProbeHistoryList(history); }

    downloadProbeHistory(): void { return configManager.downloadProbeHistory(); }

    describeSuggestionSource(detail: any): string { return configManager.describeSuggestionSource(detail); }

    updateConfigSuggestionMeta(): void { return configManager.updateConfigSuggestionMeta(); }

    /**
     * 刷新当前标签页
     */
    async refreshCurrentTab(): Promise<void> {
        switch (this.currentTab) {
            case 'dashboard':
                await this.updateDashboard();
                break;
            case 'credentials':
                await credsManager.refreshCredentials();
                break;
            case 'stats':
                await metricsManager.refreshAllData();
                break;
            case 'config':
                await this.loadConfig();
                break;
        }
    }


    // 解析 hash 中的 tab
    getTabFromHash(): string | null {
        const hash = (location.hash || '').replace(/^#/, '').trim();
        if (!hash) return null;
        // 支持 #tab 或 #/tab 两种形式
        const tab = hash.startsWith('/') ? hash.slice(1) : hash;
        return this.tabs.includes(tab) ? tab : null;
    }

    setHashForTab(tab: string): void {
        if (!this.tabs.includes(tab)) return;
        layoutSetHashForTab(tab);
    }


    // 侧边栏控制
    isMobile(): boolean {
        return layoutIsMobile();
    }

    initSidebarControls(): void {
        layoutBindSidebar({ onToggle: (_open: boolean) => { /* 保留扩展点 */ } });
    }

    toggleSidebar(open: boolean): void {
        layoutToggleSidebar(open);
    }

    /**
     * 构建快速切换器候选项
     */
    buildQuickSwitcherItems(): any[] {
        const t = (key: string) => ui.t(key);
        const sections = {
            tabs: t('quick_switch_section_tabs'),
            credentials: t('quick_switch_section_credentials'),
            models: t('quick_switch_section_models'),
            actions: t('quick_switch_section_actions'),
        };
        const items: any[] = [];
        const pushItem = (item: any) => {
            const parts = [];
            if (item.title) parts.push(String(item.title));
            if (item.meta) parts.push(String(item.meta));
            if (Array.isArray(item.keywords)) {
                item.keywords.forEach((kw: any) => {
                    if (kw !== undefined && kw !== null) {
                        parts.push(String(kw));
                    }
                });
            }
            item.searchText = parts.join(' ').toLowerCase();
            items.push(item);
        };

        this.tabs.forEach((tab) => {
            const name = t(`tab_${tab}`);
            pushItem({
                id: `tab:${tab}`,
                type: 'tab',
                section: sections.tabs,
                icon: this.getTabIcon(tab),
                title: name,
                meta: t('quick_switch_tab_meta'),
                keywords: [tab, name],
                action: () => this.switchTab(tab),
            });
        });

        const credentials = (credsManager && typeof credsManager.getCredentials === 'function')
            ? credsManager.getCredentials()
            : [];
        credentials.forEach((cred: any) => {
            const identifier = cred.filename || cred.id || cred.email || cred.project_id || '';
            const project = cred.project_id || cred.email || '';
            const health = Math.round((cred.health_score || 0) * 100);
            pushItem({
                id: `credential:${identifier}`,
                type: 'credential',
                section: sections.credentials,
                icon: '🔑',
                title: identifier || t('quick_switch_section_credentials'),
                meta: this.interpolate(t('quick_switch_cred_meta'), { project: project || 'N/A', health }),
                keywords: [identifier, cred.email, cred.project_id, cred.status, cred.banned_reason],
                action: () => {
                    this.switchTab('credentials');
                    const target = identifier;
                    setTimeout(() => credsManager.highlightCredential(target), 80);
                },
            });
        });

        const models = (registryManager && Array.isArray(registryManager.models))
            ? registryManager.models
            : [];
        models.forEach((model: any, index: number) => {
            const displayId = typeof registryManager.computeDisplayId === 'function'
                ? registryManager.computeDisplayId(model)
                : (model.id || model.base || `#${index + 1}`);
            const options = (typeof registryManager.describeOptions === 'function'
                ? registryManager.describeOptions(model)
                : '') || '-';
            pushItem({
                id: `model:${displayId}`,
                type: 'model',
                section: sections.models,
                icon: '🧩',
                title: displayId,
                meta: this.interpolate(t('quick_switch_model_meta'), { base: model.base || '-', options }),
                keywords: [displayId, model.base, options, model.group, model.family],
                action: () => {
                    this.switchTab('models');
                    const focusId = model.id || displayId;
                    setTimeout(() => registryManager.focusModel(focusId), 80);
                },
            });
        });

        pushItem({
            id: 'action:assembly',
            type: 'action',
            section: sections.actions,
            icon: '🧷',
            title: t('quick_switch_open_assembly'),
            meta: t('quick_switch_action_meta'),
            keywords: ['/assembly', 'assembly', 'external'],
            action: () => window.open('/assembly', '_blank', 'noopener'),
        });

        pushItem({
            id: 'action:routes',
            type: 'action',
            section: sections.actions,
            icon: '🗺️',
            title: t('quick_switch_open_routes'),
            meta: t('quick_switch_action_meta'),
            keywords: ['/routes', 'routes'],
            action: () => window.open('/routes', '_blank', 'noopener'),
        });

        return items;
    }

    interpolate(template: any, params: Record<string, any> = {}): string {
        if (typeof template !== 'string') {
            return '';
        }
        return template.replace(/\{(\w+)\}/g, (_: string, key: string) => {
            if (Object.prototype.hasOwnProperty.call(params, key)) {
                const val = params[key];
                return val === undefined || val === null ? '' : String(val);
            }
            return '';
        });
    }

    /**
     * 显示快速切换器
     */
    showQuickSwitcher() {
        // 委托到模块化实现，便于维护与测试
        renderQuickSwitcher(this);
    }


    showHelpModal(): void {
        const t = (key: string) => ui.t(key);
        const modal = document.createElement('div');
        modal.className = 'modal active';
        modal.innerHTML = `
            <div class="modal-content" role="dialog" aria-modal="true">
                <button type="button" class="modal-close" aria-label="${t('aria_close')}">&times;</button>
                <div class="modal-header">⌨️ 键盘快捷键</div>
                <div class="shortcuts-help">
                    <div class="shortcuts-section">
                        <h4>🚀 全局快捷键</h4>
                        <div class="shortcuts-list">
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>K</kbd></div>
                                <div class="shortcut-description">快速切换标签页</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>R</kbd></div>
                                <div class="shortcut-description">刷新当前页面</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>S</kbd></div>
                                <div class="shortcut-description">保存配置</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>E</kbd></div>
                                <div class="shortcut-description">导出当前数据</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>H</kbd></div>
                                <div class="shortcut-description">显示/隐藏侧边栏</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>G</kbd></div>
                                <div class="shortcut-description">快速跳转菜单</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>?</kbd></div>
                                <div class="shortcut-description">显示此帮助</div>
                            </div>
                        </div>
                    </div>
                    <div class="shortcuts-section">
                        <h4>📋 标签页切换</h4>
                        <div class="shortcuts-list">
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Alt</kbd><kbd>1-7</kbd></div>
                                <div class="shortcut-description">直接切换到指定标签页</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Alt</kbd><kbd>←</kbd><kbd>→</kbd></div>
                                <div class="shortcut-description">左右切换标签页</div>
                            </div>
                        </div>
                    </div>
                    <div class="shortcuts-section">
                        <h4>📝 凭证管理</h4>
                        <div class="shortcuts-list">
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>B</kbd></div>
                                <div class="shortcut-description">切换批量操作模式</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>A</kbd></div>
                                <div class="shortcut-description">全选凭证 (批量模式)</div>
                            </div>
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Delete</kbd></div>
                                <div class="shortcut-description">删除选中凭证 (批量模式)</div>
                            </div>
                        </div>
                    </div>
                    <div class="shortcuts-section">
                        <h4>⚙️ 配置管理</h4>
                        <div class="shortcuts-list">
                            <div class="shortcut-item">
                                <div class="shortcut-keys"><kbd>Ctrl</kbd><kbd>I</kbd></div>
                                <div class="shortcut-description">导入配置文件</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;

        const previousActive = document.activeElement;
        document.body.appendChild(modal);

        const focusable = Array.from(modal.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'));

        const closeModal = () => {
            modal.removeEventListener('keydown', handleKeyDown);
            modal.removeEventListener('click', handleOverlayClick);
            modal.remove();
            if (previousActive && typeof (previousActive as any).focus === 'function') {
                (previousActive as HTMLElement).focus();
            }
        };

        const handleOverlayClick = (e: MouseEvent) => {
            if (e.target === modal) {
                closeModal();
            }
        };

        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                e.preventDefault();
                closeModal();
                return;
            }
            if (e.key === 'Tab' && focusable.length > 0) {
                const first = focusable[0] as HTMLElement;
                const last = focusable[focusable.length - 1] as HTMLElement;
                if (e.shiftKey && document.activeElement === first) {
                    e.preventDefault();
                    last.focus();
                } else if (!e.shiftKey && document.activeElement === last) {
                    e.preventDefault();
                    first.focus();
                }
            }
        };

        const closeButton = modal.querySelector('.modal-close');
        if (closeButton) {
            closeButton.addEventListener('click', closeModal);
        }

        modal.addEventListener('click', handleOverlayClick);
        modal.addEventListener('keydown', handleKeyDown);

        if (focusable.length > 0) {
            (focusable[0] as HTMLElement).focus();
        }
    }

    /**
     * 获取标签图标
     */
    getTabIcon(_tab: string): string {
        // 极简风：不使用图标，专注文本
        return '';
    }

    /**
     * 显示错误消息
     */
    showErrorMessage(message: string): void {
        const container = document.getElementById('app-container');
        if (container) {
            container.innerHTML = `
                <div class="card" style="text-align: center; padding: 60px 20px;">
                    <h2 style="color: #ef4444; margin-bottom: 20px;">❌ ${ui.t('error_title')}</h2>
                    <p style="color: #666; margin-bottom: 30px;">${message}</p>
                    <button class="btn btn-primary" onclick="location.reload()">${ui.t('btn_reload_page')}</button>
                </div>
            `;
            const reloadBtn = container.querySelector('button');
            if (reloadBtn) {
                reloadBtn.focus();
            }
        }
    }

    /**
     * 销毁应用
     */
    destroy() {
        // 清理模块
        if (metricsManager) {
            metricsManager.destroy();
        }
        if (logsManager && typeof logsManager.destroy === 'function') {
            logsManager.destroy();
        }

        // 清理事件监听
        window.removeEventListener('credentialsChanged', this.updateDashboard);
        window.removeEventListener('metricsChanged', this.updateDashboard);

        if (this.detachShortcuts) {
            this.detachShortcuts();
            this.detachShortcuts = undefined;
        }

        // 清理定时器
        this.autoRefresh.stopDashboardRefresh();
    }
}
