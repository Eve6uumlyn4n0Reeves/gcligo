type NotificationType = 'success' | 'error' | 'warning' | 'info';

interface ShowNotificationOptions {
  dismissible?: boolean;
}

let notificationContainer: HTMLElement | null = null;
let notificationId = 0;

function escapeHTML(value: string | null | undefined): string {
  return String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function ensureContainer(): HTMLElement {
  if (!notificationContainer) {
    notificationContainer = document.createElement('div');
    notificationContainer.id = 'notification-container';
    notificationContainer.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      z-index: 10000;
      max-width: 400px;
    `;
    document.body.appendChild(notificationContainer);
  }
  return notificationContainer;
}

export function showNotification(
  message: string,
  type: NotificationType = 'info',
  duration = 5000,
  options: ShowNotificationOptions = {}
): number {
  const container = ensureContainer();
  const id = ++notificationId;

  const notification = document.createElement('div');
  notification.id = `notification-${id}`;
  notification.style.cssText = `
    margin-bottom: 10px;
    padding: 12px 16px;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 14px;
    line-height: 1.4;
    animation: slideIn 0.3s ease-out;
    max-width: 100%;
    word-wrap: break-word;
  `;

  const styles: Record<
    NotificationType,
    { bg: string; color: string; border: string; icon: string }
  > = {
    success: { bg: '#f0fdf4', color: '#15803d', border: '#bbf7d0', icon: '✅' },
    error: { bg: '#fef2f2', color: '#dc2626', border: '#fecaca', icon: '❌' },
    warning: { bg: '#fffbeb', color: '#d97706', border: '#fed7aa', icon: '⚠️' },
    info: { bg: '#eff6ff', color: '#2563eb', border: '#bfdbfe', icon: 'ℹ️' }
  };

  const style = styles[type] ?? styles.info;
  notification.style.backgroundColor = style.bg;
  notification.style.color = style.color;
  notification.style.border = `1px solid ${style.border}`;

  const dismissible = options.dismissible !== false;

  notification.innerHTML = `
    <div style="display: flex; align-items: center; gap: 8px; flex: 1;">
      <span style="font-size: 16px;">${style.icon}</span>
      <span>${escapeHTML(message)}</span>
    </div>
    ${
      dismissible
        ? `<button style="
              background: none;
              border: none;
              color: ${style.color};
              cursor: pointer;
              font-size: 18px;
              padding: 0 4px;
              margin-left: 12px;
              opacity: 0.7;
            " data-close="true" title="关闭">×</button>`
        : ''
    }
  `;

  if (dismissible) {
    notification
      .querySelector('[data-close]')
      ?.addEventListener('click', () => removeNotification(id));
  }

  container.appendChild(notification);

  if (duration > 0) {
    setTimeout(() => removeNotification(id), duration);
  }

  injectStyles();
  return id;
}

export function removeNotification(id: number): void {
  const notification = document.getElementById(`notification-${id}`);
  if (!notification) return;
  notification.style.animation = 'slideOut 0.3s ease-in';
  setTimeout(() => notification.remove(), 300);
}

export function clearAllNotifications(): void {
  if (notificationContainer) {
    notificationContainer.innerHTML = '';
  }
}

export const notify: Record<NotificationType, (message: string, duration?: number) => number> = {
  success: (message, duration) => showNotification(message, 'success', duration),
  error: (message, duration) => showNotification(message, 'error', duration),
  warning: (message, duration) => showNotification(message, 'warning', duration),
  info: (message, duration) => showNotification(message, 'info', duration)
};

function injectStyles(): void {
  if (document.getElementById('notification-styles')) return;
  const style = document.createElement('style');
  style.id = 'notification-styles';
  style.textContent = `
    @keyframes slideIn {
      from { transform: translateX(100%); opacity: 0; }
      to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
      from { transform: translateX(0); opacity: 1; }
      to { transform: translateX(100%); opacity: 0; }
    }
    @media (max-width: 768px) {
      #notification-container {
        left: 10px;
        right: 10px;
        top: 10px;
        max-width: none;
      }
    }
  `;
  document.head.appendChild(style);
}
