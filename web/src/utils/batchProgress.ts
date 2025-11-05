import { credentialsApi } from '../api/credentials';
import type { BatchTaskProgressEvent } from '../../types/api';

export type BatchTaskStreamHandlers = {
  onProgress?: (data: BatchTaskProgressEvent) => void;
  onDone?: (data: BatchTaskProgressEvent) => void;
  onError?: (event: MessageEvent) => void;
};

export const subscribeBatchTaskStream = (taskId: string, handlers: BatchTaskStreamHandlers = {}): EventSource => {
  const source = new EventSource(credentialsApi.getBatchTaskStreamURL(taskId));
  source.addEventListener('progress', (event: MessageEvent) => {
    if (!handlers.onProgress) return;
    try {
      handlers.onProgress(JSON.parse(event.data));
    } catch (_) {
      handlers.onProgress({
        status: 'running',
        progress: 0,
        completed: 0,
        success: 0,
        failure: 0
      });
    }
  });
  source.addEventListener('done', (event: MessageEvent) => {
    if (!handlers.onDone) return;
    try {
      handlers.onDone(JSON.parse(event.data));
    } catch (_) {
      handlers.onDone({
        status: 'completed',
        progress: 100,
        completed: 0,
        success: 0,
        failure: 0
      });
    }
    source.close();
  });
  source.addEventListener('error', (event: MessageEvent) => {
    handlers.onError?.(event);
  });
  return source;
};

export const closeBatchTaskStream = (source?: EventSource | null): void => {
  source?.close();
};
