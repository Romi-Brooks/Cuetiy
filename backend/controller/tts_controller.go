package controller

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"cuetiy-backend/model"
	"cuetiy-backend/repository"
	"cuetiy-backend/service"
	"cuetiy-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TTSController struct {
	tts       *service.TTSService
	asr       *service.ASRService
	storage   service.FileStorage
	voiceRepo *repository.VoiceRepo
}

func NewTTSController(
	tts *service.TTSService,
	asr *service.ASRService,
	storage service.FileStorage,
	voiceRepo *repository.VoiceRepo,
) *TTSController {
	return &TTSController{tts: tts, asr: asr, storage: storage, voiceRepo: voiceRepo}
}

// Synthesize POST /api/tts
func (ctl *TTSController) Synthesize(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var req struct {
		Text    string `json:"text"`
		Style   string `json:"style"`
		Emotion string `json:"emotion"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 text"})
		return
	}

	url, path, ttsDbg, err := ctl.tts.Synthesize(userID, req.Text, req.Style, req.Emotion)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "debug": ttsDbg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"url":     utils.AssetURL(c.Request, url),
		"path":    path,
		"debug":   ttsDbg,
	})
}

// Transcribe POST /api/asr  multipart file 或 JSON data_url
func (ctl *TTSController) Transcribe(c *gin.Context) {
	var data []byte
	var mime string
	lang := "zh"

	if fh, err := c.FormFile("file"); err == nil {
		f, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "读取音频失败"})
			return
		}
		defer f.Close()
		data, err = io.ReadAll(io.LimitReader(f, 12<<20))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "读取音频失败"})
			return
		}
		mime = fh.Header.Get("Content-Type")
		if mime == "" {
			if strings.ToLower(filepath.Ext(fh.Filename)) == ".mp3" {
				mime = "audio/mpeg"
			} else {
				mime = "audio/wav"
			}
		}
		if v := c.PostForm("language"); v != "" {
			lang = v
		}
	} else {
		var req struct {
			DataURL  string `json:"data_url"`
			Language string `json:"language"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.DataURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请提供 file 或 data_url"})
			return
		}
		raw := req.DataURL
		if i := strings.Index(raw, ";base64,"); i >= 0 {
			mime = strings.TrimPrefix(raw[:i], "data:")
			raw = raw[i+len(";base64,"):]
		}
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "base64 无效"})
			return
		}
		data = decoded
		if mime == "" {
			mime = "audio/wav"
		}
		if req.Language != "" {
			lang = req.Language
		}
	}

	text, err := ctl.asr.Transcribe(data, mime, lang)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok", "text": text})
}

// GetMyVoice GET /api/voice
func (ctl *TTSController) GetMyVoice(c *gin.Context) {
	userID := c.GetInt64("user_id")
	v, err := ctl.voiceRepo.FindByUserID(userID)
	if err != nil || v == nil {
		c.JSON(http.StatusOK, gin.H{"voice": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"voice": absVoice(c.Request, v)})
}

// UploadMyVoice POST /api/voice
func (ctl *TTSController) UploadMyVoice(c *gin.Context) {
	userID := c.GetInt64("user_id")
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供音频文件"})
		return
	}
	if fh.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "样本不能超过 10MB"})
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".wav" && ext != ".mp3" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 wav / mp3"})
		return
	}
	if ctl.storage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "存储未配置"})
		return
	}

	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取文件失败"})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	objectName := fmt.Sprintf("voice/%d/%s%s", userID, uuid.New().String(), ext)
	mime := "audio/wav"
	if ext == ".mp3" {
		mime = "audio/mpeg"
	}
	rec, err := ctl.storage.SaveToPath(
		objectName, userID, "voice_sample", "user", userID,
		fh.Filename, strings.NewReader(string(data)), int64(len(data)),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败: " + err.Error()})
		return
	}

	v := &model.UserVoice{
		UserID:       userID,
		StoragePath:  objectName,
		URL:          rec.URL,
		MimeType:     mime,
		OriginalName: fh.Filename,
		Size:         int64(len(data)),
	}
	if err := ctl.voiceRepo.Upsert(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入数据库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "音色已更新", "voice": absVoice(c.Request, v)})
}

// DeleteMyVoice DELETE /api/voice
func (ctl *TTSController) DeleteMyVoice(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if v, err := ctl.voiceRepo.FindByUserID(userID); err == nil && v != nil && ctl.storage != nil {
		_ = ctl.storage.Delete(&model.FileRecord{StoragePath: v.StoragePath, UserID: userID})
	}
	_ = ctl.voiceRepo.DeleteByUserID(userID)
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
