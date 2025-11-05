/**
 * 轻量 UI 工具：主题、模态、横幅
 */

import { NotificationCenter } from './components/notification';
import { DialogManager } from './components/dialog';
import { createCacheManager as createCacheService, createRefreshManager as createRefreshService } from './services/cache';
import { createEventBus as createEventBusService, throttle as throttleFn, debounce as debounceFn } from './services/shortcut';

class UIHelper {
  // Theme properties
  themeKey: string;
  currentTheme: string;

  // Language properties
  langKey: string;
  currentLang: string;
  dict: Record<string, Record<string, string>>;

  // UI components
  notificationCenter: NotificationCenter;
  dialogs: DialogManager;
  globalLoading: HTMLElement | null;

  constructor() {
    this.themeKey = 'ui:theme';
    this.currentTheme = 'minimal';
    this.langKey = 'ui:lang';
    this.currentLang = 'zh';
    this.globalLoading = null;

    // Initialize dictionary first
    this.dict = {
      en: {
        refresh: 'Refresh',
        save: 'Save',
        cancel: 'Cancel',
        confirm: 'Confirm',
        tab_dashboard: 'Dashboard',
        tab_credentials: 'Credentials',
        tab_oauth: 'OAuth',
        tab_stats: 'Statistics',
        tab_streaming: 'Streaming Insights',
        tab_logs: 'Logs',
        tab_models: 'Model Registry',
        tab_assembly: 'Router Assembly',
        tab_config: 'Settings',
        streaming_title: 'Streaming Observatory',
        streaming_lines: 'SSE lines emitted',
        streaming_disconnects: 'Disconnect reasons',
        streaming_tool_calls: 'Tool call events',
        streaming_anti_trunc: 'Anti-truncation attempts',
        streaming_fallbacks: 'Model fallbacks',
        streaming_thinking_removed: 'Thinking removed',
        streaming_last_updated: 'Last updated at {time}',
        streaming_empty: 'No data recorded yet.',
        option_theme_minimal: 'Minimal',
        nav_routes: 'Routes Overview',
        header_title: 'GCLI2API-Go Admin Console',
        header_subtitle: 'High-performance Gemini CLI to OpenAI API gateway',
        btn_toggle_nav: 'Navigation',
        badge_system_running: 'System running',
        badge_credentials: 'Credentials',
        badge_requests: 'Requests',
        badge_user: 'User',
        label_theme: 'Theme',
        label_language: 'Language',
        label_auto_refresh: 'Auto refresh',
        option_theme_gemini: 'Gemini',
        option_theme_classic: 'Classic',
        lang_zh: 'Simplified Chinese',
        lang_en: 'English',
        status_loading: 'Loading...',
        status_loading_page: 'Loading page...',
        status_loading_config: 'Loading configuration...',
        dashboard_quick_actions: 'Quick actions',
        dashboard_system_info: 'System info',
        system_version: 'Version',
        system_go_version: 'Go version',
        system_openai_port: 'OpenAI port',
        system_admin_version: 'Admin UI',
        system_uptime: 'Uptime',
        system_memory: 'Memory usage',
        dashboard_realtime_stats: 'Realtime statistics',
        dashboard_system_health: 'System health',
        btn_export: 'Export',
        oauth_heading: 'OAuth flow',
        config_heading: 'Configuration',
        config_port_hint: 'Edit this port via startup flags',
        config_calls_per_rotation: 'Credential rotation interval (requests)',
        config_retry_section: 'Retry & rate limiting',
        config_retry_enabled: 'Enable retry',
        config_retry_max: 'Max retries',
        config_rate_limit_enabled: 'Enable rate limit',
        config_rate_limit_rps: 'RPS limit',
        config_rate_limit_burst: 'Burst limit',
        config_auto_probe_section: 'Daily liveness probe',
        config_auto_probe_help: 'Runs once per day at the selected UTC hour using a flash model',
        config_auto_probe_hour: 'Trigger hour (UTC)',
        config_auto_probe_timeout: 'Timeout (seconds)',
        config_auto_probe_model: 'Probe model',
        config_auto_probe_toggle: 'Enable auto probe',
        config_preferred_section: 'Preferred base models',
        config_preferred_label: 'Default candidates for registry/assembly views',
        config_preferred_hint: 'Hold Ctrl/Cmd to multi-select. Used as defaults for registry and assembly.',
        config_auto_probe_history_section: 'Probe history',
        config_auto_probe_history_refresh: 'Refresh history',
        config_auto_probe_history_download: 'Download JSON',
        config_auto_probe_history_empty: 'No probe history yet.',
        config_auto_probe_history_error: 'Failed to load probe history',
        config_auto_probe_history_source_auto: 'Scheduled',
        config_auto_probe_history_source_manual: 'Manual',
        config_auto_probe_history_all_ok: 'All credentials healthy',
        config_auto_probe_history_unknown: 'Unknown',
        config_auto_probe_history_col_time: 'Timestamp',
        config_auto_probe_history_col_source: 'Source',
        config_auto_probe_history_col_model: 'Model',
        config_auto_probe_history_col_success: 'Success',
        config_auto_probe_history_col_duration: 'Duration',
        config_auto_probe_history_col_detail: 'Details',
        config_suggestion_upstream_missing: 'Using upstream: missing only',
        config_suggestion_upstream_missing_hint: 'Upstream models fetched successfully; showing models that are not yet registered.',
        config_suggestion_upstream: 'Using upstream catalogue',
        config_suggestion_upstream_hint: 'Upstream model list fetched successfully.',
        config_suggestion_config: 'Using config fallback',
        config_suggestion_config_hint: 'Upstream unavailable; using preferred_base_models from configuration.',
        config_suggestion_static: 'Using default candidates',
        config_suggestion_static_hint: 'Using built-in default Gemini models.',
        config_suggestion_static_error: 'Default list (upstream unavailable)',
        config_suggestion_static_error_hint: 'Both upstream and config lists were unavailable; using built-in defaults.',
        config_management_section: 'Management security',
        config_management_hash_label: 'Management key hash (bcrypt)',
        config_management_hash_placeholder: '$2b$... hashed value',
        config_management_hash_hint: 'Provide a bcrypt hash of the management key. Leave blank to keep the existing value.',
        config_save: 'Save settings',
        config_restart_hint: 'Some settings may require a restart to take effect.',
        config_update_success: 'Settings updated',
        config_update_failed: 'Failed to save settings',
        config_load_failed: 'Failed to load settings',
        placeholder_auto_probe_model: 'e.g. gemini-2.5-flash',
        error_auth_required: 'Authentication required to access the admin console',
        error_init_failed: 'Application initialization failed',
        error_title: 'Error',
        btn_reload_page: 'Reload page',
        notify_network_online: 'Network connection restored',
        notify_network_offline: 'Network connection lost',
        network_offline_banner: '⚠️ Network offline, some features are unavailable',
        aria_main_nav: 'Main navigation',
        tooltip_toggle_nav: 'Toggle navigation',
        user_status_authenticated: 'User: authenticated',
        user_status_none: 'User: unauthenticated',
        modal_quick_switch: 'Quick switch',
        quick_switch_placeholder: 'Search tabs, credentials, models…',
        quick_switch_hint: 'Arrow keys to navigate, Enter to open',
        quick_switch_section_tabs: 'Tabs',
        quick_switch_section_credentials: 'Credentials',
        quick_switch_section_models: 'Models',
        quick_switch_section_actions: 'Actions',
        quick_switch_no_results: 'No matches — try another keyword',
        quick_switch_tab_meta: 'Switch to this tab',
        quick_switch_cred_meta: 'Credential • {project} • Health {health}%',
        quick_switch_model_meta: 'Model • {base} • {options}',
        quick_switch_action_meta: 'Action',
        quick_switch_open_assembly: 'Open external assembly view',
        quick_switch_open_routes: 'Open routes overview',
        shortcut_title: 'Shortcuts',
        shortcut_quick_switch: 'Ctrl/Cmd + K: Quick switch',
        shortcut_refresh_tab: 'Ctrl/Cmd + R: Refresh current tab',
        shortcut_help: 'Shift + /: Open this help',
        aria_close: 'Close',
        page_loading: 'Loading page...'
      },
      zh: {
        refresh: '刷新数据',
        save: '保存',
        cancel: '取消',
        confirm: '确认',
        tab_dashboard: '仪表盘',
        tab_credentials: '凭证管理',
        tab_oauth: 'Google 授权',
        tab_stats: '统计数据',
        tab_streaming: '流式观测',
        tab_logs: '系统日志',
        tab_models: '模型注册中心',
        tab_assembly: '路由装配台',
        tab_config: '设置',
        streaming_title: '流式观测台',
        streaming_lines: 'SSE 行数',
        streaming_disconnects: '断开原因',
        streaming_tool_calls: '工具调用事件',
        streaming_anti_trunc: '抗截断尝试',
        streaming_fallbacks: '模型回退',
        streaming_thinking_removed: '移除思考模型',
        streaming_last_updated: '上次更新：{time}',
        streaming_empty: '暂无数据。',
        nav_routes: '路由总览（对外端点）',
        header_title: 'GCLI2API-Go 管理控制台',
        header_subtitle: '使用 Gemini Code Assist 作为唯一上游，提供 OpenAI 兼容 API 服务',
        btn_toggle_nav: '导航',
        badge_system_running: '运行中',
        badge_credentials: '凭证',
        badge_requests: '请求',
        badge_user: '用户',
        label_theme: '主题',
        label_language: '语言',
        label_auto_refresh: '自动刷新',
        option_theme_minimal: '极简风格',
        option_theme_gemini: 'Gemini 风格',
        option_theme_classic: '经典风格',
        lang_zh: '简体中文',
        user_status_authenticated: '已登录',
        status_loading: '加载中',
        status_loading_page: '正在加载…',
        page_loading: '加载中…',
        error_title: '加载失败',
        btn_reload_page: '重新加载',
        quick_switch_section_tabs: '标签页',
        quick_switch_section_credentials: '凭证',
        quick_switch_tab_meta: '快速切换',
        notify_network_online: '网络已恢复',
        notify_network_offline: '网络已断开',
        network_offline_banner: '网络离线，部分功能不可用',
        oauth_heading: 'Google 授权 (OAuth)',
        error_init_failed: '初始化失败',
        error_auth_required: '需要登录',
        tooltip_toggle_nav: '切换导航',
        user_status_none: '未登录',
        aria_close: '关闭',
        aria_main_nav: '主导航',
        quick_switch_placeholder: '搜索标签、凭证、模型…',
        quick_switch_hint: '方向键选择，Enter 打开',
        modal_quick_switch: '快速切换',
        quick_switch_no_results: '未找到匹配项，请尝试其他关键词',
        quick_switch_section_models: '模型',
        quick_switch_section_actions: '操作',
        quick_switch_open_assembly: '打开外部装配视图',
        quick_switch_open_routes: '打开路由总览',
        quick_switch_action_meta: '执行操作',
        quick_switch_model_meta: '模型 • {base} • {options}',
        quick_switch_cred_meta: '凭证 • {project} • 健康 {health}%',
        dashboard_quick_actions: '快捷操作',
        dashboard_system_info: '系统信息',
        system_version: '版本',
        system_go_version: 'Go 版本',
        system_openai_port: 'OpenAI 端口',
        system_admin_version: '管理前端',
        system_uptime: '运行时间',
        system_memory: '内存使用',
        dashboard_realtime_stats: '实时统计',
        dashboard_system_health: '系统健康状况',
        btn_export: '导出',
        config_router_cooldown_max_ms: '冷却最大时间（毫秒）',
        config_refresh_section: '刷新与凭证',
        config_refresh_ahead_seconds: '到期前刷新窗口（秒）',
        config_refresh_singleflight_timeout_sec: '并发刷新等待超时（秒）',
        config_streaming_section: '流式与抗截断',
        config_fake_streaming_enabled: '启用假流式',
        config_fake_streaming_chunk_size: '假流式文本分片大小',
        config_fake_streaming_delay_ms: '假流式分片延迟（毫秒）',
        config_anti_truncation_enabled: '启用流式抗截断',
        config_anti_truncation_max: '抗截断续写最大次数',
        config_headers_section: '指纹与头部',
        config_header_passthrough: '允许固定白名单头透传',
        config_misc_section: '其他',
        config_request_log_enabled: '启用请求日志',
        config_openai_images_include_mime: 'Images 响应包含 mime_type',
        config_tool_args_delta_chunk: '工具参数分片大小（字节）',
        config_auto_probe_section: '每日凭证健康检查（自动测活）',
        config_auto_probe_help: '每日按 UTC 小时触发，使用轻量级模型检测所有凭证可用性',
        config_auto_probe_hour: '触发小时（UTC 时区）',
        config_auto_probe_timeout: '超时时间（秒）',
        config_auto_probe_model: '测活使用的模型',
        config_auto_probe_toggle: '启用自动测活',
        config_preferred_section: '首选基础模型',
        config_preferred_label: '用于模型注册中心和快速发布的默认候选',
        config_preferred_hint: '按住 Ctrl/Cmd 可多选。这些模型将作为注册中心与快速发布页面的默认候选。',
        config_auto_probe_history_section: '测活历史',
        config_auto_probe_history_refresh: '刷新历史',
        config_auto_probe_history_download: '下载 JSON',
        config_auto_probe_history_empty: '暂无测活记录。',
        config_auto_probe_history_error: '加载测活记录失败',
        config_auto_probe_history_source_auto: '自动',
        config_auto_probe_history_source_manual: '手动',
        config_auto_probe_history_all_ok: '全部凭证健康',
        config_auto_probe_history_unknown: '未知',
        config_auto_probe_history_col_time: '时间',
        config_auto_probe_history_col_source: '来源',
        config_auto_probe_history_col_model: '模型',
        config_auto_probe_history_col_success: '成功率',
        config_auto_probe_history_col_duration: '耗时',
        config_auto_probe_history_col_detail: '详情',
        config_suggestion_upstream_missing: '已读取上游（仅显示缺失项）',
        config_suggestion_upstream_missing_hint: '成功获取上游目录，目前展示尚未注册的模型。',
        config_suggestion_upstream: '已读取上游目录',
        config_suggestion_upstream_hint: '成功获取上游模型清单。',
        config_suggestion_config: '使用配置中的候选',
        config_suggestion_config_hint: '上游暂不可用，使用配置文件中的 preferred_base_models。',
        config_suggestion_static: '使用内置默认列表',
        config_suggestion_static_hint: '使用内置的 Gemini 默认基础模型列表。',
        config_suggestion_static_error: '使用内置默认列表（上游不可用）',
        config_suggestion_static_error_hint: '上游和配置列表均不可用，已使用内置默认候选。',
        config_management_section: '管理安全',
        config_management_hash_label: '管理密钥哈希（bcrypt）',
        config_management_hash_placeholder: '$2b$… 哈希值',
        config_management_hash_hint: '粘贴管理密钥的 bcrypt 哈希，留空表示保持当前值。',
        config_save: '保存配置',
        config_restart_hint: '注意：部分配置保存后需要重启服务才会生效。',
        config_update_success: '配置已更新',
        config_update_failed: '保存配置失败',
        config_load_failed: '加载配置失败',
        placeholder_auto_probe_model: '如 gemini-2.5-flash',
        shortcut_title: '快捷键',
        shortcut_quick_switch: 'Ctrl/Cmd + K：快速切换标签',
        shortcut_refresh_tab: 'Ctrl/Cmd + R：刷新当前标签数据',
        shortcut_help: 'Shift + /：打开此帮助'
      }
    };

    // 增强UI功能
    this.notificationCenter = new NotificationCenter({
      escapeHTML: (value: string | null | undefined) => this.escapeHTML(value)
    });
    this.dialogs = new DialogManager();

    // Apply theme and language
    this.applyTheme('minimal');
    document.documentElement.setAttribute('lang', this.currentLang);

    // Initialize enhanced features
    this.notificationCenter.ensureContainer();
    this.createGlobalLoading();
    this.dialogs.ensureLegacyDialog();
  }

  /**
   * HTML转义函数
   */
  escapeHTML(str: string | null | undefined): string {
    if (str === null || str === undefined) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  loadTheme(): string | null {
    try { return localStorage.getItem(this.themeKey); } catch { return null; }
  }

  saveTheme(t: string): void {
    try { localStorage.setItem(this.themeKey, t); } catch {}
  }

  applyTheme(_name: string): void {
    // 极简黑白灰主题，忽略 name
    const root = document.documentElement;
    root.style.setProperty('--brand-start', '#f5f5f5');
    root.style.setProperty('--brand-end',   '#f5f5f5');
    root.style.setProperty('--primary',     '#111111');
    root.style.setProperty('--info',        '#111111');
    root.style.setProperty('--success',     '#10b981');
    root.style.setProperty('--warning',     '#f59e0b');
    root.style.setProperty('--danger',      '#ef4444');
    root.style.setProperty('--surface',     '#ffffff');
    root.style.setProperty('--muted',       '#6b7280');
    root.style.setProperty('--border',      '#e5e7eb');
    root.style.setProperty('--bg',          '#f5f6f7');
    this.currentTheme = 'minimal';
    try { this.saveTheme('minimal'); } catch {}
    document.body.style.background = 'var(--bg)';
  }

  loadLang(): string | null {
    try { return localStorage.getItem(this.langKey); } catch { return null; }
  }

  saveLang(l: string): void {
    try { localStorage.setItem(this.langKey, l); } catch {}
  }

  setLang(): void {}

  initLangSelect(): void {}

  t(key: string): string {
    // 兜底：即使以未绑定的函数方式调用（this 不是 UIHelper 实例），也能取到字典
    let dict: Record<string, Record<string, string>> = {};
    let lang = 'zh';
    try {
      // 优先使用绑定实例
      if (this && this.dict) { dict = this.dict; }
      else if (typeof window !== 'undefined' && window.ui) {
        const ui = window.ui as any;
        if (ui.dict) dict = ui.dict;
      }
      // 语言
      if (this && this.currentLang) { lang = this.currentLang; }
      else if (typeof window !== 'undefined' && window.ui) {
        const ui = window.ui as any;
        if (ui.currentLang) lang = ui.currentLang;
      }
    } catch (_) {}
    const d = (dict && (dict[lang] || dict.zh)) || {};
    // 优先当前语言；缺失则回退到英文；最后回退到 key
    return (d && d[key]) || (dict && dict.en && dict.en[key]) || key;
  }

  initThemeSelect(selectEl: HTMLSelectElement | null): void {
    if (selectEl && selectEl.parentElement) selectEl.parentElement.style.display = 'none';
  }

  banner(id: string, type: string, text: string): void {
    let bar = document.getElementById(id);
    if (!bar) {
      bar = document.createElement('div');
      bar.id = id;
      bar.className = `banner banner-${type}`;
      document.body.appendChild(bar);
    }
    bar.textContent = text;
    bar.style.display = 'block';
  }

  hideBanner(id: string): void {
    const bar = document.getElementById(id);
    if (bar) bar.style.display = 'none';
  }

  /**
   * 显示确认对话框
   */
  confirm(title: string, message: string, options: any = {}): Promise<boolean> {
    return this.dialogs.confirm(title, message, options);
  }

  /**
   * 获取确认框图标
   */
  getConfirmIcon(type: string): string {
    return this.dialogs.getConfirmIcon(type);
  }

  /**
   * 显示简单确认框（快捷方法）
   */
  confirmDelete(itemName: string = '此项目'): Promise<boolean> {
    return this.dialogs.confirmDelete(itemName);
  }

  /**
   * 显示警告确认框
   */
  confirmWarning(title: string, message: string, options: any = {}): Promise<boolean> {
    return this.confirm(title, message, {
      type: 'warning',
      okText: '继续',
      okClass: 'btn-warning',
      ...options
    });
  }

  // ========================================
  // 增强UI功能 (合并自enhanced_ui.js)
  // ========================================

  // ====== 基础工具 ======
  debounce(fn: Function, wait: number = 200): Function {
    let t: ReturnType<typeof setTimeout> | null = null;
    return (...args: any[]) => {
      if (t) clearTimeout(t);
      t = setTimeout(() => fn.apply(this, args), wait);
    };
  }

  getHashParams(): { path: string; params: Record<string, string> } {
    try {
      const raw = (location.hash || '').replace(/^#/, '');
      const [path, query] = raw.includes('?') ? raw.split('?') : [raw, ''];
      const params = new URLSearchParams(query);
      const obj: Record<string, string> = {};
      params.forEach((v, k) => { obj[k] = v; });
      return { path, params: obj };
    } catch {
      return { path: '', params: {} };
    }
  }

  setHashParams(patch: Record<string, any> = {}, options: { path?: string } = {}): void {
    try {
      const { path, params } = this.getHashParams();
      const next = { ...params, ...patch };
      // 删除值为空的键
      Object.keys(next).forEach(k => {
        if (next[k] === '' || next[k] == null) delete next[k];
      });
      const qs = new URLSearchParams(next).toString();
      const basePath = options.path || path || '';
      const nextHash = qs ? `#${basePath}?${qs}` : `#${basePath}`;
      if (location.hash !== nextHash) location.hash = nextHash;
    } catch {}
  }

  // ====== 轻量组件 ======
  renderSkeleton(lines: number = 3): string {
    const n = Math.max(1, Math.min(10, lines | 0));
    return `<div class="skeleton">${Array.from({ length: n }).map(() => '<div class="sk-line"></div>').join('')}</div>`;
  }

  renderEmpty(title: string = '暂无数据', hint: string = ''): string {
    return `<div class="empty"><div class="empty-icon">🗂️</div><div class="empty-title">${title}</div>${hint ? `<div class="empty-hint">${hint}</div>` : ''}</div>`;
  }

  renderErrorCard(msg: string = '加载失败', detail: string = ''): string {
    const d = detail ? `<div class="err-detail">${this.escapeHTML(detail)}</div>` : '';
    return `<div class="error-card"><div class="err-icon">⚠️</div><div class="err-title">${this.escapeHTML(msg)}</div>${d}<div class="err-actions"><button class="btn" onclick="location.reload()">重试</button></div></div>`;
  }

  /**
   * 创建全局加载覆盖层
   */
  createGlobalLoading(): void {
    if (document.querySelector('.global-loading')) return;

    const loading = document.createElement('div');
    loading.className = 'global-loading';
    loading.id = 'global-loading';
    loading.innerHTML = `
        <div class="spinner"></div>
        <div class="loading-text">加载中...</div>
    `;
    document.body.appendChild(loading);
    this.globalLoading = loading;
  }

  /**
   * 显示通知
   */
  showNotification(type: string = 'info', title: string = '', message: string = '', options: any = {}): string {
    return this.notificationCenter.show(type, title, message, options);
  }

  /**
   * 显示进度通知
   */
  showProgressNotification(title: string, message: string = '', options: any = {}): string {
    return this.notificationCenter.showProgress(title, message, options);
  }

  /**
   * 移除通知
   */
  removeNotification(id: string): void {
    this.notificationCenter.remove(id);
  }

  /**
   * 获取通知图标
   */
  getNotificationIcon(type: string): string {
    return this.notificationCenter.getIcon(type);
  }

  /**
   * 显示全局加载
   */
  showGlobalLoading(text = '加载中...') {
    if (!this.globalLoading) return;
    
    const loadingText = this.globalLoading.querySelector('.loading-text');
    if (loadingText) {
        loadingText.textContent = text;
    }
    
    this.globalLoading.classList.add('active');
    document.body.style.overflow = 'hidden';
  }

  /**
   * 隐藏全局加载
   */
  hideGlobalLoading() {
    if (!this.globalLoading) return;
    
    this.globalLoading.classList.remove('active');
    document.body.style.overflow = '';
  }

  /**
   * 显示确认对话框
   */
  showConfirmation(options: any = {}): void {
    this.dialogs.showLegacy(options.title || '', options.content || '');
  }

  /**
   * 隐藏确认对话框
   */
  hideConfirmation(): void {
    this.dialogs.hideLegacy();
  }

  /**
   * 显示通用模态框
   */
  showModal(title: string, contentHtml: string): HTMLElement | null {
    try {
      const modal = document.createElement('div');
      modal.className = 'modal active';
      modal.innerHTML = `
        <div class="modal-content" role="dialog" aria-modal="true" aria-label="${this.escapeHTML(title||'详情')}">
          <button type="button" class="modal-close" aria-label="关闭">&times;</button>
          <div class="modal-header">${this.escapeHTML(title||'详情')}</div>
          <div class="modal-body">${contentHtml||''}</div>
        </div>`;
      document.body.appendChild(modal);
      const close = ()=>{ try{ modal.remove(); document.body.style.overflow=''; }catch(_){} };
      try{ modal.addEventListener('click', (e)=>{ if(e.target===modal) close(); }); }catch(_){}
      try{ const btn = modal.querySelector('.modal-close'); if(btn) btn.addEventListener('click', close); }catch(_){}
      document.body.style.overflow = 'hidden';
      return modal;
    } catch(_) { return null; }
  }

  /**
   * 展示上游错误详情（最小加工，便于复制原文）
   */
  showErrorDetails(info: any = {}): void {
    const safe = (v: any) => this.escapeHTML(String(v == null ? '' : v));
    const hdrs = info.headers || {};
    const headersHtml = Object.keys(hdrs).length
      ? `<pre style="white-space:pre-wrap;word-break:break-all;">${this.escapeHTML(JSON.stringify(hdrs, null, 2))}</pre>`
      : '<div class="muted">(无响应头)</div>';
    const raw = typeof info.text === 'string' && info.text ? info.text
               : (info.payload ? JSON.stringify(info.payload, null, 2) : '');
    const bodyHtml = raw
      ? `<pre style="white-space:pre-wrap;word-break:break-all;">${this.escapeHTML(raw)}</pre>`
      : '<div class="muted">(无响应体)</div>';
    let retryRow = '';
    if (info.retryAfter !== undefined && info.retryAfter !== null && info.retryAfter !== '') {
      const seconds = Number(info.retryAfter);
      const display = Number.isFinite(seconds) ? `${seconds} 秒后重试` : `${info.retryAfter}`;
      retryRow = `<div class="muted">建议等待</div><div>${this.escapeHTML(display)}</div>`;
    }
    const metaHtml = `
      <div class="meta-grid" style="display:grid;grid-template-columns:120px 1fr;gap:6px 12px;margin-bottom:8px;">
        <div class="muted">状态码</div><div>${safe(info.status || '未知')}</div>
        <div class="muted">路径</div><div>${safe(info.path || info.url || '-')}</div>
        <div class="muted">时间</div><div>${new Date().toLocaleString()}</div>
        ${retryRow}
      </div>`;
    const content = `
      ${metaHtml}
      <div style="display:flex;gap:8px;flex-wrap:wrap;margin:8px 0;">
        <button class="btn btn-secondary btn-sm" id="copyErrBody">复制错误原文</button>
        <button class="btn btn-secondary btn-sm" id="copyErrHeaders">复制响应头</button>
      </div>
      <h4 style="margin:8px 0 4px;">响应头</h4>
      ${headersHtml}
      <h4 style="margin:8px 0 4px;">错误原文</h4>
      ${bodyHtml}
    `;
    const modal = this.showModal('上游错误详情', content);
    if (modal){
      const textToCopy = raw || '';
      try{
        const btnA = modal.querySelector('#copyErrBody');
        if (btnA) btnA.addEventListener('click', async ()=>{
          try{
            await navigator.clipboard.writeText(textToCopy);
            this.showNotification('success','已复制','错误原文已复制到剪贴板');
          } catch(e: unknown){
            const errorMsg = e instanceof Error ? e.message : String(e);
            alert('复制失败: '+ errorMsg);
          }
        });
      }catch(_){ }
      try{
        const btnB = modal.querySelector('#copyErrHeaders');
        if (btnB) btnB.addEventListener('click', async ()=>{
          try{
            await navigator.clipboard.writeText(JSON.stringify(hdrs, null, 2));
            this.showNotification('success','已复制','响应头已复制到剪贴板');
          } catch(e: unknown){
            const errorMsg = e instanceof Error ? e.message : String(e);
            alert('复制失败: '+ errorMsg);
          }
        });
      }catch(_){ }
    }
  }

  /**
   * 包装异步操作，自动显示加载状态
   */
  async withLoading<T>(asyncFn: () => Promise<T>, options: any = {}): Promise<T> {
    const {
        loadingText = '处理中...',
        successMessage = '',
        errorMessage = '操作失败',
        button = null
    } = options;

    try {
        if (button) {
            this.setButtonLoading(button, true);
        } else {
            this.showGlobalLoading(loadingText);
        }

        const result = await asyncFn();

        if (successMessage) {
            this.showNotification(successMessage, 'success');
        }

        return result;
    } catch (error: unknown) {
        console.error('异步操作失败:', error);
        const errorMsg = error instanceof Error ? error.message : String(error);
        this.showNotification(errorMessage + ': ' + errorMsg, 'error');
        throw error;
    } finally {
        if (button) {
            this.setButtonLoading(button, false);
        } else {
            this.hideGlobalLoading();
        }
    }
  }

  /**
   * 设置按钮加载状态
   */
  setButtonLoading(button: HTMLButtonElement | null, loading: boolean): void {
    if (!button) return;

    if (loading) {
        button.classList.add('loading');
        button.disabled = true;
    } else {
        button.classList.remove('loading');
        button.disabled = false;
    }
  }

  // ========================================
  // 数据刷新和状态管理
  // ========================================

  // Static properties
  private static _eventBus: any;
  private static _refreshManager: any;

  /**
   * 数据刷新事件管理器
   */
  static createEventBus(): any {
    return createEventBusService();
  }

  /**
   * 全局事件总线实例
   */
  static get eventBus(): any {
    if (!this._eventBus) {
      this._eventBus = this.createEventBus();
    }
    return this._eventBus;
  }

  /**
   * 节流函数
   */
  static throttle<T extends (...args: any[]) => any>(func: T, delay: number): (...args: Parameters<T>) => void {
    return throttleFn(func, delay);
  }

  /**
   * 防抖函数
   */
  static debounce<T extends (...args: any[]) => any>(func: T, delay: number): (...args: Parameters<T>) => void {
    return debounceFn(func, delay);
  }

  /**
   * 智能缓存管理器
   */
  static createCacheManager(options: any = {}): any {
    return createCacheService(options);
  }

  /**
   * 数据刷新管理器
   */
  static createRefreshManager(): any {
    return createRefreshService({
      eventBus: UIHelper.eventBus,
      cacheFactory: (opts: any) => createCacheService(opts),
      throttleFn: throttleFn as any
    });
  }

  /**
   * 全局刷新管理器实例
   */
  static get refreshManager(): any {
    if (!this._refreshManager) {
      this._refreshManager = this.createRefreshManager();
    }
    return this._refreshManager;
  }

  /**
   * 兼容旧调用：showAlert(type, title, message)
   */
  showAlert(type: string = 'info', title: string = '', message: string = ''): void {
    try {
      // 允许两参：showAlert('error','内容')
      if (!message && title) {
        this.showNotification(type, type === 'error' ? '错误' : type === 'warning' ? '提示' : '消息', title);
      } else {
        this.showNotification(type, title, message);
      }
    } catch (e) {
      try { alert(message || title || String(type)); } catch {}
    }
  }

}

export const ui = new UIHelper();
(window as any).ui = ui;
