package repository

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mocks3/shared/models"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// FileStorageNode 文件存储节点实现
type FileStorageNode struct {
	nodeID   string
	basePath string
}

// NewFileStorageNode 创建文件存储节点
func NewFileStorageNode(nodeID, basePath string) (*FileStorageNode, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory %s: %w", basePath, err)
	}

	return &FileStorageNode{
		nodeID:   nodeID,
		basePath: basePath,
	}, nil
}

// GetNodeID 获取节点ID
func (fs *FileStorageNode) GetNodeID() string {
	return fs.nodeID
}

// Write 写入对象
func (fs *FileStorageNode) Write(ctx context.Context, object *models.Object) error {
	if object == nil {
		return fmt.Errorf("object cannot be nil")
	}

	// 构建文件路径
	filePath := fs.buildFilePath(object.Bucket, object.Key)

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 写入文件
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// 计算MD5哈希
	hasher := md5.New()

	// 同时写入文件和哈希计算器
	multiWriter := io.MultiWriter(file, hasher)

	bytesWritten, err := multiWriter.Write(object.Data)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	// 验证写入的字节数
	if int64(bytesWritten) != object.Size {
		return fmt.Errorf("size mismatch: expected %d, written %d", object.Size, bytesWritten)
	}

	// 验证MD5哈希（如果提供）
	calculatedHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if object.MD5Hash != "" && object.MD5Hash != calculatedHash {
		return fmt.Errorf("MD5 hash mismatch: expected %s, calculated %s", object.MD5Hash, calculatedHash)
	}

	// 更新对象的MD5哈希
	if object.MD5Hash == "" {
		object.MD5Hash = calculatedHash
	}

	// 设置ETag（通常与MD5相同）
	if object.ETag == "" {
		object.ETag = fmt.Sprintf("\"%s\"", calculatedHash)
	}

	// 设置对象ID（如果没有）
	if object.ID == "" {
		object.ID = uuid.New().String()
	}

	return nil
}

// Read 读取对象
func (fs *FileStorageNode) Read(ctx context.Context, bucket, key string) (*models.Object, error) {
	filePath := fs.buildFilePath(bucket, key)

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found: %s/%s", bucket, key)
		}
		return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// 计算MD5哈希
	hash := md5.Sum(data)
	md5Hash := fmt.Sprintf("%x", hash)

	// 构建对象
	object := &models.Object{
		Key:         key,
		Bucket:      bucket,
		Size:        int64(len(data)),
		Data:        data,
		MD5Hash:     md5Hash,
		ETag:        fmt.Sprintf("\"%s\"", md5Hash),
		ContentType: fs.detectContentType(key),
		Headers:     make(map[string]string),
		Tags:        make(map[string]string),
		CreatedAt:   fileInfo.ModTime(),
		UpdatedAt:   fileInfo.ModTime(),
	}

	return object, nil
}

// Delete 删除对象
func (fs *FileStorageNode) Delete(ctx context.Context, bucket, key string) error {
	filePath := fs.buildFilePath(bucket, key)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，认为删除成功
			return nil
		}
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}

	// 尝试删除空目录
	fs.cleanupEmptyDirs(filepath.Dir(filePath))

	return nil
}

// IsHealthy 检查节点健康状态
func (fs *FileStorageNode) IsHealthy(ctx context.Context) bool {
	// 检查基础路径是否可访问
	if _, err := os.Stat(fs.basePath); err != nil {
		return false
	}

	// 尝试创建临时文件来测试写入权限
	tempFile := filepath.Join(fs.basePath, ".health_check")
	if err := os.WriteFile(tempFile, []byte("health_check"), 0644); err != nil {
		return false
	}

	// 清理临时文件
	os.Remove(tempFile)
	return true
}

// ListObjects 列出对象（目录遍历）
func (fs *FileStorageNode) ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]*models.ObjectInfo, error) {
	bucketPath := filepath.Join(fs.basePath, bucket)

	// 检查bucket目录是否存在
	if _, err := os.Stat(bucketPath); err != nil {
		if os.IsNotExist(err) {
			return []*models.ObjectInfo{}, nil
		}
		return nil, fmt.Errorf("failed to access bucket directory: %w", err)
	}

	var objects []*models.ObjectInfo
	count := 0

	err := filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 如果已达到限制，停止遍历
		if limit > 0 && count >= limit {
			return filepath.SkipDir
		}

		// 计算相对于bucket的key
		relPath, err := filepath.Rel(bucketPath, path)
		if err != nil {
			return err
		}

		// 将Windows路径分隔符转换为Unix风格
		key := filepath.ToSlash(relPath)

		// 检查前缀匹配
		if prefix != "" && !filepath.HasPrefix(key, prefix) {
			return nil
		}

		// 计算MD5哈希
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		hash := md5.Sum(data)
		md5Hash := fmt.Sprintf("%x", hash)

		objectInfo := &models.ObjectInfo{
			Key:         key,
			Bucket:      bucket,
			Size:        info.Size(),
			ContentType: fs.detectContentType(key),
			MD5Hash:     md5Hash,
			ETag:        fmt.Sprintf("\"%s\"", md5Hash),
			Headers:     make(map[string]string),
			Tags:        make(map[string]string),
			CreatedAt:   info.ModTime(),
			UpdatedAt:   info.ModTime(),
		}

		objects = append(objects, objectInfo)
		count++
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return objects, nil
}

// GetStats 获取节点统计信息
func (fs *FileStorageNode) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计总大小和文件数量
	var totalSize int64
	var totalFiles int64

	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += info.Size()
			totalFiles++
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate stats: %w", err)
	}

	stats["node_id"] = fs.nodeID
	stats["base_path"] = fs.basePath
	stats["total_size"] = totalSize
	stats["total_files"] = totalFiles
	stats["healthy"] = fs.IsHealthy(ctx)
	stats["timestamp"] = time.Now().Format(time.RFC3339)

	return stats, nil
}

// buildFilePath 构建文件路径
func (fs *FileStorageNode) buildFilePath(bucket, key string) string {
	return filepath.Join(fs.basePath, bucket, key)
}

// detectContentType 检测内容类型
func (fs *FileStorageNode) detectContentType(key string) string {
	ext := filepath.Ext(key)
	switch ext {
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

// cleanupEmptyDirs 清理空目录
func (fs *FileStorageNode) cleanupEmptyDirs(dirPath string) {
	// 不要删除基础路径
	if dirPath == fs.basePath {
		return
	}

	// 检查目录是否为空
	entries, err := os.ReadDir(dirPath)
	if err != nil || len(entries) > 0 {
		return
	}

	// 删除空目录
	if err := os.Remove(dirPath); err == nil {
		// 递归清理父目录
		fs.cleanupEmptyDirs(filepath.Dir(dirPath))
	}
}
