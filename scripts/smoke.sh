#!/usr/bin/env bash
set -euo pipefail

BASE=${BASE:-http://localhost:8317}
BASE_PATH=${BASE_PATH:-}

if [[ -n "$BASE_PATH" ]]; then
  if [[ "$BASE_PATH" != /* ]]; then
    BASE_PATH="/$BASE_PATH"
  fi
  # strip trailing slash from BASE to avoid double separators
  BASE="${BASE%/}${BASE_PATH%/}"
fi

BASE=${BASE%/}
KEY=${KEY:-}

echo "[1/4] Health check" && curl -fsSL "$BASE/healthz" && echo
echo "[2/4] Metrics head (first 5 lines)" && curl -fsSL "$BASE/metrics" | head -n 5
echo "[3/4] /routes overview (HTML title)" && curl -fsSL "$BASE/routes" | sed -n '1,5p' | head -n 1
if [[ -n "$KEY" ]]; then
  echo "[4/4] /v1/models with auth"
  curl -fsSL -H "Authorization: Bearer $KEY" "$BASE/v1/models" | sed -n '1,3p'
else
  echo "[4/4] /v1/models without auth (expected 401 if configured)"
  set +e
  curl -s -o /dev/null -w "%{http_code}\n" "$BASE/v1/models"
  set -e
fi
echo "Done."
