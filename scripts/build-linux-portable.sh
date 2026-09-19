#!/usr/bin/env bash
# RainYi Linux · portable（解压即用目录 + tar.gz，数据在同目录 data/）
# 建议先跑 build-linux-unified.sh，或本脚本会完整构建 unified
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND="$ROOT/frontend"
BACKEND="$ROOT/backend"
OUT="$ROOT/dist-packages"
BIN_DIR="$FRONTEND/src-tauri/binaries"
TRIPLE="$(rustc -vV | sed -n 's/^host: //p')"
STAGE="$OUT/RainYi-linux-portable"
ZIP="$OUT/RainYi-linux-portable.tar.gz"

mkdir -p "$BIN_DIR" "$OUT"
export CGO_ENABLED=0

APP="$FRONTEND/src-tauri/target/release/rain-yi"
if [[ ! -x "$APP" ]]; then
  echo "==> unified binary missing, building first"
  "$ROOT/scripts/build-linux-unified.sh"
fi

SIDE="$BIN_DIR/rainyi-backend-x86_64-unknown-linux-gnu"
if [[ ! -f "$SIDE" ]]; then
  GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o "$SIDE" "$BACKEND/cmd/main.go"
  chmod +x "$SIDE"
fi

echo "==> stage $STAGE"
rm -rf "$STAGE"
mkdir -p "$STAGE/data/files" "$STAGE/data/archives" "$STAGE/skills"
cp "$APP" "$STAGE/RainYi"
cp "$SIDE" "$STAGE/rainyi-backend"
chmod +x "$STAGE/RainYi" "$STAGE/rainyi-backend"
if [[ -d "$BACKEND/skill/default_skills" ]]; then
  cp -a "$BACKEND/skill/default_skills/." "$STAGE/skills/" 2>/dev/null || true
fi
if [[ -f "$FRONTEND/src-tauri/resources/env/.env.unified" ]]; then
  cp "$FRONTEND/src-tauri/resources/env/.env.unified" "$STAGE/.env"
else
  cat > "$STAGE/.env" <<'EOF'
DB_DRIVER=sqlite
SQLITE_PATH=./data/rainyi.db
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
JWT_SECRET=change-me-local-rainyi
STORAGE_DIR=./data/files
ARCHIVE_DIR=./data/archives
SKILLS_DIR=./skills
RUNTIME_DIR=.
REDIS_HOST=
EOF
fi

cat > "$STAGE/README-PORTABLE.txt" <<'EOF'
RainYi Linux portable (unified)

Runtime deps:
  Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0 librsvg2-2

Run:
  ./RainYi
  Backend auto-starts on http://127.0.0.1:8080 with SQLite.
  Set DeepSeek/MiMo keys in Me → Data & Server (never echoed).

Data: ./data/rainyi.db and ./data/files/  (copy the folder to backup)
EOF

echo "==> tar $ZIP"
rm -f "$ZIP"
tar -C "$OUT" -czf "$ZIP" "RainYi-linux-portable"
ls -lh "$ZIP" "$STAGE"
