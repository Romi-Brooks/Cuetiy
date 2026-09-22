# 问题与评估（issue）

> 本目录存放**尚未完全落地**的设计问题、技术方案与评估结论，作为后续评估与排期的输入。

## 用途

- 记录一个具体问题的**背景、现状、备选方案、结论摘要**，而不是使用手册。
- 每篇文档开头必须写明**状态**，例如 `已部分落地` / `评估中` / `未实施` / `已废弃`，并列出关联源码路径。
- 结论一旦落地，请回写到对应的 `docs/` 文档或 `CODING_STANDARD.md`，并在此更新状态。
- 需要对外跟踪的事项，可在 GitHub 上用 [issue 模板](../../.github/ISSUE_TEMPLATE/) 建条目，并在此文档内互相引用。

## 索引

| 文档 | 状态 | 主题 |
|------|------|------|
| [CONTEXT-SKILL-DESIGN.md](CONTEXT-SKILL-DESIGN.md) | 已部分落地（后端 Phase0–2） | 上下文分层组装、Token 预算与压缩、L2 技能触发 |
| [STREAMING-MULTITURN-EVAL.md](STREAMING-MULTITURN-EVAL.md) | 评估 / 未实施 | 流式打断、段级引用、用户多段输入合并 |

## 约定

- 文件名用**大写下划线 / 连字符**语义名，与现有 `docs/` 风格一致（如 `CONTEXT-SKILL-DESIGN.md`）。
- 方案与评估结论可用中文书写；标识符、字段名、路径保持英文。
- 与源码交叉引用时使用**仓库根目录相对路径**（如 `backend/service/skill_router.go`）。
