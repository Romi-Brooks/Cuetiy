#!/usr/bin/env bash
# 在服务器上构建 RainYi 前后端（假定代码已在 /opt/rain-yi）
set -euo pipefail

APP_DIR=${APP_DIR:-/opt/rain-yi}
cd "$APP_DIR"

echo "==> 构建后端"
cd "$APP_DIR/backend"
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rain-yi-backend ./cmd
echo "后端二进制: $APP_DIR/backend/rain-yi-backend"

echo "==> 构建前端"
cd "$APP_DIR/frontend"
# Web 部署走 nginx 反代，不注入绝对 API 地址，使用相对路径 /api
# 如需 APK，请改用 .env.production.local 并填绝对地址
export VITE_API_URL=
export VITE_WS_URL=
pnpm install
pnpm build

echo "==> 前端产物: $APP_DIR/frontend/dist"
echo "完成。请确认 backend/.env、nginx、systemd 后启动服务。"
