package controller

import (
	"net/http"

	"rain-yi-backend/repository"
	"rain-yi-backend/utils"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userRepo *repository.UserRepository
}

func NewUserController(userRepo *repository.UserRepository) *UserController {
	return &UserController{userRepo: userRepo}
}

func (ctl *UserController) GetProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	user, err := ctl.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": absUserJSON(c.Request, user),
	})
}

func (ctl *UserController) UpdateProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	user, err := ctl.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var req struct {
		Username *string `json:"username"`
		Avatar   *string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Username != nil {
		user.Username = utils.SanitizeInput(*req.Username)
	}
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}

	if err := ctl.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"user":    absUserJSON(c.Request, user),
	})
}
