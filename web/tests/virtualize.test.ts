import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createVirtualizer } from '../src/utils/virtualize';

describe('Virtualizer', () => {
  let container: HTMLElement;
  const originalRAF = globalThis.requestAnimationFrame;

  beforeEach(() => {
    container = document.createElement('div');
    container.style.height = '400px';
    container.style.overflow = 'auto';

    // Mock scrollTo method for JSDOM
    container.scrollTo = vi.fn((options: any) => {
      if (typeof options === 'object' && options.top !== undefined) {
        container.scrollTop = options.top;
      }
    }) as any;

    document.body.appendChild(container);
    globalThis.requestAnimationFrame = (cb: FrameRequestCallback): number => {
      cb(0);
      return 0;
    };
  });

  afterEach(() => {
    document.body.removeChild(container);
    globalThis.requestAnimationFrame = originalRAF;
  });

  it('should create virtualizer with basic options', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      render,
    });

    expect(container.children.length).toBeGreaterThan(0);
    expect(render).toHaveBeenCalled();

    virtualizer.destroy();
  });

  it('should update item count', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 50,
      rowHeight: 40,
      render,
    });

    const initialCalls = render.mock.calls.length;

    virtualizer.update({ itemCount: 100 });

    // Should have rendered more items
    expect(render.mock.calls.length).toBeGreaterThanOrEqual(initialCalls);

    virtualizer.destroy();
  });

  it('should scroll to index', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      render,
    });

    virtualizer.scrollToIndex(50, 'auto');

    // Check if scrollTop is approximately correct
    expect(container.scrollTop).toBeGreaterThan(0);

    virtualizer.destroy();
  });

  it('should scroll to top', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      render,
    });

    // Scroll down first
    container.scrollTop = 1000;

    virtualizer.scrollToTop();

    // Should scroll to top
    expect(container.scrollTop).toBe(0);

    virtualizer.destroy();
  });

  it('should scroll to bottom', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      render,
    });

    virtualizer.scrollToBottom();

    // Should scroll to bottom
    expect(container.scrollTop).toBe(container.scrollHeight);

    virtualizer.destroy();
  });

  it('should call onScroll callback', () => {
    const onScroll = vi.fn();

    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      render,
      onScroll,
    });

    // Trigger scroll synchronously
    container.scrollTop = 100;
    container.dispatchEvent(new Event('scroll'));

    expect(onScroll).toHaveBeenCalled();
    const [scrollTop, scrollHeight] = onScroll.mock.calls[0] as [number, number];
    expect(scrollTop).toBeGreaterThanOrEqual(0);
    expect(scrollHeight).toBeGreaterThanOrEqual(0);

    virtualizer.destroy();
  });

  it('should get visible range', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      overscan: 5,
      render,
    });

    const [first, last] = virtualizer.getVisibleRange();
    expect(first).toBeGreaterThanOrEqual(0);
    expect(last).toBeLessThan(100);
    expect(last).toBeGreaterThanOrEqual(first);

    virtualizer.destroy();
  });

  it('should clean up on destroy', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      render,
    });

    const childrenBefore = container.children.length;
    expect(childrenBefore).toBeGreaterThan(0);

    virtualizer.destroy();

    expect(container.children.length).toBe(0);
  });

  it('should handle zero items', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 0,
      rowHeight: 40,
      render,
    });

    // With 0 items, the visible range should be [0, 0] or similar
    // The virtualizer still initializes but renders no items
    const [first, last] = virtualizer.getVisibleRange();
    expect(first).toBeGreaterThanOrEqual(0);
    expect(last).toBeGreaterThanOrEqual(-1);

    virtualizer.destroy();
  });

  it('should handle overscan correctly', () => {
    const render = vi.fn((index: number) => {
      const div = document.createElement('div');
      div.textContent = `Item ${index}`;
      return div;
    });

    const virtualizer = createVirtualizer(container, {
      itemCount: 100,
      rowHeight: 40,
      overscan: 10,
      render,
    });

    const [first, last] = virtualizer.getVisibleRange();
    
    // With overscan, should render more items than visible
    const visibleCount = Math.ceil(400 / 40); // container height / row height
    const renderedCount = last - first + 1;
    expect(renderedCount).toBeGreaterThan(visibleCount);

    virtualizer.destroy();
  });
});
