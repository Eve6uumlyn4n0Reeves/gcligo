/**
 * 资源加载器
 * 处理脚本和样式表的加载，支持重试和超时
 */

/**
 * 加载资源（脚本或样式表）
 */
export function loadResourceWithRetry(
  src: string,
  type: 'script' | 'stylesheet',
  retries: number = 3
): Promise<HTMLElement> {
  const maxAttempts = Math.max(
    1,
    Number.isFinite(retries) ? Math.floor(retries) + 1 : 3
  );
  const cacheKey = `resource:${src}:${type}`;

  return new Promise((resolve, reject) => {
    let attempt = 0;
    let lastError: Error | null = null;

    // 检查是否已经加载过
    const existing =
      type === 'script'
        ? document.querySelector(`script[src="${src}"]`)
        : document.querySelector(`link[href="${src}"]`);

    if (existing) {
      resolve(existing as HTMLElement);
      return;
    }

    const tryLoad = () => {
      attempt += 1;
      console.log(
        `尝试加载资源 ${src} (第 ${attempt}/${maxAttempts} 次)`
      );

      const element =
        type === 'script'
          ? document.createElement('script')
          : document.createElement('link');

      if (type === 'script') {
        (element as HTMLScriptElement).type = 'module';
        (element as HTMLScriptElement).async = true;
        (element as HTMLScriptElement).src = src;
        element.crossOrigin = 'anonymous';
      } else {
        (element as HTMLLinkElement).rel = 'stylesheet';
        (element as HTMLLinkElement).href = src;
        element.crossOrigin = 'anonymous';
      }

      element.dataset.retryAttempt = String(attempt);
      element.dataset.loadTimestamp = String(Date.now());

      const cleanup = () => {
        element.onload = null;
        element.onerror = null;
      };

      const loadTimeout = setTimeout(() => {
        cleanup();
        if (element.parentNode) {
          element.parentNode.removeChild(element);
        }
        lastError = new Error(`加载超时: ${src}`);
        retryOrFail();
      }, 10000); // 10秒超时

      element.onload = () => {
        clearTimeout(loadTimeout);
        cleanup();
        console.log(`资源加载成功: ${src}`);

        // 记录成功加载
        try {
          sessionStorage.setItem(cacheKey, 'loaded');
        } catch (e) {
          // ignore
        }

        resolve(element);
      };

      element.onerror = () => {
        clearTimeout(loadTimeout);
        cleanup();

        if (element.parentNode) {
          element.parentNode.removeChild(element);
        }

        lastError = new Error(`加载失败: ${src} (网络错误)`);
        retryOrFail();
      };

      const retryOrFail = () => {
        if (attempt < maxAttempts) {
          // 指数退避延迟
          const delay = Math.min(1000 * Math.pow(2, attempt - 1), 8000);
          console.warn(
            `加载 ${src} 失败 (第 ${attempt}/${maxAttempts} 次)，${delay}ms 后重试...`
          );

          // 更新加载状态
          updateLoadingProgress(attempt, maxAttempts, src);

          setTimeout(tryLoad, delay);
        } else {
          console.error(`资源加载最终失败: ${src}`, lastError);
          reject(
            lastError ||
              new Error(`Failed to load ${src} after ${attempt} attempts`)
          );
        }
      };

      document.head.appendChild(element);
    };

    // 检查缓存状态
    try {
      const cached = sessionStorage.getItem(cacheKey);
      if (cached === 'loaded') {
        console.log(`从缓存加载资源: ${src}`);
      }
    } catch (e) {
      // ignore
    }

    tryLoad();
  });
}

/**
 * 更新加载进度提示
 */
function updateLoadingProgress(
  attempt: number,
  maxAttempts: number,
  resource: string
): void {
  const container = document.getElementById('app-container');
  const loading = container?.querySelector('.loading p');

  if (loading) {
    loading.textContent = `正在加载管理控制台... (重试 ${attempt}/${maxAttempts})`;

    // 添加详细信息
    let detail = container?.querySelector('.loading-detail') as HTMLElement | null;
    if (!detail) {
      detail = document.createElement('div');
      detail.className = 'loading-detail';
      (detail as HTMLElement).style.cssText =
        'font-size: 12px; color: #6b7280; margin-top: 8px;';
      loading.parentNode?.appendChild(detail);
    }
    detail.textContent = `资源: ${resource.split('/').pop()}`;
  }
}

