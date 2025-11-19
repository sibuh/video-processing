-- name: CreateVideo :one
INSERT INTO original_videos (
    user_id,     
    filename,
    title,
    description,
    bucket,
    key,
    file_size_bytes,
    content_type
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: GetVideo :one
SELECT * FROM original_videos WHERE id = $1;

-- name: ListVideos :many
SELECT * FROM original_videos ORDER BY created_at DESC;

-- name: UpdateVideo :one
UPDATE original_videos
SET 
    title = COALESCE(NULLIF($1, ''), title),
    description = COALESCE(NULLIF($2, ''), description),
    bucket = COALESCE(NULLIF($3, ''), bucket),
    key = COALESCE(NULLIF($4, ''), key),
    file_size_bytes = COALESCE(NULLIF($5, 0), file_size_bytes),
    content_type = COALESCE(NULLIF($6, ''), content_type),
    duration = COALESCE(NULLIF($7, 0), duration),
    width = COALESCE(NULLIF($8, 0), width),
    height = COALESCE(NULLIF($9, 0), height),
    metadata = COALESCE(NULLIF($10, '{}'), metadata)
WHERE id = $11 RETURNING *;

-- name: DeleteVideo :one
DELETE FROM original_videos WHERE id = $1 RETURNING *;
