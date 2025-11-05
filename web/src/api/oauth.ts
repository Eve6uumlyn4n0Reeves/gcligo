import { mg, withQuery } from './base';

export const startOAuth = (projectId: string): Promise<any> => mg('oauth/start', {
  method: 'POST',
  body: JSON.stringify({ project_id: projectId || '' })
});
export const completeOAuth = (code: string, state: string): Promise<any> => mg('oauth/callback', {
  method: 'POST',
  body: JSON.stringify({ code, state })
});
export const listOAuthProjects = (accessToken: string): Promise<any> => mg(withQuery('oauth/projects', { access_token: accessToken }));
export const getOAuthUserInfo = (accessToken: string): Promise<any> => mg(withQuery('oauth/userinfo', { access_token: accessToken }));

export const oauthApi = {
  startOAuth,
  completeOAuth,
  listOAuthProjects,
  getOAuthUserInfo
};
