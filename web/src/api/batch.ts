import {
  enableCredential,
  disableCredential,
  deleteCredential,
  probeFlash,
} from './credentials';
import { postRegistry, deleteRegistryModel } from './registry';

const DEFAULT_CHUNK_SIZE = 5;
const DEFAULT_RETRY = 2;
const CHUNK_DELAY_MS = 300;

type Settled<T> = PromiseSettledResult<T>;

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

async function executeWithRetry<T>(fn: () => Promise<T>, retries = DEFAULT_RETRY): Promise<Settled<T>> {
  let attempt = 0;
  while (attempt <= retries) {
    try {
      const value = await fn();
      return { status: 'fulfilled', value };
    } catch (error) {
      if (attempt === retries) {
        return { status: 'rejected', reason: error };
      }
      attempt += 1;
      const backoff = Math.min(1000 * Math.pow(2, attempt - 1), 5000);
      await delay(backoff);
    }
  }
  return { status: 'rejected', reason: new Error('unexpected retry loop exit') };
}

async function processInChunks<T>(
  items: T[],
  worker: (item: T) => Promise<any>,
  chunkSize = DEFAULT_CHUNK_SIZE
): Promise<Settled<any>[]> {
  const results: Settled<any>[] = [];
  for (let i = 0; i < items.length; i += chunkSize) {
    const slice = items.slice(i, i + chunkSize);
    const chunkResults = await Promise.all(
      slice.map((item) => executeWithRetry(() => worker(item)))
    );
    results.push(...chunkResults);
    if (i + chunkSize < items.length) {
      await delay(CHUNK_DELAY_MS);
    }
  }
  return results;
}

export const batchOperations = {
  bulkCredentialAction: async (actions: Array<{ id: string; action: string }>): Promise<Settled<any>[]> =>
    processInChunks(actions, async ({ id, action }: { id: string; action: string }) => {
      switch (action) {
        case 'enable':
          return enableCredential(id);
        case 'disable':
          return disableCredential(id);
        case 'delete':
          return deleteCredential(id);
        default:
          return Promise.resolve();
      }
    }),
  bulkModelAction: async (channel: string, models: any[], action: string): Promise<Settled<any>[]> =>
    processInChunks(models, async (model: any) => {
      switch (action) {
        case 'enable':
          return postRegistry(channel, { ...(model as object), enabled: true });
        case 'disable':
          return postRegistry(channel, { ...(model as object), enabled: false });
        case 'delete':
          return deleteRegistryModel(channel, (model as any).id);
        default:
          return Promise.resolve();
      }
    }),
  bulkHealthCheck: async (credentialIds: string[]): Promise<Settled<any>[]> =>
    processInChunks(credentialIds, (id: string) => probeFlash().then((result: any) => ({ id, ...result }))),
};
