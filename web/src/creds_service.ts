import { api } from './api.js';
import type { Credential } from './creds/types.js';

interface CredentialListResponse {
  credentials?: Credential[];
}

export const credsService = {
  async list(): Promise<Credential[]> {
    const data = (await api.listCredentials()) as CredentialListResponse;
    return Array.isArray(data?.credentials) ? data.credentials : [];
  },
  enable(filename: string): Promise<unknown> {
    return api.enableCredential(filename);
  },
  disable(filename: string): Promise<unknown> {
    return api.disableCredential(filename);
  },
  delete(filename: string): Promise<unknown> {
    return api.deleteCredential(filename);
  },
  recover(filename: string): Promise<unknown> {
    return api.recoverCredential(filename);
  },
  recoverAll(): Promise<unknown> {
    return api.recoverAllCredentials();
  },
  reload(): Promise<unknown> {
    return api.reloadCredentials();
  },
  probeFlash(model?: string, timeout?: number): Promise<unknown> {
    return api.probeFlash(model, timeout);
  }
};
