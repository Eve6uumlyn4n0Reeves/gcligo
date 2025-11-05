import { describe, it, beforeEach, expect } from 'vitest';
import { renderCredentialCard, populateCredentialGrid } from '../src/creds_view';
import type { Credential } from '../src/creds/types';

type MinimalManager = {
  formatTimestamp: (ts?: number, emptyText?: string) => string;
  normalizeTimestamp: (value?: number | string | null) => number;
  renderHealthBar: (score: number) => string;
};

describe('creds_view rendering', () => {
  let helpers: MinimalManager;

  beforeEach(() => {
    globalThis.requestAnimationFrame = (cb: FrameRequestCallback) => {
      cb(0);
      return 0;
    };
    helpers = {
      formatTimestamp: (ts?: number, emptyText = '从未') =>
        ts ? `ts-${ts}` : emptyText,
      normalizeTimestamp: (value?: number | string | null) => {
        if (typeof value === 'number') return value;
        if (typeof value === 'string') {
          const parsed = Number(value);
          return Number.isFinite(parsed) ? parsed : 0;
        }
        return 0;
      },
      renderHealthBar: (score: number) => `<health score="${score}"></health>`
    } as MinimalManager;
  });

  it('renders credential card snapshot', () => {
    const credential: Credential = {
      filename: 'cred-demo.json',
      email: 'demo@example.com',
      project_id: 'proj-demo',
      total_requests: 42,
      success_rate: 0.93,
      failure_weight: 0.1,
      health_score: 0.88,
      last_success_ts: 1_695_000_000,
      last_failure_ts: 1_694_900_000,
      error_codes: ['429', '500']
    };

    const html = renderCredentialCard(credential, helpers);
    expect(html).toMatchInlineSnapshot(`
      "
          <div class="credential-card " data-cred-id="cred-demo.json">
            <div class="credential-header">
              <div class="credential-name">cred-demo.json</div>
              <div class="credential-status status-active">
                活跃
              </div>
            </div>
            <div class="credential-info">项目: proj-demo</div>
            <div class="credential-info">邮箱: demo@example.com</div>
            <div class="credential-info">总调用: 42</div>
            <div class="credential-info">Gemini 2.5 Pro: 0</div>
            <div class="credential-info">成功率: 93.0%</div>
            <div class="credential-info">失败权重: 0.10</div>
            <div class="credential-info">健康评分: 88%</div>
            <div class="credential-info">最后成功: ts-1695000000</div>
            <div class="credential-info">最后失败: ts-1694900000</div>
            <div class="credential-info" style="color:#ef4444;">最近错误: 请求过多(429)、服务器错误(500)</div>
            <div class="credential-actions">
          <button class="btn btn-warning btn-sm" aria-label="禁用凭证 cred-demo.json" onclick="window.credsManager.disableCredential('cred-demo.json')">禁用</button>
          <button class="btn btn-danger btn-sm" aria-label="删除凭证 cred-demo.json" onclick="window.credsManager.deleteCredential('cred-demo.json')">删除</button></div>
          </div>"
    `);
  });

  it('populates credential grid into DOM', () => {
    const container = document.createElement('div');
    const items: Credential[] = [
      { filename: 'cred-a.json', email: 'a@example.com' },
      { filename: 'cred-b.json', email: 'b@example.com', disabled: true }
    ];

    populateCredentialGrid(container, items, helpers);
    expect(container.querySelectorAll('.credential-card').length).toBe(2);
  });
});
