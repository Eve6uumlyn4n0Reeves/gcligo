import { enhanced, mg } from './base';

export const statsApi = {
  getStats: () => mg('stats'),
  resetStats: () => mg('stats/reset', { method: 'POST' }),
  getEnhancedMetrics: () => enhanced('metrics'),
  getStreamingMetrics: () => enhanced('metrics/streaming'),
  getUsage: () => enhanced('usage')
};
