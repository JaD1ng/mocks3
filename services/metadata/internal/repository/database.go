package repository

import (
	"context"
	"database/sql"
	"fmt"
	"mocks3/services/metadata/internal/config"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Database 数据库连接管理器
type Database struct {
	db *sql.DB
}

// NewDatabase 创建数据库连接
func NewDatabase(config config.DatabaseConfig) (*Database, error) {
	dsn := config.GetDSN()
	
	db, err := sql.Open(config.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}

	// 初始化数据库表
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return database, nil
}

// GetDB 获取数据库连接
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	return d.db.Close()
}

// initTables 初始化数据库表
func (d *Database) initTables() error {
	// 创建元数据表
	metadataTable := `
	CREATE TABLE IF NOT EXISTS metadata (
		id VARCHAR(255) PRIMARY KEY,
		key VARCHAR(500) NOT NULL,
		bucket VARCHAR(255) NOT NULL,
		size BIGINT NOT NULL,
		content_type VARCHAR(255),
		md5_hash VARCHAR(32),
		etag VARCHAR(255),
		storage_nodes JSONB,
		headers JSONB,
		tags JSONB,
		status VARCHAR(50) DEFAULT 'active',
		version BIGINT DEFAULT 1,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		deleted_at TIMESTAMP WITH TIME ZONE NULL
	);
	
	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_metadata_key ON metadata(key);
	CREATE INDEX IF NOT EXISTS idx_metadata_bucket ON metadata(bucket);
	CREATE INDEX IF NOT EXISTS idx_metadata_bucket_key ON metadata(bucket, key);
	CREATE INDEX IF NOT EXISTS idx_metadata_status ON metadata(status);
	CREATE INDEX IF NOT EXISTS idx_metadata_created_at ON metadata(created_at);
	CREATE INDEX IF NOT EXISTS idx_metadata_content_type ON metadata(content_type);
	CREATE INDEX IF NOT EXISTS idx_metadata_size ON metadata(size);
	
	-- 创建唯一约束
	CREATE UNIQUE INDEX IF NOT EXISTS idx_metadata_bucket_key_unique ON metadata(bucket, key) WHERE deleted_at IS NULL;
	`

	// 创建统计表
	statsTable := `
	CREATE TABLE IF NOT EXISTS stats_cache (
		id SERIAL PRIMARY KEY,
		stats_data JSONB NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	
	-- 确保只有一行统计数据
	CREATE UNIQUE INDEX IF NOT EXISTS idx_stats_cache_single ON stats_cache((1));
	`

	// 执行SQL
	for _, tableSQL := range []string{metadataTable, statsTable} {
		if _, err := d.db.Exec(tableSQL); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// HealthCheck 健康检查
func (d *Database) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return d.db.PingContext(ctx)
}

// BeginTx 开始事务
func (d *Database) BeginTx() (*sql.Tx, error) {
	return d.db.Begin()
}

// WithTx 在事务中执行操作
func (d *Database) WithTx(fn func(*sql.Tx) error) error {
	tx, err := d.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}