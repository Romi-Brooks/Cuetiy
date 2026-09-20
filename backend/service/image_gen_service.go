package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/skill"

	"github.com/google/uuid"
)

// ImageGenDebug 本轮图片生成明细
type ImageGenDebug struct {
	Enabled     bool   `json:"enabled"`
	Triggered   bool   `json:"triggered"`
	RateLimited bool   `json:"rate_limited"`
	TriggerSrc  string `json:"trigger_src"` // skill_keyword | skill_tag | none
	SkillCats   string `json:"skill_cats"`
	Appearance  string `json:"appearance"`
	ImageStyle  string `json:"image_style"`
	Model       string `json:"model"`
	APIBase     string `json:"api_base"`
	Prompt      string `json:"prompt"`
	// PromptSrc: skills | skills+llm | skills_fallback | override | override+llm
	PromptSrc string `json:"prompt_src"`
	// LLM 场景向产物（debug）
	LLMPrompt    string `json:"llm_prompt"`
	LLMCaption   string `json:"llm_caption"`
	LLMScene     string `json:"llm_scene"`
	LLMHasPerson bool   `json:"llm_has_person"`
	Aspect       string `json:"aspect"`
	Quality      string `json:"quality"`
	HasRefImage  bool   `json:"has_ref_image"`
	TaskID       string `json:"task_id"`
	Status       string `json:"status"`
	ImageURL     string `json:"image_url"`
	LocalPath    string `json:"local_path"`
	RemoteURL    string `json:"remote_url"`
	LatencyMs    int64  `json:"latency_ms"`
	HitsInWin    int    `json:"hits_in_window"`
	MaxPerWin    int    `json:"max_per_window"`
	Error        string `json:"error,omitempty"`
}

type ImageRefSource interface {
	FindByUserID(userID int64) (*model.UserImageRef, error)
}

type RegistrySource func(personaID int64) *skill.SkillRegistry

// ImageGenService 调用 Grsai gpt-image-2.5；提示词 = Skills 约束 + LLM 场景
type ImageGenService struct {
	storage   FileStorage
	refRepo   ImageRefSource
	regSource RegistrySource
	ai        *AIService
	mu        sync.Mutex
	hits      map[int64][]time.Time
}

func NewImageGenService(storage FileStorage, refRepo ImageRefSource) *ImageGenService {
	return &ImageGenService{
		storage: storage,
		refRepo: refRepo,
		hits:    make(map[int64][]time.Time),
	}
}

// BindRegistry 绑定技能注册表来源（SkillManager.GetSkillRegistry）
func (s *ImageGenService) BindRegistry(src RegistrySource) {
	if s != nil {
		s.regSource = src
	}
}

// BindAI 绑定 AIService：Skills+LLM 组合提示词 / LLM 等待句
func (s *ImageGenService) BindAI(ai *AIService) {
	if s != nil {
		s.ai = ai
	}
}

func (s *ImageGenService) registryFor(personaID int64) *skill.SkillRegistry {
	if s == nil || s.regSource == nil {
		return nil
	}
	return s.regSource(personaID)
}

// ShouldTrigger 出图触发：技能包 keywords / 分类标签；引擎不写死业务词表
func (s *ImageGenService) ShouldTrigger(userMsg string, personaID int64, emotionTags []string) (ok bool, src string) {
	reg := s.registryFor(personaID)
	if reg == nil {
		return false, "no_registry"
	}
	if reg.ShouldTriggerImage(userMsg) {
		return true, "skill_keyword"
	}
	if reg.ShouldTriggerImageTags(emotionTags) {
		return true, "skill_tag"
	}
	return false, "none"
}

// AllowImage 发图限流：窗口内最多 max 次。max<=0 时视为不限流（由调用方判断）
func (s *ImageGenService) AllowImage(userID int64, max int, window time.Duration) (ok bool, hits int) {
	if max <= 0 {
		// 不限流：直接放行，不记账
		return true, 0
	}
	if window <= 0 {
		window = 30 * time.Minute
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cut := now.Add(-window)
	list := s.hits[userID]
	kept := list[:0]
	for _, t := range list {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	s.hits[userID] = kept
	if len(kept) >= max {
		return false, len(kept)
	}
	s.hits[userID] = append(kept, now)
	return true, len(s.hits[userID])
}

// BuildImagePrompt 组装 TTI prompt（无 LLM 时的 skills-only 路径）
func BuildImagePrompt(aiReply, emotion, style, appearance string) string {
	return BuildImagePromptWithReg(aiReply, emotion, style, appearance, nil)
}

// ComposeImagePrompt Skills + LLM 组合：
// - Skills：appearance / image_style / image_prompt 覆盖（身份与风格硬约束）
// - LLM：场景向 prompt（风景/街拍/美食等，不限于自拍）+ caption
// 最终 = [skills appearance（有人时）] + [llm.prompt 或 skills style] + 情绪 + 免水印
func ComposeImagePrompt(aiReply, emotion string, reg *skill.SkillRegistry, llm *ImagePromptLLM) (prompt, src string) {
	appearance := ""
	style := ""
	override := ""
	if reg != nil {
		appearance = strings.TrimSpace(reg.ImageAppearanceFromSkills())
		style = strings.TrimSpace(reg.ImageStyleFromSkills())
		override = strings.TrimSpace(reg.ImagePromptOverride())
	}

	// 人物是否入画：LLM 说了算；无 LLM 时有 appearance 则默认有人
	hasPerson := appearance != ""
	if llm != nil {
		hasPerson = llm.HasPerson
	}

	var b strings.Builder

	// 1) 外貌：技能包硬约束，LLM 不得改写
	if hasPerson && appearance != "" {
		b.WriteString("人物外貌：" + appearance + "。")
	}

	// 2) 场景主体
	switch {
	case llm != nil && llm.Prompt != "" && override != "":
		// 技能包整段覆盖仍保留，LLM 场景附在后面丰富细节
		b.WriteString(strings.TrimSpace(override))
		b.WriteString("。")
		b.WriteString(strings.TrimSpace(llm.Prompt))
		src = "override+llm"
	case llm != nil && llm.Prompt != "":
		b.WriteString(strings.TrimSpace(llm.Prompt))
		src = "skills+llm"
	case override != "":
		b.WriteString(override)
		if emotionPart := emotionMoodSuffix(emotion); emotionPart != "" {
			b.WriteString(emotionPart)
		}
		return strings.TrimSpace(b.String()), "override"
	case style != "":
		b.WriteString(style)
		src = "skills"
	default:
		b.WriteString("手机拍摄感照片，自然光线，生活场景")
		src = "skills_fallback"
	}

	if src != "override" {
		b.WriteString(emotionMoodSuffix(emotion))
		snip := strings.TrimSpace(aiReply)
		if r := []rune(snip); len(r) > 40 {
			snip = string(r[:40])
		}
		if snip != "" {
			b.WriteString("。配图文案意境：" + snip)
		}
	}
	b.WriteString("。不要在画面中出现文字水印。")
	return b.String(), src
}

// BuildImagePromptWithReg 无 LLM 时：优先技能包 appearance / image_style / image_prompt
func BuildImagePromptWithReg(aiReply, emotion, envStyle, envAppearance string, reg *skill.SkillRegistry) string {
	if reg != nil {
		if over := reg.ImagePromptOverride(); over != "" {
			emotionPart := emotionMoodSuffix(emotion)
			if emotionPart != "" {
				return strings.TrimSpace(over) + emotionPart
			}
			return over
		}
	}
	appearance := strings.TrimSpace(envAppearance)
	style := strings.TrimSpace(envStyle)
	if reg != nil {
		if a := reg.ImageAppearanceFromSkills(); a != "" {
			appearance = a
		}
		if s := reg.ImageStyleFromSkills(); s != "" {
			style = s
		}
	}
	if style == "" {
		style = "手机拍摄感照片，自然光线，生活场景，画面无文字水印"
	}
	var b strings.Builder
	if appearance != "" {
		b.WriteString("人物外貌：" + appearance + "。")
	}
	b.WriteString(style)
	b.WriteString(emotionMoodSuffix(emotion))
	snip := strings.TrimSpace(aiReply)
	if r := []rune(snip); len(r) > 40 {
		snip = string(r[:40])
	}
	if snip != "" {
		b.WriteString("。配图文案意境：" + snip)
	}
	b.WriteString("。不要在画面中出现文字水印。")
	return b.String()
}

// MaybeLLMPrompt 按配置决定是否调用 LLM 生成场景向提示词
func (s *ImageGenService) MaybeLLMPrompt(userMsg, aiReply, emotion string, reg *skill.SkillRegistry) *ImagePromptLLM {
	if s == nil || s.ai == nil {
		return nil
	}
	cfg := config.AppConfig
	if cfg == nil || !cfg.ImageGenLLMPrompt {
		return nil
	}
	appearance, style := "", ""
	if reg != nil {
		appearance = reg.ImageAppearanceFromSkills()
		style = reg.ImageStyleFromSkills()
	}
	llm, err := s.ai.GenerateImagePromptLLM(userMsg, aiReply, emotion, appearance, style)
	if err != nil {
		log.Printf("[image] llm prompt failed, fallback skills-only: %v", err)
		return nil
	}
	return llm
}

// MaybeLLMPromptWithPersona 按人格取 registry 后生成 LLM 提示词
func (s *ImageGenService) MaybeLLMPromptWithPersona(userMsg, aiReply, emotion string, personaID int64) *ImagePromptLLM {
	if s == nil {
		return nil
	}
	return s.MaybeLLMPrompt(userMsg, aiReply, emotion, s.registryFor(personaID))
}

// WaitPhraseFromLLM 优先 LLM caption，否则固定短句
func WaitPhraseFromLLM(llm *ImagePromptLLM) string {
	if llm != nil {
		if c := strings.TrimSpace(llm.Caption); c != "" {
			return c
		}
	}
	phrases := []string{
		"等我一下哦～",
		"马上给你看～",
		"稍等，我拍一下～",
		"好呀，你等等我～",
		"这就来，稍等一下～",
	}
	if len(phrases) == 0 {
		return "等我一下～"
	}
	return phrases[time.Now().UnixNano()%int64(len(phrases))]
}

func emotionMoodSuffix(emotion string) string {
	switch strings.ToLower(strings.TrimSpace(emotion)) {
	case "joy":
		return "，氛围轻松开心，带着笑意"
	case "sad", "tired":
		return "，居家安静，柔和光线，像刚忙完靠在沙发上的样子"
	case "anxious":
		return "，窗边安静片刻，神情放松下来"
	case "flirty":
		return "，眼神带着一点撩人的亲近感，不过分夸张"
	case "angry":
		return "，小别扭的可爱表情，仍然亲近"
	default:
		return ""
	}
}

type grsaiGenerateReq struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Images      []string `json:"images,omitempty"`
	AspectRatio string   `json:"aspectRatio,omitempty"`
	Quality     string   `json:"quality,omitempty"`
	ReplyType   string   `json:"replyType,omitempty"`
}

type grsaiGenerateResp struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Error    string `json:"error"`
	Results  []struct {
		URL string `json:"url"`
	} `json:"results"`
}

func (s *ImageGenService) loadRefBase64(userID int64) (dataURL string, ok bool) {
	if s.refRepo == nil {
		return "", false
	}
	ref, err := s.refRepo.FindByUserID(userID)
	if err != nil || ref == nil || ref.StoragePath == "" {
		return "", false
	}
	rc, err := s.storage.Get(ref.StoragePath)
	if err != nil || rc == nil {
		return "", false
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, 8<<20))
	if err != nil || len(data) == 0 {
		return "", false
	}
	mime := ref.MimeType
	if mime == "" {
		switch strings.ToLower(filepath.Ext(ref.StoragePath)) {
		case ".png":
			mime = "image/png"
		case ".webp":
			mime = "image/webp"
		case ".jpg", ".jpeg":
			mime = "image/jpeg"
		default:
			mime = "image/jpeg"
		}
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), true
}

// Generate 生成一张形象图并落到本地存储；返回可外链 URL
// prompt = Skills（appearance/style/override）+ LLM 场景（可关）；llm 可预生成避免二次调用
func (s *ImageGenService) Generate(userID, personaID int64, userMsg, aiReply, emotion string, emotionTags []string, llm *ImagePromptLLM) (url string, dbg *ImageGenDebug, err error) {
	start := time.Now()
	cfg := config.AppConfig
	dbg = &ImageGenDebug{MaxPerWin: 0}
	if cfg == nil {
		dbg.Error = "config nil"
		return "", dbg, fmt.Errorf("config nil")
	}
	dbg.Enabled = cfg.ImageGenEnabled
	dbg.MaxPerWin = cfg.ImageGenLimitMax
	dbg.Model = cfg.ImageGenModel
	dbg.APIBase = cfg.GRSAIAPIBase
	if !cfg.ImageGenEnabled {
		dbg.Error = "image gen disabled"
		return "", dbg, fmt.Errorf("图片生成未启用")
	}
	if strings.TrimSpace(cfg.GRSAIAPIKey) == "" {
		dbg.Error = "GRSAI_API_KEY missing"
		return "", dbg, fmt.Errorf("GRSAI_API_KEY 未配置")
	}

	reg := s.registryFor(personaID)
	if reg != nil {
		dbg.Appearance = reg.ImageAppearanceFromSkills()
		dbg.ImageStyle = reg.ImageStyleFromSkills()
	}

	refURL, hasRef := s.loadRefBase64(userID)
	dbg.HasRefImage = hasRef

	// Skills + LLM 组合提示词（外部已生成则复用，避免两次 LLM）
	if llm == nil {
		llm = s.MaybeLLMPrompt(userMsg, aiReply, emotion, reg)
	}
	if llm != nil {
		dbg.LLMPrompt = llm.Prompt
		dbg.LLMCaption = llm.Caption
		dbg.LLMScene = llm.SceneType
		dbg.LLMHasPerson = llm.HasPerson
	}
	prompt, promptSrc := ComposeImagePrompt(aiReply, emotion, reg, llm)
	// 无技能包 style 且无 LLM 时，用 env 兜底风格
	if promptSrc == "skills_fallback" && strings.TrimSpace(cfg.ImageGenStylePrompt) != "" {
		prompt, _ = ComposeImagePrompt(aiReply, emotion, reg, nil)
		// env style 注入
		if !strings.Contains(prompt, cfg.ImageGenStylePrompt) {
			prompt = strings.Replace(prompt, "手机拍摄感照片，自然光线，生活场景", cfg.ImageGenStylePrompt, 1)
			if !strings.Contains(prompt, cfg.ImageGenStylePrompt) {
				prompt = cfg.ImageGenStylePrompt + "。" + prompt
			}
		}
		promptSrc = "skills_env"
	}
	dbg.Prompt = prompt
	dbg.PromptSrc = promptSrc
	dbg.Aspect = cfg.ImageGenAspect
	dbg.Quality = cfg.ImageGenQuality

	body := grsaiGenerateReq{
		Model:       cfg.ImageGenModel,
		Prompt:      prompt,
		AspectRatio: cfg.ImageGenAspect,
		Quality:     cfg.ImageGenQuality,
		ReplyType:   cfg.ImageGenReplyType,
	}
	if body.ReplyType == "" {
		body.ReplyType = "json"
	}
	if hasRef && refURL != "" {
		body.Images = []string{refURL}
	}
	// gpt-image-2.5：比例或像素；quality auto
	if cfg.ImageGenModel == "gpt-image-2" && body.Quality == "" {
		body.Quality = "auto"
	}

	payload, _ := json.Marshal(body)
	base := strings.TrimRight(cfg.GRSAIAPIBase, "/")
	endpoint := base + "/v1/api/generate"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		dbg.Error = err.Error()
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return "", dbg, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.GRSAIAPIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		dbg.Error = err.Error()
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return "", dbg, fmt.Errorf("图片生成请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		var er grsaiGenerateResp
		_ = json.Unmarshal(raw, &er)
		msg := er.Error
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
			if len(msg) > 200 {
				msg = msg[:200]
			}
		}
		dbg.Error = fmt.Sprintf("API [%d]: %s", resp.StatusCode, msg)
		dbg.Status = er.Status
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return "", dbg, fmt.Errorf("图片生成 API %d: %s", resp.StatusCode, msg)
	}

	var gr grsaiGenerateResp
	if err := json.Unmarshal(raw, &gr); err != nil {
		dbg.Error = "bad response json"
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return "", dbg, fmt.Errorf("图片生成响应解析失败")
	}
	dbg.TaskID = gr.ID
	dbg.Status = gr.Status
	dbg.RemoteURL = ""
	if len(gr.Results) > 0 {
		dbg.RemoteURL = gr.Results[0].URL
	}

	// json 模式：running 时可稍后再查；当前先要求 succeeded + url（后续可加轮询）
	if strings.EqualFold(gr.Status, "violation") {
		dbg.Error = "violation"
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return "", dbg, fmt.Errorf("图片被判定违规")
	}
	if strings.EqualFold(gr.Status, "failed") || gr.Error != "" {
		if dbg.Error == "" {
			dbg.Error = gr.Error
		}
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return "", dbg, fmt.Errorf("图片生成失败: %s", gr.Error)
	}
	if dbg.RemoteURL == "" {
		// 异步：短轮询几次
		if gr.ID != "" && strings.EqualFold(gr.Status, "running") {
			u, st, perr := s.pollResult(base, cfg.GRSAIAPIKey, gr.ID, 8, 2*time.Second)
			dbg.Status = st
			if perr == nil && u != "" {
				dbg.RemoteURL = u
			}
		}
	}
	if dbg.RemoteURL == "" {
		dbg.Error = "empty image url"
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return "", dbg, fmt.Errorf("图片生成未返回链接")
	}

	// 下载并落本地（与语音条一样可走 /storage）
	imgData, mime, derr := s.downloadImage(dbg.RemoteURL)
	if derr != nil {
		// 失败时仍返回远端 URL，前端可直接展示
		dbg.Error = "download failed: " + derr.Error()
		dbg.LatencyMs = time.Since(start).Milliseconds()
		return dbg.RemoteURL, dbg, nil
	}
	ext := ".jpg"
	switch {
	case strings.Contains(mime, "png"):
		ext = ".png"
	case strings.Contains(mime, "webp"):
		ext = ".webp"
	}
	objectName := fmt.Sprintf("ai-image/%d/%s%s", userID, uuid.New().String(), ext)
	if s.storage != nil {
		rec, serr := s.storage.SaveToPath(
			objectName, userID, "ai_image", "user", userID,
			"ai-selfie"+ext, bytes.NewReader(imgData), int64(len(imgData)),
		)
		if serr == nil && rec != nil {
			dbg.LocalPath = rec.StoragePath
			dbg.ImageURL = rec.URL
			dbg.LatencyMs = time.Since(start).Milliseconds()
			return rec.URL, dbg, nil
		}
	}
	dbg.ImageURL = dbg.RemoteURL
	dbg.LatencyMs = time.Since(start).Milliseconds()
	return dbg.RemoteURL, dbg, nil
}

func (s *ImageGenService) pollResult(base, key, taskID string, tries int, wait time.Duration) (url, status string, err error) {
	base = strings.TrimRight(base, "/")
	// 文档中的异步查询接口在 apifox；这里按常见 path 轮询，失败则放弃
	paths := []string{
		"/v1/api/generate/" + taskID,
		"/v1/api/task/" + taskID,
		"/v1/api/query?id=" + taskID,
	}
	client := &http.Client{Timeout: 20 * time.Second}
	for i := 0; i < tries; i++ {
		time.Sleep(wait)
		for _, p := range paths {
			req, _ := http.NewRequest("GET", base+p, nil)
			if req == nil {
				continue
			}
			req.Header.Set("Authorization", "Bearer "+key)
			resp, rerr := client.Do(req)
			if rerr != nil {
				continue
			}
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			if resp.StatusCode >= 400 {
				continue
			}
			var gr grsaiGenerateResp
			if json.Unmarshal(raw, &gr) != nil {
				continue
			}
			status = gr.Status
			if len(gr.Results) > 0 && gr.Results[0].URL != "" {
				return gr.Results[0].URL, gr.Status, nil
			}
			if strings.EqualFold(gr.Status, "failed") || strings.EqualFold(gr.Status, "violation") {
				return "", gr.Status, fmt.Errorf("%s", gr.Status)
			}
		}
	}
	return "", status, fmt.Errorf("poll timeout")
}

func (s *ImageGenService) downloadImage(url string) ([]byte, string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/jpeg"
	}
	return data, mime, nil
}

// AbsImageURL 把相对 /storage 路径转成客户端可访问 URL
func AbsImageURL(publicBase, url string) string {
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	if publicBase == "" || !strings.HasPrefix(url, "/") {
		return url
	}
	return strings.TrimRight(publicBase, "/") + url
}
