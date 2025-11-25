-- Add columns for HLS and thumbnail metadata
ALTER TABLE video_variants
ADD COLUMN hls_playlist_key VARCHAR(255),
ADD COLUMN thumbnail_key VARCHAR(255),
ADD COLUMN width INTEGER,
ADD COLUMN height INTEGER,
ADD COLUMN bitrate_kbps INTEGER;
