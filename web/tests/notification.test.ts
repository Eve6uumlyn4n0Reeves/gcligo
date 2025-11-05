import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { NotificationCenter } from '../src/components/notification';

describe('NotificationCenter', () => {
  let center: NotificationCenter;

  beforeEach(() => {
    document.body.innerHTML = '';
    vi.useFakeTimers();
    center = new NotificationCenter({
      escapeHTML: (value) => (value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('creates container on demand', () => {
    center.ensureContainer();
    const container = document.querySelector('.notification-center');
    expect(container).toBeTruthy();
  });

  it('shows notification with provided type and text', () => {
    center.show('success', 'Done', 'Operation complete');

    const notification = document.querySelector('.notification');
    expect(notification).toBeTruthy();
    expect(notification?.classList.contains('notification-success')).toBe(true);
    expect(notification?.textContent).toContain('Operation complete');
  });

  it('auto dismisses notification after duration', () => {
    center.show('info', 'Heads up', 'Auto dismiss', { duration: 1000 });
    expect(document.querySelectorAll('.notification').length).toBe(1);

    vi.advanceTimersByTime(1000);
    expect(document.querySelectorAll('.notification').length).toBe(0);
  });

  it('allows manual dismissal via close button', () => {
    center.show('info', 'Close me', 'Manual dismiss');

    const closeButton = document.querySelector('.notification-close') as HTMLElement;
    expect(closeButton).toBeTruthy();
    closeButton.click();

    expect(document.querySelectorAll('.notification').length).toBe(0);
  });

  it('supports progress notifications without auto-dismiss', () => {
    center.showProgress('Uploading', 'Hang tight');
    vi.advanceTimersByTime(10_000);

    expect(document.querySelectorAll('.notification').length).toBe(1);
  });

  it('removes notifications explicitly', () => {
    const id = center.show('warning', 'Careful', 'Check input');
    expect(document.querySelectorAll('.notification').length).toBe(1);

    center.remove(id);
    expect(document.querySelectorAll('.notification').length).toBe(0);
  });

  it('clears all notifications', () => {
    center.show('info', 'One', 'First');
    center.show('info', 'Two', 'Second');

    center.clearAll();
    expect(document.querySelectorAll('.notification').length).toBe(0);
  });
});
