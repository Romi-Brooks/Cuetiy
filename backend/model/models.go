package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"size:100;not null;default:'用户'" json:"username"`
	Email     string    `gorm:"size:200;uniqueIndex;not null" json:"email"`
	Avatar    string    `gorm:"size:500" json:"avatar"`
	Password  string    `gorm:"size:200;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Conversation struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64      `gorm:"index;not null" json:"user_id"`
	Title       string     `gorm:"size:200;not null;default:'情感陪伴'" json:"title"`
	AINickname  string     `gorm:"size:100;not null;default:'RainYi'" json:"ai_nickname"`
	AIAvatar    string     `gorm:"size:500" json:"ai_avatar"`
	PersonaID   *int64     `gorm:"index" json:"persona_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastMessage *LastMessage `gorm:"-" json:"last_message,omitempty"`
}

type LastMessage struct {
	Content   string    `json:"content"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID             int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64  `gorm:"index;not null" json:"conversation_id"`
	Role           string `gorm:"size:20;not null" json:"role"`
	// MessageType: text | voice（voice 为语音条，不可对文字消息朗读）
	MessageType    string `gorm:"size:20;not null;default:text" json:"message_type"`
	Content        string `gorm:"type:text;not null" json:"content"`
	AudioURL       string `gorm:"size:500" json:"audio_url,omitempty"`
	AudioDurationMs int64 `gorm:"default:0" json:"audio_duration_ms,omitempty"`
	HasAttachment  bool   `gorm:"default:false" json:"has_attachment"`
	AttachmentType string `gorm:"size:20" json:"attachment_type"`
	AttachmentURL  string `gorm:"size:500" json:"attachment_url"`
	CreatedAt      time.Time `json:"created_at"`
	IsDeleted      bool      `json:"is_deleted"`
}

type Persona struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64          `gorm:"index;default:0" json:"user_id"`
	Name        string         `gorm:"size:200;not null" json:"name"`
	Nickname    string         `gorm:"size:200" json:"nickname"`
	Description string         `gorm:"size:500" json:"description"`
	DirName     string         `gorm:"size:200" json:"dir_name"`
	Avatar      string         `gorm:"size:500" json:"avatar"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type PersonaFile struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PersonaID     int64     `gorm:"index;not null" json:"persona_id"`
	FileName      string    `gorm:"size:255;not null" json:"file_name"`
	StoragePath   string    `gorm:"size:500;not null" json:"storage_path"`
	Priority      int       `gorm:"default:0" json:"priority"`
	ModuleCategory string   `gorm:"size:100" json:"module_category"`
	FileSize      int64     `json:"file_size"`
	CreatedAt     time.Time `json:"created_at"`
}

type FileRecord struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64          `gorm:"index;not null" json:"user_id"`
	FileType      string         `gorm:"size:30;not null;index" json:"file_type"`
	ReferenceID   int64          `gorm:"index" json:"reference_id"`
	ReferenceType string         `gorm:"size:30;index" json:"reference_type"`
	OriginalName  string         `gorm:"size:255" json:"original_name"`
	StoragePath   string         `gorm:"size:500;not null" json:"storage_path"`
	URL           string         `gorm:"size:500" json:"url"`
	Size          int64          `json:"size"`
	MimeType      string         `gorm:"size:100" json:"mime_type"`
	CreatedAt     time.Time      `json:"created_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// ConversationSummary 滚动摘要：更早对话的压缩纪要（与 Skills、消息表分离）
type ConversationSummary struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID   int64     `gorm:"index;not null" json:"conversation_id"`
	Content          string    `gorm:"type:text;not null" json:"content"`
	CoveredMessageID int64     `gorm:"not null;default:0" json:"covered_message_id"`
	Version          int       `gorm:"not null;default:1" json:"version"`
	TokenEstimate    int       `gorm:"not null;default:0" json:"token_estimate"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ConversationMemory 记忆卡：关于用户本身，不是人格 Skills
type ConversationMemory struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64     `gorm:"index;not null" json:"conversation_id"`
	UserID         int64     `gorm:"index;not null" json:"user_id"`
	Content        string    `gorm:"type:text;not null" json:"content"` // JSON 结构化记忆
	TokenEstimate  int       `gorm:"not null;default:0" json:"token_estimate"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ConversationSkillState 会话级技能激活状态（L2 触发注入）
type ConversationSkillState struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64     `gorm:"uniqueIndex;not null" json:"conversation_id"`
	ActiveSkills   string    `gorm:"type:text" json:"active_skills"` // JSON map[skill]expireTurn
	TurnCounter    int64     `gorm:"not null;default:0" json:"turn_counter"`
	LastEmotion    string    `gorm:"size:50" json:"last_emotion"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ChatArchive 清空聊天前的溯源归档索引
type ChatArchive struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64     `gorm:"index;not null" json:"conversation_id"`
	UserID         int64     `gorm:"index;not null" json:"user_id"`
	FilePath       string    `gorm:"size:500;not null" json:"file_path"`
	MessageCount   int       `json:"message_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// UserVoice 用户音色样本（voiceclone），每个用户一份
type UserVoice struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64      `gorm:"uniqueIndex;not null" json:"user_id"`
	StoragePath  string     `gorm:"size:500;not null" json:"storage_path"`
	URL          string     `gorm:"size:500" json:"url"`
	MimeType     string     `gorm:"size:100" json:"mime_type"`
	OriginalName string     `gorm:"size:255" json:"original_name"`
	Size         int64      `json:"size"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(
		&User{}, &Conversation{}, &Message{}, &Persona{}, &PersonaFile{}, &FileRecord{},
		&ConversationSummary{}, &ConversationMemory{}, &ConversationSkillState{}, &ChatArchive{},
		&UserVoice{},
	)
}
