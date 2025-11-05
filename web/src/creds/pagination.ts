import type { PaginatedResult } from './types.js';

export function paginate<T>(
  items: T[],
  page: number,
  pageSize: number
): PaginatedResult<T> {
  const safeSize = pageSize > 0 ? pageSize : 20;
  const total = items.length;
  const pages = Math.max(1, Math.ceil(total / safeSize));
  const safePage = Math.min(Math.max(1, page), pages);
  const start = (safePage - 1) * safeSize;
  const end = Math.min(start + safeSize, total);

  return {
    items: items.slice(start, end),
    total,
    page: safePage,
    pageSize: safeSize,
    pages
  };
}
