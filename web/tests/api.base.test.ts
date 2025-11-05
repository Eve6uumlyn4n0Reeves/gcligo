import { describe, it, expect, vi } from 'vitest';

const authMock = {
  apiRequest: vi.fn(),
  apiRequestEnhanced: vi.fn()
};

const uiMock = {
  showNotification: vi.fn(),
  banner: vi.fn(),
  showErrorDetails: vi.fn(),
  t: vi.fn((key: string) => key)
};

vi.mock('../src/auth.js', () => ({ auth: authMock }));
vi.mock('../src/ui.js', () => ({ ui: uiMock }));

describe('API base module exports', () => {
  it('exposes expected utilities and helpers', async () => {
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

    const module = await import('../src/api/base');

    expect(typeof module.apiRequestWithRetry).toBe('function');
    expect(typeof module.BatchRequestManager).toBe('function');
    expect(module.batchManager).toBeInstanceOf(module.BatchRequestManager);
    expect(typeof module.mg).toBe('function');
    expect(typeof module.enhanced).toBe('function');
    expect(typeof module.withQuery).toBe('function');
    expect(typeof module.encodeSegment).toBe('function');
    expect(module.requestCache).toBeInstanceOf(Map);
  });
});
