# Contributing to Cuetiy

[English](CONTRIBUTING.md) | [中文](docs/CONTRIBUTING-CN.md)

---
Thank you for your willingness to contribute to Cuetiy. Whether it's a bug report, a feature suggestion, a documentation improvement, or a code change, we welcome it.

Cuetiy is an AI companion chat product shipping as Web / EXE / APK packages; data can live on the server or stay local on the device. Please read this page before contributing; code changes must also follow [CODING_STANDARD.md](CODING_STANDARD.md) (Chinese: [docs/CODING_STANDARD-CN.md](docs/CODING_STANDARD-CN.md)).

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
- [Coding Standard](#coding-standard)
- [Branch Strategy](#branch-strategy)
- [Pull Request Process](#pull-request-process)
- [Commit Messages](#commit-messages)
- [Reporting Issues](#reporting-issues)

---

## Code of Conduct

When participating in this project, please:

- Use friendly, inclusive language
- Respect differing viewpoints and experiences
- Accept constructive criticism gracefully
- Act in the best interest of the project and its users

---

## How to Contribute

### Reporting Bugs

1. Search the issue list first to avoid duplicates
2. Use a clear, searchable title
3. Provide as much of the following as you can:
   - Package form (Web / EXE thin / EXE unified / portable / APK thin)
   - Operating system and version
   - Backend database (`DB_DRIVER`: sqlite / postgres / mysql) and whether Redis is enabled
   - Frontend startup method (`pnpm dev` / prebuilt package / Capacitor / Tauri)
   - Steps to reproduce
   - Expected behavior vs actual behavior
   - Relevant logs or error output (**do not paste API keys, JWTs, or plaintext `.env` contents**)

### Suggesting Features

1. Explain the problem to be solved, not just the implementation idea
2. Explain how it fits into the existing architecture (for example: context layering, the Skill protocol, the multi-platform packaging matrix)
3. If possible, provide an interface sketch or a usage example

When conversation personas / skill loading are involved, read [docs/SKILL-PROTOCOL.md](docs/SKILL-PROTOCOL.md) and [docs/issue/CONTEXT-SKILL-DESIGN.md](docs/issue/CONTEXT-SKILL-DESIGN.md) first, so as not to hard-code business personas into Go code.

### Documentation

Documentation improvements are always welcome — typos, unclear wording, missing translations, new guides, all of it. Technical design documents belong under `docs/` and should cross-reference the corresponding source paths.

---

## Coding Standard

All code **must** follow [CODING_STANDARD.md](CODING_STANDARD.md) (Chinese: [docs/CODING_STANDARD-CN.md](docs/CODING_STANDARD-CN.md)). Key points:

- **Layering**: `controller` → `service` → `repository` → `model`; HTTP / WebSocket details stay out of services, SQL stays out of controllers
- **Go naming**: exported `PascalCase`, unexported `camelCase`; JSON / table fields `snake_case`
- **TS naming**: variables/functions/interface fields `camelCase`; React component files and component names `PascalCase`; API fields align with backend JSON (mostly `snake_case`)
- **Formatting**: `gofmt` for Go (tabs); frontend TypeScript uses 2 spaces by default, never mix in tabs
- **Constructors**: `NewXxx(...)`, with dependency injection wired up in `cmd/main.go`
- **Security**: GORM parameterized queries; bcrypt for passwords; `/storage/*` paths must be confined within `STORAGE_DIR`; secrets go only into the backend `.env` and are never echoed back by the API

### Pre-commit Checklist

- [ ] `cd backend && go test ./...` passes
- [ ] `cd backend && go vet ./...` has no warnings
- [ ] `cd frontend && pnpm type-check` passes
- [ ] Changes respect layered responsibilities, with no Gin / SQL leaking into the wrong layer
- [ ] New / modified JSON fields match `frontend/src/types/api.ts`
- [ ] No API keys, passwords, or production `.env` files committed to the repo
- [ ] User-visible copy and persona behavior come from Skill MD / configuration, not hard-coded in business code
- [ ] Behavior involving clearing chats, memory cards, and export/import matches the decisions in [docs/issue/CONTEXT-SKILL-DESIGN.md](docs/issue/CONTEXT-SKILL-DESIGN.md)
- [ ] When packaging scripts or docs change, the corresponding sections under `docs/` are updated in sync

---

## Branch Strategy

- **`main`** — the releasable trunk. Merged into via PR only.
- **`feature/*`** — new features, branched from `main` and merged back into `main` when complete.
- **`fix/*`** — bug fixes, same as above.
- **`docs/*`** — documentation-only changes, can be PR'd straight to `main`.

Don't let large unfinished branches pile up on `main` for long; split big features into small steps that can be merged independently.

---

## Pull Request Process

```bash
# 1. Fork the repo (collaborators can clone the main repo directly)

# 2. Clone and enter the directory
git clone https://github.com/<your-username>/Cuetiy.git
cd Cuetiy
git remote add upstream https://github.com/<owner>/Cuetiy.git   # if there is an upstream

# 3. Branch off main
git checkout main
git pull upstream main
git checkout -b feature/<feature-name>

# 4. Develop, following CODING_STANDARD.md

# 5. Verify locally
cd backend && go test ./... && go vet ./...
cd ../frontend && pnpm type-check

# 6. Commit and push to your fork
git add -A
git commit -m "feat: short description"
git push origin feature/<feature-name>

# 7. Open a Pull Request on GitHub
#    From: feature/<feature-name> in your fork
#    To:   main in the upstream repo
```

### Before Submitting a PR

- Each commit does exactly one logical thing
- Rebase onto the latest `main` before merging:
  ```bash
  git fetch upstream
  git rebase upstream/main
  ```
- Run the relevant builds / type checks locally; for multi-platform packaging, at least verify the pipeline you changed (see [docs/PACKAGING.md](docs/PACKAGING.md))
- Keep the PR title clear; the body explains "what changed and why"; link the issue (e.g. `Closes #123`)
- Revise according to review feedback before merging

### Scope Guidelines

| Change type | Suggested scope |
|---|---|
| Conversation / context / Skill | Prefer changing the MD protocol and configuration; core routing only does matching and layering, with no hard-coded persona copy |
| API | Update `frontend/src/types/api.ts` and the relevant controller in sync |
| Packaging | Update `scripts/` and the corresponding build docs under `docs/` in sync |
| Refactoring only | No change to external JSON / packaging artifact behavior; submit as a separate PR |

---

## Commit Messages

Use the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>: <short description>

<optional body>
```

**Types:**

| Type | Purpose |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Refactoring (no external behavior change) |
| `docs` | Documentation only |
| `style` | Formatting, indentation, etc. |
| `build` | Build scripts / packaging pipeline |
| `chore` | Miscellaneous maintenance |
| `test` | Tests |
| `perf` | Performance |

**Examples:**

```
feat: skill frontmatter supports ttl_turns for sticky turn count

fix: keep memory cards by default when clearing chat, matching CONTEXT-SKILL-DESIGN decisions

docs: add EXE unified packaging steps

build: fix JDK/PATH issue in the Windows packaging script

test: add boundary cases for skill_layers compaction
```

Descriptions may be written in Chinese or English; just stay consistent within the same PR.

---

## Reporting Issues

When filing an issue, please include:

1. **Environment**: package form, OS, `DB_DRIVER`, Go / Node versions
2. **Steps to reproduce**: minimal, complete, verifiable
3. **Expected behavior**
4. **Actual behavior** (including logs; after redaction)
5. **Possible cause or fix idea** (if any)

For security-related issues (key leakage, unauthorized access to other users' sessions, path traversal), please don't publish full exploitation details publicly; you may describe the impact privately first.

---

*Thank you for helping make Cuetiy better.*
