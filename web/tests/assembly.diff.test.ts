import { describe, it, expect, vi } from 'vitest';

const modulePathMock = vi.fn();

vi.mock('../src/core/module_paths', () => ({
  modulePath: modulePathMock
}));

describe('loadAssemblyTab', () => {
  it('returns loaded module when dynamic import succeeds', async () => {
    modulePathMock.mockReturnValue('virtual:assembly-module');
    const renderPage = vi.fn(() => '<div>assembly</div>');

    vi.doMock('virtual:assembly-module', () => ({
      default: {
        renderPage,
        update: vi.fn()
      }
    }), { virtual: true });

    const { loadAssemblyTab } = await import('../src/tabs/assembly?case=success');
    const mod = await loadAssemblyTab();

    expect(await mod.renderPage()).toContain('assembly');
    expect(renderPage).toHaveBeenCalled();

    vi.unmock('virtual:assembly-module');
    modulePathMock.mockReset();
  });

  it('falls back to inline error module when import fails', async () => {
    modulePathMock.mockReturnValue('virtual:missing-assembly');
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    const { loadAssemblyTab } = await import('../src/tabs/assembly?case=failure');
    const mod = await loadAssemblyTab();
    const html = await mod.renderPage();

    expect(html).toContain('assembly_load_failed');
    errorSpy.mockRestore();
    modulePathMock.mockReset();
  });
});
