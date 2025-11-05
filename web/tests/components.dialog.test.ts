import { describe, it, expect, beforeEach } from 'vitest';
import { DialogManager } from '../src/components/dialog';

describe('DialogManager', () => {
  let manager: DialogManager;

  beforeEach(() => {
    document.body.innerHTML = '';
    manager = new DialogManager();
  });

  it('should create DialogManager instance', () => {
    expect(manager).toBeInstanceOf(DialogManager);
  });

  it('should ensure legacy dialog exists', () => {
    manager.ensureLegacyDialog();
    
    const overlay = document.getElementById('modal-overlay');
    expect(overlay).toBeTruthy();
    expect(overlay?.className).toBe('modal-overlay');
  });

  it('should not create duplicate legacy dialogs', () => {
    manager.ensureLegacyDialog();
    manager.ensureLegacyDialog();
    
    const overlays = document.querySelectorAll('#modal-overlay');
    expect(overlays.length).toBe(1);
  });

  it('should have modal structure', () => {
    manager.ensureLegacyDialog();
    
    const container = document.querySelector('.modal-container');
    const header = document.querySelector('.modal-header');
    const body = document.querySelector('.modal-body');
    const footer = document.querySelector('.modal-footer');
    
    expect(container).toBeTruthy();
    expect(header).toBeTruthy();
    expect(body).toBeTruthy();
    expect(footer).toBeTruthy();
  });

  it('should show legacy dialog', () => {
    manager.showLegacyDialog('Test Title', 'Test Content');
    
    const overlay = document.getElementById('modal-overlay');
    expect(overlay?.style.display).not.toBe('none');
    
    const title = document.querySelector('.modal-title');
    expect(title?.textContent).toBe('Test Title');
  });

  it('should close legacy dialog', () => {
    manager.showLegacyDialog('Test', 'Content');
    manager.closeLegacyDialog();
    
    const overlay = document.getElementById('modal-overlay');
    expect(overlay?.style.display).toBe('none');
  });
});

