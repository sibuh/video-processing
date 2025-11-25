-- Add a unique constraint on (video_id, variant_name) to support ON CONFLICT
ALTER TABLE video_variants
ADD CONSTRAINT video_variants_video_id_variant_name_key UNIQUE (video_id, variant_name);
