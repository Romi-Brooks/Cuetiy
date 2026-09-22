# Cuetiy — Coding Standard

[English](CODING_STANDARD.md) | [中文](docs/CODING_STANDARD-CN.md)

---
> This document is a quick reference for the Cuetiy project's architecture and code style. Please follow this guide when contributing code to this repository.  
> Go uses `gofmt` / `go vet` as the baseline for formatting and static checks; the frontend uses `pnpm type-check` as the minimum gate.

See the contributing guide: [CONTRIBUTING.md](CONTRIBUTING.md).

---

## 0. Tech Stack and Module Map

| Layer | Location | Stack |
|---|---|---|
| Backend API / business | `backend/` | Go 1.26, Gin, GORM, JWT, WebSocket |
| Skills and personas | `backend/skill/` + runtime `skills/` | Markdown + YAML frontmatter |
| Frontend Web / multi-platform shell | `frontend/` | React 18, Vite 5, TS, Zustand, Tailwind, Capacitor 7, Tauri 2 |
| Desktop host (optional) | `frontend/desktop-host/` | Go |
| Packaging scripts | `scripts/` + `dist/` | PowerShell / Bash |
| Technical docs | `docs/` | Markdown |

The product matrix (Web thin / EXE unified / portable / APK thin) determines the verification scope of a change — see [docs/PACKAGING.md](docs/PACKAGING.md).

---

## 1. Directory and File Naming

### 1.1 Backend directories

```
backend/
├── cmd/
│   └── main.go              # assembly and route registration; keep it as thin as possible
├── config/                  # environment variables, DB / Redis initialization
├── controller/              # HTTP / WebSocket request-response, auth entry point
├── middleware/              # Gin middleware (JWT, etc.)
├── model/                   # GORM models and AutoMigrate
├── repository/              # data access; the only layer allowed to touch the DB directly
├── service/                 # business orchestration (context, TTS, export, AI…)
├── skill/                   # skill registry / loader / cache / embed
├── utils/                   # pure functions with no business state
├── sql/                     # schema scripts (sqlite / pg)
├── static/                  # default static assets
└── .env.example             # configuration sample (never commit a real .env)
```

**Rules:**

- Package names are all lowercase with no underscores between words (the package containing `skill` and `image_gen_service.go` is still `service`)
- Split packages by **responsibility**, not by file type into extra `src/` or `include/` folders
- `controller` does not write SQL; `repository` does not parse HTTP; `service` does not take parameters via `*gin.Context` (use explicit parameters when a user ID is needed)
- Assembly and lifecycle live only in `cmd/main.go`

### 1.2 Frontend directories

```
frontend/src/
├── api/          # fetch wrappers and REST endpoints
├── components/   # reusable UI (ChatBubble, VoiceBar…)
├── views/        # route-level pages (Login, MainChat…)
├── stores/       # Zustand (chat, user, settings, theme…)
├── types/        # types aligned with the backend JSON
├── utils/        # pure utilities (time, URL, segmentation, local storage)
└── platform/     # smoothes over Capacitor / Tauri / Web differences
```

**Rules:**

- State goes into `stores/`, not scattered through component closures as "global truth"
- Platform differences are handled only in `platform/`; components do not touch `window.Capacitor` / `window.__TAURI__` directly
- API field changes must be mirrored in `types/api.ts`

### 1.3 File naming

| Type | Convention | Example |
|---|---|---|
| Go source files | `snake_case.go` | `skill_layers.go`, `chat_controller.go` |
| Go tests | `*_test.go` in the same directory as the file under test | `skill_layers_test.go` |
| Go constructors / methods | `NewXxx` / `PascalCase` | `NewChatController`, `HandleWebSocket` |
| React component files | `PascalCase.tsx` | `ChatBubble.tsx`, `MainChat.tsx` |
| Non-component TS | `camelCase.ts` | `chat.ts`, `segments.ts` |
| Documentation | Existing style: English kebab-case or a semantic name | `EXE-UNIFIED.md`, `docs/issue/CONTEXT-SKILL-DESIGN.md` |
| Packaging scripts | `build-<target>-<variant>.ps1/.sh` | `build-exe-unified.ps1` |
| Skill MD | Module semantic name, often hyphenated | `Emotion-Companion.md`, `SKILL-DEFAULT.md` |

If a Go type is the star of a file, name the file with the `snake_case` form of that concept (e.g. `context_assembler.go` defines `ContextAssembler`); the file name need not match the type name word for word.

---

## 2. Layering and Dependencies

```text
cmd/main.go
    ↓ assembly
controller  →  service  →  repository  →  model
                 ↓
               skill / utils / config
```

| Layer | May do | May not do |
|---|---|---|
| `controller` | parameter binding, auth reading, HTTP status codes, DTO conversion | SQL, business branching, long transactions |
| `service` | business flows, orchestrating repositories, calling AI / TTS | depending on `*gin.Context`, writing raw SQL |
| `repository` | GORM queries, pagination, soft-delete filtering | HTTP, third-party business APIs |
| `model` | structs + GORM/JSON tags | piling business methods onto the model |
| `skill` | skill package loading, registry, caching | hardcoding specific persona copy |

Once WebSocket chat is accepted inside `ChatController` / `WebSocketHub`, assembly and generation still go through `service` (`ContextAssembler`, `AIService`, `MessageWriter`, etc.), so the Hub does not become a second business layer.

---

## 3. Naming Conventions

### 3.1 Quick reference

| Category | Style | Example |
|---|---|---|
| Go package names | lowercase `snake_case` | `service`, `skill` |
| Go exported types / functions / methods | `PascalCase` | `ChatController`, `LoadSkills` |
| Go unexported identifiers | `camelCase` | `whereActive`, `voiceCooldown` |
| Go struct fields | `PascalCase` | `ConversationID`, `MessageType` |
| JSON / DB columns | `snake_case` | `conversation_id`, `keep_memory_on_clear` |
| Frontend variables / functions / store fields | `camelCase` | `resolveAssetUrl`, `randomReplyDelayMs` |
| Frontend type / interface names | `PascalCase` | `ChatState`, `ImageGenDebugInfo` |
| React components | `PascalCase` | `VoiceBubble` |
| CSS / Tailwind | Utility classes first; local classes may be `camelCase` or BEM, consistent with neighboring files | |
| Environment variables | `UPPER_SNAKE_CASE` | `DB_DRIVER`, `CONTEXT_TOKEN_BUDGET` |
| HTTP JSON fields | `snake_case` (matching the Go `json` tags) | `want_voice`, `audio_duration_ms` |
| Skill `category` | stable `snake_case` ID | `emotion_companion`, `image_appearance` |
| Skill `load_mode` | lowercase enum | `always`, `index`, `trigger` |

### 3.2 Go

```go
// Recommended
type MessageRepository struct{}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

func (r *MessageRepository) FindOlderThan(convID int64, beforeID int64, limit int) ([]model.Message, error)

// Avoid: unbounded abbreviations in exported names, camelCase in JSON, m_ prefixes
type msgRepo struct{ m_convId int64 } // ✗
```

- Constructors are uniformly `NewXxx`; required dependencies are passed as parameters and assembled in `cmd/main.go`
- Receiver names keep stable single-letter abbreviations (`r`, `ctl`, `svc`), consistent within the same type
- Return `error` and let the caller handle it or wrap it with `%w`; do not `panic` for business branching
- Field comments on table models may be written in Chinese; JSON tags must be `snake_case`

### 3.3 TypeScript / React

```ts
// Recommended
export interface Message {
  conversation_id: number
  message_type?: 'text' | 'voice' | 'image'
  created_at: string
}

export function formatConversationTime(dateStr: string): string { /* … */ }

// Component files are PascalCase.tsx
export function ChatBubble() { /* … */ }
```

- Fields aligned with backend DTOs must **not** be renamed to `camelCase` on a whim — wire fields stay `snake_case`; the presentation layer may map them
- Use `import type` for type-only imports
- Use readable prefixes such as `is` / `has` / `want` for boolean parameters / fields: `has_ref_image`, `want_voice`

### 3.4 Skill frontmatter

Follow [docs/SKILL-PROTOCOL.md](docs/SKILL-PROTOCOL.md):

```yaml
category: emotion_companion   # stable ID, used when injecting into system
load_mode: trigger            # always | index | trigger
triggers:
  emotions: [joy, sad]
  keywords: [code]
```

- Persona, tone and prohibitions belong in MD modules; they must **not** be hardcoded into Go / TS business branches
- Routing only performs `tags ∩ triggers → category` and does not understand specific persona semantics

---

## 4. Code Formatting

### 4.1 Go

- **Indentation**: `gofmt` default Tab; format before committing
- **Braces**: K&R (the opening brace does not start a new line)
- **Import grouping**: standard library → third-party → `cuetiy-backend/...`; organize with `goimports`

```go
package service

import (
	"fmt"
	"strings"

	"cuetiy-backend/skill"
)
```

- Leave field alignment to `gofmt`; do not hand-write alignment spaces and then fight the formatting tool

### 4.2 TypeScript / React

- **Indentation**: 2 spaces; no tabs
- **Semicolons**: consistent with neighboring files (most files in this repository omit them)
- **Quotes**: single quotes
- **Components**: function components + Hooks; prop types as an inline `interface` / `type` or under `types/`
- **Side effects**: WebSocket, timers and subscriptions must be released in the cleanup function

### 4.3 Errors and Null Values

```go
// Go: explicit error, avoid ambiguous zero values
if err != nil {
	return nil, err
}
```

```ts
// TS: use ? for optional fields; do not use ! to mask fields missing from the backend
audio_duration_ms?: number
```

---

## 5. API and Data Conventions

| Item | Convention |
|---|---|
| Authentication | `Authorization: Bearer <JWT>`; WS may use `?token=`, but only where a header cannot be sent |
| Error responses | JSON `{"error": "..."}`; semantic HTTP status codes |
| Time | `RFC3339` / Go `time.Time` default JSON serialization; parsed as `Date` on the frontend |
| Soft delete | `is_deleted`; queries must filter on it (see `whereActive`) |
| Pagination | Scrolling back through messages uses `before_id` + `limit`; cap the maximum `limit` (e.g. ≤100) |
| Enum strings | Lowercase English: `text` / `voice` / `image`; never use Chinese literals as stored values |
| Export format | JSON schema changes must be backward compatible or document the migration |

Adding a field:

1. Add GORM/JSON tags in `model`  
2. Mirror it in `types/api.ts`  
3. When a migration is needed, update `backend/sql/schema.sql` and `schema.pg.sql`  
4. State the compatibility in the PR description (whether old packages remain readable)

---

## 6. Security Baseline

Must be followed; reviewers treat these as hard gates:

1. **SQL**: always parameterize with GORM (`Where("id = ?", id)`); never concatenate user input into SQL strings  
2. **Passwords**: store with bcrypt; APIs never echo passwords or hashes  
3. **Secrets**: exist only in the backend `.env` / host environment variables; secrets in `.env`, `capacitor.config.json`, and production `vite.config.ts` are not committed (use `*.example`)  
4. **AI Key**: the admin API only writes and only reports "configured / not configured"; it never returns plaintext  
5. **File paths**: `/storage/*` and upload paths must be `Clean`ed and constrained under `STORAGE_DIR`; reject `..` traversal  
6. **Multi-user**: every conversation / message / persona / file query must carry `user_id` (or verify `conversation` ownership); never trust an ID sent by the frontend alone  
7. **XSS**: user and model text may go through `utils.Sanitize*` before storage; do not render unsanitized content with `dangerouslySetInnerHTML` on the frontend  
8. **CORS / cleartext**: local development may relax this; production Web uses HTTPS; APK debug may allow cleartext, release must not allow it by default  

---

## 7. Context and Skill Behavior

When implementing conversation, compaction and memory, follow the decisions already confirmed in [docs/issue/CONTEXT-SKILL-DESIGN.md](docs/issue/CONTEXT-SKILL-DESIGN.md):

| Item | Decision |
|---|---|
| Token budget | 16k by default (`CONTEXT_TOKEN_BUDGET`) |
| Compaction trigger | Assembly usage ≥ 55% (`CONTEXT_COMPACT_THRESHOLD`) |
| Compaction execution | Prefer the LLM; fall back to heuristics when there is no Key |
| Clearing a chat | Archive to `data/archives/` first; Skills are retained |
| Memory cards ≠ Skills | Memory cards are "about the user"; Skills are persona MD |

Do not hardcode specific persona role names, tone passages or business capability lists in code — use neutral placeholders when the skill package is empty (see `skill_layers.go`).

---

## 8. Testing

- Test files: `*_test.go` in the same directory as the package under test
- Prioritize coverage of: context compaction, Skill routing / layering, segmentation, export/import, path safety, permission filtering
- Naming: `TestXxx`; table-driven is fine, but keep failure messages readable
- Running:
  ```bash
  cd backend
  go test ./...
  go vet ./...
  ```
- The frontend currently gates on type checking; for new complex pure functions (such as `segments.ts`), describe the manual verification steps in the same PR
  ```bash
  cd frontend
  pnpm type-check
  ```

Do not pile up assertion-free tests just to raise coverage; do not rely on real external AI / TTS keys in unit tests.

---

## 9. Comments

Consistent with the current state of this repository: **business semantic comments may be written in Chinese**; identifiers and logs may be English or Chinese, as long as they stay consistent within a file. API / protocol field names are always English `snake_case`.

### 9.1 Files and types

```go
// ContextAssembler assembles each turn's input in layers.
// Target shape: SKILLS (layered system) + compacted HISTORY (recent) + the current user message.
type ContextAssembler struct {
	// …
}
```

```ts
/** Image generation debug: focus on prompt (the prompt actually sent to the image API) */
export interface ImageGenDebugInfo {
  // …
}
```

### 9.2 Complex methods

Write down the "why" and the invariants rather than restating the code:

```go
// FindOlderThan fetches messages older than beforeID (pagination scroll-back).
// beforeID<=0 is equivalent to fetching the newest page.
func (r *MessageRepository) FindOlderThan(...)
```

```go
// Key: on a compaction turn, the new summary must be written back into system before sending to the LLM
```

### 9.3 TODO / FIXME

```go
// TODO: WebSocketHub needs to migrate to Redis Pub/Sub for distributed deployment
// FIXME: concurrent writes to voiceCooldown need a lock
```

Include context; do not just write `TODO: fix`.

---

## 10. Multi-Platform and Packaging Changes

| Change | Must be synchronized |
|---|---|
| Environment variables | `backend/.env.example`, `frontend/.env.*.example`, related `docs/` |
| REST / WS protocol | `types/api.ts`, debug panel fields (if exposed), exported JSON |
| Packaging matrix | `scripts/build-*.ps1|.sh` and `docs/EXE-*` / `APK-THIN` / `LINUX-PACKAGING` |
| Tauri / Capacitor permissions | `tauri.conf.json` / `capabilities/` / Android config; narrow permissions rather than widen them |
| Default Skill | `backend/skill/default_skills/`, `frontend/src-tauri/resources/skills/` (the copy paths in the actual packaging scripts prevail) |

Binaries and logs under the output directories `dist-packages/` and `backend/output/` must **not** be committed to Git.

---

## 11. Pre-Commit Self-Check (corresponds to the CONTRIBUTING checklist)

```bash
cd backend && gofmt -l . && go test ./... && go vet ./...
cd ../frontend && pnpm type-check
```

- No secrets, no absolute local paths hardcoded in business code  
- JSON fields consistent between frontend and backend  
- Persona / skill semantics not hardcoded in core code  
- No regressions in permission and path validation  
