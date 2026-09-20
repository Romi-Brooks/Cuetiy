package controller

import (
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

// ImageRefController 形象参考图（图片生成 reference），接口形态对齐 /voice
type ImageRefController struct {
	storage  service.FileStorage
	refRepo  *repository.ImageRefRepo
}

func NewImageRefController(storage service.FileStorage, refRepo *repository.ImageRefRepo) *ImageRefController {
	return &ImageRefController{storage: storage, refRepo: refRepo}
}

func absImageRef(r *http.Request, v *model.UserImageRef) *model.UserImageRef {
	if v == nil {
		return nil
	}
	out := *v
	out.URL = utils.AssetURL(r, out.URL)
	return &out
}

// GetMyImageRef GET /api/image-ref
func (ctl *ImageRefController) GetMyImageRef(c *gin.Context) {
	userID := c.GetInt64("user_id")
	v, err := ctl.refRepo.FindByUserID(userID)
	if err != nil || v == nil {
		c.JSON(http.StatusOK, gin.H{"image_ref": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"image_ref": absImageRef(c.Request, v)})
}

// UploadMyImageRef POST /api/image-ref
func (ctl *ImageRefController) UploadMyImageRef(c *gin.Context) {
	userID := c.GetInt64("user_id")
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供图片文件"})
		return
	}
	if fh.Size > 8*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考图不能超过 8MB"})
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 jpg / png / webp"})
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

	objectName := fmt.Sprintf("image-ref/%d/%s%s", userID, uuid.New().String(), ext)
	mime := "image/jpeg"
	switch ext {
	case ".png":
		mime = "image/png"
	case ".webp":
		mime = "image/webp"
	}
	rec, err := ctl.storage.SaveToPath(
		objectName, userID, "image_ref", "user", userID,
		fh.Filename, strings.NewReader(string(data)), int64(len(data)),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败: " + err.Error()})
		return
	}

	v := &model.UserImageRef{
		UserID:       userID,
		StoragePath:  objectName,
		URL:          rec.URL,
		MimeType:     mime,
		OriginalName: fh.Filename,
		Size:         int64(len(data)),
	}
	if err := ctl.refRepo.Upsert(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入数据库失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "参考图已更新", "image_ref": absImageRef(c.Request, v)})
}

// DeleteMyImageRef DELETE /api/image-ref
func (ctl *ImageRefController) DeleteMyImageRef(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if v, err := ctl.refRepo.FindByUserID(userID); err == nil && v != nil && ctl.storage != nil {
		_ = ctl.storage.Delete(&model.FileRecord{StoragePath: v.StoragePath, UserID: userID})
	}
	_ = ctl.refRepo.DeleteByUserID(userID)
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
