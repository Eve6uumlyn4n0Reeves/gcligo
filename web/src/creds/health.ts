import type { Credential, HealthLevel } from './types.js';

export interface HealthThresholds {
  excellent: number;
  good: number;
  poor: number;
}

export const DEFAULT_THRESHOLDS: HealthThresholds = Object.freeze({
  excellent: 90,
  good: 70,
  poor: 50
});

export function calculateHealthScore(credential: Credential, now: number = Date.now()): number {
  let score = 100;

  if (credential.auto_banned) return 0;
  if (credential.disabled) return 25;

  const history = credential.error_history ?? {};
  const total429 = Number(history['429'] ?? 0);
  const total403 = Number(history['403'] ?? 0);
  const total5xx =
    Number(history['500'] ?? 0) +
    Number(history['502'] ?? 0) +
    Number(history['503'] ?? 0);

  score -= Math.min(total429 * 5, 30);
  score -= Math.min(total403 * 10, 40);
  score -= Math.min(total5xx * 3, 20);

  const lastSuccess =
    credential.last_success_ts ??
    credential.last_success_time ??
    credential.last_success;
  if (lastSuccess) {
    const lastSuccessDate =
      typeof lastSuccess === 'number'
        ? lastSuccess * (lastSuccess > 10_000_000_000 ? 1 : 1000)
        : Date.parse(String(lastSuccess));
    if (!Number.isNaN(lastSuccessDate)) {
      const hoursSinceSuccess = (now - lastSuccessDate) / (1000 * 60 * 60);
      if (hoursSinceSuccess > 24) {
        score -= Math.min((hoursSinceSuccess - 24) * 2, 20);
      }
    }
  }

  return Math.max(0, Math.round(score));
}

export function getHealthLevel(
  score: number,
  thresholds: HealthThresholds = DEFAULT_THRESHOLDS
): HealthLevel {
  if (score >= thresholds.excellent) return 'excellent';
  if (score >= thresholds.good) return 'good';
  if (score >= thresholds.poor) return 'poor';
  return 'critical';
}

export function getHealthColor(level: HealthLevel): string {
  switch (level) {
    case 'excellent':
      return '#10b981';
    case 'good':
      return '#3b82f6';
    case 'poor':
      return '#f59e0b';
    default:
      return '#ef4444';
  }
}
