# Cuetiy Skill 通用协议（v1）

> 目标：情绪/意图路由与具体技能包解耦。换一套 Skills 只改 MD frontmatter，不改核心路由代码。  
> 关联实现：`backend/skill/registry.go`、`loader.go`、`service/skill_router.go`、`context_assembler.go`  
> 示例技能包：`D:\Project\Repo\YiSkill\`

---

## 1. 问题

旧实现把 YiSkill 词表写死在 `skill_router.go` 正则里，并把情绪 label 硬编码成 `emotion_companion` / `professional_skills` 等 category。换技能包（客服、学习教练、其他人格）必须改 Go 代码。

## 2. 设计原则

1. **技能自描述**：触发条件写在 MD frontmatter，不在核心代码。
2. **标签开放、分类闭包**：分类器只从「当前包声明的标签 ∪ 基础情绪」中选择。
3. **路由器只做匹配**：`tags ∩ skill.triggers → category`，不理解业务语义。
4. **回退兼容**：无 registry / 无 frontmatter 时回落旧 category 约定与通用情绪词。

## 3. Frontmatter Schema

```yaml
---
name: Emotion-Companion
description: "一句话说明"
allowed-tools: None          # 可选
priority: "最高（触发时优先执行）"  # 兼容旧中文描述
priority_num: 90             # 0-100，越大越优先（同标签多模块命中时）
category: emotion_companion  # L2 注入用的稳定 ID；缺省时按文件名启发式
load_mode: trigger           # always | index | trigger
ttl_turns: 5                 # L2 激活保留轮数；0 用全局 SKILL_L2_TTL_TURNS
triggers:
  emotions: [joy, sad, anxious, angry, tired, flirty]
  intents: [need_comfort]
  domains: [code, music]     # 可选领域
  keywords: [代码, bug]      # 包专属关键词（进入规则匹配）
  tags: []                   # 自由扩展标签
examples:                    # 可选，供后续 embedding 匹配
  - "我好难过"
---
```

### 3.1 字段语义

| 字段 | 用途 |
|------|------|
| `category` | 注入 system 时的模块 ID；`GetCompiledByCategory` 按此取全文 |
| `load_mode` | `always`=L0 常驻编译；`index`=L1 仅一行；`trigger`=L2 按需全文 |
| `triggers.*` | 命中任一标签/关键词即激活该模块 |
| `priority_num` | 冲突时排序；不参与「是否注入」 |
| `ttl_turns` | 激活后粘性轮数 |
| `keywords` | 规则层直接匹配用户消息（包内领域词） |

### 3.2 load_mode 与 L0/L1/L2

| load_mode | system 中的形态 |
|-----------|-----------------|
| always | 每轮进入 L0（`CompileCorePromptFromFiles`，再 Compact） |
| index | L1 能力索引一行 |
| trigger | 仅当标签命中时 L2 注入全文 + TTL |

旧 category 回退（frontmatter 未写 `load_mode` 时）：

- `persona_base` / `persona_tone` / `forbidden_rules` → always  
- `emotion_companion` / `style_switch` / `professional_skills` / `trigger_rules` → trigger  
- 其它 → index  

## 4. 标签空间

### 4.1 基础标签（核心内置，始终可被分类器使用）

```
neutral, joy, sad, anxious, angry, tired, flirty,
need_professional, need_comfort, pet_peeve, style_switch
```

### 4.2 包声明标签

从各技能 `triggers.emotions|intents|domains|tags` 合并，供分类器 prompt 使用。

建议控制在 **15–25 个** 总标签，避免分类不稳。

### 4.3 YiSkill 当前映射（示例，非硬编码）

| 模块 | category | triggers |
|------|----------|----------|
| Persona-Base | persona_base | always |
| Persona-Tone | persona_tone | always |
| Forbidden-Rules | forbidden_rules | always |
| Emotion-Companion | emotion_companion | emotions + need_comfort |
| Professional-Skills | professional_skills | need_professional + keywords |
| Style-Switch | style_switch | style_switch + keywords |
| Trigger-Pet-Peeve | trigger_rules | pet_peeve + keywords |

## 5. 路由流程

```text
userMsg + recentHint
  → ruleTags（通用情绪词 + 包 keywords）
  → conf≥0.6 且非 neutral？ → 直接 Route
  → 否则 ClassifyEmotionTags(labels = base ∪ registry tags)
  → registry.MatchCategories(tags)
  → 注入 L2 + TTL
```

调试字段（前端 Debug 面板）：

- `emotion`：主标签  
- `emotion_reason`：`sad_kw` / `pkg_kw:代码` / AI reason  
- `activated_skills`：category 列表  

## 6. 分类模型选择

| 方案 | 适用 | 说明 |
|------|------|------|
| 规则 + DeepSeek JSON（已实现） | 默认 | 标签表动态生成；temperature=0.1，max_tokens=120 |
| 本地小模型（ERNIE nano / DistilBERT 微调） | 延迟/离线敏感 | 标签固定后再蒸馏 |
| Embedding + examples | 包内样本丰富 | frontmatter `examples` 做原型匹配 |
| 词典/fastText | 极致成本 | 领域差，仅兜底 |

**现阶段推荐**：继续 hybrid LLM；协议稳定、分类成瓶颈后再上本地小模型。

## 7. 换一套 Skills 怎么做

1. 新建目录，例如 `skills/CoachPack/*.md`  
2. 每个 MD 写齐 `category` / `load_mode` / `triggers`  
3. `SKILLS_DIR` 指向该目录（或后台上传人格包）  
4. **不改** `skill_router.go` / `context_assembler.go`  
5. Debug 面板确认 `emotion` + `activated_skills` 是否符合预期  

## 8. 配置

```env
SKILL_ROUTER_MODE=hybrid       # rule | hybrid
SKILL_L2_TTL_TURNS=5           # 模块未声明 ttl_turns 时的默认
SKILLS_DIR=./skills            # 或指向 YiSkill 目录
```

## 9. API 变更摘要

| 位置 | 变更 |
|------|------|
| `skill.SkillMeta` | 增加 `category/load_mode/priority_num/ttl_turns/triggers/examples` |
| `skill.SkillRegistry` | 新：加载时构建，MatchCategories / ClassifierLabelList / TTL |
| `SkillManager.GetSkillRegistry` | 新；文件变更后 `InvalidateRegistry` |
| `CompileCorePromptFromFiles` | L0 只编译 `load_mode=always` |
| `SkillRouter.Route` | 新主入口；`AnalyzeEmotion` 为兼容包装 |
| `AIService.ClassifyEmotionTags` | 标签集动态 |
| `context_assembler.Assemble` | 使用 registry 路由与 TTL |

## 10. 非目标（本协议不做）

- 自动从 MD 正文抽取 triggers（可后续加，需人工校对）  
- 多技能冲突消解的复杂策略（目前同 category 去重 + priority 仅排序）  
- 运行时热更新标签的远程配置中心  

## 11. 测试

```bash
cd backend
go test ./skill/ ./service/ -count=1
```

覆盖：frontmatter 解析、registry 匹配、包关键词路由、无 registry 兼容、L0 过滤。
