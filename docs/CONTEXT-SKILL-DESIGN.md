# Cuetiy 上下文与 Skill 加载技术方案

> 状态：**已部分落地（后端 Phase0–2）**  
> 关联源材料：项目内 `skills/` / `backend/skill/default_skills/` 技能 Markdown  
> 关联代码：`backend/service/context_assembler.go`、`skill_router.go`、`summary_service.go`、`archive_service.go`

---

## 0. 已确认决策（2026）

| 项 | 决策 |
|----|------|
| Token 预算 | **16k**（`CONTEXT_TOKEN_BUDGET=16384`） |
| 压缩触发 | 本轮组装占用 ≥ **55%** 预算（`CONTEXT_COMPACT_THRESHOLD=0.55`） |
| 压缩执行方 | **交给 AI**（DeepSeek 压缩器；无 Key 时降级启发式） |
| 情绪匹配 | AI 分析用户情绪/意图 + 后端匹配 **注入对应 MD**（L2） |
| 清空聊天 | **Skills 保留**；先将 **消息 + 记忆卡** 导出到 `data/archives/` 再删；记忆卡默认保留 |
| 记忆卡 ≠ Skills | 记忆卡是「关于用户」；Skills 是人格 MD；仅删除人格才动 Skills |

---

## 1. 背景与问题

当前每轮发给 DeepSeek 的消息结构是：

```text
[system] 人格 skill 整包编译 + AI 昵称
[history] 最近 20 条原文（MaxContextLength = 20）
[user]   本轮输入
```

结合 `RainSkill` 实测：

| 文件 | 大小 | 语义角色 |
|------|------|----------|
| Persona-Base.md | ~1.0KB | 核心身份（常驻） |
| Persona-Tone.md | ~0.9KB | 语气规范（常驻） |
| Forbidden-Rules.md | ~1.0KB | 禁止行为（常驻） |
| Emotion-Companion.md | ~1.6KB | 情绪陪伴（高频） |
| Style-Switch.md | ~1.4KB | 风格切换（场景） |
| Professional-Skills.md | ~1.3KB | 专业咨询（触发） |
| Trigger-Pet-Peeve.md | ~1.6KB | 雷点小脾气（触发） |
| **合计** | **~8.9KB** | 中文约 **5k～8k token 量级** |

### 1.1 已暴露的问题

1. **Skill 整包每轮重发**  
   无论用户在闲聊还是写代码，7 个模块全部进入 system。陪伴闲聊时 Professional-Skills 基本无用，却持续占 token。

2. **上下文窗口过浅**  
   20 条对「模拟女友 / 长线陪伴」几乎聊胜于无；没有滚动摘要、没有长期记忆，长聊必失忆。

3. **无压缩策略**  
   没有 token 预算，也没有「占用到阈值再压缩」的机制；条数硬切会丢关键事实。

4. **DB 查询不是主要瓶颈**  
   单机每轮读 20～200 条数据库消息是毫秒级。真正贵的是 **发给模型的 input token**。Redis 当前只是写穿镜像，读路径已回到 DB 权威，价值有限。

5. **代码里已有半成品分层**  
   `detectModuleCategory` / `isCoreModule` 已区分核心三类与触发类，但 `CompilePromptFromFiles` 仍全量拼接，分层未真正生效。

### 1.2 必须先说清的约束（容易误判）

**标准 Chat Completions API 是无状态的。**  
模型不会「第一次记住 SKILLS、后面就不用再发」。每次 HTTP 请求都是独立会话；不把 system/skill 放进 messages，模型本轮就不知道人设。

因此「只发一次」在纯 API 层面不成立。可行的等价手段是：

| 手段 | 效果 |
|------|------|
| **压缩 system**（技能摘要 + 按需注入全文） | 真正减 token，首选 |
| **稳定前缀 + 提供商 Prompt Cache** | 若 DeepSeek 对重复前缀计费折扣，可降费用/延迟（需实测） |
| **微调 / 自托管状态化服务** | 成本高，本阶段不做 |

本方案的核心不是「幻想模型记住」，而是：**每轮仍发 system，但把 system 从「全量 skill」改成「轻量核心 + 摘要索引 + 触发注入」**。

---

## 2. 目标与非目标

### 2.1 目标

1. 每轮 input token 可预算、可度量、可回收。  
2. Skill 分层：常驻精简 / 触发注入 / 偶然激活。  
3. 上下文占用达到阈值（**55% 预算**）自动 **AI 压缩**。  
4. 长期记忆（称呼、喜好、关键事件）跨压缩、跨清空保留。  
5. 清空聊天可溯源（文件归档），Skills 不受清空影响。  
6. 保持现有 DeepSeek + PostgreSQL/MySQL/SQLite（`DB_DRIVER`）+（可选）Redis 架构，不引入重型新栈。

### 2.2 非目标（本期不做）

- 微调专用陪伴模型  
- 多 Agent 编排 / 工具调用循环  
- 把上下文拉到 1M 并每轮塞满  
- 分布式多实例会话粘性（仍单机 WebSocket Hub）

---

## 3. 总体架构

```text
                    ┌─────────────────────────────────────┐
 用户消息 ──────────►│        ContextAssembler             │
                    │  1. 情绪/意图分析（规则 + 可选 AI）   │
                    │  2. Skill L2 激活 + TTL              │
                    │  3. L0 压缩 + L1 索引 + 记忆 + 摘要  │
                    │  4. 占用 ≥55% → AI 滚动摘要压缩      │
                    │  5. Recent 按 token 装填             │
                    └──────────────┬──────────────────────┘
                                   │ messages[]
                                   ▼
                            DeepSeek Chat API
```

每轮最终 messages 形态：

```text
[system]
  A. 核心人设 L0（压缩版，常驻）
  B. 能力索引 L1
  C. 长期记忆卡（用户事实）
  D. 此前对话纪要（滚动摘要）
  E. 本轮激活 Skill 全文 L2（情绪/意图命中）
[recent] 最近原文（token 预算内）
[user]   本轮输入
```

---

## 4. Skill 分层加载设计

### 4.1 三层模型

| 层级 | 名称 | 加载策略 | 对应 RainSkill |
|------|------|----------|----------------|
| L0 | 核心常驻 | 每轮必进 system（压缩版） | Persona-Base, Persona-Tone, Forbidden-Rules |
| L1 | 能力索引 | 每轮一行摘要，不进全文 | 全部非 L0 模块的名字+一句话 |
| L2 | 触发注入 | 情绪/意图命中才注入全文，带 TTL | Emotion-Companion, Style-Switch, Professional-Skills, Trigger-Pet-Peeve |

### 4.2 情绪 → MD 匹配（已确认方案）

```text
用户消息
  → 规则启发式（关键词）
  → 不确定时 AI 分类（小 max_tokens）
  → CategoriesForEmotion(emotion)
  → 注入对应 module_category 的 MD 全文 + TTL
```

| 情绪/意图标签 | 注入模块 |
|---------------|----------|
| joy / sad / anxious / angry / tired / flirty | emotion_companion |
| need_professional | professional_skills + emotion_companion |
| pet_peeve | trigger_rules + emotion_companion |
| style_switch | style_switch |

### 4.3 粘性与衰减

- 激活后 **TTL 默认 5 轮**（`SKILL_L2_TTL_TURNS`）  
- 同技能连续命中刷新 TTL  
- 状态存 `conversation_skill_states`

---

## 5. 上下文压缩（AI）

### 5.1 Token 预算（已确认）

| 项 | 值 |
|----|-----|
| `CONTEXT_TOKEN_BUDGET` | **16384** |
| 压缩触发 | **≥ 55%** × 预算 ≈ 9000 token |
| 摘要上限 | 800 汉字 |
| 近期最少保留 | 6 条 |

### 5.2 压缩流程（交给 AI）

1. 组装前估算 tokens  
2. 超过阈值时，取较旧消息段  
3. **DeepSeek 压缩器**（非女友人设 system）合并旧摘要+新段 → 新纪要  
4. 写入 `conversation_summaries`，`covered_message_id` 前移  
5. Recent 只留最新 K 条  
6. 同步/顺带用 AI 抽取记忆卡  

无 API Key 时降级为启发式拼接摘要。

### 5.3 不删业务数据

压缩 **只影响发给模型的 messages**；`messages` 表与前端分页不受影响。

---

## 6. 长期记忆卡（≠ Skills）

- 表：`conversation_memories`  
- 内容：用户事实 JSON  
- **清空聊天：保留记忆卡**；归档文件里也会带一份  
- **删除人格：不动记忆卡**；删除人格才动 Skills  

---

## 7. 清空聊天溯源（已确认）

```text
ClearMessages
  1. ArchiveConversation → data/archives/conv_{id}/chat_{ts}.json
     （含 messages + memory + summary）
  2. 写 chat_archives 索引表
  3. 软删 messages
  4. 删 summary + skill_state
  5. 保留 memory + archive + 全部 persona Skills
```

---

## 8. 存储与数据模型

| 对象 | 用途 |
|------|------|
| `messages` | 权威聊天记录 |
| `conversation_summaries` | AI 滚动摘要 |
| `conversation_memories` | 用户记忆卡 |
| `conversation_skill_states` | L2 激活 + TTL + 最近情绪 |
| `chat_archives` | 清空归档索引 |
| Redis | 可选缓存，非权威 |

---

## 9. 配置

```env
CONTEXT_TOKEN_BUDGET=16384
CONTEXT_COMPACT_THRESHOLD=0.55
RECENT_MIN_MESSAGES=6
SUMMARY_MAX_CHARS=800
MEMORY_MAX_TOKENS=400
SKILL_L2_TTL_TURNS=5
SKILL_ROUTER_MODE=hybrid          # rule | model | hybrid
ENABLE_AMBIENT_TRIGGERS=false
```

---

## 10. 实施状态

### 已落地（后端）

- [x] Token 估算 `service/token.go`  
- [x] L0 压缩 + L1 索引 + L2 触发 `skill_layers.go`  
- [x] 情绪路由（规则 + hybrid AI）`skill_router.go`  
- [x] AI 滚动摘要 / 记忆抽取 `summary_service.go`  
- [x] 组装器 16k / 55% `context_assembler.go`  
- [x] 清空归档 `archive_service.go`  
- [x] Chat WS 走 Assemble + SendMessageWithMessages  
- [x] 新表 AutoMigrate  

### 待做

- [ ] 前端 Debug：本轮 tokens / emotion / activated skills  
- [ ] 「忘记我」单独清记忆卡 API  
- [ ] 偶然触发（ambient）开关落地  
- [ ] DeepSeek Prompt Cache 实测  
- [ ] `persona_files.load_mode` 后台可配  
- [ ] 组装明细 metrics 面板  

---

## 11. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 情绪误判注入错误技能 | 规则保守 + hybrid；Emotion 高优先 |
| 摘要丢事实 | 记忆卡兜底；摘要 prompt 强制保留清单 |
| 每轮多 1～2 次分类/压缩调用 | 分类仅 hybrid 且规则不确定时；压缩仅超 55% |
| 中文 token 估算偏差 | 留 20% 余量；日志记录 estimate |

---

## 附录 A — RainSkill 映射

| 文件 | category | load_mode |
|------|----------|-----------|
| Persona-Base | persona_base | always（compact） |
| Persona-Tone | persona_tone | always（compact） |
| Forbidden-Rules | forbidden_rules | always（compact） |
| Emotion-Companion | emotion_companion | trigger |
| Style-Switch | style_switch | trigger |
| Professional-Skills | professional_skills | trigger |
| Trigger-Pet-Peeve | trigger_rules | trigger |

## 附录 B — 组装伪代码（与实现一致）

```text
Assemble(conv, userMsg):
  emotion = AnalyzeEmotion(userMsg)
  activate L2 skills with TTL
  system = L0Compact + Index + Memory + Summary + FullTexts(activated)
  msgs = messages after covered_message_id
  if tokens(system)+tokens(msgs)+tokens(user) >= 0.55 * 16k:
      summary = AI.Compress(old, older_msgs)
      save summary; msgs = recent_only; extract memory
  recent = pack by token budget
  return [system] + recent + [user]
```
