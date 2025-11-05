/**
 * 布局辅助模块
 * 提供布局相关的工具函数和常量
 */

export interface TabConfig {
    id: string;
    label: string;
    icon: string;
    ariaLabel?: string;
}

export interface LayoutConfig {
    tabs: TabConfig[];
    basePath?: string;
    version?: string;
}

/**
 * 获取标签页图标
 */
export function getTabIcon(tab: string): string {
    const icons: Record<string, string> = {
        dashboard: '📊',
        assembly: '🔧',
        credentials: '🔑',
        oauth: '🔐',
        stats: '📈',
        streaming: '🌊',
        logs: '📝',
        models: '🤖',
        config: '⚙️',
    };
    return icons[tab] || '📄';
}

/**
 * 生成导航按钮 HTML
 */
export function renderNavButton(tab: TabConfig, isActive: boolean, _t: (key: string) => string): string {
    const activeClass = isActive ? 'active' : '';
    const ariaSelected = isActive ? 'true' : 'false';
    const ariaLabel = tab.ariaLabel || tab.label;
    
    return `
        <button 
            id="tab-${tab.id}" 
            aria-selected="${ariaSelected}" 
            role="tab" 
            aria-controls="panel-${tab.id}" 
            class="nav-btn tab-button ${activeClass}" 
            data-tab="${tab.id}" 
            onclick="window.admin.switchTab('${tab.id}')"
            aria-label="${ariaLabel}"
        >
            ${tab.icon} ${tab.label}
        </button>
    `;
}

/**
 * 生成侧边栏 HTML
 */
export function renderSidebar(config: LayoutConfig, currentTab: string, t: (key: string) => string): string {
    const navButtons = config.tabs.map(tab => 
        renderNavButton(tab, tab.id === currentTab, t)
    ).join('\n');

    const basePath = config.basePath || '';
    const routesHref = basePath ? `${basePath}/routes` : '/routes';
    const version = config.version || 'v2.0.0';

    return `
        <aside class="sidebar" id="sidebar">
            <div class="sidebar-header">
                <div class="brand">GCLI2API</div>
            </div>
            <nav class="nav" role="tablist" aria-label="${t('aria_main_nav')}">
                ${navButtons}
                <a class="nav-btn external" href="${routesHref}" title="${t('tooltip_view_routes')}">${t('nav_routes')}</a>
            </nav>
            <div class="sidebar-footer">${version} · geminicli 专属</div>
        </aside>
    `;
}

/**
 * 生成头部状态栏 HTML
 */
export function renderHeader(t: (key: string) => string): string {
    return `
        <div class="header">
            <h1>${t('header_title')}</h1>
            <p>${t('header_subtitle')}</p>
            <div class="status-badges">
                <button id="sidebarToggle" class="btn btn-secondary btn-sm" title="${t('tooltip_toggle_nav')}" aria-controls="sidebar" aria-expanded="false">
                    ☰ ${t('btn_toggle_nav')}
                </button>
                <span class="badge badge-success" id="systemStatus">${t('badge_system_running')}</span>
                <span class="badge badge-info" id="credentialCount">${t('badge_credentials')}: ${t('status_loading')}</span>
                <span class="badge badge-warning" id="requestCount">${t('badge_requests')}: 0</span>
                <span class="badge badge-info" id="userInfo">${t('badge_user')}: ${t('status_loading')}</span>
                <span style="margin-left:auto"></span>
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
    `;
}

/**
 * 生成完整布局 HTML
 */
export function renderLayout(
    config: LayoutConfig, 
    currentTab: string, 
    tabContent: string,
    t: (key: string) => string
): string {
    return `
        <div class="layout">
            ${renderSidebar(config, currentTab, t)}
            <main class="main-content">
                ${renderHeader(t)}
                <div id="tabContent" role="tabpanel" aria-labelledby="tab-${currentTab}" aria-live="polite">
                    ${tabContent}
                </div>
            </main>
        </div>
    `;
}

/**
 * 更新标签按钮状态
 */
export function updateTabButtonStates(activeTab: string): void {
    document.querySelectorAll('.tab-button').forEach(btn => {
        const isActive = btn.getAttribute('data-tab') === activeTab;
        btn.classList.toggle('active', isActive);
        btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });
}

/**
 * 侧边栏控制
 */
export interface SidebarController {
    toggle: () => void;
    open: () => void;
    close: () => void;
    isOpen: () => boolean;
    isMobile: () => boolean;
}

export function createSidebarController(): SidebarController {
    let isOpen = false;

    const isMobile = (): boolean => {
        return window.innerWidth < 768;
    };

    const toggle = (): void => {
        const sidebar = document.getElementById('sidebar');
        if (!sidebar) return;

        isOpen = !isOpen;
        sidebar.classList.toggle('open', isOpen);
        
        const toggleBtn = document.getElementById('sidebarToggle');
        if (toggleBtn) {
            toggleBtn.setAttribute('aria-expanded', isOpen ? 'true' : 'false');
        }
    };

    const open = (): void => {
        if (!isOpen) toggle();
    };

    const close = (): void => {
        if (isOpen) toggle();
    };

    return {
        toggle,
        open,
        close,
        isOpen: () => isOpen,
        isMobile,
    };
}

/**
 * 绑定侧边栏控制事件
 */
export function bindSidebarControls(controller: SidebarController, onToggle?: (isOpen: boolean) => void): void {
    const toggleBtn = document.getElementById('sidebarToggle');
    if (toggleBtn) {
        toggleBtn.addEventListener('click', () => {
            controller.toggle();
            if (onToggle) {
                onToggle(controller.isOpen());
            }
        });
    }

    // 移动端点击外部关闭侧边栏
    if (controller.isMobile()) {
        document.addEventListener('click', (e) => {
            const sidebar = document.getElementById('sidebar');
            const toggleBtn = document.getElementById('sidebarToggle');
            
            if (sidebar && controller.isOpen()) {
                const target = e.target as HTMLElement;
                if (!sidebar.contains(target) && target !== toggleBtn && !toggleBtn?.contains(target)) {
                    controller.close();
                    if (onToggle) {
                        onToggle(false);
                    }
                }
            }
        });
    }
}

/**
 * 标签页配置预设
 */
export const DEFAULT_TABS: TabConfig[] = [
    { id: 'dashboard', label: '仪表板', icon: '📊' },
    { id: 'assembly', label: '路由装配台', icon: '🔧' },
    { id: 'credentials', label: '凭证管理', icon: '🔑' },
    { id: 'oauth', label: 'OAuth', icon: '🔐' },
    { id: 'stats', label: '统计', icon: '📈' },
    { id: 'streaming', label: '流式', icon: '🌊' },
    { id: 'logs', label: '日志', icon: '📝' },
    { id: 'models', label: '模型', icon: '🤖' },
    { id: 'config', label: '配置', icon: '⚙️' },
];

