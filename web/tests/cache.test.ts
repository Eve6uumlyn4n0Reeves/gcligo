import { describe, it, expect, beforeEach, vi } from 'vitest';
import { CacheService } from '../src/services/cache';

describe('CacheService', () => {
  let cache: CacheService;

  beforeEach(() => {
    cache = new CacheService();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('set and get', () => {
    it('should store and retrieve value', () => {
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

    it('should store different types of values', () => {
      cache.set('string', 'text');
      cache.set('number', 123);
      cache.set('boolean', true);
      cache.set('object', { a: 1 });
      cache.set('array', [1, 2, 3]);
      
      expect(cache.get('string')).toBe('text');
      expect(cache.get('number')).toBe(123);
      expect(cache.get('boolean')).toBe(true);
      expect(cache.get('object')).toEqual({ a: 1 });
      expect(cache.get('array')).toEqual([1, 2, 3]);
    });
  });

  describe('TTL (Time To Live)', () => {
    it('should expire value after TTL', () => {
      cache.set('key1', 'value1', 1000); // 1 second TTL
      
      expect(cache.get('key1')).toBe('value1');
      
      vi.advanceTimersByTime(1100);
      
      expect(cache.get('key1')).toBeUndefined();
    });

    it('should not expire value before TTL', () => {
      cache.set('key1', 'value1', 1000);
      
      vi.advanceTimersByTime(500);
      
      expect(cache.get('key1')).toBe('value1');
    });

    it('should handle no TTL (permanent)', () => {
      cache.set('key1', 'value1');
      
      vi.advanceTimersByTime(10000);
      
      expect(cache.get('key1')).toBe('value1');
    });
  });

  describe('has', () => {
    it('should return true for existing key', () => {
      cache.set('key1', 'value1');
      
      expect(cache.has('key1')).toBe(true);
    });

    it('should return false for non-existent key', () => {
      expect(cache.has('non-existent')).toBe(false);
    });

    it('should return false for expired key', () => {
      cache.set('key1', 'value1', 1000);
      
      vi.advanceTimersByTime(1100);
      
      expect(cache.has('key1')).toBe(false);
    });
  });

  describe('delete', () => {
    it('should delete existing key', () => {
      cache.set('key1', 'value1');
      
      cache.delete('key1');
      
      expect(cache.has('key1')).toBe(false);
      expect(cache.get('key1')).toBeUndefined();
    });

    it('should handle deleting non-existent key', () => {
      expect(() => cache.delete('non-existent')).not.toThrow();
    });
  });

  describe('clear', () => {
    it('should clear all entries', () => {
      cache.set('key1', 'value1');
      cache.set('key2', 'value2');
      cache.set('key3', 'value3');
      
      cache.clear();
      
      expect(cache.has('key1')).toBe(false);
      expect(cache.has('key2')).toBe(false);
      expect(cache.has('key3')).toBe(false);
    });

    it('should handle clearing empty cache', () => {
      expect(() => cache.clear()).not.toThrow();
    });
  });

  describe('size', () => {
    it('should return correct size', () => {
      expect(cache.size()).toBe(0);
      
      cache.set('key1', 'value1');
      expect(cache.size()).toBe(1);
      
      cache.set('key2', 'value2');
      expect(cache.size()).toBe(2);
      
      cache.delete('key1');
      expect(cache.size()).toBe(1);
      
      cache.clear();
      expect(cache.size()).toBe(0);
    });

    it('should not count expired entries', () => {
      cache.set('key1', 'value1', 1000);
      cache.set('key2', 'value2');
      
      expect(cache.size()).toBe(2);
      
      vi.advanceTimersByTime(1100);
      
      expect(cache.size()).toBe(1);
    });
  });

  describe('keys', () => {
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

    it('should not include expired keys', () => {
      cache.set('key1', 'value1', 1000);
      cache.set('key2', 'value2');
      
      vi.advanceTimersByTime(1100);
      
      const keys = cache.keys();
      expect(keys).not.toContain('key1');
      expect(keys).toContain('key2');
    });
  });

  describe('getOrSet', () => {
    it('should return existing value', () => {
      cache.set('key1', 'existing');
      
      const result = cache.getOrSet('key1', () => 'new');
      expect(result).toBe('existing');
    });

    it('should set and return new value if not exists', () => {
      const result = cache.getOrSet('key1', () => 'new');
      expect(result).toBe('new');
      expect(cache.get('key1')).toBe('new');
    });

    it('should call factory function only if needed', () => {
      const factory = vi.fn(() => 'value');
      
      cache.set('key1', 'existing');
      cache.getOrSet('key1', factory);
      
      expect(factory).not.toHaveBeenCalled();
      
      cache.getOrSet('key2', factory);
      expect(factory).toHaveBeenCalledTimes(1);
    });

    it('should handle async factory function', async () => {
      const factory = async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
        return 'async value';
      };

      const pending = cache.getOrSet('key1', factory);
      await vi.advanceTimersByTimeAsync(100);
      const result = await pending;
      expect(result).toBe('async value');
    });
  });

  describe('Memory management', () => {
    it('should limit cache size', () => {
      const limitedCache = new CacheService(3); // Max 3 entries
      
      limitedCache.set('key1', 'value1');
      limitedCache.set('key2', 'value2');
      limitedCache.set('key3', 'value3');
      limitedCache.set('key4', 'value4'); // Should evict oldest
      
      expect(limitedCache.size()).toBe(3);
      expect(limitedCache.has('key1')).toBe(false); // Oldest evicted
      expect(limitedCache.has('key4')).toBe(true);
    });

    it('should use LRU eviction policy', () => {
      const limitedCache = new CacheService(3);
      
      limitedCache.set('key1', 'value1');
      limitedCache.set('key2', 'value2');
      limitedCache.set('key3', 'value3');
      
      // Access key1 to make it recently used
      limitedCache.get('key1');
      
      // Add new key, should evict key2 (least recently used)
      limitedCache.set('key4', 'value4');
      
      expect(limitedCache.has('key1')).toBe(true);
      expect(limitedCache.has('key2')).toBe(false);
      expect(limitedCache.has('key3')).toBe(true);
      expect(limitedCache.has('key4')).toBe(true);
    });
  });

  describe('Statistics', () => {
    it('should track hit/miss ratio', () => {
      cache.set('key1', 'value1');
      
      cache.get('key1'); // hit
      cache.get('key2'); // miss
      cache.get('key1'); // hit
      cache.get('key3'); // miss
      
      const stats = cache.getStats();
      expect(stats.hits).toBe(2);
      expect(stats.misses).toBe(2);
      expect(stats.hitRate).toBe(0.5);
    });
  });
});
