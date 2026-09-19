-- RainYi MySQL 初始化 SQL（兼容旧部署）
-- 默认驱动已是 PostgreSQL：见 schema.pg.sql
-- 应用启动时 GORM AutoMigrate 会自动建表
-- 本脚本仅用于：建库 + 建账号（表结构以 AutoMigrate 为准）
-- 使用本文件时请设置 DB_DRIVER=mysql

CREATE DATABASE IF NOT EXISTS rain_yi
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

-- 可选：专用账号（密码请改成强密码）
-- CREATE USER IF NOT EXISTS 'rainyi'@'%' IDENTIFIED BY 'CHANGE_ME';
-- GRANT ALL PRIVILEGES ON rain_yi.* TO 'rainyi'@'%';
-- FLUSH PRIVILEGES;

-- 手动建表兜底（与 backend/model/models.go 对应，仅在 AutoMigrate 失败时使用）
USE rain_yi;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(100) NOT NULL DEFAULT '用户',
  email VARCHAR(200) NOT NULL,
  avatar VARCHAR(500),
  password VARCHAR(200) NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS conversations (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  title VARCHAR(200) NOT NULL DEFAULT '情感陪伴',
  ai_nickname VARCHAR(100) NOT NULL DEFAULT 'RainYi',
  ai_avatar VARCHAR(500),
  persona_id BIGINT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  KEY idx_conversations_user_id (user_id),
  KEY idx_conversations_persona_id (persona_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS messages (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  conversation_id BIGINT NOT NULL,
  role VARCHAR(20) NOT NULL,
  content TEXT NOT NULL,
  has_attachment TINYINT(1) DEFAULT 0,
  attachment_type VARCHAR(20),
  attachment_url VARCHAR(500),
  created_at DATETIME(3) NULL,
  is_deleted TINYINT(1) DEFAULT 0,
  KEY idx_messages_conversation_id (conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS personas (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT DEFAULT 0,
  name VARCHAR(200) NOT NULL,
  nickname VARCHAR(200),
  description VARCHAR(500),
  dir_name VARCHAR(200),
  avatar VARCHAR(500),
  is_active TINYINT(1) DEFAULT 1,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  KEY idx_personas_user_id (user_id),
  KEY idx_personas_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS persona_files (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  persona_id BIGINT NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  storage_path VARCHAR(500) NOT NULL,
  priority INT DEFAULT 0,
  module_category VARCHAR(100),
  file_size BIGINT,
  created_at DATETIME(3) NULL,
  KEY idx_persona_files_persona_id (persona_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS file_records (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  file_type VARCHAR(30) NOT NULL,
  reference_id BIGINT,
  reference_type VARCHAR(30),
  original_name VARCHAR(255),
  storage_path VARCHAR(500) NOT NULL,
  url VARCHAR(500),
  size BIGINT,
  mime_type VARCHAR(100),
  created_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  KEY idx_file_records_user_id (user_id),
  KEY idx_file_records_file_type (file_type),
  KEY idx_file_records_reference_id (reference_id),
  KEY idx_file_records_reference_type (reference_type),
  KEY idx_file_records_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
