type AriaPriority = 'polite' | 'assertive';

export function setRole(el: Element | null, role: string): void {
  if (el) el.setAttribute('role', role);
}

export function applyTableAria(container: Element | null, label?: string): void {
  if (!container) return;
  container.setAttribute('role', 'table');
  if (label) container.setAttribute('aria-label', label);
}

export function trapFocus(modal: HTMLElement | null): () => void {
  if (!modal) return () => {};
  const focusable = Array.from(
    modal.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )
  );
  const onKey = (e: KeyboardEvent) => {
    if (e.key !== 'Tab' || focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };
  modal.addEventListener('keydown', onKey);
  return () => modal.removeEventListener('keydown', onKey);
}

export interface ManageFocusOptions {
  trap?: boolean;
  restoreOnEscape?: boolean;
  autoFocus?: boolean;
}

export interface ManageFocusHandle {
  restore(): void;
  cleanup(): void;
}

export function manageFocus(
  element: HTMLElement | null,
  options: ManageFocusOptions = {}
): ManageFocusHandle {
  if (!element) return { restore: () => {}, cleanup: () => {} };

  const previousFocus = document.activeElement as HTMLElement | null;
  const { trap = false, restoreOnEscape = true, autoFocus = true } = options;
  let cleanup: () => void = () => {};

  if (autoFocus) {
    const firstFocusable = element.querySelector<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    if (firstFocusable) {
      setTimeout(() => firstFocusable.focus(), 0);
    }
  }

  if (trap) {
    cleanup = trapFocus(element);
  }

  if (restoreOnEscape) {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        restore();
      }
    };
    element.addEventListener('keydown', handleEscape);
    const originalCleanup = cleanup;
    cleanup = () => {
      originalCleanup();
      element.removeEventListener('keydown', handleEscape);
    };
  }

  const restore = () => {
    cleanup();
    if (previousFocus && typeof previousFocus.focus === 'function') {
      try {
        previousFocus.focus();
      } catch (error) {
        console.warn('Failed to restore focus:', error);
      }
    }
  };

  return { restore, cleanup };
}

export function announce(message: string, priority: AriaPriority = 'polite'): void {
  const announcer = getOrCreateAnnouncer(priority);
  announcer.textContent = message;
  setTimeout(() => {
    if (announcer.textContent === message) {
      announcer.textContent = '';
    }
  }, 1000);
}

function getOrCreateAnnouncer(priority: AriaPriority): HTMLElement {
  const id = `a11y-announcer-${priority}`;
  let announcer = document.getElementById(id);
  if (!announcer) {
    announcer = document.createElement('div');
    announcer.id = id;
    announcer.setAttribute('aria-live', priority);
    announcer.setAttribute('aria-atomic', 'true');
    announcer.style.cssText = `
      position: absolute;
      left: -10000px;
      width: 1px;
      height: 1px;
      overflow: hidden;
    `;
    document.body.appendChild(announcer);
  }
  return announcer;
}

export interface EnhanceButtonOptions {
  describedBy?: string;
  expanded?: boolean;
  pressed?: boolean;
  controls?: string;
  label?: string;
  shortcut?: string;
}

export function enhanceButton(
  button: HTMLElement | null,
  options: EnhanceButtonOptions = {}
): void {
  if (!button) return;
  const { describedBy, expanded, pressed, controls, label, shortcut } = options;
  if (describedBy) button.setAttribute('aria-describedby', describedBy);
  if (expanded !== undefined) button.setAttribute('aria-expanded', String(expanded));
  if (pressed !== undefined) button.setAttribute('aria-pressed', String(pressed));
  if (controls) button.setAttribute('aria-controls', controls);
  if (label) button.setAttribute('aria-label', label);
  if (shortcut) {
    const existing = button.getAttribute('title') || '';
    const newTitle = existing ? `${existing} (${shortcut})` : `快捷键: ${shortcut}`;
    button.setAttribute('title', newTitle);
  }
}

export interface EnhanceFormFieldOptions {
  label?: string;
  required?: boolean;
  invalid?: boolean;
  describedBy?: string;
  errorMessage?: string;
}

export function enhanceFormField(
  field: HTMLElement | null,
  options: EnhanceFormFieldOptions = {}
): void {
  if (!field) return;
  const { required, invalid, describedBy, errorMessage } = options;
  if (required) field.setAttribute('aria-required', 'true');
  if (invalid) {
    field.setAttribute('aria-invalid', 'true');
    if (errorMessage) {
      const errorId = `${(field as HTMLElement).id || 'field'}-error`;
      let errorEl = document.getElementById(errorId);
      if (!errorEl) {
        errorEl = document.createElement('div');
        errorEl.id = errorId;
        errorEl.className = 'error-message';
        errorEl.setAttribute('role', 'alert');
        field.parentNode?.insertBefore(errorEl, field.nextSibling);
      }
      errorEl.textContent = errorMessage;
      field.setAttribute('aria-describedby', errorId);
    }
  } else {
    field.removeAttribute('aria-invalid');
  }
  if (describedBy) {
    const existing = field.getAttribute('aria-describedby') || '';
    const combined = existing ? `${existing} ${describedBy}` : describedBy;
    field.setAttribute('aria-describedby', combined);
  }
}

export function addSkipLinks(): void {
  const skipLinks = document.createElement('div');
  skipLinks.className = 'skip-links';
  skipLinks.innerHTML = `
    <a href="#main-content" class="skip-link">跳转到主内容</a>
    <a href="#sidebar" class="skip-link">跳转到导航</a>
  `;

  const style = document.createElement('style');
  style.textContent = `
    .skip-links {
      position: absolute;
      top: -40px;
      left: 6px;
      z-index: 10001;
    }
    .skip-link {
      position: absolute;
      top: -40px;
      left: 6px;
      background: #000;
      color: #fff;
      padding: 8px;
      text-decoration: none;
      border-radius: 4px;
      z-index: 10001;
    }
    .skip-link:focus {
      top: 6px;
    }
  `;

  document.head.appendChild(style);
  document.body.insertBefore(skipLinks, document.body.firstChild);
}

export interface ArrowNavigationOptions {
  loop?: boolean;
  horizontal?: boolean;
  vertical?: boolean;
  onSelect?: (element: Element | null, index: number) => void;
}

export function handleArrowNavigation(
  container: HTMLElement | null,
  selector: string,
  options: ArrowNavigationOptions = {}
): void {
  if (!container) return;
  const { loop = true, horizontal = false, vertical = true, onSelect } = options;
  const handleKeydown = (e: KeyboardEvent) => {
    const items = Array.from(container.querySelectorAll<HTMLElement>(selector));
    if (items.length === 0) return;
    const currentIndex = items.indexOf(document.activeElement as HTMLElement);
    if (currentIndex === -1) return;
    let nextIndex = currentIndex;
    if (
      (horizontal && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) ||
      (vertical && (e.key === 'ArrowUp' || e.key === 'ArrowDown'))
    ) {
      e.preventDefault();
      if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
        nextIndex = currentIndex - 1;
        if (nextIndex < 0) nextIndex = loop ? items.length - 1 : 0;
      } else {
        nextIndex = currentIndex + 1;
        if (nextIndex >= items.length) nextIndex = loop ? 0 : items.length - 1;
      }
      items[nextIndex].focus();
    }
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      if (onSelect) onSelect(document.activeElement, currentIndex);
      else (document.activeElement as HTMLElement | null)?.click?.();
    }
  };
  container.addEventListener('keydown', handleKeydown);
}

export function initSkipLinks(): void {
  addSkipLinks();
}

export function focusMainContent(): void {
  const main = document.getElementById('main-content');
  if (main) {
    main.setAttribute('tabindex', '-1');
    (main as HTMLElement).focus();
    setTimeout(() => main.removeAttribute('tabindex'), 100);
  }
}

export function focusFirstField(form: HTMLElement | null): void {
  if (!form) return;
  const field = form.querySelector<HTMLElement>(
    'input, select, textarea, [tabindex]:not([tabindex="-1"])'
  );
  field?.focus();
}

export interface MakeDialogOptions {
  labelledBy?: string;
  describedBy?: string;
  closeSelector?: string;
  focusSelector?: string;
  autoFocus?: boolean;
}

export function makeDialogAccessible(
  modal: HTMLElement | null,
  options: MakeDialogOptions = {}
): { close: () => void } {
  if (!modal) return { close: () => {} };

  const { labelledBy, describedBy, closeSelector = '.modal-close', focusSelector, autoFocus } =
    options;
  modal.setAttribute('role', 'dialog');
  modal.setAttribute('aria-modal', 'true');
  if (labelledBy) modal.setAttribute('aria-labelledby', labelledBy);
  if (describedBy) modal.setAttribute('aria-describedby', describedBy);

  const focusTrap = manageFocus(modal, { trap: true, autoFocus: autoFocus !== false });
  const closeBtn = modal.querySelector<HTMLElement>(closeSelector);
  const focusTarget =
    focusSelector ? modal.querySelector<HTMLElement>(focusSelector) : null;

  const closeModal = () => {
    focusTrap.restore();
    modal.removeAttribute('role');
    modal.removeAttribute('aria-modal');
    modal.removeAttribute('aria-labelledby');
    modal.removeAttribute('aria-describedby');
    modal.remove();
    if (focusTarget) {
      focusTarget.focus();
    }
  };

  closeBtn?.addEventListener('click', closeModal);
  modal.addEventListener('click', (event) => {
    if (event.target === modal) closeModal();
  });

  return { close: closeModal };
}

export function scrollIntoViewIfNeeded(
  element: Element | null,
  options: ScrollIntoViewOptions = {}
): void {
  if (!element) return;
  try {
    element.scrollIntoView({ block: 'nearest', inline: 'nearest', ...options });
  } catch {
    element.scrollIntoView();
  }
}

export interface TabTrapOptions {
  loop?: boolean;
  horizontal?: boolean;
  vertical?: boolean;
  onSelect?: (element: HTMLElement, index: number) => void;
}

export function setTabTrap(
  container: HTMLElement | null,
  selector: string,
  options: TabTrapOptions = {}
): () => void {
  if (!container) return () => {};
  const { loop = true, horizontal = false, vertical = true, onSelect } = options;
  const focusable = Array.from(container.querySelectorAll<HTMLElement>(selector));
  if (focusable.length === 0) return () => {};

  const onKeydown = (e: KeyboardEvent) => {
    const currentIndex = focusable.indexOf(document.activeElement as HTMLElement);
    if (currentIndex === -1) return;
    let nextIndex = currentIndex;
    if (
      (horizontal && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) ||
      (vertical && (e.key === 'ArrowUp' || e.key === 'ArrowDown'))
    ) {
      e.preventDefault();
      if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
        nextIndex = currentIndex - 1;
        if (nextIndex < 0) nextIndex = loop ? focusable.length - 1 : 0;
      } else {
        nextIndex = currentIndex + 1;
        if (nextIndex >= focusable.length) nextIndex = loop ? 0 : focusable.length - 1;
      }
      focusable[nextIndex].focus();
    }
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      const current = focusable[currentIndex];
      if (onSelect) onSelect(current, currentIndex);
      else current.click();
    }
  };
  container.addEventListener('keydown', onKeydown);
  return () => container.removeEventListener('keydown', onKeydown);
}

export function toggleAriaExpanded(target: HTMLElement | null, expanded: boolean): void {
  if (!target) return;
  target.setAttribute('aria-expanded', String(expanded));
}

export function linkAriaControls(trigger: HTMLElement | null, targetId: string): void {
  if (!trigger) return;
  trigger.setAttribute('aria-controls', targetId);
}
