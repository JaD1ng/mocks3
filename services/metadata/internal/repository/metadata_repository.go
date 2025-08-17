package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mocks3/shared/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MetadataRepository 元数据仓库实现
type MetadataRepository struct {
	db *Database
}

// NewMetadataRepository 创建元数据仓库
func NewMetadataRepository(db *Database) *MetadataRepository {
	return &MetadataRepository{
		db: db,
	}
}

// Create 创建元数据
func (r *MetadataRepository) Create(ctx context.Context, metadata *models.Metadata) error {
	if metadata.ID == "" {
		metadata.ID = uuid.New().String()
	}

	// 序列化JSON字段
	storageNodesJSON, err := json.Marshal(metadata.StorageNodes)
	if err != nil {
		return fmt.Errorf("failed to marshal storage nodes: %w", err)
	}

	headersJSON, err := json.Marshal(metadata.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	tagsJSON, err := json.Marshal(metadata.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO metadata (
			id, key, bucket, size, content_type, md5_hash, etag,
			storage_nodes, headers, tags, status, version,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	now := time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = now
	}
	metadata.UpdatedAt = now

	_, err = r.db.GetDB().ExecContext(ctx, query,
		metadata.ID, metadata.Key, metadata.Bucket, metadata.Size,
		metadata.ContentType, metadata.MD5Hash, metadata.ETag,
		storageNodesJSON, headersJSON, tagsJSON,
		metadata.Status, metadata.Version,
		metadata.CreatedAt, metadata.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create metadata: %w", err)
	}

	return nil
}

// GetByKey 根据键获取元数据
func (r *MetadataRepository) GetByKey(ctx context.Context, bucket, key string) (*models.Metadata, error) {
	query := `
		SELECT id, key, bucket, size, content_type, md5_hash, etag,
			   storage_nodes, headers, tags, status, version,
			   created_at, updated_at, deleted_at
		FROM metadata
		WHERE bucket = $1 AND key = $2 AND deleted_at IS NULL
	`

	row := r.db.GetDB().QueryRowContext(ctx, query, bucket, key)
	
	metadata, err := r.scanMetadata(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("metadata not found: %s/%s", bucket, key)
		}
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	return metadata, nil
}

// Update 更新元数据
func (r *MetadataRepository) Update(ctx context.Context, metadata *models.Metadata) error {
	// 序列化JSON字段
	storageNodesJSON, err := json.Marshal(metadata.StorageNodes)
	if err != nil {
		return fmt.Errorf("failed to marshal storage nodes: %w", err)
	}

	headersJSON, err := json.Marshal(metadata.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	tagsJSON, err := json.Marshal(metadata.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE metadata
		SET size = $1, content_type = $2, md5_hash = $3, etag = $4,
			storage_nodes = $5, headers = $6, tags = $7, status = $8,
			version = version + 1, updated_at = $9
		WHERE bucket = $10 AND key = $11 AND deleted_at IS NULL
	`

	metadata.UpdatedAt = time.Now()
	metadata.Version++

	result, err := r.db.GetDB().ExecContext(ctx, query,
		metadata.Size, metadata.ContentType, metadata.MD5Hash, metadata.ETag,
		storageNodesJSON, headersJSON, tagsJSON, metadata.Status,
		metadata.UpdatedAt, metadata.Bucket, metadata.Key,
	)

	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("metadata not found: %s/%s", metadata.Bucket, metadata.Key)
	}

	return nil
}

// Delete 删除元数据（软删除）
func (r *MetadataRepository) Delete(ctx context.Context, bucket, key string) error {
	query := `
		UPDATE metadata
		SET deleted_at = $1, status = 'deleted', updated_at = $1
		WHERE bucket = $2 AND key = $3 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.GetDB().ExecContext(ctx, query, now, bucket, key)
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("metadata not found: %s/%s", bucket, key)
	}

	return nil
}

// List 列出元数据
func (r *MetadataRepository) List(ctx context.Context, bucket, prefix string, limit, offset int) ([]*models.Metadata, error) {
	var args []interface{}
	var conditions []string
	argIndex := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if bucket != "" {
		conditions = append(conditions, fmt.Sprintf("bucket = $%d", argIndex))
		args = append(args, bucket)
		argIndex++
	}

	if prefix != "" {
		conditions = append(conditions, fmt.Sprintf("key LIKE $%d", argIndex))
		args = append(args, prefix+"%")
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT id, key, bucket, size, content_type, md5_hash, etag,
			   storage_nodes, headers, tags, status, version,
			   created_at, updated_at, deleted_at
		FROM metadata
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata: %w", err)
	}
	defer rows.Close()

	var metadataList []*models.Metadata
	for rows.Next() {
		metadata, err := r.scanMetadata(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metadata: %w", err)
		}
		metadataList = append(metadataList, metadata)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return metadataList, nil
}

// Search 搜索元数据
func (r *MetadataRepository) Search(ctx context.Context, query string, limit int) ([]*models.Metadata, error) {
	sqlQuery := `
		SELECT id, key, bucket, size, content_type, md5_hash, etag,
			   storage_nodes, headers, tags, status, version,
			   created_at, updated_at, deleted_at
		FROM metadata
		WHERE deleted_at IS NULL AND (
			key ILIKE $1 OR
			bucket ILIKE $1 OR
			content_type ILIKE $1 OR
			tags::text ILIKE $1
		)
		ORDER BY created_at DESC
		LIMIT $2
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.GetDB().QueryContext(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search metadata: %w", err)
	}
	defer rows.Close()

	var metadataList []*models.Metadata
	for rows.Next() {
		metadata, err := r.scanMetadata(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metadata: %w", err)
		}
		metadataList = append(metadataList, metadata)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return metadataList, nil
}

// Count 计数
func (r *MetadataRepository) Count(ctx context.Context, bucket, prefix string) (int64, error) {
	var args []interface{}
	var conditions []string
	argIndex := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if bucket != "" {
		conditions = append(conditions, fmt.Sprintf("bucket = $%d", argIndex))
		args = append(args, bucket)
		argIndex++
	}

	if prefix != "" {
		conditions = append(conditions, fmt.Sprintf("key LIKE $%d", argIndex))
		args = append(args, prefix+"%")
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM metadata
		WHERE %s
	`, strings.Join(conditions, " AND "))

	var count int64
	err := r.db.GetDB().QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count metadata: %w", err)
	}

	return count, nil
}

// GetStats 获取统计信息
func (r *MetadataRepository) GetStats(ctx context.Context) (*models.Stats, error) {
	// 基础统计
	baseQuery := `
		SELECT 
			COUNT(*) as total_objects,
			COALESCE(SUM(size), 0) as total_size,
			COALESCE(AVG(size), 0) as average_size
		FROM metadata
		WHERE deleted_at IS NULL
	`

	var stats models.Stats
	err := r.db.GetDB().QueryRowContext(ctx, baseQuery).Scan(
		&stats.TotalObjects,
		&stats.TotalSize,
		&stats.AverageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get base stats: %w", err)
	}

	// 按bucket统计
	bucketQuery := `
		SELECT bucket, COUNT(*)
		FROM metadata
		WHERE deleted_at IS NULL
		GROUP BY bucket
	`
	bucketRows, err := r.db.GetDB().QueryContext(ctx, bucketQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket stats: %w", err)
	}
	defer bucketRows.Close()

	stats.BucketStats = make(map[string]int64)
	for bucketRows.Next() {
		var bucket string
		var count int64
		if err := bucketRows.Scan(&bucket, &count); err != nil {
			return nil, fmt.Errorf("failed to scan bucket stats: %w", err)
		}
		stats.BucketStats[bucket] = count
	}

	// 按内容类型统计
	contentTypeQuery := `
		SELECT content_type, COUNT(*)
		FROM metadata
		WHERE deleted_at IS NULL AND content_type IS NOT NULL
		GROUP BY content_type
	`
	ctRows, err := r.db.GetDB().QueryContext(ctx, contentTypeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get content type stats: %w", err)
	}
	defer ctRows.Close()

	stats.ContentTypes = make(map[string]int64)
	for ctRows.Next() {
		var contentType string
		var count int64
		if err := ctRows.Scan(&contentType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan content type stats: %w", err)
		}
		stats.ContentTypes[contentType] = count
	}

	stats.LastUpdated = time.Now()
	return &stats, nil
}

// scanMetadata 扫描元数据行
func (r *MetadataRepository) scanMetadata(scanner interface{}) (*models.Metadata, error) {
	var metadata models.Metadata
	var storageNodesJSON, headersJSON, tagsJSON []byte
	var deletedAt sql.NullTime

	var err error
	switch s := scanner.(type) {
	case *sql.Row:
		err = s.Scan(
			&metadata.ID, &metadata.Key, &metadata.Bucket, &metadata.Size,
			&metadata.ContentType, &metadata.MD5Hash, &metadata.ETag,
			&storageNodesJSON, &headersJSON, &tagsJSON,
			&metadata.Status, &metadata.Version,
			&metadata.CreatedAt, &metadata.UpdatedAt, &deletedAt,
		)
	case *sql.Rows:
		err = s.Scan(
			&metadata.ID, &metadata.Key, &metadata.Bucket, &metadata.Size,
			&metadata.ContentType, &metadata.MD5Hash, &metadata.ETag,
			&storageNodesJSON, &headersJSON, &tagsJSON,
			&metadata.Status, &metadata.Version,
			&metadata.CreatedAt, &metadata.UpdatedAt, &deletedAt,
		)
	default:
		return nil, fmt.Errorf("unsupported scanner type")
	}

	if err != nil {
		return nil, err
	}

	// 反序列化JSON字段
	if err := json.Unmarshal(storageNodesJSON, &metadata.StorageNodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage nodes: %w", err)
	}

	if err := json.Unmarshal(headersJSON, &metadata.Headers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &metadata.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	if deletedAt.Valid {
		metadata.DeletedAt = &deletedAt.Time
	}

	return &metadata, nil
}