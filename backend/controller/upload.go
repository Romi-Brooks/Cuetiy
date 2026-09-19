package controller

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"rain-yi-backend/model"
	"rain-yi-backend/repository"
	"rain-yi-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadController struct {
	fileRepo *repository.FileRepository
	userRepo *repository.UserRepository
	convRepo *repository.ConversationRepository
	storage  service.FileStorage
}

func NewUploadController(fileRepo *repository.FileRepository, userRepo *repository.UserRepository, convRepo *repository.ConversationRepository, storage service.FileStorage) *UploadController {
	return &UploadController{
		fileRepo: fileRepo,
		userRepo: userRepo,
		convRepo: convRepo,
		storage:  storage,
	}
}

func (ctl *UploadController) UploadFile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	fileType := c.PostForm("type")
	if fileType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件类型参数"})
		return
	}

	validTypes := map[string]bool{"avatar": true, "ai_avatar": true, "message_image": true, "message_file": true}
	if !validTypes[fileType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型: " + fileType})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供文件"})
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少选择一个文件"})
		return
	}

	var uploaded []*model.FileRecord
	var errors []string

	for _, header := range files {
		if header.Size > 10*1024*1024 {
			errors = append(errors, fmt.Sprintf("文件 %s 超过 10MB 限制", header.Filename))
			continue
		}

		record, err := service.SaveUploadedFile(ctl.storage, fileType, userID, "", 0, header)
		if err != nil {
			errors = append(errors, fmt.Sprintf("上传 %s 失败: %v", header.Filename, err))
			continue
		}

		if fileType == "avatar" {
			user, _ := ctl.userRepo.FindByID(userID)
			if user != nil {
				oldRecords, _ := ctl.fileRepo.FindByReference("user", userID)
				ctl.userRepo.UpdateAvatar(userID, record.URL)

				for i := range oldRecords {
					old := &oldRecords[i]
					if old.FileType == "avatar" && old.ID != record.ID {
						ctl.storage.Delete(old)
						ctl.fileRepo.SoftDelete(old.ID)
					}
				}
			}
		}

		uploaded = append(uploaded, record)
	}

	response := gin.H{
		"message":  fmt.Sprintf("成功上传 %d 个文件", len(uploaded)),
		"uploaded": uploaded,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

func (ctl *UploadController) DeleteFile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	fileIDStr := c.Param("id")

	var fileID int64
	if _, err := fmt.Sscanf(fileIDStr, "%d", &fileID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}

	record, err := ctl.fileRepo.FindByID(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	if record.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该文件"})
		return
	}

	if err := ctl.storage.Delete(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件删除失败"})
		return
	}

	ctl.fileRepo.SoftDelete(fileID)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (ctl *UploadController) ListFiles(c *gin.Context) {
	userID := c.GetInt64("user_id")
	fileType := c.Query("type")

	records, err := ctl.fileRepo.FindByUserID(userID, fileType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": records})
}

func (ctl *UploadController) UploadAvatar(c *gin.Context) {
	userID := c.GetInt64("user_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供文件"})
		return
	}
	defer file.Close()

	if header.Size > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件超过 1MB 限制"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	objectName := fmt.Sprintf("avatar/user/%d/%s%s", userID, uuid.New().String(), ext)

	record, err := ctl.storage.SaveToPath(objectName, userID, "avatar", "user", userID, header.Filename, strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败: " + err.Error()})
		return
	}

	ctl.userRepo.UpdateAvatar(userID, record.URL)

	oldRecords, _ := ctl.fileRepo.FindByReference("user", userID)
	for i := range oldRecords {
		o := &oldRecords[i]
		if o.FileType == "avatar" && o.ID != record.ID {
			ctl.storage.Delete(o)
			ctl.fileRepo.SoftDelete(o.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"file":    absFileRecord(c.Request, record),
	})
}

func (ctl *UploadController) UploadAIAvatar(c *gin.Context) {
	userID := c.GetInt64("user_id")
	convIDStr := c.PostForm("conversation_id")
	if convIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少会话ID"})
		return
	}

	convID, err := strconv.ParseInt(convIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话ID"})
		return
	}

	conv, err := ctl.convRepo.FindByID(convID)
	if err != nil || conv.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供文件"})
		return
	}
	defer file.Close()

	if header.Size > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件超过 1MB 限制"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	objectName := fmt.Sprintf("avatar/ai/%d/%s%s", convID, uuid.New().String(), ext)

	record, err := ctl.storage.SaveToPath(objectName, userID, "ai_avatar", "conversation", convID, header.Filename, strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败: " + err.Error()})
		return
	}

	ctl.convRepo.UpdateAIAvatar(convID, record.URL)

	oldRecords, _ := ctl.fileRepo.FindByReference("conversation", convID)
	for i := range oldRecords {
		o := &oldRecords[i]
		if o.FileType == "ai_avatar" && o.ID != record.ID {
			ctl.storage.Delete(o)
			ctl.fileRepo.SoftDelete(o.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"file":    absFileRecord(c.Request, record),
	})
}

func (ctl *UploadController) UploadImage(c *gin.Context) {
	ctl.handleSingleUpload(c, "message_image", "message")
}

func (ctl *UploadController) handleSingleUpload(c *gin.Context, fileType string, referenceType string) {
	userID := c.GetInt64("user_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供文件"})
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件超过 10MB 限制"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	record, err := ctl.storage.Save(fileType, userID, referenceType, 0, header.Filename, strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败: " + err.Error()})
		return
	}

	if fileType == "avatar" {
		oldRecords, _ := ctl.fileRepo.FindByReference("user", userID)
		ctl.userRepo.UpdateAvatar(userID, record.URL)
		for i := range oldRecords {
			o := &oldRecords[i]
			if o.FileType == "avatar" && o.ID != record.ID {
				ctl.storage.Delete(o)
				ctl.fileRepo.SoftDelete(o.ID)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"file":    absFileRecord(c.Request, record),
	})
}
