import { enhanced, mg, encodeSegment } from './base';
import type {
  BatchCredentialRequest,
  BatchOperationResponse,
  BatchTaskDetail,
  BatchTaskListResponse
} from '../../types/api';

export const listCredentials = (): Promise<any> => mg('credentials');
export const enableCredential = (id: string): Promise<any> => mg(`credentials/${encodeSegment(id)}/enable`, { method: 'POST' });
export const disableCredential = (id: string): Promise<any> => mg(`credentials/${encodeSegment(id)}/disable`, { method: 'POST' });
export const deleteCredential = (id: string): Promise<any> => mg(`credentials/${encodeSegment(id)}`, { method: 'DELETE' });
export const recoverCredential = (id: string): Promise<any> => mg(`credentials/${encodeSegment(id)}/recover`, { method: 'POST' });
export const recoverAllCredentials = (): Promise<any> => mg('credentials/recover-all', { method: 'POST' });
export const reloadCredentials = (): Promise<any> => mg('credentials/reload', { method: 'POST' });
export const uploadCredential = (payload: any): Promise<any> => mg('credentials', { method: 'POST', body: JSON.stringify(payload) });
export const uploadCredentialFiles = (formData: FormData): Promise<any> => mg('credentials/upload', { method: 'POST', body: formData });
export const probeFlash = (model: string = 'gemini-2.5-flash', timeout: number = 10): Promise<any> => enhanced('credentials/probe', {
  method: 'POST',
  body: JSON.stringify({ model, timeout_sec: timeout })
});

// Batch operations
const buildBatchPayload = (ids: string[], concurrency?: number): BatchCredentialRequest => {
  if (concurrency !== undefined && concurrency !== null) {
    return { ids, concurrency };
  }
  return { ids };
};

export const batchEnableCredentials = (ids: string[], concurrency?: number): Promise<BatchOperationResponse> =>
  mg('credentials/batch-enable', {
    method: 'POST',
    body: JSON.stringify(buildBatchPayload(ids, concurrency))
  });

export const batchDisableCredentials = (ids: string[], concurrency?: number): Promise<BatchOperationResponse> =>
  mg('credentials/batch-disable', {
    method: 'POST',
    body: JSON.stringify(buildBatchPayload(ids, concurrency))
  });

export const batchDeleteCredentials = (ids: string[], concurrency?: number): Promise<BatchOperationResponse> =>
  mg('credentials/batch-delete', {
    method: 'POST',
    body: JSON.stringify(buildBatchPayload(ids, concurrency))
  });

export const batchRecoverCredentials = (ids: string[], concurrency?: number): Promise<BatchOperationResponse> =>
  mg('credentials/batch-recover', {
    method: 'POST',
    body: JSON.stringify(buildBatchPayload(ids, concurrency))
  });

export const listBatchTasks = (): Promise<BatchTaskListResponse> =>
  mg('credentials/batch-tasks');

export const getBatchTask = (taskId: string): Promise<BatchTaskDetail> =>
  mg(`credentials/batch-tasks/${encodeSegment(taskId)}`);

export const getBatchTaskResult = (taskId: string): Promise<BatchTaskDetail> =>
  mg(`credentials/batch-tasks/${encodeSegment(taskId)}/results`);

export const cancelBatchTask = (taskId: string): Promise<any> =>
  mg(`credentials/batch-tasks/${encodeSegment(taskId)}`, { method: 'DELETE' });

export const getBatchTaskStreamURL = (taskId: string): string =>
  `/routes/api/management/credentials/batch-tasks/${encodeSegment(taskId)}/stream`;

export const credentialsApi = {
  listCredentials,
  enableCredential,
  disableCredential,
  deleteCredential,
  recoverCredential,
  recoverAllCredentials,
  reloadCredentials,
  uploadCredential,
  uploadCredentialFiles,
  probeFlash,
  // Batch operations
  batchEnableCredentials,
  batchDisableCredentials,
  batchDeleteCredentials,
  batchRecoverCredentials,
  listBatchTasks,
  getBatchTask,
  getBatchTaskResult,
  cancelBatchTask,
  getBatchTaskStreamURL
};
