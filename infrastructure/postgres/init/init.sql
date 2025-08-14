-- 创建数据库
CREATE DATABASE IF NOT EXISTS micro_s3;

-- 创建用户
CREATE USER IF NOT EXISTS micro_s3 WITH PASSWORD 'micro_s3_password';

-- 授权
GRANT ALL PRIVILEGES ON DATABASE micro_s3 TO micro_s3;

-- 连接到 micro_s3 数据库
\c micro_s3;

-- 创建元数据表
CREATE TABLE IF NOT EXISTS metadata_entries (
    id VARCHAR(255) PRIMARY KEY,
    key VARCHAR(1024) UNIQUE NOT NULL,
    bucket VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    content_type VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    md5_hash VARCHAR(32),
    storage_nodes JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_metadata_bucket ON metadata_entries(bucket);
CREATE INDEX IF NOT EXISTS idx_metadata_created_at ON metadata_entries(created_at);
CREATE INDEX IF NOT EXISTS idx_metadata_size ON metadata_entries(size);
CREATE INDEX IF NOT EXISTS idx_metadata_content_type ON metadata_entries(content_type);

-- 创建全文搜索索引
CREATE INDEX IF NOT EXISTS idx_metadata_key_search ON metadata_entries USING gin(to_tsvector('english', key));
CREATE INDEX IF NOT EXISTS idx_metadata_bucket_search ON metadata_entries USING gin(to_tsvector('english', bucket));

-- 创建更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_metadata_entries_updated_at 
    BEFORE UPDATE ON metadata_entries 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- 插入一些测试数据
INSERT INTO metadata_entries (id, key, bucket, size, content_type, md5_hash, storage_nodes) 
VALUES 
    ('test-1', 'test-bucket/sample1.txt', 'test-bucket', 1024, 'text/plain', 'abc123def456', '["stg1", "stg2", "stg3"]'),
    ('test-2', 'test-bucket/sample2.jpg', 'test-bucket', 2048, 'image/jpeg', 'def456ghi789', '["stg1", "stg2", "stg3"]'),
    ('test-3', 'docs/readme.md', 'docs', 512, 'text/markdown', 'ghi789jkl012', '["stg1", "stg2", "stg3"]')
ON CONFLICT (key) DO NOTHING;

-- 授权给 micro_s3 用户
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO micro_s3;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO micro_s3;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO micro_s3;