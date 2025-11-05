/**
 * Notification Center Component
 * Manages toast notifications and alerts
 */

export interface NotificationOptions {
  duration?: number;
  closable?: boolean;
  onClose?: () => void;
  progress?: boolean;
}

export class NotificationCenter {
  private container: HTMLElement | null = null;
  private notifications: Map<string, HTMLElement> = new Map();
  private escapeHTML: (value: string | null | undefined) => string;

  constructor(options: { escapeHTML: (value: string | null | undefined) => string }) {
    this.escapeHTML = options.escapeHTML;
  }

  /**
   * Ensure notification container exists
   */
  ensureContainer(): void {
    if (this.container) return;

    this.container = document.createElement('div');
    this.container.className = 'notification-center';
    this.container.id = 'notification-center';
    document.body.appendChild(this.container);
  }

  /**
   * Show a notification
   */
  show(type: string = 'info', title: string = '', message: string = '', options: NotificationOptions = {}): string {
    this.ensureContainer();

    const id = `notification-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.id = id;

    const icon = this.getIcon(type);
    const titleHtml = title ? `<div class="notification-title">${this.escapeHTML(title)}</div>` : '';
    const messageHtml = message ? `<div class="notification-message">${this.escapeHTML(message)}</div>` : '';
    const closeBtn = options.closable !== false ? `<button class="notification-close" aria-label="Close">&times;</button>` : '';

    notification.innerHTML = `
      <div class="notification-icon">${icon}</div>
      <div class="notification-content">
        ${titleHtml}
        ${messageHtml}
      </div>
      ${closeBtn}
    `;

    if (this.container) {
      this.container.appendChild(notification);
    }
    this.notifications.set(id, notification);

    // Auto-remove after duration
    const duration = options.duration ?? 5000;
    if (duration > 0) {
      setTimeout(() => this.remove(id), duration);
    }

    // Close button handler
    const closeButton = notification.querySelector('.notification-close');
    if (closeButton) {
      closeButton.addEventListener('click', () => {
        this.remove(id);
        if (options.onClose) options.onClose();
      });
    }

    return id;
  }

  /**
   * Show a progress notification
   */
  showProgress(title: string, message: string = '', options: NotificationOptions = {}): string {
    return this.show('info', title, message, { ...options, progress: true, duration: 0 });
  }

  /**
   * Remove a notification
   */
  remove(id: string): void {
    const notification = this.notifications.get(id);
    if (notification) {
      // Remove immediately to avoid timing issues and ensure deterministic tests
      notification.remove();
      this.notifications.delete(id);
    }
  }

  /**
   * Get icon for notification type
   */
  getIcon(type: string): string {
    const icons: Record<string, string> = {
      success: '✓',
      error: '✕',
      warning: '⚠',
      info: 'ℹ'
    };
    return icons[type] || icons.info;
  }

  /**
   * Clear all notifications
   */
  clearAll(): void {
    this.notifications.forEach((_, id) => this.remove(id));
  }

  /**
   * Backwards-compatible alias
   */
  close(id: string): void {
    this.remove(id);
  }
}

