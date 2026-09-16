#!/usr/bin/env bash
# RainYi 云服务器一键初始化（Ubuntu/Debian）
# 用法: sudo bash deploy/setup-server.sh
set -euo pipefail

APP_DIR=/opt/rain-yi

echo "==> 1. 安装系统依赖"
apt-get update -y
apt-get install -y curl git nginx redis-server mysql-client unzip

echo "==> 2. 安装 Go 1.22（若未安装）"
if ! command -v go >/dev/null 2>&1; then
  GO_VER=1.22.12
  curl -fsSL "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz" -o /tmp/go.tgz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tgz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
fi
go version

echo "==> 3. 安装 Node.js 20（若未安装）"
if ! command -v node >/dev/null 2>&1; then
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y nodejs
fi
node -v
npm -v
npm install -g pnpm

echo "==> 4. 启用 Redis"
systemctl enable redis-server
systemctl start redis-server
redis-cli ping

echo "==> 5. 准备应用目录"
mkdir -p "$APP_DIR"

echo "==> 6. 安装 MinIO（可选，文件上传需要）"
if ! command -v mc >/dev/null 2>&1 || ! docker ps >/dev/null 2>&1; then
  echo "MinIO 建议用 Docker 部署。若本机无 Docker，可执行:"
  echo "  curl -fsSL https://get.docker.com | sh"
  echo "  docker run -d --name minio --restart always \\"
  echo "    -p 9000:9000 -p 9001:9001 \\"
  echo "    -e MINIO_ROOT_USER=minioadmin \\"
  echo "    -e MINIO_ROOT_PASSWORD='请改成强密码' \\"
  echo "    -v /opt/minio/data:/data \\"
  echo "    quay.io/minio/minio server /data --console-address \":9001\""
else
  echo "检测到 Docker，可直接启动 MinIO（见上方命令）"
fi

echo "==> 完成。下一步："
echo "  1) 创建 MySQL 库: mysql -h 127.0.0.1 -u root -p < database/schema.sql"
echo "  2) 配置 backend/.env（参考 deploy/env.production.example）"
echo "  3) 本地构建后上传，或在服务器 git clone 后构建"
echo "  4) 部署 nginx 配置 deploy/nginx.conf"
echo "  5) 启动 systemd: deploy/rain-yi-backend.service"
