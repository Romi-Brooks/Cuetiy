package skill

import (
	"fmt"
	"sync"
	"time"

	"cuetiy-backend/config"
)

type PromptCache struct {
	mu           sync.RWMutex
	cache        map[int64]*cachedPrompt
	personaCache *PersonaCache
	personaStg   MDFileStorage
}

type cachedPrompt struct {
	Prompt    string
	CreatedAt time.Time
}

func NewPromptCache(personaCache *PersonaCache, personaStg MDFileStorage) *PromptCache {
	return &PromptCache{
		cache:        make(map[int64]*cachedPrompt),
		personaCache: personaCache,
		personaStg:   personaStg,
	}
}

func (pc *PromptCache) Get(personaID int64) (string, bool) {
	pc.mu.RLock()
	cp, ok := pc.cache[personaID]
	pc.mu.RUnlock()
	if ok {
		return cp.Prompt, true
	}

	if config.RDB != nil {
		val, err := config.RDB.Get(config.RedisCtx, fmt.Sprintf("skill:prompt:%d", personaID)).Result()
		if err == nil && val != "" {
			pc.mu.Lock()
			pc.cache[personaID] = &cachedPrompt{Prompt: val, CreatedAt: time.Now()}
			pc.mu.Unlock()
			return val, true
		}
	}

	return "", false
}

func (pc *PromptCache) Set(personaID int64, prompt string) {
	pc.mu.Lock()
	pc.cache[personaID] = &cachedPrompt{Prompt: prompt, CreatedAt: time.Now()}
	pc.mu.Unlock()

	if config.RDB != nil {
		config.RDB.Set(config.RedisCtx, fmt.Sprintf("skill:prompt:%d", personaID), prompt, 0)
	}
}

func (pc *PromptCache) Invalidate(personaID int64) {
	pc.mu.Lock()
	delete(pc.cache, personaID)
	pc.mu.Unlock()

	if config.RDB != nil {
		pipe := config.RDB.Pipeline()
		pipe.Del(config.RedisCtx, fmt.Sprintf("skill:prompt:%d", personaID))
		pipe.Exec(config.RedisCtx)
	}

	if pc.personaCache != nil {
		pc.personaCache.InvalidateFileIndex(personaID)
		pc.personaCache.InvalidateAllPersonaMD(personaID)
	}
}

func (pc *PromptCache) CompileAndCache(personaID int64) string {
	prompt := pc.buildFromStorage(personaID)
	if prompt != "" {
		pc.Set(personaID, prompt)
	}
	return prompt
}

func (pc *PromptCache) buildFromStorage(personaID int64) string {
	if pc.personaCache == nil || pc.personaStg == nil {
		return ""
	}

	files, err := pc.personaCache.GetFileIndex(personaID)
	if err != nil || len(files) == 0 {
		return ""
	}

	return CompilePromptFromFiles(files, pc.personaStg, pc.personaCache, personaID)
}

func (pc *PromptCache) Warmup(personas []struct {
	ID   int64
	Name string
}) {
	for _, p := range personas {
		prompt := pc.CompileAndCache(p.ID)
		if prompt != "" {
			pc.Set(p.ID, prompt)
		}
	}
}
