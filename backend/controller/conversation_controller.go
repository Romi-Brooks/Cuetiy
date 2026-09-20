package controller

import (
	"net/http"
	"strconv"

	"cuetiy-backend/model"
	"cuetiy-backend/repository"
	"cuetiy-backend/service"
	"cuetiy-backend/utils"

	"github.com/gin-gonic/gin"
)

type ConversationController struct {
	convRepo       *repository.ConversationRepository
	msgRepo        *repository.MessageRepository
	contextManager *service.ContextManager
	ctxRepo        *repository.ContextRepo
	archive        *service.ArchiveService
	msgWriter      *service.MessageWriter
}

func NewConversationController(
	convRepo *repository.ConversationRepository,
	msgRepo *repository.MessageRepository,
	contextManager *service.ContextManager,
	ctxRepo *repository.ContextRepo,
	archive *service.ArchiveService,
	msgWriter *service.MessageWriter,
) *ConversationController {
	return &ConversationController{
		convRepo:       convRepo,
		msgRepo:        msgRepo,
		contextManager: contextManager,
		ctxRepo:        ctxRepo,
		archive:        archive,
		msgWriter:      msgWriter,
	}
}

func (ctl *ConversationController) GetConversations(c *gin.Context) {
	userID := c.GetInt64("user_id")

	conversations, err := ctl.convRepo.FindByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话列表失败"})
		return
	}

	if len(conversations) == 0 {
		conv, err := ctl.convRepo.FindOrCreateDefault(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建默认会话失败"})
			return
		}
		conversations = append(conversations, *conv)
	}

	for i := range conversations {
		lastMsg, err := ctl.msgRepo.GetLastMessage(conversations[i].ID)
		if err == nil && lastMsg != nil {
			conversations[i].LastMessage = &model.LastMessage{
				Content:   lastMsg.Content,
				Role:      lastMsg.Role,
				CreatedAt: lastMsg.CreatedAt,
			}
		}
	}

	out := make([]model.Conversation, len(conversations))
	for i := range conversations {
		out[i] = absConversation(c.Request, &conversations[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": out,
	})
}

func (ctl *ConversationController) GetMessages(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := strconv.ParseInt(convIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID无效"})
		return
	}

	conv, err := ctl.convRepo.FindByID(convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	userID := c.GetInt64("user_id")
	if conv.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该会话"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	beforeIDStr := c.DefaultQuery("before_id", "0")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	beforeID, _ := strconv.ParseInt(beforeIDStr, 10, 64)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var messages []model.Message
	var msgErr error
	if beforeID > 0 || offset == 0 {
		messages, msgErr = ctl.msgRepo.FindOlderThan(convID, beforeID, limit)
	} else {
		messages, msgErr = ctl.msgRepo.FindByConversationID(convID, limit, offset)
	}
	if msgErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败"})
		return
	}

	total, _ := ctl.msgRepo.CountByConversationID(convID)

	c.JSON(http.StatusOK, gin.H{
		"messages":     absMessages(c.Request, messages),
		"total":        total,
		"conversation": absConversation(c.Request, conv),
	})
}

func (ctl *ConversationController) ClearMessages(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := strconv.ParseInt(convIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID无效"})
		return
	}

	conv, err := ctl.convRepo.FindByID(convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	userID := c.GetInt64("user_id")
	if conv.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该会话"})
		return
	}

	// keep_memory：query 优先，否则用会话配置；默认 false = 全新开局
	keepMemory := conv.KeepMemoryOnClear
	if q := c.Query("keep_memory"); q != "" {
		keepMemory = q == "true" || q == "1"
	}

	// 先归档消息 + 记忆卡 + 摘要到文件（Skills/人格不动），再物理删除库内上下文
	archivePath := ""
	archiveCount := 0
	if ctl.archive != nil {
		path, n, aerr := ctl.archive.ArchiveConversation(conv)
		if aerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "归档失败: " + aerr.Error()})
			return
		}
		archivePath, archiveCount = path, n
	}

	// 丢掉异步落库镜像，避免下一轮 Assemble 把「已清空」消息再拼进 history
	if ctl.msgWriter != nil {
		ctl.msgWriter.ClearPending(convID)
	}

	// Redis 软上下文 + 物理删消息（归档已留档）
	if err := ctl.contextManager.ResetContext(convID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空记录失败"})
		return
	}
	if err := ctl.msgRepo.HardDeleteByConversationID(convID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除消息失败"})
		return
	}
	if ctl.ctxRepo != nil {
		_ = ctl.ctxRepo.DeleteSummary(convID)
		_ = ctl.ctxRepo.ResetSkillState(convID)
		if !keepMemory {
			_ = ctl.ctxRepo.DeleteMemory(convID)
		}
	}

	msg := "聊天记录已清空（已归档，全新会话）"
	if keepMemory {
		msg = "聊天记录已清空（已归档，记忆卡已保留）"
	} else {
		msg = "聊天记录与记忆卡已清空（已归档，全新会话）"
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        msg,
		"archive_path":   archivePath,
		"archive_count":  archiveCount,
		"keep_memory":    keepMemory,
	})
}

func (ctl *ConversationController) UpdateConfig(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := strconv.ParseInt(convIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID无效"})
		return
	}

	conv, err := ctl.convRepo.FindByID(convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	userID := c.GetInt64("user_id")
	if conv.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该会话"})
		return
	}

	var req struct {
		AINickname        *string `json:"ai_nickname"`
		AIAvatar          *string `json:"ai_avatar"`
		Title             *string `json:"title"`
		PersonaID         *int64  `json:"persona_id"`
		KeepMemoryOnClear *bool   `json:"keep_memory_on_clear"`
		NSFWEnabled       *bool   `json:"nsfw_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.PersonaID != nil {
		conv.PersonaID = req.PersonaID
	}
	if req.AINickname != nil {
		conv.AINickname = utils.SanitizeInput(*req.AINickname)
	}
	if req.AIAvatar != nil {
		conv.AIAvatar = *req.AIAvatar
	}
	if req.Title != nil {
		conv.Title = utils.SanitizeInput(*req.Title)
	}
	if req.KeepMemoryOnClear != nil {
		conv.KeepMemoryOnClear = *req.KeepMemoryOnClear
	}
	if req.NSFWEnabled != nil {
		conv.NSFWEnabled = *req.NSFWEnabled
	}

	if err := ctl.convRepo.Update(conv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "更新成功",
		"conversation": absConversation(c.Request, conv),
	})
}

func (ctl *ConversationController) CreateConversation(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Title = "情感陪伴"
	}

	conv := &model.Conversation{
		UserID:     userID,
		Title:      utils.SanitizeInput(req.Title),
		AINickname: "Cuetiy",
		AIAvatar:   "/static/default-avatar.svg",
	}

	if err := ctl.convRepo.Create(conv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "创建成功",
		"conversation": absConversation(c.Request, conv),
	})
}
