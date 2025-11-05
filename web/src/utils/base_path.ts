export const DEFAULT_BASE_PATH_MARKERS: string[] = ['/admin', '/assembly', '/routes'];

export interface BasePathOptions {
  markers?: string[];
  fallback?: string;
}

declare global {
  interface Window {
    __BASE_PATH__?: string;
  }
}

export function normalizeBasePath(path: string | null | undefined): string {
  if (!path || path === '/') return '';
  let next = String(path);
  if (!next.startsWith('/')) next = `/${next}`;
  while (next.endsWith('/') && next.length > 1) {
    next = next.slice(0, -1);
  }
  return next === '/' ? '' : next;
}

export function detectBasePath(options: BasePathOptions = {}): string {
  const { markers = DEFAULT_BASE_PATH_MARKERS, fallback = '' } = options;
  if (typeof window === 'undefined') {
    return normalizeBasePath(fallback);
  }
  if (typeof window.__BASE_PATH__ === 'string') {
    return normalizeBasePath(window.__BASE_PATH__);
  }
  return detectBasePathFromLocation(markers, fallback);
}

export function detectBasePathFromLocation(markers: string[] = DEFAULT_BASE_PATH_MARKERS, fallback = ''): string {
  if (typeof window === 'undefined') {
    return normalizeBasePath(fallback);
  }
  const pathname = window.location?.pathname || '';
  if (!pathname || pathname === '/') {
    return normalizeBasePath(fallback);
  }
  for (const marker of markers || []) {
    if (!marker) continue;
    const idx = pathname.indexOf(marker);
    if (idx >= 0) {
      return normalizeBasePath(pathname.slice(0, idx));
    }
  }
  const lastSlash = pathname.lastIndexOf('/');
  if (lastSlash > 0) {
    return normalizeBasePath(pathname.slice(0, lastSlash));
  }
  return '';
}

export function setBasePath(basePath: string): string {
  const normalized = normalizeBasePath(basePath);
  if (typeof window !== 'undefined') {
    window.__BASE_PATH__ = normalized;
  }
  return normalized;
}

export function ensureBasePath(options: BasePathOptions = {}): string {
  const base = detectBasePath(options);
  return setBasePath(base);
}

export function getBasePath(): string {
  if (typeof window === 'undefined') {
    return '';
  }
  if (typeof window.__BASE_PATH__ === 'string') {
    return normalizeBasePath(window.__BASE_PATH__);
  }
  return '';
}

export function joinBasePath(basePath: string, suffix: string | null | undefined): string {
  const base = normalizeBasePath(basePath);
  const tail = suffix || '';
  if (!tail) {
    return base || '';
  }
  if (!base) {
    return tail.startsWith('/') ? tail : `/${tail}`;
  }
  if (tail === '/') {
    return `${base}/`;
  }
  if (tail.startsWith('/')) {
    return `${base}${tail}`;
  }
  return `${base}/${tail}`;
}

export function buildStaticAssetPath(assetPath: string | null | undefined): string {
  const base = getBasePath();
  if (!assetPath) {
    return base;
  }
  if (!base) {
    return assetPath.startsWith('/') ? assetPath : `/${assetPath}`;
  }
  return joinBasePath(base, assetPath);
}
