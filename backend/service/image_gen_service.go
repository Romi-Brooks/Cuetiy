package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	Aspect      string `json:"aspect"`
	Quality     string `json:"quality"`
	HasRefImage bool   `json:"has_ref_image"`
	TaskID      string `json:"task_id"`
	Status      string `json:"status"`
	ImageURL    string `json:"image_url"`
	LocalPath   string `json:"local_path"`
	RemoteURL   string `json:"remote_url"`
	LatencyMs   int64  `json:"latency_ms"`
	HitsInWin   int    `json:"hits_in_window"`
	MaxPerWin   int    `json:"max_per_window"`
	Error       string `json:"error,omitempty"`
}

type ImageRefSource interface {
	FindByUserID(userID int64) (*model.UserImageRef, error)
}

type RegistrySource func(personaID int64) *skill.SkillRegistry

// ImageGenService 调用 Grsai gpt-image-2.5；触发/外貌由技能包驱动 + 限流 + 可选参考图
type ImageGenService struct {
	storage     FileStorage
	refRepo     ImageRefSource
	regSource   RegistrySource
	mu          sync.Mutex
	hits        map[int64][]time.Time
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

// AllowImage 发图限流：窗口内最多 max 次
func (s *ImageGenService) AllowImage(userID int64, max int, window time.Duration) (ok bool, hits int) {
	if max <= 0 {
		max = 2
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

// BuildImagePrompt 组装 TTI prompt：
// 1) 技能包 image_prompt 覆盖 → 2) appearance + image_style（技能包） + 情绪微调 + 回复意境
// 引擎只保留通用连接词与情绪修饰，不写死具体人设外貌。
func BuildImagePrompt(aiReply, emotion, style, appearance string) string {
	return BuildImagePromptWithReg(aiReply, emotion, style, appearance, nil)
}

// BuildImagePromptWithReg 优先使用技能包 appearance / image_style / image_prompt
func BuildImagePromptWithReg(aiReply, emotion, envStyle, envAppearance string, reg *skill.SkillRegistry) string {
	if reg != nil {
		if over := reg.ImagePromptOverride(); over != "" {
			// 覆盖式仍可附上情绪一句，避免完全僵硬
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
		// 仅中性兜底，不含具体长相
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
func (s *ImageGenService) Generate(userID, personaID int64, aiReply, emotion string, emotionTags []string) (url string, dbg *ImageGenDebug, err error) {
	start := time.Now()
	cfg := config.AppConfig
	dbg = &ImageGenDebug{MaxPerWin: 2}
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
		if reg.ShouldTriggerImage("") {
			// no-op
		}
	}

	refURL, hasRef := s.loadRefBase64(userID)
	dbg.HasRefImage = hasRef
	prompt := BuildImagePromptWithReg(aiReply, emotion, cfg.ImageGenStylePrompt, "", reg)
	dbg.Prompt = prompt
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
