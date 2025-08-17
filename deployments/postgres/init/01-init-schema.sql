-- MockS3 PostgreSQL 初始化脚本
-- 创建数据库架构和初始数据

-- 设置数据库配置
SET timezone = 'Asia/Shanghai';
SET default_text_search_config = 'pg_catalog.english';

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- 创建元数据表
CREATE TABLE IF NOT EXISTS object_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bucket VARCHAR(255) NOT NULL,
    key TEXT NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    etag VARCHAR(64) NOT NULL,
    content_type VARCHAR(255) DEFAULT 'application/octet-stream',
    content_encoding VARCHAR(100),
    content_disposition VARCHAR(500),
    cache_control VARCHAR(200),
    expires TIMESTAMP WITH TIME ZONE,
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    storage_class VARCHAR(50) DEFAULT 'STANDARD',
    version_id VARCHAR(100),
    delete_marker BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    tags JSONB DEFAULT '{}',
    
    -- 约束
    CONSTRAINT object_metadata_bucket_key_unique UNIQUE (bucket, key),
    CONSTRAINT object_metadata_size_check CHECK (size >= 0),
    CONSTRAINT object_metadata_bucket_check CHECK (bucket ~ '^[a-z0-9][a-z0-9.-]*[a-z0-9]$'),
    CONSTRAINT object_metadata_key_check CHECK (length(key) > 0 AND length(key) <= 1024)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_object_metadata_bucket ON object_metadata(bucket);
CREATE INDEX IF NOT EXISTS idx_object_metadata_key ON object_metadata(key);
CREATE INDEX IF NOT EXISTS idx_object_metadata_last_modified ON object_metadata(last_modified DESC);
CREATE INDEX IF NOT EXISTS idx_object_metadata_created_at ON object_metadata(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_object_metadata_size ON object_metadata(size);
CREATE INDEX IF NOT EXISTS idx_object_metadata_content_type ON object_metadata(content_type);
CREATE INDEX IF NOT EXISTS idx_object_metadata_storage_class ON object_metadata(storage_class);

-- GIN 索引用于 JSONB 查询
CREATE INDEX IF NOT EXISTS idx_object_metadata_metadata_gin ON object_metadata USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_object_metadata_tags_gin ON object_metadata USING GIN (tags);

-- 全文搜索索引
CREATE INDEX IF NOT EXISTS idx_object_metadata_key_trgm ON object_metadata USING GIN (key gin_trgm_ops);

-- 创建存储桶统计视图
CREATE OR REPLACE VIEW bucket_statistics AS
SELECT 
    bucket,
    COUNT(*) as object_count,
    SUM(size) as total_size,
    AVG(size) as avg_size,
    MIN(size) as min_size,
    MAX(size) as max_size,
    MIN(created_at) as first_object_created,
    MAX(last_modified) as last_object_modified,
    COUNT(DISTINCT content_type) as content_type_count
FROM object_metadata 
WHERE delete_marker = FALSE
GROUP BY bucket;

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建触发器
DROP TRIGGER IF EXISTS trigger_update_updated_at ON object_metadata;
CREATE TRIGGER trigger_update_updated_at
    BEFORE UPDATE ON object_metadata
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- 创建分区表函数 (按月分区)
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name TEXT, start_date DATE)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    end_date DATE;
BEGIN
    partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM');
    end_date := (start_date + INTERVAL '1 month')::DATE;
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I 
                    FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;

-- 创建当前月份的分区
-- SELECT create_monthly_partition('object_metadata', date_trunc('month', CURRENT_DATE)::DATE);

-- 插入示例数据 (仅开发环境)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_database WHERE datname = current_database() AND datname LIKE '%dev%') THEN
        INSERT INTO object_metadata (bucket, key, size, etag, content_type, metadata, tags) VALUES
        ('test-bucket', 'documents/readme.txt', 1024, 'abcd1234', 'text/plain', 
         '{"author": "admin", "project": "mocks3"}', '{"category": "documentation"}'),
        ('test-bucket', 'images/logo.png', 2048, 'efgh5678', 'image/png',
         '{"width": 256, "height": 256}', '{"type": "logo"}'),
        ('data-bucket', 'exports/data.csv', 4096, 'ijkl9012', 'text/csv',
         '{"rows": 1000, "encoding": "utf-8"}', '{"source": "analytics"}'),
        ('backup-bucket', 'backups/db_20240101.sql', 10485760, 'mnop3456', 'application/sql',
         '{"compressed": true, "database": "mocks3"}', '{"type": "backup", "date": "2024-01-01"}');
        
        RAISE NOTICE 'Sample data inserted for development environment';
    END IF;
END $$;

-- 创建统计信息收集函数
CREATE OR REPLACE FUNCTION collect_table_stats()
RETURNS TABLE(
    table_name TEXT,
    row_count BIGINT,
    total_size TEXT,
    index_size TEXT,
    last_vacuum TIMESTAMP WITH TIME ZONE,
    last_analyze TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        schemaname||'.'||tablename as table_name,
        n_tup_ins + n_tup_upd - n_tup_del as row_count,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
        pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as index_size,
        last_vacuum,
        last_analyze
    FROM pg_stat_user_tables 
    WHERE schemaname = 'public';
END;
$$ LANGUAGE plpgsql;

-- 设置数据库配置优化
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
ALTER SYSTEM SET track_activity_query_size = 2048;
ALTER SYSTEM SET log_min_duration_statement = 1000;
ALTER SYSTEM SET log_checkpoints = on;
ALTER SYSTEM SET log_connections = on;
ALTER SYSTEM SET log_disconnections = on;
ALTER SYSTEM SET log_lock_waits = on;

-- 提交配置更改
SELECT pg_reload_conf();

-- 创建监控用户 (仅用于指标收集)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_user WHERE usename = 'monitoring') THEN
        CREATE USER monitoring WITH PASSWORD 'monitoring_password';
        GRANT CONNECT ON DATABASE mocks3 TO monitoring;
        GRANT USAGE ON SCHEMA public TO monitoring;
        GRANT SELECT ON ALL TABLES IN SCHEMA public TO monitoring;
        GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO monitoring;
        
        -- 授权访问系统视图
        GRANT SELECT ON pg_stat_database TO monitoring;
        GRANT SELECT ON pg_stat_user_tables TO monitoring;
        GRANT SELECT ON pg_stat_user_indexes TO monitoring;
        GRANT SELECT ON pg_statio_user_tables TO monitoring;
        
        RAISE NOTICE 'Monitoring user created';
    END IF;
END $$;

-- 输出初始化完成信息
DO $$
BEGIN
    RAISE NOTICE 'MockS3 database schema initialized successfully';
    RAISE NOTICE 'Tables created: object_metadata';
    RAISE NOTICE 'Views created: bucket_statistics';
    RAISE NOTICE 'Functions created: update_updated_at, create_monthly_partition, collect_table_stats';
    RAISE NOTICE 'Extensions enabled: uuid-ossp, pg_trgm, btree_gin';
END $$;