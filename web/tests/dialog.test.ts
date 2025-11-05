import { describe, it, expect, beforeEach } from 'vitest';
import { DialogManager } from '../src/components/dialog';

describe('DialogManager interactions', () => {
  let manager: DialogManager;

  beforeEach(() => {
    document.body.innerHTML = '';
    manager = new DialogManager();
  });

  it('opens legacy dialog with text content and closes it', () => {
    manager.open('legacy', 'Title', 'Plain message');

    const overlay = document.getElementById('modal-overlay');
    expect(overlay).toBeTruthy();
    expect(overlay?.style.display).toBe('flex');

    const title = document.querySelector('.modal-title');
    const body = document.querySelector('.modal-body');
    expect(title?.textContent).toBe('Title');
    expect(body?.textContent).toBe('Plain message');

    manager.closeLegacyDialog();
    expect(overlay?.style.display).toBe('none');
    expect(document.body.style.overflow).toBe('');
  });

  it('renders HTML when allowHTML option is enabled', () => {
    manager.open('legacy', 'HTML Title', '<strong>bold</strong>', { allowHTML: true });

    const body = document.querySelector('.modal-body');
    expect(body?.innerHTML).toContain('<strong>bold</strong>');
  });

  it('resolves confirm promise on button clicks', async () => {
    const cancelPromise = manager.confirm('Confirm', 'Are you sure?');

    const footer = document.querySelector('.modal-footer');
    const buttons = footer?.querySelectorAll('button');
    expect(buttons?.length).toBe(2);

    buttons?.[0].dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await expect(cancelPromise).resolves.toBe(false);

    const acceptPromise = manager.confirm('Confirm', 'Proceed?');
    const acceptButton = document.querySelector('.modal-footer .btn-primary') as HTMLButtonElement;
    expect(acceptButton).toBeTruthy();
    acceptButton.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await expect(acceptPromise).resolves.toBe(true);
  });

  it('confirmDelete uses danger button styling and resolves true', async () => {
    const promise = manager.confirmDelete('凭证A');

    const dangerButton = document.querySelector('.modal-footer .btn-danger') as HTMLButtonElement;
    expect(dangerButton).toBeTruthy();
    expect(dangerButton.textContent).toContain('删除');

    dangerButton.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await expect(promise).resolves.toBe(true);
  });

  it('closeAll hides dialogs and resets state', () => {
    manager.open('legacy', 'Title', 'Content');
    expect(manager.isOpen()).toBe(true);

    manager.closeAll();
    expect(manager.isOpen()).toBe(false);
    const overlay = document.getElementById('modal-overlay');
    expect(overlay?.style.display).toBe('none');
  });
});
