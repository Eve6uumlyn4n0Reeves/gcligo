import { describe, it, expect } from 'vitest';
import { filterCredentials, collectProjects } from '../src/creds/filter';
import type { Credential, CredentialFilters } from '../src/creds/types';

const credentials: Credential[] = [
  {
    filename: 'cred-a.json',
    email: 'a@example.com',
    project_id: 'proj-a',
    disabled: false,
    auto_banned: false
  },
  {
    filename: 'cred-b.json',
    email: 'b@example.com',
    project_id: 'proj-b',
    disabled: true,
    auto_banned: false
  },
  {
    filename: 'cred-c.json',
    email: 'c@example.com',
    project_id: 'proj-c',
    disabled: false,
    auto_banned: true
  }
];

const baseFilters: CredentialFilters = {
  search: '',
  status: 'all',
  health: 'all',
  project: 'all'
};

describe('creds/filter helpers', () => {
  it('filters by keyword and status', () => {
    const result = filterCredentials(credentials, {
      ...baseFilters,
      search: 'cred-b',
      status: 'disabled'
    });
    expect(result).toHaveLength(1);
    expect(result[0]?.filename).toBe('cred-b.json');
  });

  it('filters by banned status', () => {
    const result = filterCredentials(credentials, {
      ...baseFilters,
      status: 'banned'
    });
    expect(result).toHaveLength(1);
    expect(result[0]?.filename).toBe('cred-c.json');
  });

  it('collects sorted project list', () => {
    const projects = collectProjects(credentials);
    expect(projects).toEqual(['proj-a', 'proj-b', 'proj-c']);
  });
});
