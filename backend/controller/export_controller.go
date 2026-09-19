package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"rain-yi-backend/service"

	"github.com/gin-gonic/gin"
)

type ExportController struct {
	exportSvc        *service.ExportService
	importSvc        *service.ImportService
	findConversation func(id int64) (ownerUserID int64, ok bool)
}

func NewExportController(
	exportSvc *service.ExportService,
	importSvc *service.ImportService,
	findConversation func(id int64) (int64, bool),
) *ExportController {
	return &ExportController{
		exportSvc:        exportSvc,
		importSvc:        importSvc,
		findConversation: findConversation,
	}
}

func (ctl *ExportController) ExportConversation(c *gin.Context) {
	userID := c.GetInt64("user_id")
	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || convID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID无效"})
		return
	}
	ownerID, ok := ctl.findConversation(convID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权导出该会话"})
		return
	}

	payload, err := ctl.exportSvc.ExportConversationByID(convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败: " + err.Error()})
		return
	}

	name := fmt.Sprintf("rainyi-chat-%d-%s.json", convID, time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, payload)
}

func (ctl *ExportController) ExportAll(c *gin.Context) {
	userID := c.GetInt64("user_id")
	bundle, err := ctl.exportSvc.ExportUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败: " + err.Error()})
		return
	}
	name := fmt.Sprintf("rainyi-chat-all-%s.json", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, bundle)
}

func (ctl *ExportController) ImportChat(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不是合法 JSON"})
		return
	}

	var probe struct {
		Format        string            `json:"format"`
		Conversations []json.RawMessage `json:"conversations"`
		Messages      []json.RawMessage `json:"messages"`
		Conversation  json.RawMessage   `json:"conversation"`
		Note          string            `json:"note"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON 解析失败"})
		return
	}

	// 批量包
	if len(probe.Conversations) > 0 {
		var bundle service.ChatExportBundle
		if err := json.Unmarshal(raw, &bundle); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "批量导出包格式错误"})
			return
		}
		results, err := ctl.importSvc.ImportBundle(userID, &bundle)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("已导入 %d 个会话", len(results)),
			"results": results,
		})
		return
	}

	// 单会话 / 旧清空归档（conversation_id + messages + 可选 conversation 对象）
	var single service.ChatExportPayload
	if err := json.Unmarshal(raw, &single); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导出包格式错误"})
		return
	}
	if single.Conversation.Title == "" && single.ConversationID > 0 {
		single.Conversation.Title = fmt.Sprintf("导入会话 #%d", single.ConversationID)
	}
	if len(single.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导出包中没有可导入的消息"})
		return
	}

	result, err := ctl.importSvc.ImportOne(userID, &single)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导入失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "导入成功",
		"result":  result,
	})
}
