import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

const apiRequestMock = vi.fn();
const apiRequestEnhancedMock = vi.fn();
const showNotification = vi.fn();
const banner = vi.fn();
const showErrorDetails = vi.fn();
const translate = vi.fn((key: string) => key);

vi.mock('../src/auth.js', () => ({
  auth: {
    apiRequest: apiRequestMock,
    apiRequestEnhanced: apiRequestEnhancedMock
  }
}));

vi.mock('../src/ui.js', () => ({
  ui: {
    showNotification,
    banner,
    showErrorDetails,
    t: translate
  }
}));

describe('api/base utilities', () => {
  let base: typeof import('../src/api/base');

  beforeEach(async () => {
    vi.clearAllMocks();
    vi.resetModules();

    Object.defineProperty(navigator, 'onLine', {
      value: true,
      configurable: true
    });

    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: vi.fn().mockResolvedValue(undefined)
      },
      configurable: true
    });

    base = await import('../src/api/base');
    base.requestCache.clear();
  });

  afterEach(() => {
    base.requestCache.clear();
  });

  it('apiRequestWithRetry caches successful GET requests', async () => {
    const requestFn = vi.fn().mockResolvedValue('ok');
    const context = { method: 'GET', cacheKey: 'cache:test' };

    const first = await base.apiRequestWithRetry(requestFn, context);
    const second = await base.apiRequestWithRetry(requestFn, context);

    expect(first).toBe('ok');
    expect(second).toBe('ok');
    expect(requestFn).toHaveBeenCalledTimes(1);
  });

  it('apiRequestWithRetry retries transient errors', async () => {
    vi.useFakeTimers();
    const error = { status: 500, errorInfo: {} };
    const requestFn = vi.fn()
      .mockRejectedValueOnce(error)
      .mockResolvedValueOnce('ok');

    const promise = base.apiRequestWithRetry(requestFn, { method: 'GET' });
    await vi.runAllTimersAsync();
    const result = await promise;

    expect(result).toBe('ok');
    expect(requestFn).toHaveBeenCalledTimes(2);
    vi.useRealTimers();
  });

  it('BatchRequestManager executes queued requests', async () => {
    const manager = new base.BatchRequestManager();
    manager.batchDelay = 0;

    const req1 = vi.fn().mockResolvedValue('a');
    const req2 = vi.fn().mockResolvedValue('b');

    const results = await Promise.all([
      manager.add(req1, {}),
      manager.add(req2, {})
    ]);

    expect(results).toEqual(['a', 'b']);
    expect(req1).toHaveBeenCalledTimes(1);
    expect(req2).toHaveBeenCalledTimes(1);
  });

  it('mg delegates to auth.apiRequest', async () => {
    apiRequestMock.mockResolvedValueOnce({ ok: true });
    const result = await base.mg('/test', { method: 'POST', cache: false });

    expect(result).toEqual({ ok: true });
    expect(apiRequestMock).toHaveBeenCalledWith('/test', { method: 'POST', cache: false });
  });

  it('enhanced delegates to apiRequestEnhanced', async () => {
    apiRequestEnhancedMock.mockResolvedValueOnce({ ok: true });
    const result = await base.enhanced('/enh', { method: 'GET' });

    expect(result).toEqual({ ok: true });
    expect(apiRequestEnhancedMock).toHaveBeenCalled();
  });

  it('withQuery builds query strings and encodeSegment escapes paths', () => {
    expect(base.withQuery('/path', { q: 'value', empty: '' })).toBe('/path?q=value');
    expect(base.encodeSegment('a b/c')).toBe('a%20b%2Fc');
  });
});
