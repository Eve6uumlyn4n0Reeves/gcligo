#!/usr/bin/env node
import { statSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '..');

const checks = [
  { key: 'admin.js', file: path.join(projectRoot, 'web/js/admin.js'), limit: 80 * 1024 }
];

let hasFailure = false;

for (const check of checks) {
  try {
    const stats = statSync(check.file);
    const size = stats.size;
    if (size > check.limit) {
      console.error(
        `[bundle-size] ${check.key} exceeded limit: ${size} bytes (limit ${check.limit})`
      );
      hasFailure = true;
    } else {
      console.log(
        `[bundle-size] ${check.key} OK: ${size} bytes (limit ${check.limit})`
      );
    }
  } catch (err) {
    console.error(`[bundle-size] Failed to read ${check.file}:`, err);
    hasFailure = true;
  }
}

if (hasFailure) {
  process.exitCode = 1;
  process.exit(1);
}

console.log('[bundle-size] All bundle size checks passed.');
