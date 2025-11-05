/**
 * Dialog Manager Component
 * Manages modal dialogs and popups
 */

export interface DialogOptions {
  width?: string;
  height?: string;
  closable?: boolean;
  backdrop?: boolean;
  onClose?: () => void;
  buttons?: DialogButton[];
  // When true, content passed to open() will be set as HTML; otherwise as plain text.
  allowHTML?: boolean;
}

export interface DialogButton {
  text: string;
  type?: 'primary' | 'secondary' | 'danger';
  onClick?: () => void;
}

export class DialogManager {
  private dialogs: Map<string, HTMLElement> = new Map();
  private legacyDialog: HTMLElement | null = null;

  /**
   * Ensure legacy dialog container exists
   */
  ensureLegacyDialog(): void {
    if (this.legacyDialog) return;

    this.legacyDialog = document.createElement('div');
    this.legacyDialog.className = 'modal-overlay';
    this.legacyDialog.id = 'modal-overlay';
    this.legacyDialog.style.display = 'none';
    this.legacyDialog.innerHTML = `
      <div class="modal-container">
        <div class="modal-header">
          <h3 class="modal-title"></h3>
          <button class="modal-close" aria-label="Close">&times;</button>
        </div>
        <div class="modal-body"></div>
        <div class="modal-footer"></div>
      </div>
    `;
    document.body.appendChild(this.legacyDialog);

    // Close on backdrop click
    this.legacyDialog.addEventListener('click', (e) => {
      if (e.target === this.legacyDialog) {
        this.closeLegacyDialog();
      }
    });

    // Close button handler
    const closeButton = this.legacyDialog.querySelector('.modal-close');
    if (closeButton) {
      closeButton.addEventListener('click', () => this.closeLegacyDialog());
    }
  }

  /**
   * Open a dialog
   */
  open(_id: string, title: string, content: string, options: DialogOptions = {}): void {
    this.ensureLegacyDialog();

    if (!this.legacyDialog) return;

    const titleEl = this.legacyDialog.querySelector('.modal-title');
    const bodyEl = this.legacyDialog.querySelector('.modal-body');
    const footerEl = this.legacyDialog.querySelector('.modal-footer');

    if (titleEl) titleEl.textContent = title;
    if (bodyEl) {
      if (options.allowHTML) bodyEl.innerHTML = content;
      else bodyEl.textContent = content;
    }

    // Add buttons if provided
    if (footerEl && options.buttons) {
      footerEl.innerHTML = '';
      options.buttons.forEach(button => {
        const btn = document.createElement('button');
        btn.className = `btn btn-${button.type || 'secondary'}`;
        btn.textContent = button.text;
        if (button.onClick) {
          btn.addEventListener('click', () => {
            button.onClick!();
            this.closeLegacyDialog();
          });
        }
        footerEl.appendChild(btn);
      });
    }

    this.legacyDialog.style.display = 'flex';
    document.body.style.overflow = 'hidden';
  }

  /**
   * Close legacy dialog
   */
  closeLegacyDialog(): void {
    if (this.legacyDialog) {
      this.legacyDialog.style.display = 'none';
      document.body.style.overflow = '';
    }
  }

  /**
   * Close a specific dialog
   */
  close(id: string): void {
    const dialog = this.dialogs.get(id);
    if (dialog) {
      dialog.remove();
      this.dialogs.delete(id);
    }
  }

  /**
   * Close all dialogs
   */
  closeAll(): void {
    this.dialogs.forEach((dialog) => dialog.remove());
    this.dialogs.clear();
    this.closeLegacyDialog();
  }

  /**
   * Check if any dialog is open
   */
  isOpen(): boolean {
    return this.dialogs.size > 0 || (this.legacyDialog?.style.display === 'flex');
  }

  /**
   * Show legacy dialog
   */
  showLegacy(title: string, contentHtml: string): void {
    this.open('legacy', title, contentHtml);
  }

  /**
   * Hide legacy dialog
   */
  hideLegacy(): void {
    this.closeLegacyDialog();
  }

  /**
   * Backwards-compatible wrappers
   */
  showLegacyDialog(title: string, content: string, options: DialogOptions = {}): void {
    this.open('legacy', title, content, options);
  }

  hideLegacyDialog(): void {
    this.hideLegacy();
  }

  /**
   * Confirm dialog
   */
  confirm(title: string, message: string, options: any = {}): Promise<boolean> {
    return new Promise((resolve) => {
      const buttons: DialogButton[] = [
        {
          text: options.cancelText || '取消',
          type: 'secondary',
          onClick: () => resolve(false)
        },
        {
          text: options.okText || '确定',
          type: options.okClass?.includes('danger') ? 'danger' : 'primary',
          onClick: () => resolve(true)
        }
      ];

      this.open('confirm', title, message, { ...options, buttons });
    });
  }

  /**
   * Get confirm icon
   */
  getConfirmIcon(type: string): string {
    const icons: Record<string, string> = {
      warning: '⚠️',
      danger: '🗑️',
      info: 'ℹ️',
      success: '✓'
    };
    return icons[type] || icons.info;
  }

  /**
   * Confirm delete
   */
  confirmDelete(itemName: string = '此项目'): Promise<boolean> {
    return this.confirm(
      '确认删除',
      `确定要删除 ${itemName} 吗？此操作无法撤销。`,
      {
        type: 'danger',
        okText: '删除',
        okClass: 'btn-danger'
      }
    );
  }
}

