package service

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"

	"github.com/google/uuid"
)

type FileStorage interface {
	Save(fileType string, userID int64, referenceType string, referenceID int64, originalName string, data io.Reader, size int64) (*model.FileRecord, error)
	Delete(record *model.FileRecord) error
	GetURL(path string) string
	Get(path string) (io.ReadCloser, error)
	SaveToPath(objectName string, userID int64, fileType string, referenceType string, referenceID int64, originalName string, data io.Reader, size int64) (*model.FileRecord, error)
}

type LocalStorage struct {
	root string
	repo *repository.FileRepository
}

func NewLocalStorage(cfg *config.Config, repo *repository.FileRepository) FileStorage {
	root := cfg.StorageDir
	if root == "" {
		root = "./data/files"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		log.Printf("警告: 创建本地存储目录失败: %v", err)
		return nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	log.Printf("本地文件存储已就绪: %s", abs)
	return &LocalStorage{root: abs, repo: repo}
}

func (s *LocalStorage) resolve(path string) (string, error) {
	clean := filepath.Clean("/" + path)
	full := filepath.Join(s.root, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
	if !strings.HasPrefix(full, s.root) {
		return "", fmt.Errorf("非法路径: %s", path)
	}
	return full, nil
}

func (s *LocalStorage) writeObject(objectName string, data io.Reader) error {
	full, err := s.resolve(objectName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, data); err != nil {
		return fmt.Errorf("写入内容失败: %w", err)
	}
	return nil
}

func (s *LocalStorage) Save(fileType string, userID int64, referenceType string, referenceID int64, originalName string, data io.Reader, size int64) (*model.FileRecord, error) {
	ext := filepath.Ext(originalName)
	objectName := fmt.Sprintf("%s/%d/%s%s", fileType, userID, uuid.New().String(), ext)
	return s.SaveToPath(objectName, userID, fileType, referenceType, referenceID, originalName, data, size)
}

func (s *LocalStorage) SaveToPath(objectName string, userID int64, fileType string, referenceType string, referenceID int64, originalName string, data io.Reader, size int64) (*model.FileRecord, error) {
	if err := s.writeObject(objectName, data); err != nil {
		return nil, err
	}

	contentType := detectContentType(filepath.Ext(originalName))
	if contentType == "application/octet-stream" {
		contentType = detectContentType(filepath.Ext(objectName))
	}

	record := &model.FileRecord{
		UserID:        userID,
		FileType:      fileType,
		ReferenceID:   referenceID,
		ReferenceType: referenceType,
		OriginalName:  originalName,
		StoragePath:   objectName,
		URL:           s.GetURL(objectName),
		Size:          size,
		MimeType:      contentType,
	}

	if err := s.repo.Create(record); err != nil {
		return nil, fmt.Errorf("FileRecord 入库失败: %w", err)
	}
	return record, nil
}

func (s *LocalStorage) Delete(record *model.FileRecord) error {
	full, err := s.resolve(record.StoragePath)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

func (s *LocalStorage) GetURL(path string) string {
	return "/storage/" + strings.TrimPrefix(path, "/")
}

func (s *LocalStorage) Get(path string) (io.ReadCloser, error) {
	full, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

type localFileObject struct {
	*os.File
}

func (l *localFileObject) Size() int64 {
	info, err := l.Stat()
	if err != nil {
		return -1
	}
	return info.Size()
}

func detectContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".mp4":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

func SaveUploadedFile(storage FileStorage, fileType string, userID int64, referenceType string, referenceID int64, fileHeader *multipart.FileHeader) (*model.FileRecord, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取上传文件失败: %w", err)
	}

	return storage.Save(fileType, userID, referenceType, referenceID, fileHeader.Filename, bytes.NewReader(data), int64(len(data)))
}
