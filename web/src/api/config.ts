import { enhanced } from './base';

export const getConfig = (): Promise<any> => enhanced('config');
export const updateConfig = (payload: any): Promise<any> => enhanced('config', { method: 'PUT', body: JSON.stringify(payload) });
export const reloadConfig = (): Promise<any> => enhanced('config/reload', { method: 'POST' });

export const configApi = {
  getConfig,
  updateConfig,
  reloadConfig
};
