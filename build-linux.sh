#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

ARCH="${1:-$(uname -m)}"
case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *)
    echo "[ERROR] Unsupported architecture: $ARCH (use amd64 or arm64)" >&2
    exit 1
    ;;
esac

echo "==> Checking prerequisites..."
command -v go >/dev/null 2>&1 || {
  echo "[ERROR] Go was not found in PATH. Install Go 1.26+ first." >&2
  exit 1
}
command -v npm >/dev/null 2>&1 || {
  echo "[ERROR] npm was not found in PATH. Install Node.js 20+ first." >&2
  exit 1
}

if [ ! -d frontend/node_modules ]; then
  echo "==> Installing frontend dependencies..."
  npm install --prefix frontend
fi

echo "==> Building frontend..."
npm run build --prefix frontend

mkdir -p bin
OUT="bin/resume-helper-linux-${ARCH}"

build() {
  GOOS=linux GOARCH="$ARCH" go build -trimpath "$@" -ldflags="-s -w" -o "$OUT" .
}

echo "==> Building Linux (${ARCH}) binary..."
# Wails 3 defaults to GTK4 / WebKitGTK 6.0. On older distributions set
# WAILS_TAGS=gtk3 to build against GTK3 / WebKit2GTK 4.1 instead.
if [ -n "${WAILS_TAGS:-}" ]; then
  build -tags "$WAILS_TAGS"
else
  build
fi

echo
echo "[OK] Built $OUT"
ls -lh "$OUT"
