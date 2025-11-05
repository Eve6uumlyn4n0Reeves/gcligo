import type { Credential, CredentialFilters } from './types.js';
import type { HealthThresholds } from './health.js';
import { DEFAULT_THRESHOLDS, calculateHealthScore, getHealthLevel } from './health.js';

export interface FilterOptions {
  thresholds?: HealthThresholds;
  now?: number;
}

export function filterCredentials(
  credentials: Credential[],
  filters: CredentialFilters,
  options: FilterOptions = {}
): Credential[] {
  const { search, status, health, project } = filters;
  const kw = (search ?? '').toLowerCase();
  const thresholds = options.thresholds ?? DEFAULT_THRESHOLDS;
  const now = options.now ?? Date.now();

  return credentials.filter((cred) => {
    const matchesKeyword =
      !kw ||
      [cred.filename, cred.email, cred.project_id, cred.display_name]
        .filter(Boolean)
        .some((field) => String(field).toLowerCase().includes(kw));

    if (!matchesKeyword) return false;

    let matchesStatus = true;
    if (status === 'active') {
      matchesStatus = !cred.disabled && !cred.auto_banned;
    } else if (status === 'disabled') {
      matchesStatus = Boolean(cred.disabled) && !cred.auto_banned;
    } else if (status === 'banned') {
      matchesStatus = Boolean(cred.auto_banned);
    }
    if (!matchesStatus) return false;

    if (health !== 'all') {
      const score = calculateHealthScore(cred, now);
      const level = getHealthLevel(score, thresholds);
      if (level !== health) return false;
    }

    if (project !== 'all' && project) {
      if (cred.project_id !== project) return false;
    }

    return true;
  });
}

export function collectProjects(credentials: Credential[]): string[] {
  const projects = new Set<string>();
  for (const cred of credentials) {
    if (cred.project_id) {
      projects.add(String(cred.project_id));
    }
  }
  return Array.from(projects).sort();
}
