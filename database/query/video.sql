-- name: CreateVideo :one
INSERT INTO videos (
    user_id,     
    title,
    description,
    bucket,
    key,
    file_size_bytes,
    content_type
) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetVideo :one
SELECT * FROM videos WHERE id = $1;

-- name: ListVideos :many
SELECT * FROM videos ORDER BY created_at DESC;

-- name: UpdateVideo :one
UPDATE videos
SET 
    title = COALESCE(NULLIF($1, ''), title),
    description = COALESCE(NULLIF($2, ''), description),
    bucket = COALESCE(NULLIF($3, ''), bucket),
    key = COALESCE(NULLIF($4, ''), key),
    file_size_bytes = COALESCE(NULLIF($5, 0), file_size_bytes),
    content_type = COALESCE(NULLIF($6, ''), content_type)
WHERE id = $1 RETURNING *;

-- name: DeleteVideo :one
DELETE FROM videos WHERE id = $1 RETURNING *;

-- name: UpdateVideoStatus :one
UPDATE videos
SET 
    status = $1
WHERE id = $2 RETURNING *;

-- name: SaveProcessedVideoMetadata :one
INSERT INTO video_variants (
    video_id,
    variant_name,
    bucket,
    key,
    content_type,
    hls_playlist_key,
    thumbnail_key,
    width,
    height,
    bitrate_kbps
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
ON CONFLICT (video_id, variant_name) 
DO UPDATE SET 
    bucket = EXCLUDED.bucket,
    key = EXCLUDED.key,
    content_type = EXCLUDED.content_type,
    hls_playlist_key = EXCLUDED.hls_playlist_key,
    thumbnail_key = EXCLUDED.thumbnail_key,
    width = EXCLUDED.width,
    height = EXCLUDED.height,
    bitrate_kbps = EXCLUDED.bitrate_kbps
RETURNING *;