import type { AdminApp } from './app';

export function bindAdminShortcuts(app: AdminApp, isFormInput: (target: any) => boolean): () => void {
	const handler = (e: KeyboardEvent) => {
		const target = e.target as HTMLElement | null;
		if (isFormInput && isFormInput(target)) {
			return;
		}

		// Ctrl/Cmd + K: 快速切换标签
		if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
			e.preventDefault();
			app.showQuickSwitcher();
			return;
		}

		// Ctrl/Cmd + R: 刷新数据（不刷新页面）
		if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
			e.preventDefault();
			app.refreshCurrentTab();
			return;
		}

		// Ctrl/Cmd + S: 保存当前配置
		if ((e.ctrlKey || e.metaKey) && e.key === 's') {
			e.preventDefault();
			if ((app as any).currentTab === 'config') {
				app.saveConfig();
			}
			return;
		}

		// Ctrl/Cmd + E: 导出当前数据
		if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
			e.preventDefault();
			app.exportCurrentTabData();
			return;
		}

		// Ctrl/Cmd + I: 导入配置
		if ((e.ctrlKey || e.metaKey) && e.key === 'i') {
			e.preventDefault();
			const win = window as any;
			if ((app as any).currentTab === 'config' && win.configManager) {
				win.configManager.importConfig();
			}
			return;
		}

		// Ctrl/Cmd + B: 切换批量模式
		if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
			e.preventDefault();
			if ((app as any).currentTab === 'credentials' && window.credsManager) {
				const isBatchMode = document.body.classList.contains('batch-mode');
				if (isBatchMode) {
					window.credsManager.hideBatchMode();
				} else {
					window.credsManager.showBatchMode();
				}
			}
			return;
		}

		// Ctrl/Cmd + H: 显示/隐藏侧边栏
		if ((e.ctrlKey || e.metaKey) && e.key === 'h') {
			e.preventDefault();
			// Toggle sidebar - check current state and invert
			const sidebar = document.querySelector('.sidebar');
			const isOpen = sidebar?.classList.contains('open') ?? false;
			app.toggleSidebar(!isOpen);
			return;
		}

		// F5: 强制刷新页面
		if (e.key === 'F5') {
			e.preventDefault();
			if (window.ui && window.ui.showConfirmation) {
				window.ui
					.showConfirmation({
						title: '刷新页面',
						message: '确定要刷新整个页面吗？未保存的更改将丢失。',
						type: 'warning',
						confirmText: '刷新'
					})
					.then((confirmed: boolean) => {
						if (confirmed) {
							window.location.reload();
						}
					});
			} else {
				window.location.reload();
			}
			return;
		}

		// Shift + / (?): 快捷键帮助
		if (e.shiftKey && e.key === '?') {
			e.preventDefault();
			app.showHelpModal();
			return;
		}

		// Alt + 数字 切换标签
		if (e.altKey && /^[1-7]$/.test(e.key)) {
			e.preventDefault();
			const tabs = Array.isArray((app as any).tabs) ? (app as any).tabs : [];
			const idx = parseInt(e.key, 10) - 1;
			const tab = tabs[idx];
			if (tab) app.switchTab(tab);
			return;
		}

		// 方向键导航
		if (e.altKey && e.key === 'ArrowLeft') {
			e.preventDefault();
			app.navigateTab(-1);
			return;
		}

		if (e.altKey && e.key === 'ArrowRight') {
			e.preventDefault();
			app.navigateTab(1);
			return;
		}

		// G 键快速跳转
		if (e.key === 'g' && !e.ctrlKey && !e.metaKey && !e.altKey) {
			e.preventDefault();
			app.showGotoMenu();
		}
	};

	document.addEventListener('keydown', handler);
	return () => {
		document.removeEventListener('keydown', handler);
	};
}
