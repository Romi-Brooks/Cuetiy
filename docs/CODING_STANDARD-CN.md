# Cuetiy — 代码规范

[English](../CODING_STANDARD.md) | [中文](CODING_STANDARD-CN.md)

---
> 本文件是 Cuetiy 项目架构与代码风格的快速参考。向本仓库贡献代码时请遵守本指南。  
> Go 以 `gofmt` / `go vet` 为格式与静态检查基准；前端以 `pnpm type-check` 为最低门禁。

参见贡献指南：[English](../CONTRIBUTING.md) | [中文](CONTRIBUTING-CN.md)

---

## 0. 技术栈与模块地图

| 层 | 位置 | 栈 |
|---|---|---|
| 后端 API / 业务 | `backend/` | Go 1.26, Gin, GORM, JWT, WebSocket |
| 技能与人设 | `backend/skill/` + 运行时 `skills/` | Markdown + YAML frontmatter |
| 前端 Web / 多端壳 | `frontend/` | React 18, Vite 5, TS, Zustand, Tailwind, Capacitor 7, Tauri 2 |
| 桌面宿主（可选） | `frontend/desktop-host/` | Go |
| 打包脚本 | `scripts/` + `dist/` | PowerShell / Bash |
| 技术文档 | `docs/` | Markdown |

产品矩阵（Web thin / EXE unified / portable / APK thin）决定改动的验证范围——见 [PACKAGING.md](PACKAGING.md)。

---

## 1. 目录与文件命名

### 1.1 后端目录

```
backend/
├── cmd/
│   └── main.go              # 装配与路由注册，尽量薄
├── config/                  # 环境变量、DB / Redis 初始化
├── controller/              # HTTP / WebSocket 入参出参、鉴权入口
├── middleware/              # Gin 中间件（JWT 等）
├── model/                   # GORM 模型与 AutoMigrate
├── repository/              # 数据访问，唯一允许直接摸 DB 的层
├── service/                 # 业务编排（上下文、TTS、导出、AI…）
├── skill/                   # 技能 registry / loader / 缓存 / embed
├── utils/                   # 无业务状态的纯函数
├── sql/                     # schema 脚本（sqlite / pg）
├── static/                  # 默认静态资源
└── .env.example             # 配置样例（勿提交真实 .env）
```

**规则：**

- 包名全小写、单词不混下划线（`skill`、`image_gen_service.go` 所在包仍是 `service`）
- 按**职责**分包，不按文件类型再套 `src/`、`include/`
- `controller` 不写 SQL；`repository` 不解析 HTTP；`service` 不依赖 `*gin.Context` 传参（需要用户 ID 时用显式参数）
- 装配与生命周期只放在 `cmd/main.go`

### 1.2 前端目录

```
frontend/src/
├── api/          # fetch 封装与 REST 端点
├── components/   # 可复用 UI（ChatBubble、VoiceBar…）
├── views/        # 路由级页面（Login、MainChat…）
├── stores/       # Zustand（chat、user、settings、theme…）
├── types/        # 与后端 JSON 对齐的类型
├── utils/        # 纯工具（时间、URL、分段、本地存储）
└── platform/     # Capacitor / Tauri / Web 差异抹平
```

**规则：**

- 状态进 `stores/`，不散落在组件闭包里做“全局真相”
- 平台差异只在 `platform/` 处理；组件不直接 `window.Capacitor` / `window.__TAURI__`
- API 字段变更必须同步 `types/api.ts`

### 1.3 文件命名

| 类型 | 约定 | 示例 |
|---|---|---|
| Go 源文件 | `snake_case.go` | `skill_layers.go`, `chat_controller.go` |
| Go 测试 | `*_test.go` 与被测文件同目录 | `skill_layers_test.go` |
| Go 构造 / 方法 | `NewXxx` / `PascalCase` | `NewChatController`, `HandleWebSocket` |
| React 组件文件 | `PascalCase.tsx` | `ChatBubble.tsx`, `MainChat.tsx` |
| TS 非组件 | `camelCase.ts` | `chat.ts`, `segments.ts` |
| 文档 | 现有风格：英文短横线或语义名 | `EXE-UNIFIED.md`, `issue/CONTEXT-SKILL-DESIGN.md` |
| 打包脚本 | `build-<target>-<variant>.ps1/.sh` | `build-exe-unified.ps1` |
| Skill MD | 模块语义名，常带连字符 | `Emotion-Companion.md`, `SKILL-DEFAULT.md` |

一个 Go 类型若是一文件主角，文件名用该概念的 `snake_case`（如 `context_assembler.go` 定义 `ContextAssembler`）；不必强求文件名与类型名逐字相同。

---

## 2. 分层与依赖

```text
cmd/main.go
    ↓ 装配
controller  →  service  →  repository  →  model
                 ↓
               skill / utils / config
```

| 层 | 可以做 | 不可以做 |
|---|---|---|
| `controller` | 参数绑定、鉴权读取、HTTP 状态码、DTO 转换 | SQL、业务分支、长事务 |
| `service` | 业务流程、编排 repository、调用 AI / TTS | 依赖 `*gin.Context`、拼裸 SQL |
| `repository` | GORM 查询、分页、软删过滤 | HTTP、第三方业务 API |
| `model` | 结构体 + GORM/JSON 标签 | 业务方法堆在模型里 |
| `skill` | 技能包加载、registry、缓存 | 具体人设文案写死 |

WebSocket 聊天在 `ChatController` / `WebSocketHub` 内完成接入后，组装与生成逻辑仍走 `service`（`ContextAssembler`、`AIService`、`MessageWriter` 等），避免 Hub 变成第二个业务层。

---

## 3. 命名约定

### 3.1 速查

| 类别 | 风格 | 示例 |
|---|---|---|
| Go 包名 | `snake_case` 小写 | `service`, `skill` |
| Go 导出类型 / 函数 / 方法 | `PascalCase` | `ChatController`, `LoadSkills` |
| Go 非导出标识符 | `camelCase` | `whereActive`, `voiceCooldown` |
| Go 结构体字段 | `PascalCase` | `ConversationID`, `MessageType` |
| JSON / DB 列 | `snake_case` | `conversation_id`, `keep_memory_on_clear` |
| 前端变量 / 函数 / store 字段 | `camelCase` | `resolveAssetUrl`, `randomReplyDelayMs` |
| 前端类型 / 接口名 | `PascalCase` | `ChatState`, `ImageGenDebugInfo` |
| React 组件 | `PascalCase` | `VoiceBubble` |
| CSS / Tailwind | 工具类优先；局部类 `camelCase` 或 BEM 均可，与邻近文件一致 | |
| 环境变量 | `UPPER_SNAKE_CASE` | `DB_DRIVER`, `CONTEXT_TOKEN_BUDGET` |
| HTTP JSON 字段 | `snake_case`（与 Go `json` 标签一致） | `want_voice`, `audio_duration_ms` |
| Skill `category` | `snake_case` 稳定 ID | `emotion_companion`, `image_appearance` |
| Skill `load_mode` | 小写枚举 | `always`, `index`, `trigger` |

### 3.2 Go

```go
// 推荐
type MessageRepository struct{}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

func (r *MessageRepository) FindOlderThan(convID int64, beforeID int64, limit int) ([]model.Message, error)

// 避免：导出名缩写无节制、JSON 用 camelCase、m_ 前缀
type msgRepo struct{ m_convId int64 } // ✗
```

- 构造函数统一 `NewXxx`，需要的依赖作参数传入，在 `cmd/main.go` 装配
- 接收者名单字母简写保持稳定（`r`、`ctl`、`svc`），同一类型内一致
- 错误返回 `error`，调用方处理或用 `%w` 包装；不要 `panic` 做业务分支
- 表模型字段注释可写中文；JSON 标签必须 `snake_case`

### 3.3 TypeScript / React

```ts
// 推荐
export interface Message {
  conversation_id: number
  message_type?: 'text' | 'voice' | 'image'
  created_at: string
}

export function formatConversationTime(dateStr: string): string { /* … */ }

// 组件文件 PascalCase.tsx
export function ChatBubble() { /* … */ }
```

- 与后端 DTO 对齐的字段**不要**擅自改成 `camelCase`——网络字段保持 `snake_case`，展示层可再映射
- 类型导入用 `import type`（纯类型）
- 布尔参数 / 字段用 `is` / `has` / `want` 等可读前缀：`has_ref_image`, `want_voice`

### 3.4 Skill frontmatter

遵守 [SKILL-PROTOCOL.md](SKILL-PROTOCOL.md)：

```yaml
category: emotion_companion   # 稳定 ID，注入 system 时使用
load_mode: trigger            # always | index | trigger
triggers:
  emotions: [joy, sad]
  keywords: [代码]
```

- 人设、语气、禁止项写在 MD 模块里，**禁止**再写死进 Go / TS 业务分支
- 路由只做 `tags ∩ triggers → category`，不理解具体人格语义

---

## 4. 代码格式

### 4.1 Go

- **缩进**：`gofmt` 默认 Tab；提交前请 format
- **括号**：K&R（开括号不换行）
- **导入分组**：标准库 → 第三方 → `cuetiy-backend/...`；用 `goimports` 整理

```go
package service

import (
	"fmt"
	"strings"

	"cuetiy-backend/skill"
)
```

- 字段对齐交给 `gofmt`；不要手写对齐空格然后和 format 工具打架

### 4.2 TypeScript / React

- **缩进**：2 空格；不用 Tab
- **分号**：与邻近文件一致（本仓库多数文件省略分号）
- **引号**：单引号
- **组件**：函数组件 + Hooks；props 类型就地 `interface` / `type` 或放 `types/`
- **副作用**：WebSocket、定时器、订阅必须在清理函数里释放

### 4.3 误差与空值

```go
// Go：显式 error，避免模糊零值
if err != nil {
	return nil, err
}
```

```ts
// TS：可选字段用 ?；不要用 ! 强断掩盖后端缺字段
audio_duration_ms?: number
```

---

## 5. API 与数据约定

| 项 | 约定 |
|---|---|
| 认证 | `Authorization: Bearer <JWT>`；WS 可用 `?token=`，但仅限无法带 Header 的场景 |
| 响应错误 | JSON `{"error": "..."}`；HTTP 状态码语义化 |
| 时间 | `RFC3339` / Go `time.Time` 默认 JSON 序列化；前端 `Date` 解析 |
| 软删 | `is_deleted`；查询必须过滤（参考 `whereActive`） |
| 分页 | 消息上翻用 `before_id` + `limit`；限制 `limit` 上限（如 ≤100） |
| 枚举字符串 | 小写英文：`text` / `voice` / `image`，不用中文章面当存储值 |
| 导出格式 | JSON schema 变更需向后兼容或在文档中写明迁移 |

新增字段：

1. `model` 加 GORM/JSON 标签  
2. `types/api.ts` 同步  
3. 需要迁移时更新 `backend/sql/schema.sql` 与 `schema.pg.sql`  
4. 在 PR 描述写明兼容性（旧包是否可读）

---

## 6. 安全基线

必须遵守，评审时按硬门槛看：

1. **SQL**：一律 GORM 参数化（`Where("id = ?", id)`）；禁止拼接用户输入进 SQL 字符串  
2. **密码**：bcrypt 存储；接口不回显密码或哈希  
3. **密钥**：只存在于后端 `.env` / 宿主环境变量；`.env`、`capacitor.config.json` 中的密钥、生产 `vite.config.ts` 不入库（用 `*.example`）  
4. **AI Key**：管理接口只写入、只报告「已配置 / 未配置」，不返回明文  
5. **文件路径**：`/storage/*` 与上传路径必须 `Clean` 并约束在 `STORAGE_DIR` 下，拒绝 `..` 穿越  
6. **多用户**：所有会话 / 消息 / 人格 / 文件查询必须带 `user_id`（或校验 `conversation` 归属），禁止只信前端传的 ID  
7. **XSS**：用户与模型文本入库前可走 `utils.Sanitize*`；前端渲染富文本时不要 `dangerouslySetInnerHTML` 未消毒内容  
8. **CORS / 明文**：本地开发可放宽；生产 Web 用 HTTPS；APK debug 允许 cleartext，release 不得默认允许  

---

## 7. 上下文与 Skill 行为

实现对话、压缩、记忆时以 [issue/CONTEXT-SKILL-DESIGN.md](issue/CONTEXT-SKILL-DESIGN.md) 已确认决策为准：

| 项 | 决策 |
|---|---|
| Token 预算 | 默认 16k（`CONTEXT_TOKEN_BUDGET`） |
| 压缩触发 | 组装占用 ≥ 55%（`CONTEXT_COMPACT_THRESHOLD`） |
| 压缩执行 | 优先 LLM；无 Key 降级启发式 |
| 清空聊天 | 先归档到 `data/archives/`；Skills 保留 |
| 记忆卡 ≠ Skills | 记忆卡是「关于用户」；Skills 是人格 MD |

代码中不要再硬编码具体人设角色名、语气段落、业务能力清单——空技能包时用中性占位（见 `skill_layers.go`）。

---

## 8. 测试

- 测试文件：与被测包同目录的 `*_test.go`
- 优先覆盖：上下文压缩、Skill 路由 / 分层、分段、导出导入、路径安全、权限过滤
- 命名：`TestXxx`；表驱动可用，但保持失败信息可读
- 运行：
  ```bash
  cd backend
  go test ./...
  go vet ./...
  ```
- 前端暂以类型检查为门禁；新增复杂纯函数（如 `segments.ts`）建议在同 PR 说明手工验证步骤
  ```bash
  cd frontend
  pnpm type-check
  ```

不要为了覆盖率堆无断言测试；不要在单测里依赖外网 AI / TTS 真实密钥。

---

## 9. 注释

与本仓库现状一致：**业务语义注释可用中文**；标识符、日志英文或中文均可，同一文件保持一致。API / 协议字段名始终英文 `snake_case`。

### 9.1 文件与类型

```go
// ContextAssembler 负责分层组装每轮输入。
// 目标形态：SKILLS(system 分层) + 压缩后 HISTORY(recent) + 当前用户消息。
type ContextAssembler struct {
	// …
}
```

```ts
/** 图片生成 debug：重点看 prompt（真正发给 image API 的提示词） */
export interface ImageGenDebugInfo {
  // …
}
```

### 9.2 复杂方法

写清「为什么」和不变量，而不是复述代码：

```go
// FindOlderThan 取比 beforeID 更旧的消息（分页上翻）。
// beforeID<=0 时等价于取最新一页。
func (r *MessageRepository) FindOlderThan(...)
```

```go
// 关键：压缩当轮必须把新摘要写回 system，再发给 LLM
```

### 9.3 TODO / FIXME

```go
// TODO: 分布式部署时 WebSocketHub 需迁移到 Redis Pub/Sub
// FIXME: 并发写 voiceCooldown 时需加锁
```

带上下文，不要只写 `TODO: fix`。

---

## 10. 多端与打包改动

| 变更 | 必须同步 |
|---|---|
| 环境变量 | `backend/.env.example`、`frontend/.env.*.example`、相关 `docs/` |
| REST / WS 协议 | `types/api.ts`、调试面板字段（若暴露）、导出 JSON |
| 打包矩阵 | `scripts/build-*.ps1|.sh` 与 `docs/EXE-*` / `APK-THIN` / `LINUX-PACKAGING` |
| Tauri / Capacitor 权限 | `tauri.conf.json` / `capabilities/` / Android 配置；收窄而非扩大权限 |
| 默认 Skill | `backend/skill/default_skills/`、`frontend/src-tauri/resources/skills/`（以实际打包脚本拷贝路径为准） |

产物目录 `dist-packages/`、`backend/output/` 中的二进制与日志**不要**提交进 Git。

---

## 11. 提交前自查（与 CONTRIBUTING 清单对应）

```bash
cd backend && gofmt -l . && go test ./... && go vet ./...
cd ../frontend && pnpm type-check
```

- 无密钥、无绝对本机路径写死在业务代码  
- JSON 字段前后端一致  
- 人设 / 技能语义不在核心代码里写死  
- 权限与路径校验无回归  
