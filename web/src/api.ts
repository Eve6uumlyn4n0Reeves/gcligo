import { statsApi } from './api/stats';
import { configApi } from './api/config';
import { credentialsApi } from './api/credentials';
import { registryApi } from './api/registry';
import { assemblyApi } from './api/assembly';
import { oauthApi } from './api/oauth';
import { batchOperations } from './api/batch';
import { cacheApi } from './api/cache';

export const api = {
  ...statsApi,
  ...configApi,
  ...credentialsApi,
  ...registryApi,
  ...assemblyApi,
  ...oauthApi,
  batchOperations,
  cache: cacheApi
};
