package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"
	"cuetiy-backend/service"
	"cuetiy-backend/skill"
	"cuetiy-backend/utils"

	"github.com/gin-gonic/gin"
)

type PersonaController struct {
	personaRepo  *repository.PersonaRepository
	pfRepo       *repository.PersonaFileRepository
	convRepo     *repository.ConversationRepository
	promptCache  *skill.PromptCache
	personaCache *skill.PersonaCache
	personaStg   *service.PersonaStorage
}

func NewPersonaController(
	personaRepo *repository.PersonaRepository,
	pfRepo *repository.PersonaFileRepository,
	convRepo *repository.ConversationRepository,
	promptCache *skill.PromptCache,
	personaCache *skill.PersonaCache,
	personaStg *service.PersonaStorage,
) *PersonaController {
	return &PersonaController{
		personaRepo:  personaRepo,
		pfRepo:       pfRepo,
		convRepo:     convRepo,
		promptCache:  promptCache,
		personaCache: personaCache,
		personaStg:   personaStg,
	}
}

func (ctl *PersonaController) GetPersonas(c *gin.Context) {
	userID := c.GetInt64("user_id")

	personas, err := ctl.personaRepo.FindAllVisible(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取人格列表失败"})
		return
	}

	type personaWithCount struct {
		model.Persona
		FileCount int64 `json:"file_count"`
		IsBuiltIn bool  `json:"is_built_in"`
	}

	result := make([]personaWithCount, 0, len(personas))
	for _, p := range personas {
		fileCount, _ := ctl.pfRepo.CountByPersonaID(p.ID)
		result = append(result, personaWithCount{
			Persona:   absPersona(c.Request, &p),
			FileCount: fileCount,
			IsBuiltIn: p.UserID == 0,
		})
	}

	c.JSON(http.StatusOK, gin.H{"personas": result})
}

func (ctl *PersonaController) GetPersona(c *gin.Context) {
	userID := c.GetInt64("user_id")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格ID无效"})
		return
	}

	persona, err := ctl.personaRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
		return
	}

	// 仅允许查看内置人格或自己的人格
	if persona.UserID != 0 && persona.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该人格"})
		return
	}

	files, _ := ctl.pfRepo.FindByPersonaID(id)

	c.JSON(http.StatusOK, gin.H{
		"persona":       absPersona(c.Request, persona),
		"persona_files": files,
	})
}

func (ctl *PersonaController) CreatePersona(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req struct {
		Name        string `json:"name"`
		Nickname    string `json:"nickname"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	displayName := strings.TrimSpace(req.Nickname)
	if displayName == "" {
		displayName = strings.TrimSpace(req.Name)
	}
	if displayName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格名称不能为空"})
		return
	}

	dirName := service.SanitizeDirName(displayName)
	if dirName == "" {
		dirName = "persona"
	}

	persona := &model.Persona{
		UserID:      userID,
		Name:        utils.SanitizeInput(req.Name),
		Nickname:    utils.SanitizeInput(displayName),
		Description: utils.SanitizeInput(req.Description),
		DirName:     dirName,
		IsActive:    true,
	}
	if persona.Name == "" {
		persona.Name = dirName
	}
	if err := ctl.personaRepo.Create(persona); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建人格失败"})
		return
	}

	if ctl.personaCache != nil {
		ctl.personaCache.UpsertPersona(persona)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "创建成功",
		"persona": absPersona(c.Request, persona),
	})
}

func (ctl *PersonaController) UpdatePersona(c *gin.Context) {
	userID := c.GetInt64("user_id")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格ID无效"})
		return
	}

	persona, err := ctl.personaRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
		return
	}

	if persona.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改该人格"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Nickname    *string `json:"nickname"`
		Description *string `json:"description"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Name != nil {
		persona.Name = utils.SanitizeInput(*req.Name)
	}
	if req.Nickname != nil {
		persona.Nickname = utils.SanitizeInput(*req.Nickname)
	}
	if req.Description != nil {
		persona.Description = utils.SanitizeInput(*req.Description)
	}
	if req.IsActive != nil {
		persona.IsActive = *req.IsActive
	}

	if err := ctl.personaRepo.Update(persona); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	if ctl.personaCache != nil {
		ctl.personaCache.UpsertPersona(persona)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"persona": absPersona(c.Request, persona),
	})
}

func (ctl *PersonaController) DeletePersona(c *gin.Context) {
	userID := c.GetInt64("user_id")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格ID无效"})
		return
	}

	persona, err := ctl.personaRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
		return
	}

	if persona.UserID != userID && persona.UserID != 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除该人格"})
		return
	}

	if persona.UserID == 0 && userID != 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除内置人格"})
		return
	}

	if ctl.personaStg != nil {
		ctl.personaStg.DeleteAllByPersona(persona)
	}

	ctl.personaRepo.HardDelete(id)

	if ctl.personaCache != nil {
		ctl.personaCache.DeletePersona(id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (ctl *PersonaController) UploadSkillFile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格ID无效"})
		return
	}

	persona, err := ctl.personaRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
		return
	}

	if persona.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该人格"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供技能文件"})
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少选择一个 .md 文件"})
		return
	}

	var uploaded []string
	var errors []string

	for _, header := range files {
		if !strings.HasSuffix(header.Filename, ".md") {
			errors = append(errors, fmt.Sprintf("跳过非 .md 文件: %s", header.Filename))
			continue
		}

		if ctl.personaStg == nil {
			errors = append(errors, "文件存储未配置，无法上传")
			continue
		}

		file, err := header.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("打开文件失败 %s: %v", header.Filename, err))
			continue
		}

		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("读取文件失败 %s: %v", header.Filename, err))
			continue
		}

		parsed, _ := skill.ParseSkillContent(header.Filename, string(content))
		filePriority := 0
		if parsed != nil {
			filePriority = parsed.Meta.NumericPriority()
		}

		if _, err := ctl.personaStg.UploadMD(persona, header.Filename, content, filePriority); err != nil {
			errors = append(errors, fmt.Sprintf("上传失败 %s: %v", header.Filename, err))
			continue
		}

		uploaded = append(uploaded, header.Filename)
	}

	if ctl.promptCache != nil && len(uploaded) > 0 {
		ctl.promptCache.Invalidate(id)
	}

	response := gin.H{"message": fmt.Sprintf("成功上传 %d 个文件", len(uploaded))}
	if len(uploaded) > 0 {
		response["uploaded"] = uploaded
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

func (ctl *PersonaController) DeleteSkillFile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	personaIDStr := c.Param("id")
	fileIDStr := c.Param("fileId")

	personaID, err := strconv.ParseInt(personaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格ID无效"})
		return
	}

	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}

	persona, err := ctl.personaRepo.FindByID(personaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
		return
	}

	if persona.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该人格"})
		return
	}

	if ctl.personaStg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件存储未配置"})
		return
	}

	pf, pfErr := ctl.pfRepo.FindByID(fileID)
	if pfErr != nil || pf.PersonaID != personaID {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	if err := ctl.personaStg.DeleteMD(pf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败"})
		return
	}

	if ctl.promptCache != nil {
		ctl.promptCache.Invalidate(personaID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (ctl *PersonaController) UploadPersonaAvatar(c *gin.Context) {
	userID := c.GetInt64("user_id")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格ID无效"})
		return
	}

	persona, err := ctl.personaRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
		return
	}

	if persona.UserID != userID && persona.UserID != 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	if ctl.personaStg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件存储未配置"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供图片文件"})
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

	url, err := ctl.personaStg.UploadAvatar(persona.ID, header.Filename, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败: " + err.Error()})
		return
	}

	persona.Avatar = url
	ctl.personaRepo.Update(persona)

	if ctl.personaCache != nil {
		ctl.personaCache.UpsertPersona(persona)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"avatar":  utils.AssetURL(c.Request, url),
	})
}

func (ctl *PersonaController) SetConversationPersona(c *gin.Context) {
	userID := c.GetInt64("user_id")
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

	if conv.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该会话"})
		return
	}

	var req struct {
		PersonaID *int64 `json:"persona_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.PersonaID != nil {
		persona, pErr := ctl.personaRepo.FindByID(*req.PersonaID)
		if pErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
			return
		}
		if persona.UserID != 0 && persona.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该人格"})
			return
		}
	}

	if err := ctl.personaRepo.SetConversationPersona(convID, req.PersonaID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置失败"})
		return
	}

	conv.PersonaID = req.PersonaID

	c.JSON(http.StatusOK, gin.H{
		"message":      "设置成功",
		"conversation": absConversation(c.Request, conv),
	})
}

// OpenConversation 打开（或创建）该人格对应的专属会话：一人格一会话
func (ctl *PersonaController) OpenConversation(c *gin.Context) {
	userID := c.GetInt64("user_id")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人格ID无效"})
		return
	}

	persona, err := ctl.personaRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人格不存在"})
		return
	}
	if persona.UserID != 0 && persona.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该人格"})
		return
	}

	nickname := persona.Nickname
	if nickname == "" {
		nickname = persona.Name
	}
	title := nickname
	if title == "" {
		title = "对话"
	}

	conv, created, err := ctl.convRepo.FindOrCreateByPersona(userID, id, title, nickname, persona.Avatar)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开会话失败"})
		return
	}

	// 已有会话时同步最新头像/昵称（人格改名后会话侧跟着变）
	if !created {
		changed := false
		if persona.Avatar != "" && conv.AIAvatar != persona.Avatar {
			conv.AIAvatar = persona.Avatar
			changed = true
		}
		if nickname != "" && conv.AINickname != nickname {
			conv.AINickname = nickname
			changed = true
		}
		if changed {
			_ = ctl.convRepo.Update(conv)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "ok",
		"conversation": absConversation(c.Request, conv),
		"created":      created,
		"persona":      absPersona(c.Request, persona),
	})
}

func (ctl *PersonaController) GetConversationPersona(c *gin.Context) {
	userID := c.GetInt64("user_id")
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

	if conv.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该会话"})
		return
	}

	if conv.PersonaID == nil {
		c.JSON(http.StatusOK, gin.H{"persona": nil})
		return
	}

	persona, err := ctl.personaRepo.FindByID(*conv.PersonaID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"persona": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"persona": absPersona(c.Request, persona)})
}

func (ctl *PersonaController) DebugPrompt(c *gin.Context) {
	userID := c.GetInt64("user_id")
	convIDStr := c.Param("id")
	convID, err := strconv.ParseInt(convIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID无效"})
		return
	}

	conv, err := ctl.convRepo.FindByID(convID)
	if err != nil || conv.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	systemPrompt := ""
	if conv.PersonaID != nil && ctl.promptCache != nil {
		systemPrompt = ctl.promptCache.CompileAndCache(*conv.PersonaID)
	}

	c.JSON(http.StatusOK, gin.H{
		"system_prompt": systemPrompt,
	})
}

func (ctl *PersonaController) LoadFromDirectory(c *gin.Context) {
	userID := c.GetInt64("user_id")

	skillsDir := config.AppConfig.SkillsDir
	if skillsDir == "" {
		skillsDir = "./skills"
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("读取技能目录失败: %v", err)})
		return
	}

	loadedCount := 0
	loadedPersonas := make(map[string]int64)

	for _, entry := range entries {
		if entry.IsDir() {
			dirName := entry.Name()

			existing, _ := ctl.personaRepo.FindByDirName(dirName)
			if existing != nil {
				if existing.UserID != 0 && existing.UserID != userID {
					continue
				}
				if ctl.personaStg != nil {
					ctl.personaStg.DeleteAllByPersona(existing)
				}
				ctl.personaRepo.HardDelete(existing.ID)
			}

			persona := &model.Persona{
				UserID:      0,
				Name:        dirName,
				Description: fmt.Sprintf("从 %s 目录加载", dirName),
				DirName:     dirName,
				IsActive:    true,
			}
			if err := ctl.personaRepo.Create(persona); err != nil {
				continue
			}

			if err := loadSkillsDir(persona, ctl.personaStg); err != nil {
				continue
			}

			loadedPersonas[dirName] = persona.ID
			loadedCount++

			if ctl.personaCache != nil {
				ctl.personaCache.UpsertPersona(persona)
			}
		}
	}

	if ctl.promptCache != nil {
		for _, personaID := range loadedPersonas {
			ctl.promptCache.Invalidate(personaID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("成功加载 %d 个人格", loadedCount),
		"count":   loadedCount,
	})
}

func loadSkillsDir(persona *model.Persona, personaStg *service.PersonaStorage) error {
	if personaStg == nil {
		return fmt.Errorf("persona storage not available")
	}

	skillsDir := config.AppConfig.SkillsDir
	if skillsDir == "" {
		skillsDir = "./skills"
	}

	dirPath := filepath.Join(skillsDir, persona.DirName)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			continue
		}
		personaStg.UploadMD(persona, entry.Name(), content, i)
	}

	return nil
}
