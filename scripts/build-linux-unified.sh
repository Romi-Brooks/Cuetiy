#!/usr/bin/env bash
# Cuetiy Linux · unified（内嵌后端 + SQLite，安装包 deb/AppImage/rpm）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND="$ROOT/frontend"
BACKEND="$ROOT/backend"
OUT="$ROOT/dist-packages"
BIN_DIR="$FRONTEND/src-tauri/binaries"
TRIPLE="$(rustc -vV | sed -n 's/^host: //p')"

mkdir -p "$BIN_DIR" "$OUT"
export CGO_ENABLED=0

echo "==> backend sidecar ($TRIPLE)"
GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' \
  -o "$BIN_DIR/cuetiy-backend-x86_64-unknown-linux-gnu" "$BACKEND/cmd/main.go"
chmod +x "$BIN_DIR/cuetiy-backend-x86_64-unknown-linux-gnu"
if [[ "$TRIPLE" != "x86_64-unknown-linux-gnu" ]]; then
  cp "$BIN_DIR/cuetiy-backend-x86_64-unknown-linux-gnu" "$BIN_DIR/cuetiy-backend-${TRIPLE}"
  chmod +x "$BIN_DIR/cuetiy-backend-${TRIPLE}"
fi

echo "==> frontend + tauri unified"
cd "$FRONTEND"
pnpm install --prefer-offline
pnpm type-check
export CUETIY_PACKAGE=unified
pnpm exec tauri build

echo "==> copy bundles"
find "$FRONTEND/src-tauri/target/release/bundle" -type f \
  \( -name '*.deb' -o -name '*.AppImage' -o -name '*.rpm' \) \
  -exec cp -v {} "$OUT/" \; || true
echo "==> $OUT"
