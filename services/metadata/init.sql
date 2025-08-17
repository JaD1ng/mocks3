-- 初始化元数据数据库

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建元数据表
CREATE TABLE IF NOT EXISTS metadata (
    id VARCHAR(255) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    key VARCHAR(500) NOT NULL,
    bucket VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    content_type VARCHAR(255),
    md5_hash VARCHAR(32),
    etag VARCHAR(255),
    storage_nodes JSONB DEFAULT '[]'::jsonb,
    headers JSONB DEFAULT '{}'::jsonb,
    tags JSONB DEFAULT '{}'::jsonb,
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
CREATE INDEX IF NOT EXISTS idx_metadata_deleted_at ON metadata(deleted_at);

-- 创建唯一约束（同一bucket下key唯一，排除已删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS idx_metadata_bucket_key_unique 
ON metadata(bucket, key) 
WHERE deleted_at IS NULL;

-- 创建GIN索引用于JSON字段搜索
CREATE INDEX IF NOT EXISTS idx_metadata_tags_gin ON metadata USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_metadata_headers_gin ON metadata USING gin(headers);
CREATE INDEX IF NOT EXISTS idx_metadata_storage_nodes_gin ON metadata USING gin(storage_nodes);

-- 创建复合索引用于常见查询
CREATE INDEX IF NOT EXISTS idx_metadata_bucket_status_created 
ON metadata(bucket, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_metadata_content_type_created 
ON metadata(content_type, created_at DESC) 
WHERE deleted_at IS NULL;

-- 创建统计缓存表
CREATE TABLE IF NOT EXISTS stats_cache (
    id SERIAL PRIMARY KEY,
    stats_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 确保只有一行统计数据
CREATE UNIQUE INDEX IF NOT EXISTS idx_stats_cache_single ON stats_cache((1));

-- 创建触发器函数用于更新updated_at字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 创建触发器
DROP TRIGGER IF EXISTS update_metadata_updated_at ON metadata;
CREATE TRIGGER update_metadata_updated_at
    BEFORE UPDATE ON metadata
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_stats_cache_updated_at ON stats_cache;
CREATE TRIGGER update_stats_cache_updated_at
    BEFORE UPDATE ON stats_cache
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 插入一些示例数据（可选）
INSERT INTO metadata (key, bucket, size, content_type, md5_hash, etag, storage_nodes, headers, tags, status)
VALUES 
    ('test/file1.txt', 'test-bucket', 1024, 'text/plain', 'd41d8cd98f00b204e9800998ecf8427e', '"d41d8cd98f00b204e9800998ecf8427e"', '["stg1", "stg2", "stg3"]'::jsonb, '{"Cache-Control": "max-age=3600"}'::jsonb, '{"env": "test", "type": "demo"}'::jsonb, 'active'),
    ('images/photo.jpg', 'media-bucket', 2048576, 'image/jpeg', 'a1b2c3d4e5f6789012345678901234567', '"a1b2c3d4e5f6789012345678901234567"', '["stg1", "stg2"]'::jsonb, '{"Content-Disposition": "inline"}'::jsonb, '{"category": "photo", "public": "true"}'::jsonb, 'active'),
    ('docs/readme.md', 'docs-bucket', 4096, 'text/markdown', 'f1e2d3c4b5a6978563214789012345678', '"f1e2d3c4b5a6978563214789012345678"', '["stg1", "stg3"]'::jsonb, '{"Cache-Control": "no-cache"}'::jsonb, '{"type": "documentation", "lang": "en"}'::jsonb, 'active')
ON CONFLICT (bucket, key) DO NOTHING;

-- 初始化统计缓存
INSERT INTO stats_cache (stats_data)
VALUES ('{"total_objects": 0, "total_size": 0, "average_size": 0, "bucket_stats": {}, "content_types": {}, "last_updated": "2024-01-01T00:00:00Z"}'::jsonb)
ON CONFLICT ((1)) DO NOTHING;