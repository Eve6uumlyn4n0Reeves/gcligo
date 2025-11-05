/**
 * 组件相关类型定义
 * 定义所有 UI 组件的接口和类型
 */

/**
 * 通知中心接口
 */
export interface NotificationCenter {
  /**
   * 显示通知
   */
  show(
    type: NotificationType,
    title?: string,
    message?: string,
    options?: NotificationOptions
  ): string;

  /**
   * 关闭通知
   */
  close(id: string): void;

  /**
   * 关闭所有通知
   */
  closeAll(): void;

  /**
   * 确保容器存在
   */
  ensureContainer(): void;
}

/**
 * 通知选项
 */
export interface NotificationOptions {
  duration?: number;
  closable?: boolean;
  onClose?: () => void;
  progress?: boolean;
  actions?: NotificationAction[];
}

/**
 * 通知操作
 */
export interface NotificationAction {
  id: string;
  text: string;
  handler: () => void | Promise<void>;
}

/**
 * 对话框管理器接口
 */
export interface DialogManager {
  /**
   * 显示对话框
   */
  show(options: DialogOptions): Promise<DialogResult>;

  /**
   * 显示确认对话框
   */
  confirm(title: string, message: string, options?: ConfirmOptions): Promise<boolean>;

  /**
   * 显示提示对话框
   */
  prompt(title: string, message: string, defaultValue?: string): Promise<string | null>;

  /**
   * 显示警告对话框
   */
  alert(title: string, message: string): Promise<void>;

  /**
   * 关闭对话框
   */
  close(id?: string): void;

  /**
   * 关闭所有对话框
   */
  closeAll(): void;

  /**
   * 显示旧版对话框（兼容性）
   */
  showLegacyDialog(title: string, content: string): void;

  /**
   * 关闭旧版对话框
   */
  closeLegacyDialog(): void;
}

/**
 * 对话框结果
 */
export interface DialogResult {
  confirmed: boolean;
  value?: string;
  action?: string;
}

/**
 * 确认对话框选项
 */
export interface ConfirmOptions {
  type?: 'info' | 'warning' | 'danger';
  confirmText?: string;
  cancelText?: string;
  confirmClass?: string;
}

/**
 * 对话框按钮
 */
export interface DialogButton {
  text: string;
  type?: 'primary' | 'secondary' | 'danger';
  onClick?: () => void | Promise<void>;
}

/**
 * 加载指示器接口
 */
export interface LoadingIndicator {
  /**
   * 显示加载指示器
   */
  show(message?: string): void;

  /**
   * 隐藏加载指示器
   */
  hide(): void;

  /**
   * 更新加载消息
   */
  updateMessage(message: string): void;

  /**
   * 是否正在显示
   */
  isShowing(): boolean;
}

/**
 * 进度条接口
 */
export interface ProgressBar {
  /**
   * 设置进度
   */
  setProgress(value: number): void;

  /**
   * 获取进度
   */
  getProgress(): number;

  /**
   * 显示进度条
   */
  show(): void;

  /**
   * 隐藏进度条
   */
  hide(): void;

  /**
   * 重置进度
   */
  reset(): void;
}

/**
 * 标签页管理器接口
 */
export interface TabManager {
  /**
   * 切换到指定标签页
   */
  switchTo(tabId: string): void;

  /**
   * 注册标签页
   */
  register(tabId: string, handler: TabHandler): void;

  /**
   * 注销标签页
   */
  unregister(tabId: string): void;

  /**
   * 获取当前标签页
   */
  getCurrentTab(): string;

  /**
   * 获取所有标签页
   */
  getAllTabs(): string[];
}

/**
 * 标签页处理器
 */
export interface TabHandler {
  /**
   * 渲染标签页内容
   */
  render(container: HTMLElement): void;

  /**
   * 刷新标签页数据
   */
  refresh?(): Promise<void>;

  /**
   * 销毁标签页
   */
  destroy?(): void;

  /**
   * 标签页激活时调用
   */
  onActivate?(): void;

  /**
   * 标签页停用时调用
   */
  onDeactivate?(): void;
}

/**
 * 表格组件接口
 */
export interface TableComponent<T = any> {
  /**
   * 设置数据
   */
  setData(data: T[]): void;

  /**
   * 获取数据
   */
  getData(): T[];

  /**
   * 刷新表格
   */
  refresh(): void;

  /**
   * 排序
   */
  sort(column: string, order: 'asc' | 'desc'): void;

  /**
   * 过滤
   */
  filter(predicate: (item: T) => boolean): void;

  /**
   * 获取选中的行
   */
  getSelectedRows(): T[];

  /**
   * 清除选择
   */
  clearSelection(): void;
}

/**
 * 表单组件接口
 */
export interface FormComponent {
  /**
   * 获取表单值
   */
  getValues(): Record<string, any>;

  /**
   * 设置表单值
   */
  setValues(values: Record<string, any>): void;

  /**
   * 验证表单
   */
  validate(): boolean;

  /**
   * 获取验证错误
   */
  getErrors(): Record<string, string>;

  /**
   * 重置表单
   */
  reset(): void;

  /**
   * 提交表单
   */
  submit(): Promise<void>;
}

/**
 * 下拉菜单接口
 */
export interface DropdownMenu {
  /**
   * 显示菜单
   */
  show(anchor: HTMLElement): void;

  /**
   * 隐藏菜单
   */
  hide(): void;

  /**
   * 切换显示状态
   */
  toggle(anchor: HTMLElement): void;

  /**
   * 添加菜单项
   */
  addItem(item: DropdownMenuItem): void;

  /**
   * 移除菜单项
   */
  removeItem(id: string): void;
}

/**
 * 下拉菜单项
 */
export interface DropdownMenuItem {
  id: string;
  text: string;
  icon?: string;
  disabled?: boolean;
  onClick?: () => void | Promise<void>;
  separator?: boolean;
}

/**
 * 工具提示接口
 */
export interface Tooltip {
  /**
   * 显示工具提示
   */
  show(element: HTMLElement, content: string, options?: TooltipOptions): void;

  /**
   * 隐藏工具提示
   */
  hide(): void;
}

/**
 * 工具提示选项
 */
export interface TooltipOptions {
  position?: 'top' | 'bottom' | 'left' | 'right';
  delay?: number;
  maxWidth?: string;
}

/**
 * 侧边栏接口
 */
export interface Sidebar {
  /**
   * 显示侧边栏
   */
  show(): void;

  /**
   * 隐藏侧边栏
   */
  hide(): void;

  /**
   * 切换侧边栏
   */
  toggle(): void;

  /**
   * 是否显示
   */
  isVisible(): boolean;
}

