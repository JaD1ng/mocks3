package db

import (
	"encoding/json"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MetadataEntry 元数据条目
type MetadataEntry struct {
	ID           string      `json:"id" gorm:"primaryKey"`
	Key          string      `json:"key" gorm:"uniqueIndex;not null"`
	Bucket       string      `json:"bucket" gorm:"index;not null"`
	Size         int64       `json:"size"`
	ContentType  string      `json:"content_type"`
	MD5Hash      string      `json:"md5_hash"`
	StorageNodes StringArray `json:"storage_nodes" gorm:"type:jsonb"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// TableName 指定表名
func (MetadataEntry) TableName() string {
	return "metadata"
}

// StringArray 用于存储字符串数组到 JSONB
type StringArray []string

// Scan 实现 Scanner 接口
func (s *StringArray) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// Value 实现 Valuer 接口
func (s StringArray) Value() (any, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// NewPostgresDB 创建 PostgreSQL 连接
func NewPostgresDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&MetadataEntry{})
}

// MetadataRepository 元数据仓库
type MetadataRepository struct {
	db *gorm.DB
}

// NewMetadataRepository 创建元数据仓库
func NewMetadataRepository(db *gorm.DB) *MetadataRepository {
	return &MetadataRepository{db: db}
}

// Save 保存元数据
func (r *MetadataRepository) Save(entry *MetadataEntry) error {
	entry.UpdatedAt = time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	return r.db.Save(entry).Error
}

// GetByKey 根据 key 获取元数据
func (r *MetadataRepository) GetByKey(key string) (*MetadataEntry, error) {
	var entry MetadataEntry
	err := r.db.Where("key = ?", key).First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// Update 更新元数据
func (r *MetadataRepository) Update(key string, updates map[string]any) error {
	updates["updated_at"] = time.Now()
	return r.db.Model(&MetadataEntry{}).Where("key = ?", key).Updates(updates).Error
}

// Delete 删除元数据
func (r *MetadataRepository) Delete(key string) error {
	return r.db.Where("key = ?", key).Delete(&MetadataEntry{}).Error
}

// List 列出元数据
func (r *MetadataRepository) List(limit, offset int) ([]MetadataEntry, error) {
	var entries []MetadataEntry
	err := r.db.Limit(limit).Offset(offset).Order("created_at desc").Find(&entries).Error
	return entries, err
}

// Search 搜索元数据
func (r *MetadataRepository) Search(query string, limit int) ([]MetadataEntry, error) {
	var entries []MetadataEntry
	searchPattern := "%" + query + "%"
	err := r.db.Where("key ILIKE ? OR bucket ILIKE ? OR content_type ILIKE ?",
		searchPattern, searchPattern, searchPattern).
		Limit(limit).
		Order("created_at desc").
		Find(&entries).Error
	return entries, err
}

// GetStats 获取统计信息
func (r *MetadataRepository) GetStats() (map[string]any, error) {
	stats := make(map[string]any)

	// 总对象数
	var totalObjects int64
	err := r.db.Model(&MetadataEntry{}).Count(&totalObjects).Error
	if err != nil {
		return nil, err
	}
	stats["total_objects"] = totalObjects

	// 总存储大小
	var totalSize int64
	err = r.db.Model(&MetadataEntry{}).Select("COALESCE(SUM(size), 0)").Scan(&totalSize).Error
	if err != nil {
		return nil, err
	}
	stats["total_size"] = totalSize

	// 按 bucket 分组统计
	var bucketStats []struct {
		Bucket string `json:"bucket"`
		Count  int64  `json:"count"`
		Size   int64  `json:"size"`
	}
	err = r.db.Model(&MetadataEntry{}).
		Select("bucket, COUNT(*) as count, COALESCE(SUM(size), 0) as size").
		Group("bucket").
		Find(&bucketStats).Error
	if err != nil {
		return nil, err
	}
	stats["bucket_stats"] = bucketStats

	return stats, nil
}
