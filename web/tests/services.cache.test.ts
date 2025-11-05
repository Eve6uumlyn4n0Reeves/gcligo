import { describe, it, expect, beforeEach, vi } from 'vitest';
import { CacheManager } from '../src/services/cache';

describe('CacheManager', () => {
  let cache: CacheManager<string>;

  beforeEach(() => {
    cache = new CacheManager<string>({ ttl: 1000, maxSize: 5 });
  });

  it('should create CacheManager instance', () => {
    expect(cache).toBeInstanceOf(CacheManager);
  });

  it('should set and get value', () => {
    cache.set('key1', 'value1');
    
    const result = cache.get('key1');
    expect(result).toBe('value1');
  });

  it('should return undefined for non-existent key', () => {
    const result = cache.get('non-existent');
    expect(result).toBeUndefined();
  });

  it('should overwrite existing value', () => {
    cache.set('key1', 'value1');
    cache.set('key1', 'value2');
    
    const result = cache.get('key1');
    expect(result).toBe('value2');
  });

  it('should expire value after TTL', () => {
    vi.useFakeTimers();
    
    cache.set('key1', 'value1', 1000);
    
    expect(cache.get('key1')).toBe('value1');
    
    vi.advanceTimersByTime(1100);
    
    expect(cache.get('key1')).toBeUndefined();
    
    vi.restoreAllMocks();
  });

  it('should check if key exists', () => {
    cache.set('key1', 'value1');
    
    expect(cache.has('key1')).toBe(true);
    expect(cache.has('non-existent')).toBe(false);
  });

  it('should delete key', () => {
    cache.set('key1', 'value1');
    
    expect(cache.has('key1')).toBe(true);
    
    cache.delete('key1');
    
    expect(cache.has('key1')).toBe(false);
    expect(cache.get('key1')).toBeUndefined();
  });

  it('should clear all entries', () => {
    cache.set('key1', 'value1');
    cache.set('key2', 'value2');
    cache.set('key3', 'value3');
    
    expect(cache.size()).toBe(3);
    
    cache.clear();
    
    expect(cache.size()).toBe(0);
    expect(cache.has('key1')).toBe(false);
    expect(cache.has('key2')).toBe(false);
    expect(cache.has('key3')).toBe(false);
  });

  it('should return correct size', () => {
    expect(cache.size()).toBe(0);
    
    cache.set('key1', 'value1');
    expect(cache.size()).toBe(1);
    
    cache.set('key2', 'value2');
    expect(cache.size()).toBe(2);
    
    cache.delete('key1');
    expect(cache.size()).toBe(1);
  });

  it('should return all keys', () => {
    cache.set('key1', 'value1');
    cache.set('key2', 'value2');
    cache.set('key3', 'value3');
    
    const keys = cache.keys();
    expect(keys).toContain('key1');
    expect(keys).toContain('key2');
    expect(keys).toContain('key3');
    expect(keys.length).toBe(3);
  });

  it('should enforce max size', () => {
    const smallCache = new CacheManager<string>({ maxSize: 3 });
    
    smallCache.set('key1', 'value1');
    smallCache.set('key2', 'value2');
    smallCache.set('key3', 'value3');
    smallCache.set('key4', 'value4'); // Should evict oldest
    
    expect(smallCache.size()).toBe(3);
  });

  it('should handle different data types', () => {
    const anyCache = new CacheManager<any>();
    
    anyCache.set('string', 'text');
    anyCache.set('number', 123);
    anyCache.set('boolean', true);
    anyCache.set('object', { a: 1 });
    anyCache.set('array', [1, 2, 3]);
    
    expect(anyCache.get('string')).toBe('text');
    expect(anyCache.get('number')).toBe(123);
    expect(anyCache.get('boolean')).toBe(true);
    expect(anyCache.get('object')).toEqual({ a: 1 });
    expect(anyCache.get('array')).toEqual([1, 2, 3]);
  });
});
