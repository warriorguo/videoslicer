-- PostgreSQL 数据库初始化脚本
-- 请以超级用户身份运行此脚本

-- 创建用户（如果不存在）
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'videoslicer') THEN
    CREATE ROLE videoslicer WITH LOGIN PASSWORD 'password';
  END IF;
END
$$;

-- 创建数据库（如果不存在）
SELECT 'CREATE DATABASE videoslicer OWNER videoslicer'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'videoslicer')\gexec

-- 连接到 videoslicer 数据库并创建表
\c videoslicer

-- 给用户权限
GRANT ALL PRIVILEGES ON DATABASE videoslicer TO videoslicer;
GRANT ALL PRIVILEGES ON SCHEMA public TO videoslicer;

-- 创建表
CREATE TABLE IF NOT EXISTS video_tasks (
    task_id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    stage VARCHAR(30) NOT NULL DEFAULT '',
    progress_percent DECIMAL(5,2) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    source_path TEXT NOT NULL,
    source_size BIGINT NOT NULL DEFAULT 0,
    
    result_path TEXT,
    result_size BIGINT NOT NULL DEFAULT 0,
    manifest_path TEXT,
    
    segment_sec INTEGER NOT NULL DEFAULT 8,
    frame_interval_sec DOUBLE PRECISION NOT NULL DEFAULT 2,
    frame_format VARCHAR(10) NOT NULL DEFAULT 'jpg',
    bg_color VARCHAR(20) NOT NULL DEFAULT 'black',
    zip_format VARCHAR(10) NOT NULL DEFAULT 'zip',
    
    error_code VARCHAR(50),
    error_message TEXT,
    
    lease_owner VARCHAR(100),
    lease_expires_at TIMESTAMP,
    
    callback_url TEXT
);

CREATE INDEX IF NOT EXISTS idx_video_tasks_status ON video_tasks(status);
CREATE INDEX IF NOT EXISTS idx_video_tasks_created_at ON video_tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_video_tasks_lease ON video_tasks(status, lease_expires_at) WHERE status = 'processing';

-- 创建更新时间戳的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 创建触发器
DROP TRIGGER IF EXISTS update_video_tasks_updated_at ON video_tasks;
CREATE TRIGGER update_video_tasks_updated_at 
    BEFORE UPDATE ON video_tasks 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- 给表权限
GRANT ALL PRIVILEGES ON TABLE video_tasks TO videoslicer;

-- 显示创建结果
\echo '数据库初始化完成!'
\echo '用户: videoslicer'
\echo '数据库: videoslicer' 
\echo '表: video_tasks'