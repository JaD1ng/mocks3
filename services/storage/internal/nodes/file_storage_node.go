package nodes

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"mocks3/shared/utils"
)

// FileObject 文件对象
type FileObject struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Bucket      string    `json:"bucket"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	MD5Hash     string    `json:"md5_hash"`
	Data        []byte    `json:"data"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FileStorageNode 文件存储节点
type FileStorageNode struct {
	id       string
	rootPath string
}

// NewFileStorageNode 创建文件存储节点
func NewFileStorageNode(id, rootPath string) (*FileStorageNode, error) {
	// 确保根目录存在
	err := os.MkdirAll(rootPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create root directory %s: %v", rootPath, err)
	}

	return &FileStorageNode{
		id:       id,
		rootPath: rootPath,
	}, nil
}

// Write 写入文件对象
func (n *FileStorageNode) Write(obj *FileObject) error {
	if obj == nil {
		return fmt.Errorf("file object cannot be nil")
	}

	// 构建文件路径
	filePath := n.buildFilePath(obj.Key)

	// 确保目录存在
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// 写入文件
	err = os.WriteFile(filePath, obj.Data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %v", filePath, err)
	}

	// 验证写入的数据
	if obj.MD5Hash != "" {
		expectedHash := utils.CalculateMD5(obj.Data)
		if expectedHash != obj.MD5Hash {
			// 删除损坏的文件
			os.Remove(filePath)
			return fmt.Errorf("MD5 hash mismatch: expected %s, got %s", obj.MD5Hash, expectedHash)
		}
	}

	return nil
}

// Read 读取文件对象
func (n *FileStorageNode) Read(key string) (*FileObject, error) {
	filePath := n.buildFilePath(key)

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to stat file %s: %v", filePath, err)
	}

	// 读取文件数据
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	// 计算 MD5 哈希
	md5Hash := utils.CalculateMD5(data)

	// 解析 key 获取 bucket 信息
	bucket := ""
	if filepath.Dir(key) != "." {
		bucket = filepath.Dir(key)
	}

	return &FileObject{
		Key:         key,
		Bucket:      bucket,
		Size:        info.Size(),
		ContentType: "application/octet-stream", // 默认类型
		MD5Hash:     md5Hash,
		Data:        data,
		CreatedAt:   info.ModTime(),
		UpdatedAt:   info.ModTime(),
	}, nil
}

// Delete 删除文件对象
func (n *FileStorageNode) Delete(key string) error {
	filePath := n.buildFilePath(key)

	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", key)
		}
		return fmt.Errorf("failed to delete file %s: %v", filePath, err)
	}

	// 尝试清理空的父目录
	dir := filepath.Dir(filePath)
	n.cleanupEmptyDirs(dir)

	return nil
}

// GetNodeID 获取节点 ID
func (n *FileStorageNode) GetNodeID() string {
	return n.id
}

// GetStatus 获取节点状态
func (n *FileStorageNode) GetStatus() *NodeStatus {
	capacity, used := n.getDiskUsage()
	health := "healthy"

	// 简单的健康检查
	if used > int64(float64(capacity)*0.9) { // 使用率超过 90%
		health = "warning"
	}
	if used > int64(float64(capacity)*0.95) { // 使用率超过 95%
		health = "critical"
	}

	return &NodeStatus{
		ID:       n.id,
		Status:   "active",
		Capacity: capacity,
		Used:     used,
		Health:   health,
	}
}

// GetCapacity 获取存储容量
func (n *FileStorageNode) GetCapacity() int64 {
	capacity, _ := n.getDiskUsage()
	return capacity
}

// GetUsed 获取已使用空间
func (n *FileStorageNode) GetUsed() int64 {
	_, used := n.getDiskUsage()
	return used
}

// buildFilePath 构建文件路径
func (n *FileStorageNode) buildFilePath(key string) string {
	// 将 key 中的 '/' 转换为路径分隔符
	return filepath.Join(n.rootPath, key)
}

// getDiskUsage 获取磁盘使用情况
func (n *FileStorageNode) getDiskUsage() (capacity, used int64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(n.rootPath, &stat)
	if err != nil {
		// 如果获取失败，返回默认值
		return 10 * 1024 * 1024 * 1024, 0 // 10GB 容量，0 使用
	}

	// 计算总容量和已使用空间
	capacity = int64(stat.Blocks) * int64(stat.Bsize)
	available := int64(stat.Bavail) * int64(stat.Bsize)
	used = capacity - available

	return capacity, used
}

// cleanupEmptyDirs 清理空的父目录
func (n *FileStorageNode) cleanupEmptyDirs(dir string) {
	// 不删除根目录
	if dir == n.rootPath || dir == filepath.Dir(n.rootPath) {
		return
	}

	// 检查目录是否为空
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return
	}

	// 删除空目录
	err = os.Remove(dir)
	if err == nil {
		// 递归清理父目录
		n.cleanupEmptyDirs(filepath.Dir(dir))
	}
}

// ListFiles 列出节点中的所有文件（用于调试）
func (n *FileStorageNode) ListFiles() ([]string, error) {
	var files []string

	err := filepath.WalkDir(n.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			// 转换为相对于根目录的路径
			relPath, err := filepath.Rel(n.rootPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}
