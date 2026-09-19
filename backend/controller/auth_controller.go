package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"rain-yi-backend/config"
	"rain-yi-backend/middleware"
	"rain-yi-backend/model"
	"rain-yi-backend/repository"
	"rain-yi-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	userRepo *repository.UserRepository
}

func NewAuthController(userRepo *repository.UserRepository) *AuthController {
	return &AuthController{userRepo: userRepo}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=1,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (ctl *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	existing, _ := ctl.userRepo.FindByEmail(req.Email)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已注册"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user := &model.User{
		Username: utils.SanitizeInput(req.Username),
		Email:    req.Email,
		Password: string(hashedPassword),
		Avatar:   "/static/default-avatar.svg",
	}

	if err := ctl.userRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败: " + err.Error()})
		return
	}

	token, err := ctl.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "令牌生成失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"token":   token,
		"user":    absUserJSON(c.Request, user),
	})
}

func (ctl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	user, err := ctl.userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	token, err := ctl.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "令牌生成失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"token":   token,
		"user":    absUserJSON(c.Request, user),
	})
}

func (ctl *AuthController) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		c.JSON(http.StatusBadRequest, gin.H{"error": "认证格式错误"})
		return
	}

	claims := &middleware.Claims{}
	token, _ := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if token == nil || !token.Valid {
		c.JSON(http.StatusOK, gin.H{"message": "已登出"})
		return
	}

	if config.RDB != nil && claims.ID != "" {
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			config.RDB.Set(config.RedisCtx, fmt.Sprintf("auth:blacklist:%s", claims.ID), "1", ttl)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "已登出"})
}

func (ctl *AuthController) generateToken(user *model.User) (string, error) {
	claims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			// jti 用于登出后的 Redis 黑名单
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}
