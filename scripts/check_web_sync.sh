#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"

if [[ ! -d "${WEB_DIR}/dist" ]]; then
  echo "web/dist directory not found. Run 'npm install' and 'npm run build:ts' under web/." >&2
  exit 1
fi

# Rebuild TypeScript output unless SKIP_WEB_BUILD is set.
if [[ -z "${SKIP_WEB_BUILD:-}" ]]; then
  (cd "$WEB_DIR" && npm run build:ts >/dev/null)
fi

if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  if git -C "$ROOT_DIR" diff --quiet -- web/dist; then
    echo "web-sync OK"
  else
    echo "web/dist is out of sync with the compiled TypeScript output." >&2
    echo "Run 'make web-sync' and commit the updated files." >&2
    git -C "$ROOT_DIR" --no-pager diff --stat -- web/dist >&2 || true
    exit 1
  fi
else
  echo "web-sync check skipped (no git repository detected); assuming current tree is up to date)."
fi
