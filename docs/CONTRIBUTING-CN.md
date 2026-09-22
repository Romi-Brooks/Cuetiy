# 为 Cuetiy 做贡献

[English](../CONTRIBUTING.md) | [中文](CONTRIBUTING-CN.md)

---
感谢你愿意为 Cuetiy 贡献力量。无论是缺陷报告、功能建议、文档改进，还是代码变更，我们都欢迎。

Cuetiy 是一个 AI 陪伴聊天产品，覆盖 Web / EXE / APK 多端打包；数据可放在服务端，也可留在设备本地。贡献前请先读完本页，代码变更还需遵守 `CODING_STANDARD.md`（[English](../CODING_STANDARD.md) / [中文](CODING_STANDARD-CN.md)）。

---

## 目录

- [行为准则](#行为准则)
- [如何贡献](#如何贡献)
- [代码规范](#代码规范)
- [分支策略](#分支策略)
- [Pull Request 流程](#pull-request-流程)
- [提交信息](#提交信息)
- [报告问题](#报告问题)

---

## 行为准则

参与本项目时，请做到：

- 使用友善、包容的语言
- 尊重不同观点与经验
- 坦然接受建设性批评
- 以对项目与用户最有利为准

---

## 如何贡献

### 报告缺陷

1. 先在 issue 列表中搜索，避免重复
2. 使用清晰、可检索的标题
3. 尽量提供：
   - 打包形态（Web / EXE thin / EXE unified / portable / APK thin）
   - 操作系统与版本
   - 后端数据库（`DB_DRIVER`：sqlite / postgres / mysql）与 Redis 是否启用
   - 前端启动方式（`pnpm dev` / 预编译包 / Capacitor / Tauri）
   - 复现步骤
   - 期望行为 vs 实际行为
   - 相关日志或错误输出（**不要粘贴 API Key、JWT、`.env` 明文**）

### 建议功能

1. 说明要解决的问题，而不只是实现想法
2. 说明它如何融入现有架构（例如：上下文分层、Skill 协议、多端打包矩阵）
3. 如有可能，给出接口草图或使用示例

涉及对话人格 / 技能加载时，请先读 [SKILL-PROTOCOL.md](SKILL-PROTOCOL.md) 与 [issue/CONTEXT-SKILL-DESIGN.md](issue/CONTEXT-SKILL-DESIGN.md)，避免把业务人设写死进 Go 代码。

### 文档

文档改进始终欢迎——错别字、表述不清、缺失翻译、新指南都可以。技术方案文档优先放在 `docs/`，并与对应源码路径交叉引用。

---

## 代码规范

所有代码**必须**遵守 `CODING_STANDARD.md`（[English](../CODING_STANDARD.md) / [中文](CODING_STANDARD-CN.md)）。要点：

- **分层**：`controller` → `service` → `repository` → `model`；HTTP / WebSocket 细节不进 service，SQL 不进 controller
- **Go 命名**：导出 `PascalCase`，非导出 `camelCase`；JSON / 表字段 `snake_case`
- **TS 命名**：变量/函数/接口字段 `camelCase`；React 组件文件与组件名 `PascalCase`；API 字段与后端 JSON 对齐（多为 `snake_case`）
- **格式**：Go 用 `gofmt`（Tab）；前端 TypeScript 默认 2 空格，不混用 Tab
- **构造函数**：`NewXxx(...)`，依赖注入写在 `cmd/main.go` 装配
- **安全**：GORM 参数化查询；密码 bcrypt；`/storage/*` 路径必须限制在 `STORAGE_DIR` 内；密钥只进后端 `.env`，接口不回显

### 提交前检查清单

- [ ] `cd backend && go test ./...` 通过
- [ ] `cd backend && go vet ./...` 无告警
- [ ] `cd frontend && pnpm type-check` 通过
- [ ] 改动符合分层职责，没有把 Gin / SQL 泄漏到错误层
- [ ] 新增 / 修改的 JSON 字段与 `frontend/src/types/api.ts` 一致
- [ ] 没有把 API Key、口令、生产 `.env` 写进仓库
- [ ] 用户可见文案与人设行为来自 Skill MD / 配置，而不是写死在业务代码里
- [ ] 涉及清空聊天、记忆卡、导出导入的行为与 [issue/CONTEXT-SKILL-DESIGN.md](issue/CONTEXT-SKILL-DESIGN.md) 决策一致
- [ ] 打包脚本或文档有变更时，同步更新 `docs/` 下对应章节

---

## 分支策略

- **`main`** — 可发布主干。只通过 PR 合入。
- **`feature/*`** — 新功能，从 `main` 拉出，完成后合回 `main`。
- **`fix/*`** — 缺陷修复，同上。
- **`docs/*`** — 纯文档变更，可直接 PR 到 `main`。

不要把未完成的大分支长期堆在 `main` 上；大功能拆成可独立合入的小步。

---

## Pull Request 流程

```bash
# 1. Fork 仓库（协作者可直接 clone 主仓库）

# 2. 克隆并进入目录
git clone https://github.com/<your-username>/Cuetiy.git
cd Cuetiy
git remote add upstream https://github.com/<owner>/Cuetiy.git   # 如有 upstream

# 3. 从 main 拉出功能分支
git checkout main
git pull upstream main
git checkout -b feature/<feature-name>

# 4. 开发，遵守 CODING_STANDARD.md

# 5. 本地验证
cd backend && go test ./... && go vet ./...
cd ../frontend && pnpm type-check

# 6. 提交并推送到你的 fork
git add -A
git commit -m "feat: 简短说明"
git push origin feature/<feature-name>

# 7. 在 GitHub 发起 Pull Request
#    From: 你的 fork 的 feature/<feature-name>
#    To:   上游仓库的 main
```

### 提交 PR 前

- 每个 commit 只做一件逻辑上的事
- 合并前 rebase 到最新 `main`：
  ```bash
  git fetch upstream
  git rebase upstream/main
  ```
- 本地跑通相关构建 / 类型检查；涉及多端打包时至少验证你改动的那条链路（见 [PACKAGING.md](PACKAGING.md)）
- PR 标题清晰；正文说明「改了什么、为什么」；关联 issue（如 `Closes #123`）
- 根据评审意见修改后再合并

### 范围约定

| 变更类型 | 建议范围 |
|---|---|
| 对话 / 上下文 / Skill | 优先改 MD 协议与配置；核心路由只做匹配与分层，不写死人设文案 |
| API | 同步更新 `frontend/src/types/api.ts` 与相关 controller |
| 打包 | 同步更新 `scripts/` 与 `docs/` 对应构建文档 |
| 仅重构 | 不改对外 JSON / 打包产物行为，单独 PR |

---

## 提交信息

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>: <简短描述>

<可选正文>
```

**类型：**

| Type | 用途 |
|---|---|
| `feat` | 新功能 |
| `fix` | 缺陷修复 |
| `refactor` | 重构（不改对外行为） |
| `docs` | 仅文档 |
| `style` | 格式、缩进等 |
| `build` | 构建脚本 / 打包链路 |
| `chore` | 杂项维护 |
| `test` | 测试 |
| `perf` | 性能 |

**示例：**

```
feat: 技能 frontmatter 支持 ttl_turns 粘性轮数

fix: 清空聊天时记忆卡默认保留，与 CONTEXT-SKILL-DESIGN 决策对齐

docs: 补充 EXE unified 打包步骤

build: 修复 Windows 打包脚本 JDK/PATH 问题

test: 为 skill_layers 压缩补边界用例
```

描述可用中文或英文，同一 PR 内保持一致即可。

---

## 报告问题

提 issue 时建议包含：

1. **环境**：打包形态、OS、`DB_DRIVER`、Go / Node 版本
2. **复现步骤**：最小、完整、可验证
3. **期望行为**
4. **实际行为**（含日志；脱敏后）
5. **可能的原因或修复思路**（如有）

安全相关问题（密钥泄漏、越权读取他人会话、路径穿越）请勿公开发完整利用细节，可先私下说明影响面。

---

*感谢你帮助 Cuetiy 变得更好。*
