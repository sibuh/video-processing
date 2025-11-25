-- Remove the added columns
ALTER TABLE video_variants
DROP COLUMN IF EXISTS hls_playlist_key,
DROP COLUMN IF EXISTS thumbnail_key,
DROP COLUMN IF EXISTS width,
DROP COLUMN IF EXISTS height,
DROP COLUMN IF EXISTS bitrate_kbps;
