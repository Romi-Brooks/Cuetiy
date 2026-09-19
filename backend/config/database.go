package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase 按 DB_DRIVER 打开数据库：
//   postgres — 云端 / 服务器（默认）
//   mysql    — 兼容旧部署
//   sqlite   — 本地 EXE/APK 统一包（离线，文件库）
func InitDatabase() *gorm.DB {
	cfg := AppConfig
	driver := strings.ToLower(strings.TrimSpace(cfg.DBDriver))
	if driver == "" {
		driver = "postgres"
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	var (
		dialector gorm.Dialector
		target    string
	)

	switch driver {
	case "postgres", "pg", "postgresql":
		sslMode := cfg.DBSSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
			cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, sslMode,
		)
		dialector = postgres.Open(dsn)
		target = fmt.Sprintf("%s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
		)
		dialector = mysql.Open(dsn)
		target = fmt.Sprintf("%s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	case "sqlite":
		path := cfg.SQLitePath
		if path == "" {
			if cfg.RuntimeDir == "" || cfg.RuntimeDir == "." {
				path = filepath.Join("data", "cuetiy.db")
			} else {
				path = filepath.Join(cfg.RuntimeDir, "data", "cuetiy.db")
			}
		}
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0o755)
		}
		// busy_timeout / WAL：本地多协程写入更稳，也便于打包后单机使用
		dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
		dialector = sqlite.Open(dsn)
		target = path

	default:
		panic(fmt.Sprintf("unsupported DB_DRIVER=%q (use postgres | mysql | sqlite)", cfg.DBDriver))
	}

	var err error
	DB, err = gorm.Open(dialector, gormCfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect database [%s] %s: %v", driver, target, err))
	}

	sqlDB, err := DB.DB()
	if err != nil {
		panic(fmt.Sprintf("Failed to get sql.DB: %v", err))
	}

	if driver == "sqlite" {
		// SQLite 单写者：连接数压到 1，避免 database is locked
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}

	return DB
}
