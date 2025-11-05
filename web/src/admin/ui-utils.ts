/**
 * 管理控制台 UI 工具
 * 处理模态框、emoji 移除等 UI 相关功能
 */

/**
 * 关闭模态框
 */
export function closeModal(): void {
  const modal = document.getElementById('modal');
  if (modal) {
    modal.classList.remove('active');
  }
}

/**
 * 打开模态框
 */
export function openModal(title: string, content: string, allowHTML: boolean = true): void {
  const modalTitle = document.getElementById('modalTitle');
  const modalBody = document.getElementById('modalBody');
  const modal = document.getElementById('modal');

  if (modalTitle) {
    modalTitle.textContent = title;
  }
  if (modalBody) {
    if (allowHTML) modalBody.innerHTML = content;
    else modalBody.textContent = content;
  }
  if (modal) {
    modal.classList.add('active');
  }
}

/**
 * 从元素中移除前导 emoji
 */
function stripEmojiFromElement(el: Element): void {
  if (!el) return;
  el.innerHTML = el.innerHTML.replace(/^[^\p{L}\p{N}\u4e00-\u9fa5]+\s*/u, '');
}

/**
 * 移除页面中的前导 emoji
 */
function stripEmojiOnce(): void {
  try {
    document
      .querySelectorAll('.tab-button')
      .forEach((el) => stripEmojiFromElement(el));
    document
      .querySelectorAll('.nav-btn.external')
      .forEach((el) => stripEmojiFromElement(el));
    document
      .querySelectorAll('.header h1')
      .forEach((el) => stripEmojiFromElement(el));
    document
      .querySelectorAll('h1, h2, h3')
      .forEach((el) => stripEmojiFromElement(el));
  } catch (e) {
    // ignore
  }
}

/**
 * 初始化 emoji 移除
 */
export function initializeEmojiStripping(): void {
  const run = () => {
    stripEmojiOnce();
    setTimeout(stripEmojiOnce, 500);
    setTimeout(stripEmojiOnce, 1500);

    // 监听后续 DOM 变更（5s 内），避免后续重渲染又出现表情
    try {
      const obs = new MutationObserver(() => stripEmojiOnce());
      obs.observe(document.body, { childList: true, subtree: true });
      setTimeout(() => {
        try {
          obs.disconnect();
        } catch {
          // ignore
        }
      }, 5000);
    } catch {
      // ignore
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', run);
  } else {
    run();
  }
}

/**
 * 初始化全局错误处理
 */
export function initializeGlobalErrorHandling(): void {
  window.addEventListener('error', (event) => {
    console.error('全局错误:', event.error);
  });

  window.addEventListener('unhandledrejection', (event) => {
    console.error('未处理的Promise拒绝:', event.reason);
  });
}

/**
 * 渲染浏览器不支持错误
 */
export function renderBrowserNotSupportedError(): void {
  const container = document.getElementById('app-container');
  if (container) {
    container.innerHTML = `
      <div class="card" style="text-align: center; padding: 60px 20px;">
        <h2 style="color: #ef4444; margin-bottom: 20px;">浏览器不支持</h2>
        <p style="color: #666; margin-bottom: 30px;">请使用支持ES6模块的现代浏览器访问此页面</p>
        <p style="color: #999;">推荐使用最新版本的 Chrome、Firefox、Safari 或 Edge</p>
      </div>
    `;
  }
}

