# Cuetiy

> **English** | [中文](docs/README-CN.md)

AI companion chat. Supports Web / EXE / APK. Data can live on a server or on-device; chat supports JSON export/import.

## Product matrix

| Package | Backend | Data | How users connect |
|---------|---------|------|-------------------|
| Web / thin clients | Server Go | PostgreSQL or SQLite | Set **API URL** in the app |
| **EXE unified** | Embedded Go sidecar | Local SQLite + `data/` | Default `http://127.0.0.1:8080` |
| **EXE portable zip** | Same as unified | Folder `data/` | Unzip and run |
| **APK thin** | Remote server | Server DB | Set API URL in the app |

**AI keys**: after login, open **Me → Data & Server**. Keys are never echoed; the UI only shows configured / not configured. Saved to backend `.env` and applied immediately.

## Features

- Social-app style UI, WebSocket streaming, humanized typing rhythm
- Context: L0 persona + L1 skill index + rolling summary + L2 triggered skills + **compacted HISTORY**
- Persona/skill files, TTS voice bars, ASR, archive before clear
- JWT multi-user isolation
- Chat export/import
- Full package vs frontend-only package for easier deployment

## Tech stack

| Layer | Stack |
|-------|--------|
| Frontend | React 18, Vite 5, TS, Zustand, Tailwind, Capacitor 7, Tauri 2 |
| Backend | Go 1.26, Gin, GORM, JWT, WebSocket |
| DB | PostgreSQL / MySQL / SQLite |
| Cache | Redis optional |
| AI | DeepSeek; MiMo TTS/ASR |

## Development

```bash
# Backend
cd backend
cp .env.example .env   # DB_DRIVER=sqlite or postgres
go run cmd/main.go

# Frontend
cd frontend
pnpm install
pnpm dev
# Login page → server settings → http://127.0.0.1:8080
```

SQLite example:

```env
DB_DRIVER=sqlite
SQLITE_PATH=./data/cuetiy.db
SERVER_PORT=8080
REDIS_HOST=
```

## Packaging

Official releases are built via CI. To build yourself, see:

| Doc | Topic |
|-----|--------|
| [docs/EXE-BUILD.md](docs/EXE-BUILD.md) | EXE thin |
| [docs/EXE-UNIFIED.md](docs/EXE-UNIFIED.md) | EXE unified + portable |
| [docs/APK-THIN.md](docs/APK-THIN.md) | APK thin |
| [docs/LINUX-PACKAGING.md](docs/LINUX-PACKAGING.md) | Linux thin / unified |
| [docs/PG-MIGRATION.md](docs/PG-MIGRATION.md) | PostgreSQL setup / MySQL migration |

Scripts (run from repo root after env is ready):

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

Output: `dist-packages/` (do not commit binaries).

## Contributing

- [CONTRIBUTING.md](CONTRIBUTING.md) — how to report issues, branch, and open PRs (English)
- [docs/CONTRIBUTING-CN.md](docs/CONTRIBUTING-CN.md) — 中文贡献指南
- [CODING_STANDARD.md](CODING_STANDARD.md) — layering, naming, security baseline (English)
- [docs/CODING_STANDARD-CN.md](docs/CODING_STANDARD-CN.md) — 中文代码规范
- [docs/issue/](docs/issue/) — open design questions and evaluation documents

## Notes

- GORM parameterized SQL; passwords bcrypt
- `/storage/*` path cleaned under `STORAGE_DIR`
- APK debug allows cleartext HTTP; use HTTPS in production
