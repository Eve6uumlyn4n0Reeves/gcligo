import { describe, it, expect } from 'vitest';
import { normalizeBasePath, joinBasePath } from '../src/utils/base_path';

describe('base_path utils', () => {
  it('normalizes base path', () => {
    expect(normalizeBasePath('')).toBe('');
    expect(normalizeBasePath('/')).toBe('');
    expect(normalizeBasePath('api')).toBe('/api');
    expect(normalizeBasePath('/api/')).toBe('/api');
  });
  it('joins base and suffix', () => {
    expect(joinBasePath('', '/admin')).toBe('/admin');
    expect(joinBasePath('/root', 'admin')).toBe('/root/admin');
    expect(joinBasePath('/root', '/admin')).toBe('/root/admin');
    expect(joinBasePath('/root', '')).toBe('/root');
  });
});
