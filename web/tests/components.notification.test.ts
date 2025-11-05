import { describe, it, expect, beforeEach, vi } from 'vitest';
import { NotificationCenter } from '../src/components/notification';

describe('NotificationCenter', () => {
  let center: NotificationCenter;
  
  const mockEscapeHTML = (value: string | null | undefined): string => {
    if (!value) return '';
    return String(value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  };

  beforeEach(() => {
    document.body.innerHTML = '';
    center = new NotificationCenter({ escapeHTML: mockEscapeHTML });
  });

  it('should create NotificationCenter instance', () => {
    expect(center).toBeInstanceOf(NotificationCenter);
  });

  it('should ensure container exists', () => {
    center.ensureContainer();
    
    const container = document.getElementById('notification-center');
    expect(container).toBeTruthy();
    expect(container?.className).toBe('notification-center');
  });

  it('should not create duplicate containers', () => {
    center.ensureContainer();
    center.ensureContainer();
    
    const containers = document.querySelectorAll('#notification-center');
    expect(containers.length).toBe(1);
  });

  it('should show notification', () => {
    const id = center.show('info', 'Test Title', 'Test Message');
    
    expect(id).toBeTruthy();
    expect(id).toMatch(/^notification-/);
    
    const notification = document.getElementById(id);
    expect(notification).toBeTruthy();
    expect(notification?.className).toContain('notification-info');
  });

  it('should show different notification types', () => {
    const infoId = center.show('info', 'Info', 'Info message');
    const successId = center.show('success', 'Success', 'Success message');
    const warningId = center.show('warning', 'Warning', 'Warning message');
    const errorId = center.show('error', 'Error', 'Error message');
    
    expect(document.getElementById(infoId)?.className).toContain('notification-info');
    expect(document.getElementById(successId)?.className).toContain('notification-success');
    expect(document.getElementById(warningId)?.className).toContain('notification-warning');
    expect(document.getElementById(errorId)?.className).toContain('notification-error');
  });

  it('should escape HTML in messages', () => {
    const id = center.show('info', '<script>alert("xss")</script>', '<b>Bold</b>');
    
    const notification = document.getElementById(id);
    const html = notification?.innerHTML || '';
    
    expect(html).toContain('&lt;script&gt;');
    expect(html).toContain('&lt;b&gt;');
    expect(html).not.toContain('<script>');
  });

  it('should close notification', () => {
    const id = center.show('info', 'Test', 'Message');
    
    expect(document.getElementById(id)).toBeTruthy();
    
    center.close(id);
    
    // Wait for animation
    setTimeout(() => {
      expect(document.getElementById(id)).toBeFalsy();
    }, 100);
  });

  it('should auto-close notification with duration', () => {
    vi.useFakeTimers();
    
    const id = center.show('info', 'Test', 'Message', { duration: 1000 });
    
    expect(document.getElementById(id)).toBeTruthy();
    
    vi.advanceTimersByTime(1100);
    
    expect(document.getElementById(id)).toBeFalsy();
    
    vi.restoreAllMocks();
  });
});

