import { enhanced, withQuery, encodeSegment } from './base';

export const getRegistry = (channel: string = 'openai'): Promise<any> => enhanced(`models/${encodeSegment(channel)}/registry`);
export const putRegistry = (channel: string, models: any[]): Promise<any> => enhanced(`models/${encodeSegment(channel)}/registry`, {
  method: 'PUT',
  body: JSON.stringify({ models })
});
export const postRegistry = (channel: string, payload: any): Promise<any> => enhanced(`models/${encodeSegment(channel)}/registry`, {
  method: 'POST',
  body: JSON.stringify(payload)
});
export const deleteRegistryModel = (channel: string, id: string): Promise<any> => enhanced(`models/${encodeSegment(channel)}/registry/${encodeSegment(id)}`, {
  method: 'DELETE'
});
export const seedDefaults = (channel: string): Promise<any> => enhanced(`models/${encodeSegment(channel)}/registry/seed-defaults`, { method: 'POST' });
export const getTemplate = (channel: string): Promise<any> => enhanced(`models/${encodeSegment(channel)}/template`);
export const updateTemplate = (channel: string, payload: any): Promise<any> => enhanced(`models/${encodeSegment(channel)}/template`, {
  method: 'PUT',
  body: JSON.stringify(payload)
});
export const getGroups = (channel: string): Promise<any> => enhanced(`models/${encodeSegment(channel)}/groups`);
export const createGroup = (channel: string, name: string): Promise<any> => enhanced(`models/${encodeSegment(channel)}/groups`, {
  method: 'POST',
  body: JSON.stringify({ name, enabled: true, order: 0 })
});
export const bulkEnable = (channel: string, group: string): Promise<any> => enhanced(withQuery(`models/${encodeSegment(channel)}/registry/bulk-enable`, { group }), {
  method: 'POST'
});
export const bulkDisable = (channel: string, group: string): Promise<any> => enhanced(withQuery(`models/${encodeSegment(channel)}/registry/bulk-disable`, { group }), {
  method: 'POST'
});
export const importRegistryJSON = (channel: string, models: any[], mode: string = 'append'): Promise<any> => enhanced(withQuery(`models/${encodeSegment(channel)}/registry/import`, { mode }), {
  method: 'POST',
  body: JSON.stringify({ models })
});
export const importRegistryFiles = (channel: string, formData: FormData, mode: string = 'append'): Promise<any> => enhanced(withQuery(`models/${encodeSegment(channel)}/registry/import`, { mode }), {
  method: 'POST',
  body: formData
});
export const exportRegistry = (channel: string): Promise<any> => enhanced(`models/${encodeSegment(channel)}/registry/export`);
export const getUpstreamSuggestions = (): Promise<any> => enhanced('models/upstream-suggest');
export const refreshUpstreamModels = (force: boolean = false, timeout: number = 30): Promise<any> => enhanced('models/upstream-refresh', {
  method: 'POST',
  body: JSON.stringify({ force, timeout }),
  cache: false
});

export const registryApi = {
  getRegistry,
  putRegistry,
  postRegistry,
  deleteRegistryModel,
  seedDefaults,
  getTemplate,
  updateTemplate,
  getGroups,
  createGroup,
  bulkEnable,
  bulkDisable,
  importRegistryJSON,
  importRegistryFiles,
  exportRegistry,
  getUpstreamSuggestions,
  refreshUpstreamModels
};
