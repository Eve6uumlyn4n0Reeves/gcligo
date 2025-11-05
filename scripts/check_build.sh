#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"
export GOCACHE="${ROOT_DIR}/.gocache"
export GOMODCACHE="${ROOT_DIR}/.gomodcache"
export GOSUMDB=${GOSUMDB:-sum.golang.org}

mkdir -p "${GOCACHE}" "${GOMODCACHE}"

GO_LDFLAGS=${GO_LDFLAGS:--s -w}
GO_TRIMPATH_FLAG=${GO_TRIMPATH_FLAG:=true}
BUILD_FLAGS=()
if [ "${GO_TRIMPATH_FLAG}" = "true" ]; then
  BUILD_FLAGS+=("-trimpath")
fi
BUILD_FLAGS+=("-ldflags" "${GO_LDFLAGS}")

echo "==> go build ./cmd/server (flags: ${BUILD_FLAGS[*]})"
(cd "${ROOT_DIR}" && go build "${BUILD_FLAGS[@]}" ./cmd/server)

echo "==> go test ./..."
(cd "${ROOT_DIR}" && go test ./...)

if [ -f "${ROOT_DIR}/web/package.json" ]; then
  echo "==> npm install (web)"
  (cd "${ROOT_DIR}/web" && npm install)

  echo "==> npm test --runInBand (web)"
  (cd "${ROOT_DIR}/web" && npm test -- --runInBand)
fi

echo "==> npm run lint"
(cd "${ROOT_DIR}" && npm run lint)

echo "✅ build & test completed"
