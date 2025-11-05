const GLOBAL_ASSET_VERSION: string =
  (typeof window !== 'undefined' && (window as any).__ASSET_VERSION__) || '';

const VERSION_OVERRIDES: Record<string, string> = {
  auth: '20251026',
  api: '20251026',
  ui: '20251026c',
  oauth: '20251026c',
  creds: '20251026c',
  dashboard: '20251026c',
  logs: '20251026',
  registry: '20251026',
  config: '20251026c',
  quickswitcher: '20251026',
  upstream: '20251026',
  layout: '20251026',
  keyboard: '20251026',
  a11y: '20251026',
  metrics: '20251026c',
  assemblyPage: '20251026',
  streaming: '20251026c',
};

export const moduleVersion = (key: string): string =>
  VERSION_OVERRIDES[key] || GLOBAL_ASSET_VERSION;

export const modulePath = (key: string, path: string): string => {
  const version = moduleVersion(key);
  return version ? `${path}?v=${version}` : path;
};

export const setModuleVersionOverride = (key: string, version: string): void => {
  VERSION_OVERRIDES[key] = version;
};

export const moduleVersionOverrides = VERSION_OVERRIDES;
