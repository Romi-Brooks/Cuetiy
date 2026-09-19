package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"cuetiy-backend/config"
	"cuetiy-backend/controller"
	"cuetiy-backend/middleware"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"
	"cuetiy-backend/service"
	"cuetiy-backend/skill"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	if err := skill.EnsureSkillsDir(cfg.SkillsDir); err != nil {
		log.Printf("警告: 初始化 skills 目录失败: %v", err)
	}

	db := config.InitDatabase()
	model.AutoMigrate(db)

	config.InitRedis(cfg)

	userRepo := repository.NewUserRepository()
	convRepo := repository.NewConversationRepository()
	msgRepo := repository.NewMessageRepository()
	personaRepo := repository.NewPersonaRepository()
	pfRepo := repository.NewPersonaFileRepository()
	fileRepo := repository.NewFileRepository()

	personaStg := service.NewPersonaStorage(pfRepo)
	fileStorage := service.NewLocalStorage(cfg, fileRepo)

	skillManager := skill.NewSkillManager(personaRepo, pfRepo, personaStg)
	if err := skillManager.LoadSkills(); err != nil {
		log.Printf("警告: 技能加载失败: %v", err)
	}

	aiService := service.NewAIService(skillManager)
	contextManager := service.NewContextManager(msgRepo)
	hub := service.NewWebSocketHub()

	ctxRepo := repository.NewContextRepo()
	skillRouter := service.NewSkillRouter(aiService)
	summarizer := service.NewSummaryService()
	msgWriter := service.NewMessageWriter(msgRepo)
	assembler := service.NewContextAssembler(ctxRepo, skillManager, skillRouter, summarizer, msgWriter)
	archiveSvc := service.NewArchiveService(msgRepo, ctxRepo)
	voiceRepo := repository.NewVoiceRepo()
	ttsService := service.NewTTSService(fileStorage, voiceRepo)
	asrService := service.NewASRService()

	exportSvc := service.NewExportService(convRepo, msgRepo, ctxRepo, personaRepo)
	importSvc := service.NewImportService(convRepo, msgRepo, ctxRepo, personaRepo)
	exportController := controller.NewExportController(exportSvc, importSvc, func(id int64) (int64, bool) {
		conv, err := convRepo.FindByID(id)
		if err != nil || conv == nil {
			return 0, false
		}
		return conv.UserID, true
	})

	authController := controller.NewAuthController(userRepo)
	conversationController := controller.NewConversationController(convRepo, msgRepo, contextManager, ctxRepo, archiveSvc)
	chatController := controller.NewChatController(msgRepo, convRepo, aiService, contextManager, assembler, msgWriter, ttsService, hub)
	personaController := controller.NewPersonaController(
		personaRepo,
		pfRepo,
		convRepo,
		skillManager.PromptCache(),
		skillManager.PersonaCache(),
		personaStg,
		skillManager,
	)
	userController := controller.NewUserController(userRepo)
	uploadController := controller.NewUploadController(fileRepo, userRepo, convRepo, fileStorage)
	ttsController := controller.NewTTSController(ttsService, asrService, fileStorage, voiceRepo)
	secretsController := controller.NewSecretsController()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	r.Static("/static", "./static")

	if fileStorage != nil {
		r.GET("/storage/*filepath", func(c *gin.Context) {
			objPath := strings.TrimPrefix(c.Param("filepath"), "/")
			if objPath == "" {
				c.Status(http.StatusNotFound)
				return
			}

			objPath = strings.TrimPrefix(objPath, "/")

			obj, err := fileStorage.Get(objPath)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			defer obj.Close()

			ext := strings.ToLower(filepath.Ext(objPath))
			contentType := "application/octet-stream"
			switch ext {
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".png":
				contentType = "image/png"
			case ".gif":
				contentType = "image/gif"
			case ".webp":
				contentType = "image/webp"
			case ".svg":
				contentType = "image/svg+xml"
			case ".md":
				contentType = "text/markdown; charset=utf-8"
			case ".wav":
				contentType = "audio/wav"
			case ".mp3":
				contentType = "audio/mpeg"
			case ".ogg":
				contentType = "audio/ogg"
			}

			var fileSize int64 = -1
			if sized, ok := obj.(interface{ Size() int64 }); ok {
				fileSize = sized.Size()
			}

			c.Header("Cache-Control", "public, max-age=86400")
			if fileSize >= 0 {
				c.DataFromReader(http.StatusOK, fileSize, contentType, obj, nil)
			} else {
				c.Header("Content-Type", contentType)
				io.Copy(c.Writer, obj)
			}
		})
	}

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
			auth.POST("/logout", authController.Logout)
		}

		api.GET("/ws/chat", chatController.HandleWebSocket)

		authorized := api.Group("")
		authorized.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			authorized.GET("/user/profile", userController.GetProfile)
			authorized.PUT("/user/profile", userController.UpdateProfile)
			authorized.GET("/secrets/status", secretsController.GetStatus)
			authorized.PUT("/secrets", secretsController.Update)

			authorized.GET("/conversations", conversationController.GetConversations)
			authorized.POST("/conversations", conversationController.CreateConversation)
			authorized.GET("/conversations/:id/messages", conversationController.GetMessages)
			authorized.DELETE("/conversations/:id/messages", conversationController.ClearMessages)
			authorized.PUT("/conversations/:id/config", conversationController.UpdateConfig)
			authorized.GET("/conversations/:id/export", exportController.ExportConversation)
			authorized.GET("/export/chat", exportController.ExportAll)
			authorized.POST("/import/chat", exportController.ImportChat)

			personas := authorized.Group("/personas")
			{
				personas.GET("", personaController.GetPersonas)
				personas.GET("/:id", personaController.GetPersona)
				personas.GET("/:id/debug", personaController.DebugPrompt)
				personas.POST("", personaController.CreatePersona)
				personas.PUT("/:id", personaController.UpdatePersona)
				personas.DELETE("/:id", personaController.DeletePersona)
				personas.POST("/:id/files", personaController.UploadSkillFile)
				personas.DELETE("/:id/files/:fileId", personaController.DeleteSkillFile)
				personas.POST("/:id/avatar", personaController.UploadPersonaAvatar)
				personas.POST("/load", personaController.LoadFromDirectory)
			personas.POST("/:id/conversation", personaController.OpenConversation)
			}

			authorized.GET("/conversations/:id/persona", personaController.GetConversationPersona)
			authorized.PUT("/conversations/:id/persona", personaController.SetConversationPersona)

			authorized.POST("/tts", ttsController.Synthesize)
			authorized.POST("/asr", ttsController.Transcribe)
			authorized.GET("/voice", ttsController.GetMyVoice)
			authorized.POST("/voice", ttsController.UploadMyVoice)
			authorized.DELETE("/voice", ttsController.DeleteMyVoice)

			uploads := authorized.Group("/upload")
			{
				uploads.POST("", uploadController.UploadFile)
				uploads.POST("/avatar", uploadController.UploadAvatar)
				uploads.POST("/avatar/ai", uploadController.UploadAIAvatar)
				uploads.POST("/image", uploadController.UploadImage)
				uploads.DELETE("/:id", uploadController.DeleteFile)
				uploads.GET("/list", uploadController.ListFiles)
			}
		}
	}

	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Cuetiy 服务启动于 %s", addr)
	log.Printf("前端地址: %s", cfg.FrontendURL)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
