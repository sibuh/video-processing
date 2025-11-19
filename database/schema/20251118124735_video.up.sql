
CREATE TABLE original_videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id), 
    status VARCHAR(20) NOT NULL DEFAULT 'queued',     
    filename VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description VARCHAR NOT NULL,
    bucket VARCHAR(255) NOT NULL,
    key VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    duration INT NOT NULL DEFAULT 0,
    width INT NOT NULL DEFAULT 0,
    height INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL  DEFAULT NOW()
);

CREATE TABLE processed_videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES original_videos(id) ON DELETE CASCADE,
    -- What is this file?
    asset_type VARCHAR(20) NOT NULL, -- e.g., '1080p', '720p', 'thumbnail', 'hls'
    -- Location
    bucket VARCHAR(255) NOT NULL,
    key VARCHAR(255) NOT NULL,    
    -- Technicals
    width INT NOT NULL DEFAULT 0,
    height INT NOT NULL DEFAULT 0,
    file_size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);