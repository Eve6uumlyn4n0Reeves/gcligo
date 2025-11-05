// 虚拟滚动（固定行高版）
// 用法：
//  const vt = createVirtualizer(container, {
//    itemCount: data.length,
//    rowHeight: 40,
//    overscan: 6,
//    render: (index) => /* return HTMLElement for row */,
//    onScroll: (scrollTop, scrollHeight) => { /* optional callback */ }
//  });
//  vt.update({ itemCount: newCount });
//  vt.scrollToIndex(50); // 滚动到指定索引
//  vt.destroy();

export interface VirtualizerOptions {
  itemCount: number;
  rowHeight: number;
  overscan?: number;
  render: (index: number) => HTMLElement;
  onScroll?: (scrollTop: number, scrollHeight: number) => void;
  enableSmoothScroll?: boolean;
}

export interface VirtualizerHandle {
  update(next: Partial<Pick<VirtualizerOptions, 'itemCount'>>): void;
  scrollToIndex(index: number, behavior?: ScrollBehavior): void;
  scrollToTop(): void;
  scrollToBottom(): void;
  getVisibleRange(): [number, number];
  destroy(): void;
}

export function createVirtualizer(
  container: HTMLElement,
  opts: VirtualizerOptions
): VirtualizerHandle {
  const rowHeight = Math.max(1, Number(opts.rowHeight || 40));
  const overscan = Math.max(0, Number(opts.overscan || 6));
  let itemCount = Math.max(0, Number(opts.itemCount || 0));
  const render =
    typeof opts.render === 'function' ? opts.render : () => document.createElement('div');
  const onScrollCallback = opts.onScroll;
  const enableSmoothScroll = opts.enableSmoothScroll !== false;

  // set up container
  container.style.overflow = container.style.overflow || 'auto';
  container.style.position = container.style.position || 'relative';

  const spacer = document.createElement('div'); // total height holder
  spacer.style.height = `${itemCount * rowHeight}px`;
  spacer.style.position = 'relative';
  spacer.style.width = '100%';
  container.innerHTML = '';
  container.appendChild(spacer);

  let first = 0, last = -1; // rendered range [first,last]
  const nodes = new Map<number, HTMLElement>(); // index -> element

  function computeRange(): [number, number] {
    const scrollTop = container.scrollTop;
    const viewH = container.clientHeight || 1;
    const start = Math.floor(scrollTop / rowHeight) - overscan;
    const end = Math.ceil((scrollTop + viewH) / rowHeight) + overscan;
    return [clamp(start, 0, itemCount-1), clamp(end, 0, itemCount-1)];
  }

  function clamp(v: number, min: number, max: number): number {
    if (max < min) return min;
    return Math.max(min, Math.min(max, v));
  }

  function mountRow(index: number): HTMLElement {
    let el = nodes.get(index);
    if (!el) {
      el = render(index) || document.createElement('div');
      el.style.position = 'absolute';
      el.style.top = `${index * rowHeight}px`;
      el.style.left = '0';
      el.style.right = '0';
      nodes.set(index, el);
      spacer.appendChild(el);
    }
    return el;
  }

  function unmountRow(index: number): void {
    const el = nodes.get(index);
    if (el && el.parentNode === spacer) spacer.removeChild(el);
    nodes.delete(index);
  }

  function rerender(): void {
    const [nextFirst, nextLast] = computeRange();
    // remove rows outside new range
    for (const [idx] of Array.from(nodes)) {
      if (idx < nextFirst || idx > nextLast) unmountRow(idx);
    }
    // add missing rows
    for (let i = nextFirst; i <= nextLast; i++) mountRow(i);
    first = nextFirst; last = nextLast;
  }

  function onScroll(): void {
    requestAnimationFrame(() => {
      rerender();
      if (onScrollCallback) {
        onScrollCallback(container.scrollTop, container.scrollHeight);
      }
    });
  }

  container.addEventListener('scroll', onScroll, { passive: true });

  // 添加 resize 监听以处理容器大小变化
  let resizeObserver: ResizeObserver | null = null;
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(rerender);
    });
    resizeObserver.observe(container);
  }

  // initial paint
  rerender();

  return {
    update(next: Partial<Pick<VirtualizerOptions, 'itemCount'>>) {
      if (typeof next.itemCount === 'number') {
        itemCount = Math.max(0, next.itemCount);
        spacer.style.height = `${itemCount * rowHeight}px`;
        // prune out-of-range
        for (const [idx] of Array.from(nodes)) { if (idx >= itemCount) unmountRow(idx); }
        rerender();
      }
    },

    scrollToIndex(index: number, behavior: ScrollBehavior = 'smooth') {
      const clampedIndex = clamp(index, 0, itemCount - 1);
      const targetScrollTop = clampedIndex * rowHeight;
      container.scrollTo({
        top: targetScrollTop,
        behavior: enableSmoothScroll ? behavior : 'auto',
      });
    },

    scrollToTop() {
      container.scrollTo({
        top: 0,
        behavior: enableSmoothScroll ? 'smooth' : 'auto',
      });
    },

    scrollToBottom() {
      container.scrollTo({
        top: container.scrollHeight,
        behavior: enableSmoothScroll ? 'smooth' : 'auto',
      });
    },

    getVisibleRange(): [number, number] {
      return [first, last];
    },

    destroy() {
      container.removeEventListener('scroll', onScroll);
      if (resizeObserver) {
        resizeObserver.disconnect();
      }
      nodes.forEach((el) => { if (el.parentNode === spacer) spacer.removeChild(el); });
      nodes.clear();
      container.innerHTML = '';
    }
  };
}
