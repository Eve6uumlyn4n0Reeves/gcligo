import { describe, it, expect, beforeEach, vi } from 'vitest';
import { Router, createRouter } from '../src/admin/router';

describe('Router', () => {
  let router: Router;

  beforeEach(() => {
    // Mock window.location
    delete (window as any).location;
    window.location = { hash: '' } as any;

    router = createRouter({
      tabs: ['dashboard', 'credentials', 'oauth', 'stats'],
      defaultTab: 'dashboard',
    });
  });

  describe('getTabFromHash', () => {
    it('should return null for empty hash', () => {
      window.location.hash = '';
      expect(router.getTabFromHash()).toBeNull();
    });

    it('should return tab name from hash', () => {
      window.location.hash = '#credentials';
      expect(router.getTabFromHash()).toBe('credentials');
    });

    it('should return null for invalid tab', () => {
      window.location.hash = '#invalid';
      expect(router.getTabFromHash()).toBeNull();
    });

    it('should handle hash with path', () => {
      window.location.hash = '#credentials/detail/123';
      expect(router.getTabFromHash()).toBe('credentials');
    });
  });

  describe('setHashForTab', () => {
    it('should set hash for valid tab', () => {
      router.setHashForTab('credentials');
      expect(window.location.hash).toBe('#credentials');
    });

    it('should not set hash for invalid tab', () => {
      window.location.hash = '#dashboard';
      router.setHashForTab('invalid' as any);
      expect(window.location.hash).toBe('#dashboard');
    });
  });

  describe('switchTo', () => {
    it('should switch to valid tab', () => {
      const onTabChange = vi.fn();
      router = createRouter({
        tabs: ['dashboard', 'credentials'],
        defaultTab: 'dashboard',
        onTabChange,
      });

      router.switchTo('credentials');
      expect(router.getCurrentTab()).toBe('credentials');
      expect(onTabChange).toHaveBeenCalledWith('credentials');
    });

    it('should not switch to invalid tab', () => {
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      router.switchTo('invalid' as any);
      expect(router.getCurrentTab()).toBe('dashboard');
      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it('should not trigger callback if already on tab', () => {
      const onTabChange = vi.fn();
      router = createRouter({
        tabs: ['dashboard', 'credentials'],
        defaultTab: 'dashboard',
        onTabChange,
      });

      router.switchTo('dashboard');
      expect(onTabChange).not.toHaveBeenCalled();
    });
  });

  describe('navigate', () => {
    beforeEach(() => {
      router = createRouter({
        tabs: ['dashboard', 'credentials', 'oauth', 'stats'],
        defaultTab: 'dashboard',
      });
    });

    it('should navigate to next tab', () => {
      router.next();
      expect(router.getCurrentTab()).toBe('credentials');
    });

    it('should navigate to previous tab', () => {
      router.switchTo('credentials');
      router.prev();
      expect(router.getCurrentTab()).toBe('dashboard');
    });

    it('should wrap around when navigating forward from last tab', () => {
      router.switchTo('stats');
      router.next();
      expect(router.getCurrentTab()).toBe('dashboard');
    });

    it('should wrap around when navigating backward from first tab', () => {
      router.prev();
      expect(router.getCurrentTab()).toBe('stats');
    });
  });

  describe('getTabs', () => {
    it('should return copy of tabs array', () => {
      const tabs = router.getTabs();
      expect(tabs).toEqual(['dashboard', 'credentials', 'oauth', 'stats']);
      
      // Verify it's a copy
      tabs.push('new-tab' as any);
      expect(router.getTabs()).toEqual(['dashboard', 'credentials', 'oauth', 'stats']);
    });
  });

  describe('getVisibleRange', () => {
    it('should return current tab', () => {
      expect(router.getCurrentTab()).toBe('dashboard');
      router.switchTo('credentials');
      expect(router.getCurrentTab()).toBe('credentials');
    });
  });
});
