/**
 * 凭证批量操作模块
 * 提供批量操作相关的功能和UI
 */

import type { CredentialBatchAction } from './types.js';

export interface BatchProgress {
    total: number;
    completed: number;
    failed: number;
    current?: string;
}

export interface BatchOptions {
    confirmTitle?: string;
    confirmMessage?: string;
    successMessage?: string;
    operationName?: string;
    concurrency?: number;
}

/**
 * 批量操作管理器
 */
export class BatchOperationManager {
    private selectedItems: Set<string> = new Set();
    private batchMode = false;

    /**
     * 切换批量模式
     */
    toggleBatchMode(): void {
        this.batchMode = !this.batchMode;
        
        if (this.batchMode) {
            this.showBatchUI();
        } else {
            this.hideBatchUI();
            this.clearSelection();
        }
    }

    /**
     * 显示批量操作UI
     */
    showBatchUI(): void {
        document.body.classList.add('batch-mode');
        this.addBatchCheckboxes();
        this.updateBatchUI();
    }

    /**
     * 隐藏批量操作UI
     */
    hideBatchUI(): void {
        document.body.classList.remove('batch-mode');
        this.removeBatchCheckboxes();
    }

    /**
     * 添加批量选择复选框
     */
    private addBatchCheckboxes(): void {
        document.querySelectorAll<HTMLElement>('.credential-card').forEach((card) => {
            if (card.querySelector('.batch-checkbox')) return;
            
            const filename = card.dataset.filename || card.dataset.credId || '';
            if (!filename) return;

            const overlay = document.createElement('div');
            overlay.className = 'batch-checkbox-overlay';
            overlay.innerHTML = `
                <input 
                    type="checkbox" 
                    class="batch-checkbox" 
                    data-item-id="${filename}"
                    onchange="batchManager.toggleSelection('${filename}')"
                />
            `;
            
            card.style.position = 'relative';
            card.insertBefore(overlay, card.firstChild);
        });
    }

    /**
     * 移除批量选择复选框
     */
    private removeBatchCheckboxes(): void {
        document.querySelectorAll('.batch-checkbox-overlay').forEach(overlay => {
            overlay.remove();
        });
    }

    /**
     * 切换选择状态
     */
    toggleSelection(itemId: string): void {
        if (this.selectedItems.has(itemId)) {
            this.selectedItems.delete(itemId);
        } else {
            this.selectedItems.add(itemId);
        }
        this.updateBatchUI();
    }

    /**
     * 全选
     */
    selectAll(): void {
        document.querySelectorAll<HTMLInputElement>('.batch-checkbox:not([disabled])').forEach((checkbox) => {
            checkbox.checked = true;
            const itemId = checkbox.dataset.itemId;
            if (itemId) {
                this.selectedItems.add(itemId);
            }
        });
        this.updateBatchUI();
    }

    /**
     * 清除选择
     */
    clearSelection(): void {
        this.selectedItems.clear();
        document.querySelectorAll<HTMLInputElement>('.batch-checkbox').forEach((checkbox) => {
            checkbox.checked = false;
        });
        this.updateBatchUI();
    }

    /**
     * 获取选中项
     */
    getSelectedItems(): string[] {
        return Array.from(this.selectedItems);
    }

    /**
     * 获取选中数量
     */
    getSelectionCount(): number {
        return this.selectedItems.size;
    }

    /**
     * 更新批量操作UI
     */
    private updateBatchUI(): void {
        const count = this.selectedItems.size;
        const countElement = document.querySelector('.batch-count');
        if (countElement) {
            countElement.textContent = `已选择 ${count} 项`;
        }

        // 更新批量操作按钮状态
        document.querySelectorAll<HTMLButtonElement>('.batch-actions button').forEach(btn => {
            btn.disabled = count === 0;
        });
    }

    /**
     * 显示进度条
     */
    showProgress(progress: BatchProgress): void {
        const progressBar = document.querySelector('.batch-progress');
        if (progressBar) {
            progressBar.classList.add('active');
            
            const fill = progressBar.querySelector<HTMLElement>('.progress-fill');
            const text = progressBar.querySelector('.progress-text');
            
            const percent = Math.round((progress.completed / progress.total) * 100);
            
            if (fill) {
                fill.style.width = `${percent}%`;
            }
            
            if (text) {
                text.textContent = `${progress.completed} / ${progress.total} (${progress.failed} 失败)`;
            }
        }
    }

    /**
     * 隐藏进度条
     */
    hideProgress(): void {
        const progressBar = document.querySelector('.batch-progress');
        if (progressBar) {
            progressBar.classList.remove('active');
        }
    }

    /**
     * 是否处于批量模式
     */
    isBatchMode(): boolean {
        return this.batchMode;
    }
}

/**
 * 渲染批量操作工具栏
 */
export function renderBatchToolbar(): string {
    return `
        <div class="batch-toolbar">
            <div class="batch-selection-info">
                <span class="batch-count">已选择 0 项</span>
                <button type="button" class="btn btn-link btn-sm" onclick="batchManager.selectAll()">
                    全选
                </button>
                <button type="button" class="btn btn-link btn-sm" onclick="batchManager.clearSelection()">
                    清除选择
                </button>
            </div>
            <div class="batch-actions">
                <div class="btn-group">
                    <button type="button" class="btn btn-success btn-sm" onclick="batchManager.performAction('enable')" disabled>
                        <i class="icon">✓</i> 启用
                    </button>
                    <button type="button" class="btn btn-warning btn-sm" onclick="batchManager.performAction('disable')" disabled>
                        <i class="icon">⏸</i> 禁用
                    </button>
                    <button type="button" class="btn btn-danger btn-sm" onclick="batchManager.performAction('delete')" disabled>
                        <i class="icon">🗑</i> 删除
                    </button>
                </div>
                <div class="btn-group">
                    <button type="button" class="btn btn-info btn-sm" onclick="batchManager.performAction('health-check')" disabled>
                        <i class="icon">🔍</i> 快速测活
                    </button>
                    <button type="button" class="btn btn-secondary btn-sm" onclick="batchManager.performAction('export')" disabled>
                        <i class="icon">📥</i> 导出数据
                    </button>
                </div>
            </div>
            <button type="button" class="batch-close" onclick="batchManager.toggleBatchMode()" aria-label="关闭批量模式">
                ×
            </button>
        </div>
        <div class="batch-progress">
            <div class="progress-bar">
                <div class="progress-fill"></div>
            </div>
            <div class="progress-text">0 / 0</div>
        </div>
    `;
}

/**
 * 渲染批量操作切换按钮
 */
export function renderBatchToggleButton(): string {
    return `
        <button 
            id="batch-mode-toggle-btn" 
            class="batch-mode-toggle" 
            title="批量操作"
            aria-label="切换批量操作模式"
            onclick="batchManager.toggleBatchMode()"
        >
            ☑
        </button>
    `;
}

/**
 * 执行批量操作
 */
export async function executeBatchOperation(
    action: CredentialBatchAction | 'export',
    items: string[],
    executor: (action: CredentialBatchAction | 'export', item: string) => Promise<void>,
    onProgress?: (progress: BatchProgress) => void
): Promise<{ success: number; failed: number }> {
    const total = items.length;
    let completed = 0;
    let failed = 0;

    for (const item of items) {
        try {
            await executor(action, item);
            completed++;
        } catch (error) {
            console.error(`Failed to ${action} ${item}:`, error);
            failed++;
        }

        if (onProgress) {
            onProgress({
                total,
                completed: completed + failed,
                failed,
                current: item,
            });
        }
    }

    return { success: completed, failed };
}

/**
 * 获取批量操作的确认消息
 */
export function getBatchConfirmMessage(action: CredentialBatchAction | 'export', count: number): {
    title: string;
    message: string;
} {
    const actionNames: Record<string, string> = {
        enable: '启用',
        disable: '禁用',
        delete: '删除',
        'health-check': '测活',
        export: '导出',
    };

    const actionName = actionNames[action] || action;

    return {
        title: `批量${actionName}`,
        message: `确定要${actionName} ${count} 个凭证吗？`,
    };
}

/**
 * 导出凭证数据
 */
export function exportCredentials(credentials: any[]): void {
    const data = JSON.stringify(credentials, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    
    const a = document.createElement('a');
    a.href = url;
    a.download = `credentials-${Date.now()}.json`;
    a.click();
    
    URL.revokeObjectURL(url);
}

