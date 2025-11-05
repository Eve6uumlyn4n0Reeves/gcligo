/**
 * Cache Manager Service
 * Provides caching functionality for API responses and data
 */

export interface CacheOptions {
  ttl?: number; // Time to live in milliseconds
  maxSize?: number; // Maximum number of entries
}

export interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number;
}

export interface CacheStats {
  hits: number;
  misses: number;
  hitRate: number;
}

export class CacheManager<T = any> {
  private cache: Map<string, CacheEntry<T>> = new Map();
  private options: CacheOptions;
  private hits = 0;
  private misses = 0;

  constructor(options: CacheOptions = {}) {
    this.options = {
      ttl: options.ttl || 5 * 60 * 1000, // Default 5 minutes
      maxSize: options.maxSize || 100
    };
  }

  private resolveTTL(ttl?: number): number {
    return ttl ?? this.options.ttl ?? 5 * 60 * 1000;
  }

  private readEntry(key: string): CacheEntry<T> | undefined {
    const entry = this.cache.get(key);
    if (!entry) return undefined;

    if (Date.now() - entry.timestamp > entry.ttl) {
      this.cache.delete(key);
      return undefined;
    }
    return entry;
  }

  private cleanupExpiredEntries(): void {
    const now = Date.now();
    for (const [key, entry] of this.cache.entries()) {
      if (now - entry.timestamp > entry.ttl) {
        this.cache.delete(key);
      }
    }
  }

  private touch(key: string, entry: CacheEntry<T>): void {
    this.cache.delete(key);
    this.cache.set(key, entry);
  }

  /**
   * Get value from cache
   */
  get(key: string): T | undefined {
    const entry = this.readEntry(key);
    if (!entry) {
      this.misses++;
      return undefined;
    }
    this.hits++;
    this.touch(key, entry);
    return entry.data;
  }

  /**
   * Set value in cache
   */
  set(key: string, data: T, ttl?: number): void {
    // Enforce max size
    if (this.cache.size >= (this.options.maxSize || 100)) {
      // Remove oldest entry
      const firstKey = this.cache.keys().next().value;
      if (firstKey) this.cache.delete(firstKey);
    }

    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      ttl: this.resolveTTL(ttl)
    });
  }

  /**
   * Check if key exists and is not expired
   */
  has(key: string): boolean {
    const entry = this.readEntry(key);
    if (!entry) {
      return false;
    }
    this.touch(key, entry);
    return true;
  }

  /**
   * Delete a key from cache
   */
  delete(key: string): void {
    this.cache.delete(key);
  }

  /**
   * Clear all cache
   */
  clear(): void {
    this.cache.clear();
  }

  /**
   * Get cache size
   */
  size(): number {
    this.cleanupExpiredEntries();
    return this.cache.size;
  }

  /**
   * Get all keys
   */
  keys(): string[] {
    this.cleanupExpiredEntries();
    return Array.from(this.cache.keys());
  }

  /**
   * Get or set cache entry with factory
   */
  getOrSet(key: string, factory: () => T | Promise<T>, ttl?: number): T | Promise<T> {
    const existing = this.get(key);
    if (existing !== undefined) {
      return existing;
    }
    const value = factory();
    if (value && typeof (value as Promise<T>).then === 'function') {
      return (value as Promise<T>).then((resolved) => {
        this.set(key, resolved, ttl);
        return resolved;
      });
    }
    this.set(key, value as T, ttl);
    return value as T;
  }

  /**
   * Cache hit/miss statistics
   */
  getStats(): CacheStats {
    const total = this.hits + this.misses;
    return {
      hits: this.hits,
      misses: this.misses,
      hitRate: total === 0 ? 0 : this.hits / total
    };
  }
}

/**
 * Create a cache manager instance
 */
export function createCacheManager<T = any>(options: CacheOptions = {}): CacheManager<T> {
  return new CacheManager<T>(options);
}

/**
 * Legacy-friendly cache service wrapper
 */
export class CacheService<T = any> extends CacheManager<T> {
  constructor(maxSizeOrOptions: number | CacheOptions = {}) {
    if (typeof maxSizeOrOptions === 'number') {
      super({ maxSize: maxSizeOrOptions });
    } else {
      super(maxSizeOrOptions);
    }
  }
}

/**
 * Refresh Manager Service
 * Manages automatic refresh of data
 */

export interface RefreshManagerOptions {
  eventBus?: any;
  cacheFactory?: (opts: CacheOptions) => CacheManager;
  throttleFn?: (func: Function, delay: number) => Function;
}

export class RefreshManager {
  private intervals: Map<string, number> = new Map();

  constructor(_options: RefreshManagerOptions = {}) {
  }

  /**
   * Register a refresh handler
   */
  register(key: string, handler: () => void | Promise<void>, interval: number): void {
    // Clear existing interval if any
    this.unregister(key);

    // Set new interval
    const intervalId = window.setInterval(handler, interval);
    this.intervals.set(key, intervalId);
  }

  /**
   * Unregister a refresh handler
   */
  unregister(key: string): void {
    const intervalId = this.intervals.get(key);
    if (intervalId !== undefined) {
      clearInterval(intervalId);
      this.intervals.delete(key);
    }
  }

  /**
   * Unregister all handlers
   */
  unregisterAll(): void {
    this.intervals.forEach((intervalId) => clearInterval(intervalId));
    this.intervals.clear();
  }

  /**
   * Check if a key is registered
   */
  isRegistered(key: string): boolean {
    return this.intervals.has(key);
  }

  /**
   * Get number of registered handlers
   */
  count(): number {
    return this.intervals.size;
  }
}

/**
 * Create a refresh manager instance
 */
export function createRefreshManager(options: RefreshManagerOptions = {}): RefreshManager {
  return new RefreshManager(options);
}
