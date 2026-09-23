# Cuetiy Linux（thin / unified / portable）

在 Linux 上运行完整桌面程序（不只是后端）。Go sidecar 纯 Go SQLite，无需 CGO。

| 模式 | 脚本 | 数据 |
|------|------|------|
| thin | `scripts/build-linux-thin.sh` | 远程服务器（App 内填 API） |
| unified | `scripts/build-linux-unified.sh` | 本机 SQLite |
| portable | `scripts/build-linux-portable.sh` | 同目录 `data/`，tar.gz |

## 构建机依赖（Debian/Ubuntu）

```bash
sudo apt install -y build-essential libwebkit2gtk-4.1-dev libgtk-3-dev \
  librsvg2-dev patchelf pkg-config
# 另需: Go 1.26+, Node 18+, pnpm, Rust (rustup)
```

在 **Linux 或 WSL** 仓库根目录执行，例如：

```bash
chmod +x scripts/build-linux-*.sh
./scripts/build-linux-thin.sh
./scripts/build-linux-unified.sh
./scripts/build-linux-portable.sh
```

产物在 `dist-packages/`（deb / AppImage / rpm / tar.gz）。

Windows 无法完整交叉编译 Linux GUI；只能编 Linux 后端：

```powershell
cd backend
$env:CGO_ENABLED='0'; $env:GOOS='linux'; $env:GOARCH='amd64'
go build -ldflags '-s -w' -o output/cuetiy-backend-linux-amd64 ./cmd/main.go
```

## 用户安装

**thin / unified（.deb）**

```bash
sudo dpkg -i Cuetiy_*.deb
# 运行库: sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0 librsvg2-2
```

thin：登录页填服务器 API。unified：默认本机 `127.0.0.1:8080`。

**portable**

```bash
tar -xzf Cuetiy-linux-portable.tar.gz
cd Cuetiy-linux-portable
./Cuetiy
```

**仅后端（给 thin/网页用）**

```bash
DB_DRIVER=sqlite SQLITE_PATH=./data/cuetiy.db \
SERVER_HOST=0.0.0.0 SERVER_PORT=8080 ./cuetiy-backend
```

服务器 + PostgreSQL 见 [PG-MIGRATION.md](PG-MIGRATION.md)。

## sidecar 文件名

Tauri 要求：

```text
frontend/src-tauri/binaries/cuetiy-backend-x86_64-unknown-linux-gnu
```

脚本已自动处理。
