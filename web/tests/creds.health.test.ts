import { describe, it, expect } from 'vitest';
import {
  calculateHealthScore,
  getHealthLevel,
  getHealthColor,
  DEFAULT_THRESHOLDS
} from '../src/creds/health';
import type { Credential } from '../src/creds/types';

const baseCredential: Credential = {
  filename: 'cred.json',
  success_rate: 0.95
};

describe('creds/health helpers', () => {
  it('returns zero for auto banned credentials', () => {
    const score = calculateHealthScore({ ...baseCredential, auto_banned: true });
    expect(score).toBe(0);
  });

  it('reduces score according to error history', () => {
    const score = calculateHealthScore({
      ...baseCredential,
      error_history: { '429': 3, '403': 1, '500': 2 }
    });
    // 3*5 = 15, 1*10 = 10, 2*3 = 6 => 100 - 31 = 69
    expect(score).toBe(69);
  });

  it('applies recency penalty for long inactive credentials', () => {
    const twoDaysAgo = Date.now() - 1000 * 60 * 60 * 48;
    const score = calculateHealthScore(
      { ...baseCredential, last_success_time: new Date(twoDaysAgo).toISOString() },
      Date.now()
    );
    expect(score).toBeLessThan(100);
  });

  it('maps score to health level and color', () => {
    expect(getHealthLevel(95, DEFAULT_THRESHOLDS)).toBe('excellent');
    expect(getHealthLevel(75, DEFAULT_THRESHOLDS)).toBe('good');
    expect(getHealthLevel(65, DEFAULT_THRESHOLDS)).toBe('poor');
    expect(getHealthLevel(10, DEFAULT_THRESHOLDS)).toBe('critical');
    expect(getHealthColor('excellent')).toBe('#10b981');
    expect(getHealthColor('critical')).toBe('#ef4444');
  });
});
