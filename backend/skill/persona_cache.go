package skill

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"
)

const (
	RedisKeyPersonaList = "persona:list"
	RedisKeyPersonaMD   = "persona:md:%d:%d"

	mdContentTTL = 1 * time.Hour
)

type PersonaCache struct {
	mu        sync.RWMutex
	localList map[int64]*cachedPersonaMeta
	repo      *repository.PersonaRepository
	pfRepo    *repository.PersonaFileRepository
}

type cachedPersonaMeta struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DirName     string `json:"dir_name"`
	IsActive    bool   `json:"is_active"`
	UserID      int64  `json:"user_id"`
}

func NewPersonaCache(personaRepo *repository.PersonaRepository, pfRepo *repository.PersonaFileRepository) *PersonaCache {
	return &PersonaCache{
		localList: make(map[int64]*cachedPersonaMeta),
		repo:      personaRepo,
		pfRepo:    pfRepo,
	}
}

func (pc *PersonaCache) cacheKeyPersonaFiles(personaID int64) string {
	return fmt.Sprintf("persona:files:%d", personaID)
}

func (pc *PersonaCache) cacheKeyMDContent(personaID int64, fileID int64) string {
	return fmt.Sprintf(RedisKeyPersonaMD, personaID, fileID)
}

func (pc *PersonaCache) WarmupList(personas []model.Persona) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for _, p := range personas {
		meta := &cachedPersonaMeta{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			DirName:     p.DirName,
			IsActive:    p.IsActive,
			UserID:      p.UserID,
		}
		pc.localList[p.ID] = meta
	}

	if config.RDB != nil {
		pipe := config.RDB.Pipeline()
		for _, p := range personas {
			data, _ := json.Marshal(cachedPersonaMeta{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				DirName:     p.DirName,
				IsActive:    p.IsActive,
				UserID:      p.UserID,
			})
			pipe.HSet(config.RedisCtx, RedisKeyPersonaList, fmt.Sprintf("%d", p.ID), data)
		}
		pipe.Exec(config.RedisCtx)
	}
}

func (pc *PersonaCache) GetPersonaList() ([]model.Persona, error) {
	pc.mu.RLock()
	if len(pc.localList) > 0 {
		result := make([]model.Persona, 0, len(pc.localList))
		for _, meta := range pc.localList {
			result = append(result, model.Persona{
				ID:          meta.ID,
				Name:        meta.Name,
				Description: meta.Description,
				DirName:     meta.DirName,
				IsActive:    meta.IsActive,
				UserID:      meta.UserID,
			})
		}
		pc.mu.RUnlock()
		return result, nil
	}
	pc.mu.RUnlock()

	if config.RDB != nil {
		cached, err := config.RDB.HGetAll(config.RedisCtx, RedisKeyPersonaList).Result()
		if err == nil && len(cached) > 0 {
			pc.mu.Lock()
			result := make([]model.Persona, 0, len(cached))
			for _, val := range cached {
				var meta cachedPersonaMeta
				if json.Unmarshal([]byte(val), &meta) == nil {
					pc.localList[meta.ID] = &meta
					result = append(result, model.Persona{
						ID:          meta.ID,
						Name:        meta.Name,
						Description: meta.Description,
						DirName:     meta.DirName,
						IsActive:    meta.IsActive,
						UserID:      meta.UserID,
					})
				}
			}
			pc.mu.Unlock()
			return result, nil
		}
	}

	personas, err := pc.repo.FindActive()
	if err != nil {
		return nil, err
	}

	pc.WarmupList(personas)

	result := make([]model.Persona, len(personas))
	copy(result, personas)
	return result, nil
}

func (pc *PersonaCache) GetPersona(id int64) (*model.Persona, error) {
	pc.mu.RLock()
	meta, ok := pc.localList[id]
	pc.mu.RUnlock()
	if ok {
		return &model.Persona{
			ID:          meta.ID,
			Name:        meta.Name,
			Description: meta.Description,
			DirName:     meta.DirName,
			IsActive:    meta.IsActive,
			UserID:      meta.UserID,
		}, nil
	}

	if config.RDB != nil {
		data, err := config.RDB.HGet(config.RedisCtx, RedisKeyPersonaList, fmt.Sprintf("%d", id)).Result()
		if err == nil && data != "" {
			var meta cachedPersonaMeta
			if json.Unmarshal([]byte(data), &meta) == nil {
				pc.mu.Lock()
				pc.localList[id] = &meta
				pc.mu.Unlock()
				return &model.Persona{
					ID:          meta.ID,
					Name:        meta.Name,
					Description: meta.Description,
					DirName:     meta.DirName,
					IsActive:    meta.IsActive,
					UserID:      meta.UserID,
				}, nil
			}
		}
	}

	return pc.repo.FindByID(id)
}

func (pc *PersonaCache) UpsertPersona(persona *model.Persona) {
	meta := &cachedPersonaMeta{
		ID:          persona.ID,
		Name:        persona.Name,
		Description: persona.Description,
		DirName:     persona.DirName,
		IsActive:    persona.IsActive,
		UserID:      persona.UserID,
	}

	pc.mu.Lock()
	pc.localList[persona.ID] = meta
	pc.mu.Unlock()

	if config.RDB != nil {
		data, _ := json.Marshal(meta)
		config.RDB.HSet(config.RedisCtx, RedisKeyPersonaList, fmt.Sprintf("%d", persona.ID), data)
	}
}

func (pc *PersonaCache) DeletePersona(id int64) {
	pc.mu.Lock()
	delete(pc.localList, id)
	pc.mu.Unlock()

	if config.RDB != nil {
		pipe := config.RDB.Pipeline()
		pipe.HDel(config.RedisCtx, RedisKeyPersonaList, fmt.Sprintf("%d", id))
		pipe.Del(config.RedisCtx, pc.cacheKeyPersonaFiles(id))
		pipe.Exec(config.RedisCtx)
	}
}

func (pc *PersonaCache) InvalidatePersona(id int64) {
	pc.mu.Lock()
	delete(pc.localList, id)
	pc.mu.Unlock()

	if config.RDB != nil {
		pipe := config.RDB.Pipeline()
		pipe.HDel(config.RedisCtx, RedisKeyPersonaList, fmt.Sprintf("%d", id))
		pipe.Del(config.RedisCtx, pc.cacheKeyPersonaFiles(id))
		pipe.Exec(config.RedisCtx)
	}
}

func (pc *PersonaCache) GetFileIndex(personaID int64) ([]model.PersonaFile, error) {
	cacheKey := pc.cacheKeyPersonaFiles(personaID)

	if config.RDB != nil {
		cached, err := config.RDB.HGetAll(config.RedisCtx, cacheKey).Result()
		if err == nil && len(cached) > 0 {
			files := make([]model.PersonaFile, 0, len(cached))
			for _, val := range cached {
				var pf model.PersonaFile
				if json.Unmarshal([]byte(val), &pf) == nil {
					files = append(files, pf)
				}
			}
			return files, nil
		}
	}

	files, err := pc.pfRepo.FindByPersonaID(personaID)
	if err != nil {
		return nil, err
	}

	if config.RDB != nil && len(files) > 0 {
		pipe := config.RDB.Pipeline()
		for _, f := range files {
			data, _ := json.Marshal(f)
			pipe.HSet(config.RedisCtx, cacheKey, fmt.Sprintf("%d", f.ID), data)
		}
		pipe.Exec(config.RedisCtx)
	}

	return files, nil
}

func (pc *PersonaCache) InvalidateFileIndex(personaID int64) {
	if config.RDB != nil {
		config.RDB.Del(config.RedisCtx, pc.cacheKeyPersonaFiles(personaID))
	}
}

func (pc *PersonaCache) GetMDContent(personaID int64, pf *model.PersonaFile) (string, error) {
	cacheKey := pc.cacheKeyMDContent(personaID, pf.ID)

	if config.RDB != nil {
		content, err := config.RDB.Get(config.RedisCtx, cacheKey).Result()
		if err == nil && content != "" {
			return content, nil
		}
	}

	return "", fmt.Errorf("md content not cached")
}

func (pc *PersonaCache) SetMDContent(personaID int64, pf *model.PersonaFile, content string) {
	cacheKey := pc.cacheKeyMDContent(personaID, pf.ID)

	if config.RDB != nil {
		ttl := mdContentTTL
		if isCoreModule(pf.ModuleCategory) {
			ttl = 0
		}
		config.RDB.Set(config.RedisCtx, cacheKey, content, ttl)
	}
}

func (pc *PersonaCache) InvalidateMDContent(personaID int64, fileID int64) {
	if config.RDB != nil {
		config.RDB.Del(config.RedisCtx, pc.cacheKeyMDContent(personaID, fileID))
	}
}

func (pc *PersonaCache) InvalidateAllPersonaMD(personaID int64) {
	files, err := pc.pfRepo.FindByPersonaID(personaID)
	if err != nil {
		return
	}
	if config.RDB != nil {
		pipe := config.RDB.Pipeline()
		for _, f := range files {
			pipe.Del(config.RedisCtx, pc.cacheKeyMDContent(personaID, f.ID))
		}
		pipe.Exec(config.RedisCtx)
	}
}

func isCoreModule(category string) bool {
	switch category {
	case "persona_base", "persona_tone", "forbidden_rules":
		return true
	}
	return false
}

func (pc *PersonaCache) InvalidateAll() {
	pc.mu.Lock()
	pc.localList = make(map[int64]*cachedPersonaMeta)
	pc.mu.Unlock()

	if config.RDB != nil {
		pipe := config.RDB.Pipeline()
		pipe.Del(config.RedisCtx, RedisKeyPersonaList)

		iter := config.RDB.Scan(config.RedisCtx, 0, "persona:files:*", 0).Iterator()
		for iter.Next(config.RedisCtx) {
			pipe.Del(config.RedisCtx, iter.Val())
		}

		iter2 := config.RDB.Scan(config.RedisCtx, 0, "persona:md:*", 0).Iterator()
		for iter2.Next(config.RedisCtx) {
			pipe.Del(config.RedisCtx, iter2.Val())
		}

		pipe.Exec(config.RedisCtx)
	}
}

func (pc *PersonaCache) GetPersonaNames() ([]string, error) {
	personas, err := pc.GetPersonaList()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(personas))
	for _, p := range personas {
		names = append(names, p.Name)
	}
	return names, nil
}

func (pc *PersonaCache) RefreshFromDB() error {
	personas, err := pc.repo.FindActive()
	if err != nil {
		return err
	}

	pc.InvalidateAll()
	pc.WarmupList(personas)

	log.Printf("PersonaCache: 已刷新 %d 个人格缓存", len(personas))
	return nil
}

func (pc *PersonaCache) GetPersonaByDirName(dirName string) (*model.Persona, error) {
	pc.mu.RLock()
	for _, meta := range pc.localList {
		if meta.DirName == dirName {
			persona := &model.Persona{
				ID:          meta.ID,
				Name:        meta.Name,
				Description: meta.Description,
				DirName:     meta.DirName,
				IsActive:    meta.IsActive,
				UserID:      meta.UserID,
			}
			pc.mu.RUnlock()
			return persona, nil
		}
	}
	pc.mu.RUnlock()

	return pc.repo.FindByDirName(dirName)
}
