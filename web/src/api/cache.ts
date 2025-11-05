import { requestCache } from './base';

export const cacheApi = {
  clear: () => requestCache.clear(),
  clearPattern: (pattern: string): void => {
    const regex = new RegExp(pattern);
    for (const [key] of requestCache) {
      if (regex.test(key)) {
        requestCache.delete(key);
      }
    }
  },
  size: () => requestCache.size
};
