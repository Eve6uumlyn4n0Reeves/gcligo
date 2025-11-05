/**
 * 路由管理模块
 * 负责 URL hash 路由和标签页切换逻辑
 */

export interface RouterConfig {
    tabs: string[];
    defaultTab: string;
    onTabChange?: (tab: string) => void;
}

export class Router {
    private tabs: string[];
    private currentTab: string;
    private onTabChange?: (tab: string) => void;

    constructor(config: RouterConfig) {
        this.tabs = config.tabs;
        this.currentTab = config.defaultTab;
        this.onTabChange = config.onTabChange;
    }

    /**
     * 从 URL hash 获取标签页名称
     */
    getTabFromHash(): string | null {
        const hash = window.location.hash.slice(1);
        if (!hash) return null;
        
        const tab = hash.split('/')[0];
        return this.tabs.includes(tab) ? tab : null;
    }

    /**
     * 设置 URL hash
     */
    setHashForTab(tab: string): void {
        if (!this.tabs.includes(tab)) return;
        window.location.hash = `#${tab}`;
    }

    /**
     * 初始化 hash 路由监听
     */
    initHashRouter(getTabFromHash: () => string | null, onHashChange: (tab: string) => void): void {
        // 监听 hash 变化
        window.addEventListener('hashchange', () => {
            const tab = getTabFromHash();
            if (tab) {
                onHashChange(tab);
            }
        });

        // 初始化时检查 hash
        const initialTab = getTabFromHash();
        if (initialTab) {
            onHashChange(initialTab);
        }
    }

    /**
     * 切换到指定标签页
     */
    switchTo(tab: string): void {
        if (!this.tabs.includes(tab)) {
            console.warn(`Invalid tab: ${tab}`);
            return;
        }

        if (this.currentTab === tab) {
            return;
        }

        this.currentTab = tab;
        this.setHashForTab(tab);

        if (this.onTabChange) {
            this.onTabChange(tab);
        }
    }

    /**
     * 获取当前标签页
     */
    getCurrentTab(): string {
        return this.currentTab;
    }

    /**
     * 获取所有标签页
     */
    getTabs(): string[] {
        return [...this.tabs];
    }

    /**
     * 导航到下一个/上一个标签页
     */
    navigate(direction: 1 | -1): void {
        const currentIndex = this.tabs.indexOf(this.currentTab);
        const newIndex = (currentIndex + direction + this.tabs.length) % this.tabs.length;
        const newTab = this.tabs[newIndex];
        if (newTab) {
            this.switchTo(newTab);
        }
    }

    /**
     * 导航到下一个标签页
     */
    next(): void {
        this.navigate(1);
    }

    /**
     * 导航到上一个标签页
     */
    prev(): void {
        this.navigate(-1);
    }
}

/**
 * 创建路由实例
 */
export function createRouter(config: RouterConfig): Router {
    return new Router(config);
}

