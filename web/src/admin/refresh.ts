
type AutoRefreshContext = {
	getCurrentTab: () => string;
	updateDashboard: () => void;
	getModules: () => Record<string, any>;
	getMetricsManager: () => any;
};

export type AutoRefreshManager = {
	startDashboardRefresh: () => void;
	stopDashboardRefresh: () => void;
	isEnabled: () => boolean;
	getInterval: () => number;
	initControls: () => void;
};

const DEFAULT_INTERVAL = 30000;

export function createAutoRefreshManager(ctx: AutoRefreshContext): AutoRefreshManager {
	let dashboardInterval: number | null = null;

	const readEnabled = () => {
		try {
			const value = localStorage.getItem('ui:autoRefresh');
			return value === null ? true : value === '1';
		} catch {
			return true;
		}
	};

	const readInterval = () => {
		try {
			const value = parseInt(localStorage.getItem('ui:autoRefreshInterval') || `${DEFAULT_INTERVAL}`, 10);
			return Number.isFinite(value) ? value : DEFAULT_INTERVAL;
		} catch {
			return DEFAULT_INTERVAL;
		}
	};

	const startDashboardRefresh = () => {
		if (dashboardInterval) {
			clearInterval(dashboardInterval);
		}
		dashboardInterval = window.setInterval(() => {
			if (ctx.getCurrentTab() === 'dashboard') {
				ctx.updateDashboard();
			}
		}, DEFAULT_INTERVAL);
	};

	const stopDashboardRefresh = () => {
		if (dashboardInterval) {
			clearInterval(dashboardInterval);
			dashboardInterval = null;
		}
	};

	const initControls = () => {
		const toggle = document.getElementById('autoRefreshToggle') as HTMLInputElement | null;
		const select = document.getElementById('autoRefreshInterval') as HTMLSelectElement | null;
		if (!toggle || !select) {
			return;
		}

		// 初始化控件状态
		toggle.checked = readEnabled();
		const interval = readInterval();
		if ([15000, 30000, 60000].includes(interval)) {
			select.value = String(interval);
		}

		toggle.addEventListener('change', () => {
			try {
				localStorage.setItem('ui:autoRefresh', toggle.checked ? '1' : '0');
			} catch {
				// 忽略本地存储失败
			}
			const metrics = ctx.getMetricsManager();
			const modules = ctx.getModules();
			const streaming = modules?.streaming;
			const currentTab = ctx.getCurrentTab();
			const intervalMs = readInterval();

			if (currentTab === 'dashboard' || currentTab === 'stats') {
				if (toggle.checked) {
					metrics?.startAutoRefresh?.(intervalMs);
				} else {
					metrics?.stopAutoRefresh?.();
				}
			}

			if (currentTab === 'streaming' && streaming) {
				if (toggle.checked && typeof streaming.startAutoRefresh === 'function') {
					streaming.startAutoRefresh(intervalMs);
				} else if (typeof streaming.stopAutoRefresh === 'function') {
					streaming.stopAutoRefresh();
				}
			}
		});

		select.addEventListener('change', () => {
			try {
				localStorage.setItem('ui:autoRefreshInterval', select.value);
			} catch {
				// 忽略本地存储失败
			}
			const metrics = ctx.getMetricsManager();
			const modules = ctx.getModules();
			const streaming = modules?.streaming;
			const currentTab = ctx.getCurrentTab();
			const intervalMs = parseInt(select.value, 10);

			if (toggle.checked && (currentTab === 'dashboard' || currentTab === 'stats')) {
				metrics?.startAutoRefresh?.(intervalMs);
			}

			if (
				toggle.checked &&
				currentTab === 'streaming' &&
				streaming &&
				typeof streaming.startAutoRefresh === 'function'
			) {
				streaming.startAutoRefresh(intervalMs);
			}
		});
	};

	return {
		startDashboardRefresh,
		stopDashboardRefresh,
		isEnabled: readEnabled,
		getInterval: readInterval,
		initControls
	};
}
