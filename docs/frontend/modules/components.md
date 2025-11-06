# Components 模块文档

## 模块定位与职责

Components 模块是 GCLI2API-Go 前端的**UI 组件库**，提供可复用的 UI 组件，包括通知中心、对话框管理器等核心交互组件。

### 核心职责

1. **通知管理**：提供 Toast 通知、进度通知、自动关闭等功能
2. **对话框管理**：提供模态对话框、确认对话框、删除确认等功能
3. **HTML 安全**：防止 XSS 攻击，自动转义用户输入
4. **生命周期管理**：自动创建/销毁 DOM 元素，管理组件状态
5. **事件处理**：支持自定义事件回调（onClose、onClick）
6. **无障碍支持**：提供 ARIA 标签，支持键盘操作
7. **向后兼容**：提供 Legacy API 兼容旧代码

---

## 目录结构与文件职责

```
web/src/components/
├── notification.ts  # 通知中心组件（129 行）- Toast 通知、进度通知
└── dialog.ts        # 对话框管理器（215 行）- 模态对话框、确认对话框
```

### 文件职责说明

| 文件 | 核心职责 | 关键类 | 主要方法 |
|------|---------|--------|---------|
| **notification.ts** | 通知中心管理 | `NotificationCenter` | `show`、`showProgress`、`remove`、`clearAll` |
| **dialog.ts** | 对话框管理 | `DialogManager` | `open`、`close`、`confirm`、`confirmDelete` |

---

## 核心设计与数据流

### 1. 通知中心架构

```
用户调用 show()
    ↓
ensureContainer() - 确保容器存在
    ↓
创建通知元素（notification）
    ↓
设置类型、标题、消息、图标
    ↓
添加到 DOM（container.appendChild）
    ↓
存储到 Map（notifications.set）
    ↓
设置自动关闭定时器（setTimeout）
    ↓
绑定关闭按钮事件
    ↓
返回通知 ID
```

### 2. 对话框生命周期

```
用户调用 open()
    ↓
ensureLegacyDialog() - 确保对话框容器存在
    ↓
设置标题、内容、按钮
    ↓
显示对话框（display: flex）
    ↓
禁用页面滚动（overflow: hidden）
    ↓
用户交互（点击按钮/背景/关闭按钮）
    ↓
触发回调（onClick）
    ↓
关闭对话框（display: none）
    ↓
恢复页面滚动（overflow: ''）
```

### 3. 确认对话框流程

```
用户调用 confirm()
    ↓
创建 Promise
    ↓
构建按钮数组（取消、确定）
    ↓
调用 open() 显示对话框
    ↓
用户点击按钮
    ↓
触发 onClick 回调
    ↓
resolve(true/false)
    ↓
关闭对话框
    ↓
返回 Promise 结果
```

### 4. HTML 安全处理

```
用户输入内容
    ↓
escapeHTML() 转义特殊字符
    ↓
< → &lt;
> → &gt;
& → &amp;
" → &quot;
' → &#x27;
    ↓
安全的 HTML 字符串
    ↓
插入到 DOM
```

---

## 关键类型与接口

### 1. NotificationCenter 类

```typescript
export class NotificationCenter {
  private container: HTMLElement | null = null;
  private notifications: Map<string, HTMLElement> = new Map();
  private escapeHTML: (value: string | null | undefined) => string;

  constructor(options: { escapeHTML: (value: string | null | undefined) => string });
  
  ensureContainer(): void;                                    // 确保容器存在
  show(type, title, message, options): string;                // 显示通知
  showProgress(title, message, options): string;              // 显示进度通知
  remove(id: string): void;                                   // 移除通知
  getIcon(type: string): string;                              // 获取图标
  clearAll(): void;                                           // 清除所有通知
  close(id: string): void;                                    // 向后兼容别名
}
```

### 2. NotificationOptions 接口

```typescript
export interface NotificationOptions {
  duration?: number;        // 自动关闭时长（毫秒），0 表示不自动关闭
  closable?: boolean;       // 是否显示关闭按钮
  onClose?: () => void;     // 关闭回调
  progress?: boolean;       // 是否显示进度条
}
```

### 3. DialogManager 类

```typescript
export class DialogManager {
  private dialogs: Map<string, HTMLElement> = new Map();
  private legacyDialog: HTMLElement | null = null;

  ensureLegacyDialog(): void;                                 // 确保对话框容器存在
  open(id, title, content, options): void;                    // 打开对话框
  close(id: string): void;                                    // 关闭对话框
  closeAll(): void;                                           // 关闭所有对话框
  isOpen(): boolean;                                          // 检查是否有对话框打开
  confirm(title, message, options): Promise<boolean>;         // 确认对话框
  confirmDelete(itemName): Promise<boolean>;                  // 删除确认对话框
  getConfirmIcon(type: string): string;                       // 获取确认图标
  
  // Legacy API
  showLegacy(title, contentHtml): void;
  hideLegacy(): void;
  showLegacyDialog(title, content, options): void;
  hideLegacyDialog(): void;
}
```

### 4. DialogOptions 接口

```typescript
export interface DialogOptions {
  width?: string;           // 对话框宽度
  height?: string;          // 对话框高度
  closable?: boolean;       // 是否可关闭
  backdrop?: boolean;       // 是否显示背景遮罩
  onClose?: () => void;     // 关闭回调
  buttons?: DialogButton[]; // 按钮数组
  allowHTML?: boolean;      // 是否允许 HTML（默认 false，使用 textContent）
}
```

### 5. DialogButton 接口

```typescript
export interface DialogButton {
  text: string;                           // 按钮文本
  type?: 'primary' | 'secondary' | 'danger'; // 按钮类型
  onClick?: () => void;                   // 点击回调
}
```

---

## 重要配置项

### NotificationCenter 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `duration` | `number` | `5000` | 自动关闭时长（毫秒），0 表示不自动关闭 |
| `closable` | `boolean` | `true` | 是否显示关闭按钮 |
| `progress` | `boolean` | `false` | 是否显示进度条 |

### DialogManager 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `allowHTML` | `boolean` | `false` | 是否允许 HTML（默认使用 textContent） |
| `closable` | `boolean` | `true` | 是否可关闭 |
| `backdrop` | `boolean` | `true` | 是否显示背景遮罩 |

---

## 与其他模块的依赖关系

### 依赖的模块

无直接依赖（纯 UI 组件）

### 被依赖的模块

Components 模块被以下模块依赖：

- **UI 模块**：通过 `ui.ts` 导入并封装为全局 UI 工具
- **Admin 模块**：使用通知和对话框显示消息
- **Creds 模块**：使用确认对话框进行批量操作确认
- **API 模块**：使用通知显示错误消息

---

## 可执行示例

### 示例 1：显示成功通知

```typescript
import { NotificationCenter } from './components/notification';

// 创建通知中心
const escapeHTML = (str: string | null | undefined) => 
  (str || '').replace(/[&<>"']/g, (m) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#x27;'
  }[m] || m));

const notificationCenter = new NotificationCenter({ escapeHTML });

// 显示成功通知
const id = notificationCenter.show(
  'success',
  '操作成功',
  '凭证已成功启用',
  { duration: 3000 }
);

// 手动关闭
setTimeout(() => {
  notificationCenter.remove(id);
}, 1000);
```

### 示例 2：显示错误通知

```typescript
// 显示错误通知（不自动关闭）
const id = notificationCenter.show(
  'error',
  '操作失败',
  '凭证启用失败：网络错误',
  { duration: 0, closable: true }
);
```

### 示例 3：显示进度通知

```typescript
// 显示进度通知
const progressId = notificationCenter.showProgress(
  '批量操作进行中',
  '正在处理 50 个凭证...'
);

// 操作完成后关闭
setTimeout(() => {
  notificationCenter.remove(progressId);
  notificationCenter.show('success', '批量操作完成', '成功处理 50 个凭证');
}, 5000);
```

### 示例 4：打开对话框

```typescript
import { DialogManager } from './components/dialog';

const dialogManager = new DialogManager();

// 打开对话框
dialogManager.open(
  'my-dialog',
  '凭证详情',
  '这是凭证的详细信息...',
  {
    width: '600px',
    buttons: [
      { text: '关闭', type: 'secondary', onClick: () => console.log('关闭') }
    ]
  }
);
```

### 示例 5：确认对话框

```typescript
// 显示确认对话框
const confirmed = await dialogManager.confirm(
  '确认操作',
  '确定要启用此凭证吗？',
  {
    okText: '启用',
    cancelText: '取消'
  }
);

if (confirmed) {
  console.log('用户确认');
} else {
  console.log('用户取消');
}
```

---

### 示例 6：删除确认对话框

```typescript
// 显示删除确认对话框
const confirmed = await dialogManager.confirmDelete('凭证 user@example.com');

if (confirmed) {
  // 执行删除操作
  await credentialsApi.deleteCredential('cred-001');
  notificationCenter.show('success', '删除成功', '凭证已删除');
} else {
  console.log('用户取消删除');
}
```

### 示例 7：自定义按钮对话框

```typescript
// 显示自定义按钮对话框
dialogManager.open(
  'custom-dialog',
  '批量操作',
  '选择要执行的操作：',
  {
    buttons: [
      {
        text: '启用',
        type: 'primary',
        onClick: () => {
          console.log('批量启用');
        }
      },
      {
        text: '禁用',
        type: 'secondary',
        onClick: () => {
          console.log('批量禁用');
        }
      },
      {
        text: '删除',
        type: 'danger',
        onClick: () => {
          console.log('批量删除');
        }
      }
    ]
  }
);
```

### 示例 8：通知类型图标

```typescript
// 获取不同类型的图标
const successIcon = notificationCenter.getIcon('success'); // ✓
const errorIcon = notificationCenter.getIcon('error');     // ✕
const warningIcon = notificationCenter.getIcon('warning'); // ⚠
const infoIcon = notificationCenter.getIcon('info');       // ℹ

console.log(successIcon, errorIcon, warningIcon, infoIcon);
```

### 示例 9：清除所有通知

```typescript
// 显示多个通知
notificationCenter.show('info', '通知 1', '消息 1');
notificationCenter.show('info', '通知 2', '消息 2');
notificationCenter.show('info', '通知 3', '消息 3');

// 清除所有通知
setTimeout(() => {
  notificationCenter.clearAll();
}, 2000);
```

### 示例 10：检查对话框状态

```typescript
// 打开对话框
dialogManager.open('test-dialog', '测试', '内容');

// 检查是否有对话框打开
if (dialogManager.isOpen()) {
  console.log('有对话框打开');
}

// 关闭所有对话框
dialogManager.closeAll();

// 再次检查
if (!dialogManager.isOpen()) {
  console.log('所有对话框已关闭');
}
```

---

## 架构示意图

```mermaid
graph TB
    subgraph "通知中心架构"
        A[调用 show] --> B[ensureContainer]
        B --> C[创建通知元素]
        C --> D[设置类型/标题/消息]
        D --> E[添加图标]
        E --> F[添加关闭按钮]
        F --> G[添加到 DOM]
        G --> H[存储到 Map]
        H --> I{duration > 0?}
        I -->|是| J[设置自动关闭定时器]
        I -->|否| K[不自动关闭]
        J --> L[返回通知 ID]
        K --> L
    end

    subgraph "对话框生命周期"
        M[调用 open] --> N[ensureLegacyDialog]
        N --> O[设置标题]
        O --> P{allowHTML?}
        P -->|是| Q[innerHTML 设置内容]
        P -->|否| R[textContent 设置内容]
        Q --> S[渲染按钮]
        R --> S
        S --> T[显示对话框]
        T --> U[禁用页面滚动]
        U --> V[等待用户交互]
        V --> W[触发回调]
        W --> X[关闭对话框]
        X --> Y[恢复页面滚动]
    end

    subgraph "确认对话框流程"
        Z[调用 confirm] --> AA[创建 Promise]
        AA --> AB[构建按钮数组]
        AB --> AC[取消按钮 → resolve false]
        AB --> AD[确定按钮 → resolve true]
        AC --> AE[调用 open]
        AD --> AE
        AE --> AF[用户点击按钮]
        AF --> AG{点击哪个?}
        AG -->|取消| AH[resolve false]
        AG -->|确定| AI[resolve true]
        AH --> AJ[关闭对话框]
        AI --> AJ
        AJ --> AK[返回 Promise]
    end

    subgraph "HTML 安全处理"
        AL[用户输入] --> AM[escapeHTML]
        AM --> AN["< → &lt;"]
        AM --> AO["> → &gt;"]
        AM --> AP["& → &amp;"]
        AM --> AQ["\" → &quot;"]
        AM --> AR["' → &#x27;"]
        AN --> AS[安全的 HTML]
        AO --> AS
        AP --> AS
        AQ --> AS
        AR --> AS
        AS --> AT[插入到 DOM]
    end
```

## 已知限制

### 1. 通知容器位置固定
**限制**：通知容器位置硬编码（右上角）
**影响**：无法自定义通知位置
**解决方案**：通过 CSS 变量或配置项支持自定义位置

### 2. 对话框不支持嵌套
**限制**：Legacy 对话框只支持单个实例
**影响**：无法同时显示多个对话框
**解决方案**：使用 `dialogs` Map 支持多对话框

### 3. 通知无优先级队列
**限制**：通知按时间顺序显示
**影响**：重要通知可能被淹没
**解决方案**：实现优先级队列

### 4. 对话框无动画
**限制**：对话框显示/隐藏无过渡动画
**影响**：用户体验不够流畅
**解决方案**：使用 CSS transition 或 animation

### 5. 通知无分组功能
**限制**：相同类型的通知无法分组
**影响**：大量通知时界面混乱
**解决方案**：实现通知分组和折叠

### 6. 对话框无拖拽功能
**限制**：对话框位置固定，无法拖拽
**影响**：用户无法调整对话框位置
**解决方案**：实现拖拽功能

### 7. 通知无持久化
**限制**：刷新页面后通知消失
**影响**：用户可能错过重要通知
**解决方案**：使用 localStorage 持久化通知

### 8. 对话框无键盘导航
**限制**：对话框不支持 Tab 键导航
**影响**：键盘用户体验不佳
**解决方案**：实现焦点管理和键盘导航

---

## 最佳实践

### 1. 使用类型安全的通知
**建议**：使用 TypeScript 类型定义
**原因**：提供编译时类型检查
**示例**：
```typescript
// 推荐
const options: NotificationOptions = {
  duration: 3000,
  closable: true
};
notificationCenter.show('success', '标题', '消息', options);

// 不推荐
notificationCenter.show('success', '标题', '消息', { duration: 3000 }); // 无类型检查
```

### 2. 始终转义用户输入
**建议**：使用 escapeHTML 转义用户输入
**原因**：防止 XSS 攻击
**示例**：
```typescript
// 推荐
const userInput = '<script>alert("XSS")</script>';
const escaped = escapeHTML(userInput);
notificationCenter.show('info', '用户输入', escaped);

// 不推荐
notificationCenter.show('info', '用户输入', userInput); // 可能导致 XSS
```

### 3. 使用合理的通知时长
**建议**：根据消息重要性设置时长
**原因**：提升用户体验
**示例**：
```typescript
// 成功消息：3 秒
notificationCenter.show('success', '操作成功', '凭证已启用', { duration: 3000 });

// 错误消息：不自动关闭
notificationCenter.show('error', '操作失败', '网络错误', { duration: 0 });

// 信息消息：5 秒
notificationCenter.show('info', '提示', '正在处理...', { duration: 5000 });
```

### 4. 使用确认对话框防止误操作
**建议**：危险操作前显示确认对话框
**原因**：防止误操作
**示例**：
```typescript
// 推荐
const confirmed = await dialogManager.confirmDelete('凭证 user@example.com');
if (confirmed) {
  await deleteCredential();
}

// 不推荐
await deleteCredential(); // 直接删除，无确认
```

### 5. 清理不再需要的通知
**建议**：及时清理通知
**原因**：避免内存泄漏
**示例**：
```typescript
// 推荐
const id = notificationCenter.show('info', '加载中', '正在加载...');
await loadData();
notificationCenter.remove(id);

// 不推荐
notificationCenter.show('info', '加载中', '正在加载...', { duration: 0 }); // 永不关闭
```

### 6. 使用语义化的按钮类型
**建议**：根据操作类型选择按钮类型
**原因**：提升可读性和用户体验
**示例**：
```typescript
// 推荐
dialogManager.open('dialog', '操作', '内容', {
  buttons: [
    { text: '取消', type: 'secondary' },
    { text: '确定', type: 'primary' },
    { text: '删除', type: 'danger' }
  ]
});
```

### 7. 提供有意义的通知标题
**建议**：通知标题简洁明了
**原因**：用户快速理解通知内容
**示例**：
```typescript
// 推荐
notificationCenter.show('success', '凭证启用成功', 'user@example.com 已启用');

// 不推荐
notificationCenter.show('success', '成功', '操作成功'); // 标题不明确
```

### 8. 使用 Promise 处理确认对话框
**建议**：使用 async/await 处理确认对话框
**原因**：代码更清晰
**示例**：
```typescript
// 推荐
const confirmed = await dialogManager.confirm('确认', '确定吗？');
if (confirmed) {
  await performAction();
}
```

### 9. 关闭对话框前执行清理
**建议**：关闭对话框前执行必要的清理
**原因**：避免内存泄漏
**示例**：
```typescript
// 推荐
dialogManager.open('dialog', '标题', '内容', {
  onClose: () => {
    // 清理资源
    clearInterval(intervalId);
  }
});
```

### 10. 使用通知中心单例
**建议**：全局使用单个通知中心实例
**原因**：避免多个容器
**示例**：
```typescript
// 推荐
// ui.ts
export const notificationCenter = new NotificationCenter({ escapeHTML });

// 其他文件
import { notificationCenter } from './ui';
notificationCenter.show('info', '标题', '消息');
```

---

## 常见问题

### Q1: 如何自定义通知位置？
**A**: 通过 CSS 修改通知容器位置：
```css
.notification-center {
  position: fixed;
  top: 20px;      /* 修改为 bottom: 20px 可以显示在底部 */
  right: 20px;    /* 修改为 left: 20px 可以显示在左侧 */
  z-index: 9999;
}
```

### Q2: 如何实现通知分组？
**A**: 扩展 NotificationCenter 类：
```typescript
class GroupedNotificationCenter extends NotificationCenter {
  private groups: Map<string, string[]> = new Map();

  showGrouped(group: string, type: string, title: string, message: string) {
    const id = this.show(type, title, message);
    if (!this.groups.has(group)) {
      this.groups.set(group, []);
    }
    this.groups.get(group)!.push(id);
    return id;
  }

  clearGroup(group: string) {
    const ids = this.groups.get(group) || [];
    ids.forEach(id => this.remove(id));
    this.groups.delete(group);
  }
}
```

### Q3: 如何实现对话框键盘导航？
**A**: 添加键盘事件监听：
```typescript
document.addEventListener('keydown', (e) => {
  if (dialogManager.isOpen()) {
    if (e.key === 'Escape') {
      dialogManager.closeAll();
    } else if (e.key === 'Enter') {
      const primaryButton = document.querySelector('.btn-primary');
      if (primaryButton) {
        (primaryButton as HTMLButtonElement).click();
      }
    }
  }
});
```

### Q4: 如何限制通知数量？
**A**: 扩展 NotificationCenter 类：
```typescript
class LimitedNotificationCenter extends NotificationCenter {
  private maxNotifications = 5;

  show(type: string, title: string, message: string, options: NotificationOptions = {}): string {
    // 如果超过最大数量，移除最旧的通知
    if (this.notifications.size >= this.maxNotifications) {
      const oldestId = this.notifications.keys().next().value;
      this.remove(oldestId);
    }

    return super.show(type, title, message, options);
  }
}
```

### Q5: 如何实现通知持久化？
**A**: 使用 localStorage 持久化：
```typescript
class PersistentNotificationCenter extends NotificationCenter {
  private storageKey = 'notifications';

  show(type: string, title: string, message: string, options: NotificationOptions = {}): string {
    const id = super.show(type, title, message, options);

    // 保存到 localStorage
    const notifications = JSON.parse(localStorage.getItem(this.storageKey) || '[]');
    notifications.push({ id, type, title, message, timestamp: Date.now() });
    localStorage.setItem(this.storageKey, JSON.stringify(notifications));

    return id;
  }

  loadPersisted() {
    const notifications = JSON.parse(localStorage.getItem(this.storageKey) || '[]');
    notifications.forEach((n: any) => {
      this.show(n.type, n.title, n.message);
    });
  }
}
```

---

## 性能优化建议

1. **限制通知数量**：同时显示的通知不超过 5 个
2. **使用 CSS 动画**：使用 CSS transition 代替 JavaScript 动画
3. **延迟加载对话框**：仅在需要时创建对话框 DOM
4. **批量清理通知**：使用 `clearAll()` 批量清理通知
5. **使用事件委托**：对话框按钮使用事件委托

---

## 相关文档

- [Admin 模块文档](./admin.md) - 应用核心
- [API 模块文档](./api.md) - 后端通信层
- [Creds 模块文档](./creds.md) - 凭证管理 UI

