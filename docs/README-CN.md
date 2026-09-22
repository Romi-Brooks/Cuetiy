# Cuetiy

> [English](../README.md) | **中文**

AI 情感陪伴聊天。支持 Web / EXE/ APK 。数据可在服务器或本机流通，且聊天支持 JSON 导出/导入。

## 产品矩阵

| 包 | 后端 | 数据 | 用户如何连接 |
|----|------|------|----------------|
| Web / thin 客户端 | 服务器 Go | PostgreSQL 或 SQLite | App 内运行时填 **API 地址** |
| **EXE unified** | 内嵌 Go sidecar | 本机 SQLite + `data/` | 默认 `http://127.0.0.1:8080` |
| **EXE 便携 zip** | 同 unified | 同目录 `data/` | 解压即用 |
| **APK thin** | 远程服务器 | 服务器库 | 运行时填 API |

**AI 密钥**：登录后「我的 → 数据与服务器」填写 DeepSeek / MiMo Key。接口**不回显**明文，只显示是否已配置；保存后写入后端 `.env` 并立即生效。

## 功能

- 模拟社交APP UI、WebSocket 流式回复、拟人输入节奏
- 上下文：L0 人设 + L1 能力索引 + 滚动摘要 + L2 触发技能 + **压缩后 HISTORY**
- 人格/技能文件、TTS 语音条、ASR、清空前归档
- JWT 多用户隔离
- 聊天导出/导入
- 分为完整包和仅前端包，便于部署

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | React 18、Vite 5、TS、Zustand、Tailwind、Capacitor 7、Tauri 2 |
| 后端 | Go 1.26、Gin、GORM、JWT、WebSocket |
| 数据库 | PostgreSQL / MySQL / SQLite |
| 缓存 | Redis 可选 |
| AI | DeepSeek；MiMo TTS/ASR |

## 开发启动

```bash
# 后端
cd backend
cp .env.example .env   # DB_DRIVER=sqlite 或 postgres
go run cmd/main.go

# 前端
cd frontend
pnpm install
pnpm dev
# 登录页 → 服务器设置 → http://127.0.0.1:8080
```

SQLite 示例：

```env
DB_DRIVER=sqlite
SQLITE_PATH=./data/cuetiy.db
SERVER_PORT=8080
REDIS_HOST=
```

## 打包

正式包由 CI 构建。本地自行打包见：

| 文档 | 内容 |
|------|------|
| [EXE-BUILD.md](EXE-BUILD.md) | EXE thin |
| [EXE-UNIFIED.md](EXE-UNIFIED.md) | EXE unified + 便携包 |
| [APK-THIN.md](APK-THIN.md) | APK thin |
| [LINUX-PACKAGING.md](LINUX-PACKAGING.md) | Linux thin / unified |
| [PG-MIGRATION.md](PG-MIGRATION.md) | PostgreSQL / MySQL 迁移 |

脚本（在仓库根目录、环境就绪后执行）：

```bash
# Windows
scripts/build-exe-thin.ps1
scripts/build-exe-unified.ps1
scripts/build-portable-unified.ps1
scripts/build-apk-thin.ps1
scripts/clean-build.ps1

# Linux
scripts/build-linux-thin.sh
scripts/build-linux-unified.sh
scripts/build-linux-portable.sh
```

产物：`dist-packages/`（二进制不要提交）。

## 参与贡献

- [贡献指南（中文）](CONTRIBUTING-CN.md) / [English](../CONTRIBUTING.md)
- [代码规范（中文）](CODING_STANDARD-CN.md) / [English](../CODING_STANDARD.md)
- [问题与评估文档](issue/) — 待评估的设计问题与评估结论

## 注意

- GORM 参数化 SQL；密码 bcrypt
- `/storage/*` 路径清洗
- APK 调试允许明文 HTTP；公网请 HTTPS
