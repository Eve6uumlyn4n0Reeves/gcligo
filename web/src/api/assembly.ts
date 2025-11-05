import { enhanced, encodeSegment } from './base';

const withReasonHeader = (reason?: string) => {
  if (!reason) return undefined;
  return { 'X-Change-Reason': reason };
};

export const listAssemblyPlans = () => enhanced('assembly/plans');
export const getAssemblyPlan = (name: string) => enhanced(`assembly/plans/${encodeSegment(name)}`);
export const saveAssemblyPlan = (
  name: string,
  include: Record<string, boolean> = { models: true, variants: true, routing: true }
) =>
  enhanced('assembly/plans', {
    method: 'POST',
    body: JSON.stringify({ name, include }),
  });

export const applyAssemblyPlan = (name: string, options: { reason?: string } = {}) =>
  enhanced(`assembly/plans/${encodeSegment(name)}/apply`, {
    method: 'PUT',
    cache: false,
    headers: withReasonHeader(options.reason),
  });

export const rollbackAssemblyPlan = (name: string, options: { reason?: string } = {}) =>
  enhanced(`assembly/plans/${encodeSegment(name)}/rollback`, {
    method: 'PUT',
    cache: false,
    headers: withReasonHeader(options.reason),
  });

export const getAssemblyRouting = () => enhanced('assembly/routing');

export const dryRunAssemblyPlan = (plan: Record<string, unknown>, options: { reason?: string } = {}) =>
  enhanced('assembly/dry-run', {
    method: 'POST',
    body: JSON.stringify({ plan }),
    cache: false,
    headers: withReasonHeader(options.reason),
  });

export const clearRoutingCooldowns = (
  payload: Record<string, unknown> = {},
  options: { reason?: string } = {}
) =>
  enhanced('assembly/cooldowns/clear', {
    method: 'POST',
    body: JSON.stringify(payload),
    cache: false,
    headers: withReasonHeader(options.reason),
  });

export const assemblyApi = {
  listAssemblyPlans,
  getAssemblyPlan,
  saveAssemblyPlan,
  applyAssemblyPlan,
  rollbackAssemblyPlan,
  getAssemblyRouting,
  dryRunAssemblyPlan,
  clearRoutingCooldowns
};
