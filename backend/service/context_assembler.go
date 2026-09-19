package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"
)

// PromptPart system 分段（Debug 用）
type PromptPart struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Tokens  int    `json:"tokens"`
}

// AssembleDebug 前端 Debug 面板数据
type AssembleDebug struct {
	TokenBudget      int          `json:"token_budget"`
	CompactThreshold float64      `json:"compact_threshold"`
	TriggerTokens    int          `json:"trigger_tokens"`
	EstimatedTokens  int          `json:"estimated_tokens"`
	Occupancy        float64      `json:"occupancy"`
	SystemTokens     int          `json:"system_tokens"`
	RecentTokens     int          `json:"recent_tokens"`
	UserTokens       int          `json:"user_tokens"`
	SystemParts      []PromptPart `json:"system_parts"`
	Summary          string       `json:"summary"`
	SummaryVersion   int          `json:"summary_version"`
	Compacted        bool         `json:"compacted"`
	Memory           string       `json:"memory"`
	ActivatedSkills  []string     `json:"activated_skills"`
	Emotion          string       `json:"emotion"`
	EmotionReason    string       `json:"emotion_reason"`
	RecentCount      int          `json:"recent_count"`
	CoveredMessageID int64        `json:"covered_message_id"`
	FullSystem       string       `json:"full_system"`
	// 本轮真正发给 LLM 的 messages（system + recent + 当前 user）
	SentMessages []ChatMessage `json:"sent_messages"`
	LLMPreview   string        `json:"llm_preview"`
}

// AssembleResult 组装结果 + 调试元数据
type AssembleResult struct {
	Messages        []ChatMessage  `json:"-"`
	EstimatedTokens int            `json:"estimated_tokens"`
	SystemTokens    int            `json:"system_tokens"`
	RecentTokens    int            `json:"recent_tokens"`
	ActivatedSkills []string       `json:"activated_skills"`
	Emotion         string         `json:"emotion"`
	SummaryVersion  int            `json:"summary_version"`
	Compacted       bool           `json:"compacted"`
	RecentCount     int            `json:"recent_count"`
	Debug           *AssembleDebug `json:"debug"`
}

// ContextAssembler 负责分层组装每轮输入
type ContextAssembler struct {
	ctxRepo    *repository.ContextRepo
	skillMgr   skillPromptSource
	router     *SkillRouter
	summarizer *SummaryService
	msgWriter  *MessageWriter
}

type skillPromptSource interface {
	GetSystemPromptByPersona(personaID *int64) string
	GetCompiledByCategory(personaID int64, category string) string
	ListModuleSummaries(personaID int64) map[string]string
}

func NewContextAssembler(
	ctxRepo *repository.ContextRepo,
	skillMgr skillPromptSource,
	router *SkillRouter,
	summarizer *SummaryService,
	msgWriter *MessageWriter,
) *ContextAssembler {
	return &ContextAssembler{
		ctxRepo:    ctxRepo,
		skillMgr:   skillMgr,
		router:     router,
		summarizer: summarizer,
		msgWriter:  msgWriter,
	}
}

// buildSystem 按当前 summary/memory/active 生成 system 与 debug 分段。
// 压缩成功后必须再调一次，保证本轮 LLM 输入带新摘要。
func (a *ContextAssembler) buildSystem(
	conv *model.Conversation,
	summaryText, memoryText string,
	active map[string]int64,
) (parts []PromptPart, systemStr, fullSys string, activated []string) {
	addPart := func(name, content string) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		parts = append(parts, PromptPart{
			Name:    name,
			Content: content,
			Tokens:  EstimateTokens(content),
		})
	}

	l0 := CompactL0Prompt(a.skillMgr.GetSystemPromptByPersona(conv.PersonaID))
	addPart("L0 核心人设", l0)
	addPart("L1 能力索引", CapabilityIndex(a.skillMgr.ListModuleSummaries(personaIDInt(conv.PersonaID))))
	if memoryText != "" {
		addPart("长期记忆卡", TruncateRunes(memoryText, 800))
	}
	if summaryText != "" {
		addPart("滚动摘要", summaryText)
	}
	for cat := range active {
		full := a.skillMgr.GetCompiledByCategory(personaIDInt(conv.PersonaID), cat)
		if full == "" {
			continue
		}
		addPart("L2 技能 · "+cat, full)
		activated = append(activated, cat)
	}

	var sys strings.Builder
	if conv.AINickname != "" {
		sys.WriteString("你的名字是")
		sys.WriteString(conv.AINickname)
		sys.WriteString("。\n")
	}
	for i, p := range parts {
		if i > 0 {
			sys.WriteString("\n\n")
		}
		sys.WriteString(p.Content)
	}
	var full strings.Builder
	for i, p := range parts {
		if i > 0 {
			full.WriteString("\n\n")
		}
		full.WriteString("### ")
		full.WriteString(p.Name)
		full.WriteString("\n")
		full.WriteString(p.Content)
	}
	return parts, sys.String(), full.String(), activated
}

func packMsgTokens(list []model.Message) int {
	t := 0
	for _, m := range list {
		t += EstimateTokens(m.Content) + 4
	}
	return t
}

// Assemble 组装本轮 messages。
// 目标形态：SKILLS(system 分层) + 压缩后 HISTORY(recent) + 当前用户消息。
func (a *ContextAssembler) Assemble(conv *model.Conversation, userMsg string) (*AssembleResult, error) {
	cfg := config.AppConfig
	budget := 16384
	threshold := 0.55
	minRecent := 6
	if cfg != nil {
		if cfg.ContextTokenBudget > 0 {
			budget = cfg.ContextTokenBudget
		}
		if cfg.ContextCompactThreshold > 0 {
			threshold = cfg.ContextCompactThreshold
		}
		if cfg.RecentMinMessages > 0 {
			minRecent = cfg.RecentMinMessages
		}
	}

	summary, _ := a.ctxRepo.GetSummary(conv.ID)
	summaryText := ""
	summaryVersion := 0
	coveredID := int64(0)
	if summary != nil {
		summaryText = summary.Content
		summaryVersion = summary.Version
		coveredID = summary.CoveredMessageID
	}

	memoryText := ""
	if mem, err := a.ctxRepo.GetMemory(conv.ID); err == nil && mem != nil {
		memoryText = mem.Content
	}

	hint := summaryText
	emotion := a.router.AnalyzeEmotion(userMsg, hint)
	cats := CategoriesForEmotion(emotion)

	state, _ := a.ctxRepo.GetSkillState(conv.ID)
	if state == nil {
		state = &model.ConversationSkillState{ConversationID: conv.ID}
	}
	state.TurnCounter++
	active := parseActiveSkills(state.ActiveSkills)
	ttl := 5
	if cfg != nil && cfg.SkillL2TTLTurns > 0 {
		ttl = cfg.SkillL2TTLTurns
	}
	expireAt := state.TurnCounter + int64(ttl)
	for _, c := range cats {
		active[c] = expireAt
	}
	for k, exp := range active {
		if state.TurnCounter > exp {
			delete(active, k)
		}
	}
	state.LastEmotion = string(emotion.Label)
	state.ActiveSkills = marshalActiveSkills(active)
	if err := a.ctxRepo.SaveSkillState(state); err != nil {
		log.Printf("save skill state: %v", err)
	}

	// 首次组装 system（用于触发阈值粗算；压缩后会重建）
	sysParts, systemStr, fullSysStr, activated := a.buildSystem(conv, summaryText, memoryText, active)
	systemTokens := EstimateTokens(systemStr)

	recentBudget := budget - systemTokens - EstimateTokens(userMsg) - 200
	if recentBudget < 800 {
		recentBudget = 800
	}
	triggerTokens := int(float64(budget) * threshold)

	msgs, err := a.ctxRepo.GetMessagesAfterID(conv.ID, coveredID, 300)
	if err != nil {
		return nil, err
	}
	if a.msgWriter != nil {
		pending := a.msgWriter.PendingMessages(conv.ID)
		if len(pending) > 0 {
			seen := make(map[int64]bool, len(msgs))
			for _, m := range msgs {
				seen[m.ID] = true
			}
			for _, p := range pending {
				if p.ID != 0 && seen[p.ID] {
					continue
				}
				msgs = append(msgs, p)
			}
			for i := 1; i < len(msgs); i++ {
				for j := i; j > 0 && msgs[j].CreatedAt.Before(msgs[j-1].CreatedAt); j-- {
					msgs[j], msgs[j-1] = msgs[j-1], msgs[j]
				}
			}
		}
	}

	compacted := false
	allTokens := packMsgTokens(msgs) + systemTokens + EstimateTokens(userMsg)
	if allTokens >= triggerTokens && len(msgs) > minRecent {
		keep := minRecent
		if keep > len(msgs) {
			keep = len(msgs)
		}
		toCompress := msgs[:len(msgs)-keep]
		keepMsgs := msgs[len(msgs)-keep:]
		maxChars := 800
		if cfg != nil && cfg.SummaryMaxChars > 0 {
			maxChars = cfg.SummaryMaxChars
		}
		newSummary, cerr := a.summarizer.Compress(summaryText, toCompress, maxChars)
		if cerr == nil && newSummary != "" {
			last := toCompress[len(toCompress)-1]
			next := &model.ConversationSummary{
				ConversationID:   conv.ID,
				Content:          newSummary,
				CoveredMessageID: last.ID,
				Version:          summaryVersion + 1,
				TokenEstimate:    EstimateTokens(newSummary),
			}
			if summary != nil {
				next.ID = summary.ID
				next.CreatedAt = summary.CreatedAt
			}
			if serr := a.ctxRepo.SaveSummary(next); serr != nil {
				log.Printf("save summary: %v", serr)
			} else {
				summaryText = newSummary
				summaryVersion = next.Version
				coveredID = next.CoveredMessageID
				msgs = keepMsgs
				compacted = true
				if memJSON := a.summarizer.ExtractMemory(memoryText, toCompress, 600); memJSON != "" && memJSON != memoryText {
					memoryText = memJSON
					mem := &model.ConversationMemory{
						ConversationID: conv.ID,
						UserID:         conv.UserID,
						Content:        memJSON,
						TokenEstimate:  EstimateTokens(memJSON),
					}
					if old, err := a.ctxRepo.GetMemory(conv.ID); err == nil && old != nil {
						mem.ID = old.ID
						mem.CreatedAt = old.CreatedAt
					}
					_ = a.ctxRepo.SaveMemory(mem)
				}

				// 关键：压缩当轮必须把新摘要写回 system，再发给 LLM
				sysParts, systemStr, fullSysStr, activated = a.buildSystem(conv, summaryText, memoryText, active)
				systemTokens = EstimateTokens(systemStr)
				recentBudget = budget - systemTokens - EstimateTokens(userMsg) - 200
				if recentBudget < 800 {
					recentBudget = 800
				}
			}
		}
	}

	// HISTORY：只装摘要覆盖之后（压缩后 keep）的消息
	var recent []model.Message
	acc := 0
	for i := len(msgs) - 1; i >= 0; i-- {
		t := EstimateTokens(msgs[i].Content) + 4
		if acc+t > recentBudget && len(recent) >= minRecent {
			break
		}
		if acc+t > recentBudget*2 && len(recent) > 0 {
			break
		}
		recent = append([]model.Message{msgs[i]}, recent...)
		acc += t
	}

	// 最终 LLM 输入 = system(含新摘要) + compressed history + 当前提问
	messages := []ChatMessage{{Role: "system", Content: systemStr}}
	for _, m := range recent {
		messages = append(messages, ChatMessage{Role: m.Role, Content: m.Content})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: userMsg})

	var llmPrev strings.Builder
	llmPrev.WriteString("=== system（SKILLS/记忆/摘要） ===\n")
	llmPrev.WriteString(systemStr)
	llmPrev.WriteString("\n\n=== history（压缩后 recent） ===\n")
	if len(recent) == 0 {
		llmPrev.WriteString("（无）\n")
	}
	for _, m := range recent {
		llmPrev.WriteString(m.Role)
		llmPrev.WriteString(": ")
		llmPrev.WriteString(m.Content)
		llmPrev.WriteString("\n")
	}
	llmPrev.WriteString("\n=== user（当前提问） ===\n")
	llmPrev.WriteString(userMsg)
	if compacted {
		llmPrev.WriteString("\n\n[compact] 本轮已压缩：旧消息并入滚动摘要 v")
		llmPrev.WriteString(fmt.Sprintf("%d", summaryVersion))
		llmPrev.WriteString("，history 仅 ")
		llmPrev.WriteString(fmt.Sprintf("%d", len(recent)))
		llmPrev.WriteString(" 条；covered_message_id=")
		llmPrev.WriteString(fmt.Sprintf("%d", coveredID))
	} else if summaryText != "" {
		llmPrev.WriteString("\n\n[summary] 滚动摘要 v")
		llmPrev.WriteString(fmt.Sprintf("%d", summaryVersion))
		llmPrev.WriteString(" 已在 system；covered_message_id=")
		llmPrev.WriteString(fmt.Sprintf("%d", coveredID))
	}

	recentTokens := packMsgTokens(recent)
	userTokens := EstimateTokens(userMsg)
	total := systemTokens + recentTokens + userTokens
	occupancy := 0.0
	if budget > 0 {
		occupancy = float64(total) / float64(budget)
	}

	sentCopy := make([]ChatMessage, len(messages))
	copy(sentCopy, messages)

	return &AssembleResult{
		Messages:        messages,
		EstimatedTokens: total,
		SystemTokens:    systemTokens,
		RecentTokens:    recentTokens,
		ActivatedSkills: activated,
		Emotion:         string(emotion.Label),
		SummaryVersion:  summaryVersion,
		Compacted:       compacted,
		RecentCount:     len(recent),
		Debug: &AssembleDebug{
			TokenBudget:      budget,
			CompactThreshold: threshold,
			TriggerTokens:    triggerTokens,
			EstimatedTokens:  total,
			Occupancy:        occupancy,
			SystemTokens:     systemTokens,
			RecentTokens:     recentTokens,
			UserTokens:       userTokens,
			SystemParts:      sysParts,
			Summary:          summaryText,
			SummaryVersion:   summaryVersion,
			Compacted:        compacted,
			Memory:           memoryText,
			ActivatedSkills:  activated,
			Emotion:          string(emotion.Label),
			EmotionReason:    emotion.Reason,
			RecentCount:      len(recent),
			CoveredMessageID: coveredID,
			FullSystem:       fullSysStr,
			SentMessages:     sentCopy,
			LLMPreview:       llmPrev.String(),
		},
	}, nil
}

func personaIDInt(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func parseActiveSkills(s string) map[string]int64 {
	m := map[string]int64{}
	if s == "" {
		return m
	}
	_ = json.Unmarshal([]byte(s), &m)
	return m
}

func marshalActiveSkills(m map[string]int64) string {
	b, _ := json.Marshal(m)
	return string(b)
}
