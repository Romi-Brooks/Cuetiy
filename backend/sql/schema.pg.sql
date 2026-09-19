-- Cuetiy PostgreSQL 初始化
-- 表结构以 backend/model/models.go + GORM AutoMigrate 为准
-- 本脚本仅用于：建库 + 建账号（可选）

-- CREATE DATABASE cuetiy
--   WITH ENCODING 'UTF8'
--   LC_COLLATE='en_US.utf8'
--   LC_CTYPE='en_US.utf8'
--   TEMPLATE=template0;

-- 可选：专用账号
-- CREATE USER cuetiy WITH PASSWORD 'CHANGE_ME';
-- GRANT ALL PRIVILEGES ON DATABASE cuetiy TO cuetiy;

-- 手动建表兜底（与 models.go 对应；AutoMigrate 失败时使用）
-- 注意：bool / timestamptz / text 为 PG 语义

CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(100) NOT NULL DEFAULT '用户',
  email VARCHAR(200) NOT NULL,
  avatar VARCHAR(500),
  password VARCHAR(200) NOT NULL,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE TABLE IF NOT EXISTS conversations (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  title VARCHAR(200) NOT NULL DEFAULT '情感陪伴',
  ai_nickname VARCHAR(100) NOT NULL DEFAULT 'Cuetiy',
  ai_avatar VARCHAR(500),
  persona_id BIGINT NULL,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations (user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_persona_id ON conversations (persona_id);

CREATE TABLE IF NOT EXISTS messages (
  id BIGSERIAL PRIMARY KEY,
  conversation_id BIGINT NOT NULL,
  role VARCHAR(20) NOT NULL,
  message_type VARCHAR(20) NOT NULL DEFAULT 'text',
  content TEXT NOT NULL,
  audio_url VARCHAR(500),
  audio_duration_ms BIGINT DEFAULT 0,
  has_attachment BOOLEAN DEFAULT FALSE,
  attachment_type VARCHAR(20),
  attachment_url VARCHAR(500),
  created_at TIMESTAMPTZ NULL,
  is_deleted BOOLEAN DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages (conversation_id);

CREATE TABLE IF NOT EXISTS personas (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT DEFAULT 0,
  name VARCHAR(200) NOT NULL,
  nickname VARCHAR(200),
  description VARCHAR(500),
  dir_name VARCHAR(200),
  avatar VARCHAR(500),
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL,
  deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_personas_user_id ON personas (user_id);
CREATE INDEX IF NOT EXISTS idx_personas_deleted_at ON personas (deleted_at);

CREATE TABLE IF NOT EXISTS persona_files (
  id BIGSERIAL PRIMARY KEY,
  persona_id BIGINT NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  storage_path VARCHAR(500) NOT NULL,
  priority INT DEFAULT 0,
  module_category VARCHAR(100),
  file_size BIGINT,
  created_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_persona_files_persona_id ON persona_files (persona_id);

CREATE TABLE IF NOT EXISTS file_records (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  file_type VARCHAR(30) NOT NULL,
  reference_id BIGINT,
  reference_type VARCHAR(30),
  original_name VARCHAR(255),
  storage_path VARCHAR(500) NOT NULL,
  url VARCHAR(500),
  size BIGINT,
  mime_type VARCHAR(100),
  created_at TIMESTAMPTZ NULL,
  deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_file_records_user_id ON file_records (user_id);
CREATE INDEX IF NOT EXISTS idx_file_records_file_type ON file_records (file_type);
CREATE INDEX IF NOT EXISTS idx_file_records_reference_id ON file_records (reference_id);
CREATE INDEX IF NOT EXISTS idx_file_records_reference_type ON file_records (reference_type);
CREATE INDEX IF NOT EXISTS idx_file_records_deleted_at ON file_records (deleted_at);

CREATE TABLE IF NOT EXISTS conversation_summaries (
  id BIGSERIAL PRIMARY KEY,
  conversation_id BIGINT NOT NULL,
  content TEXT NOT NULL,
  covered_message_id BIGINT NOT NULL DEFAULT 0,
  version INT NOT NULL DEFAULT 1,
  token_estimate INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_conversation_summaries_conversation_id ON conversation_summaries (conversation_id);

CREATE TABLE IF NOT EXISTS conversation_memories (
  id BIGSERIAL PRIMARY KEY,
  conversation_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  content TEXT NOT NULL,
  token_estimate INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_conversation_memories_conversation_id ON conversation_memories (conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_memories_user_id ON conversation_memories (user_id);

CREATE TABLE IF NOT EXISTS conversation_skill_states (
  id BIGSERIAL PRIMARY KEY,
  conversation_id BIGINT NOT NULL,
  active_skills TEXT,
  turn_counter BIGINT NOT NULL DEFAULT 0,
  last_emotion VARCHAR(50),
  updated_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_skill_states_conversation_id
  ON conversation_skill_states (conversation_id);

CREATE TABLE IF NOT EXISTS chat_archives (
  id BIGSERIAL PRIMARY KEY,
  conversation_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  file_path VARCHAR(500) NOT NULL,
  message_count INT,
  created_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_chat_archives_conversation_id ON chat_archives (conversation_id);
CREATE INDEX IF NOT EXISTS idx_chat_archives_user_id ON chat_archives (user_id);

CREATE TABLE IF NOT EXISTS user_voices (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  storage_path VARCHAR(500) NOT NULL,
  url VARCHAR(500),
  mime_type VARCHAR(100),
  original_name VARCHAR(255),
  size BIGINT,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL,
  deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_voices_user_id ON user_voices (user_id);
