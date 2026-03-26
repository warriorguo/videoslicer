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
    
    segment_sec DOUBLE PRECISION NOT NULL DEFAULT 8,
    frame_interval_sec DOUBLE PRECISION NOT NULL DEFAULT 2,
    frame_format VARCHAR(10) NOT NULL DEFAULT 'jpg',
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

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_video_tasks_updated_at 
    BEFORE UPDATE ON video_tasks 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();