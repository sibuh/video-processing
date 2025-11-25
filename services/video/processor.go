package video

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"video-processing/database/db"
	"video-processing/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
)

/*
This program:
1) Downloads source video from MinIO to a local temp file.
2) For each target quality:
   - Transcodes the source into an MP4 at that resolution/bitrate.
   - Generates an HLS playlist + segments from the MP4.
   - Generates a thumbnail image for that quality.
   - Uploads the MP4, HLS files (.m3u8 + .ts) and thumbnail back to MinIO under a results prefix.

Usage:
  go run main.go <bucket> <source-object> <results-prefix>

Example:
  go run main.go my-bucket uploads/input.mp4 processed/input-uuid/
*/

// Variant represents a video variant configuration
type Variant struct {
	Name    string // logical name like "1080p"
	Width   int
	Height  int
	Bitrate string // e.g., "4000k"
}

// ProcessingTask represents a single video processing task
type ProcessingTask struct {
	Variant    Variant
	WorkDir    string
	SourcePath string
	DestPrefix string
	Bucket     string
	VideoID    string
}

// UploadTask represents a file to be uploaded to MinIO
type UploadTask struct {
	SourcePath  string
	ObjectKey   string
	ContentType string
	Bucket      string
}

// ProcessingResult represents the result of processing a single variant
type ProcessingResult struct {
	Variant  Variant
	VideoID  string
	WorkDir  string
	Success  bool
	Error    error
	Files    []UploadTask
	Metadata db.SaveProcessedVideoMetadataParams
}

var variants = []Variant{
	{Name: "1080p", Width: 1920, Height: 1080, Bitrate: "4000k"},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: "2000k"},
	{Name: "480p", Width: 854, Height: 480, Bitrate: "1000k"},
	{Name: "360p", Width: 640, Height: 360, Bitrate: "500k"},
	{Name: "240p", Width: 426, Height: 240, Bitrate: "250k"},
	{Name: "144p", Width: 256, Height: 144, Bitrate: "100k"},
}

// processVariant processes a single video variant
func (rc *redisConsumer) processVariant(ctx context.Context, task ProcessingTask, resultChan chan<- ProcessingResult, wg *sync.WaitGroup) {
	defer wg.Done()

	result := ProcessingResult{
		Variant: task.Variant,
		VideoID: task.VideoID,
		WorkDir: task.WorkDir,
		Success: true,
	}

	// Create variant-specific directory
	varDir := filepath.Join(task.WorkDir, task.Variant.Name)
	if err := os.MkdirAll(varDir, 0o755); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to create variant directory: %w", err)
		resultChan <- result
		return
	}

	// 1. Transcode to MP4
	mp4Path := filepath.Join(varDir, fmt.Sprintf("%s.mp4", task.Variant.Name))
	if err := transcodeToMP4(ctx, task.SourcePath, mp4Path, task.Variant); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("transcode failed: %w", err)
		resultChan <- result
		return
	}

	// 2. Generate HLS in the variant directory (same level as thumbnail)
	hlsDir := varDir // Store HLS files directly in the variant directory
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to create variant directory for HLS: %w", err)
		resultChan <- result
		return
	}

	if err := generateHLS(ctx, mp4Path, hlsDir); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("HLS generation failed: %w", err)
		resultChan <- result
		return
	}

	// 3. Generate thumbnail
	thumbPath := filepath.Join(varDir, fmt.Sprintf("%s-thumb.jpg", task.Variant.Name))
	if err := generateThumbnail(ctx, mp4Path, thumbPath, 5); err != nil {
		rc.logger.Warn("thumbnail generation failed", "error", err, "variant", task.Variant.Name)
		// Don't fail the whole process if thumbnail fails
	}

	// Prepare upload tasks
	destPrefix := filepath.Join(task.DestPrefix, task.Variant.Name)
	destPrefix = filepath.ToSlash(destPrefix) // Normalize to forward slashes

	// Add MP4 file to upload tasks
	result.Files = append(result.Files, UploadTask{
		SourcePath:  mp4Path,
		ObjectKey:   filepath.ToSlash(filepath.Join(destPrefix, fmt.Sprintf("%s.mp4", task.Variant.Name))),
		ContentType: "video/mp4",
		Bucket:      task.Bucket,
	})

	// Add thumbnail to upload tasks
	if _, err := os.Stat(thumbPath); err == nil {
		result.Files = append(result.Files, UploadTask{
			SourcePath:  thumbPath,
			ObjectKey:   filepath.ToSlash(filepath.Join(destPrefix, fmt.Sprintf("%s-thumb.jpg", task.Variant.Name))),
			ContentType: "image/jpeg",
			Bucket:      task.Bucket,
		})
	}

	// Add HLS files to upload tasks (now at the same level as other files)
	hlsFiles, err := filepath.Glob(filepath.Join(hlsDir, "*"))
	if err != nil {
		rc.logger.Warn("failed to list HLS files", "error", err, "variant", task.Variant.Name)
	} else {
		for _, hlsFile := range hlsFiles {
			// Skip the MP4 and thumbnail files that are already added
			if strings.HasSuffix(hlsFile, ".mp4") || strings.HasSuffix(hlsFile, "-thumb.jpg") {
				continue
			}
			ext := filepath.Ext(hlsFile)
			contentType := mimeTypeByExt(ext)
			// Get just the filename to maintain flat structure
			_, fileName := filepath.Split(hlsFile)
			result.Files = append(result.Files, UploadTask{
				SourcePath:  hlsFile,
				ObjectKey:   filepath.ToSlash(filepath.Join(destPrefix, fileName)),
				ContentType: contentType,
				Bucket:      task.Bucket,
			})
		}
	}

	// Prepare metadata for database
	bitrateStr := strings.TrimSuffix(task.Variant.Bitrate, "k")
	bitrate, _ := strconv.ParseInt(bitrateStr, 10, 32)

	videoUUID, err := uuid.Parse(task.VideoID)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("invalid video ID: %w", err)
		resultChan <- result
		return
	}

	// Prepare metadata with updated HLS path (now at the same level)
	hlsPlaylistPath := filepath.ToSlash(filepath.Join(destPrefix, "index.m3u8"))
	thumbnailPath := filepath.ToSlash(filepath.Join(destPrefix, fmt.Sprintf("%s-thumb.jpg", task.Variant.Name)))

	result.Metadata = db.SaveProcessedVideoMetadataParams{
		VideoID:     videoUUID,
		VariantName: task.Variant.Name,
		Bucket:      task.Bucket,
		Key:         filepath.ToSlash(filepath.Join(destPrefix, fmt.Sprintf("%s.mp4", task.Variant.Name))),
		ContentType: "video/mp4",
		HlsPlaylistKey: pgtype.Text{
			String: hlsPlaylistPath,
			Valid:  true,
		},
		ThumbnailKey: pgtype.Text{
			String: thumbnailPath,
			Valid:  true,
		},
		Width: pgtype.Int4{
			Int32: int32(task.Variant.Width),
			Valid: true,
		},
		Height: pgtype.Int4{
			Int32: int32(task.Variant.Height),
			Valid: true,
		},
		BitrateKbps: pgtype.Int4{
			Int32: int32(bitrate),
			Valid: true,
		},
	}

	rc.logger.Info("prepared variant metadata", 
		"variant", task.Variant.Name,
		"hls_playlist", hlsPlaylistPath,
		"thumbnail", thumbnailPath,
	)

	resultChan <- result
}

// uploadWorker processes upload tasks from the upload channel
func (rc *redisConsumer) uploadWorker(ctx context.Context, uploadCh <-chan UploadTask, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range uploadCh {
		file, err := os.Open(task.SourcePath)
		if err != nil {
			rc.logger.Error("failed to open file for upload", "path", task.SourcePath, "error", err)
			continue
		}

		_, err = rc.mc.PutObject(ctx, task.Bucket, task.ObjectKey, file, -1, minio.PutObjectOptions{
			ContentType: task.ContentType,
		})
		file.Close()

		if err != nil {
			rc.logger.Error("upload failed", "object", task.ObjectKey, "error", err)
		} else {
			rc.logger.Info("upload successful", "object", task.ObjectKey)
		}
	}
}

// saveVariantMetadata saves variant metadata to the database
func (rc *redisConsumer) saveVariantMetadata(ctx context.Context, result ProcessingResult) {
	if !result.Success || result.Error != nil {
		rc.logger.Error("skipping metadata save for failed variant",
			"variant", result.Variant.Name,
			"error", result.Error)
		return
	}

	_, err := rc.db.SaveProcessedVideoMetadata(ctx, result.Metadata)
	if err != nil {
		rc.logger.Error("failed to save variant metadata",
			"variant", result.Variant.Name,
			"error", err)
	} else {
		rc.logger.Info("saved variant metadata",
			"variant", result.Variant.Name,
			"videoID", result.VideoID)
	}
}

func (rc *redisConsumer) ProcessVideo(ctx context.Context, values map[string]interface{}) error {
	// Extract input parameters
	bucket := values["bucket"].(string)
	sourceObj := values["key"].(string)
	videoID := values["video_id"].(string)
	resultsPrefix := fmt.Sprintf("processed/%s", uuid.New().String())

	// Create a temp working dir for the job; cleaned up on exit
	workDir, err := os.MkdirTemp("", "video-job-*")
	if err != nil {
		return models.Error{
			Code:        http.StatusInternalServerError,
			Message:     "internal server error",
			Description: "failed to create working directory",
			Params:      fmt.Sprintf("bucket: %v, sourceObj: %v", bucket, sourceObj),
			Err:         fmt.Errorf("failed to create temp dir: %w", err),
		}
	}
	defer os.RemoveAll(workDir)

	rc.logger.Info("starting video processing",
		"videoID", videoID,
		"source", sourceObj,
		"workDir", workDir)

	// Step 1: Download source video from MinIO
	localSourcePath := filepath.Join(workDir, "source"+filepath.Ext(sourceObj))
	rc.logger.Info("downloading source video",
		"source", fmt.Sprintf("s3://%s/%s", bucket, sourceObj),
		"destination", localSourcePath)

	if err := downloadFromMinio(ctx, rc.mc, bucket, sourceObj, localSourcePath); err != nil {
		return models.Error{
			Code:        http.StatusInternalServerError,
			Message:     "download failed",
			Description: "failed to download source video",
			Params:      fmt.Sprintf("bucket: %v, source: %v", bucket, sourceObj),
			Err:         err,
		}
	}

	rc.logger.Info("source download complete", "path", localSourcePath)

	// Create channels for the pipeline
	resultCh := make(chan ProcessingResult, len(variants))
	uploadCh := make(chan UploadTask, 100) // Buffer some upload tasks

	// Start the upload workers
	var uploadWg sync.WaitGroup
	numUploadWorkers := 3 // Number of concurrent uploads
	for i := 0; i < numUploadWorkers; i++ {
		uploadWg.Add(1)
		go rc.uploadWorker(ctx, uploadCh, &uploadWg)
	}

	// Start a goroutine to process results and queue uploads
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for result := range resultCh {
			if result.Success && len(result.Files) > 0 {
				// Queue uploads for this variant
				for _, file := range result.Files {
					select {
					case <-ctx.Done():
						rc.logger.Warn("context done, stopping upload queue", "variant", result.Variant.Name)
						return
					case uploadCh <- file:
						// File queued for upload
					}
				}
				// Save metadata to database
				rc.saveVariantMetadata(ctx, result)
			} else if !result.Success {
				rc.logger.Error("variant processing failed",
					"variant", result.Variant.Name,
					"error", result.Error)
			}
		}
	}()

	// Process each variant in parallel
	var processWg sync.WaitGroup
	for _, variant := range variants {
		processWg.Add(1)
		task := ProcessingTask{
			Variant:    variant,
			WorkDir:    workDir,
			SourcePath: localSourcePath,
			DestPrefix: resultsPrefix,
			Bucket:     bucket,
			VideoID:    videoID,
		}
		go func(t ProcessingTask) {
			rc.processVariant(ctx, t, resultCh, &processWg)
		}(task)
	}

	// Wait for all variants to be processed
	processWg.Wait()
	close(resultCh) // This will signal the result processor to exit

	// Wait for all processing to complete
	resultWg.Wait()

	rc.logger.Debug("all variants processed, waiting for uploads to complete", "videoID", videoID)

	// Close upload channel and wait for uploads to complete
	close(uploadCh)
	uploadWg.Wait()

	rc.logger.Info("all processing and uploads completed", "videoID", videoID)

	// Clean up working directory
	if err := os.RemoveAll(workDir); err != nil {
		rc.logger.Error("failed to clean up working directory", "error", err, "workDir", workDir)
	} else {
		rc.logger.Debug("cleaned up working directory", "workDir", workDir)
	}

	rc.logger.Info("video processing completed", "videoID", videoID)
	return nil
}

// ...
// downloadFromMinio downloads an object to a local file path using FGetObject (server-side streaming to disk)
func downloadFromMinio(ctx context.Context, client *minio.Client, bucket, object, destPath string) error {
	// FGetObject will stream object directly to the destination path on disk.
	// This avoids loading the whole object into memory.
	opts := minio.GetObjectOptions{}
	if err := client.FGetObject(ctx, bucket, object, destPath, opts); err != nil {
		return fmt.Errorf("FGetObject error: %w", err)
	}
	return nil
}

// uploadDirToMinio walks a local directory and uploads files preserving relative paths under destPrefix.
// Example: uploadDirToMinio(..., "processed/uuid/1080p", "/tmp/job/1080p")
// will upload "/tmp/job/1080p/index.m3u8" -> "processed/uuid/1080p/index.m3u8" in bucket
func (rc *redisConsumer) uploadDirToMinio(ctx context.Context, client *minio.Client, bucket, destPrefix, dir string, videoID uuid.UUID) error {
	// Walk local directory
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// skip directories
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// objectName should use forward slashes
		objectName := filepath.ToSlash(filepath.Join(destPrefix, rel))

		// choose content type by extension (simple)
		contentType := mimeTypeByExt(filepath.Ext(path))

		// FPutObject uploads local file from disk; efficient and uses multipart when large
		_, err = client.FPutObject(ctx, bucket, objectName, path, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("FPutObject %s -> %s: %w", path, objectName, err)
		}
		log.Printf("uploaded %s -> s3://%s/%s", path, bucket, objectName)
		return nil
	})
}

/* ----------------------------
   FFmpeg helpers
   ---------------------------- */

// transcodeToMP4 transcodes input -> output MP4 using x264 + aac with scaling and bitrate.
// This writes to a local output file (mp4Path).
func transcodeToMP4(ctx context.Context, inputPath, mp4Path string, v Variant) error {
	// ffmpeg command:
	// ffmpeg -y -i input -vf scale=WIDTH:HEIGHT -c:v libx264 -b:v BITRATE -preset fast -c:a aac -ac 2 -ar 44100 output.mp4
	args := []string{
		"-y", // overwrite output if exists
		"-nostdin",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=%d:%d", v.Width, v.Height),
		"-c:v", "libx264",
		"-b:v", v.Bitrate,
		"-preset", "fast",
		"-c:a", "aac",
		"-ac", "2",
		"-ar", "44100",
		mp4Path,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	// Optional: capture combined output for logging
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg transcode error: %v, output: %s", err, string(out))
	}
	return nil
}

// generateHLS creates HLS playlist and .ts segments from an mp4.
// It outputs index.m3u8 and segment_###.ts files into outDir.
func generateHLS(ctx context.Context, mp4Path, outDir string) error {
	// ffmpeg command:
	// ffmpeg -y -i input.mp4 -c:v libx264 -c:a aac -vf "format=yuv420p" -hls_time 6 -hls_playlist_type vod \
	//   -hls_segment_filename "outDir/segment_%03d.ts" outDir/index.m3u8
	playlistPath := filepath.Join(outDir, "index.m3u8")
	segmentPattern := filepath.Join(outDir, "segment_%03d.ts")

	args := []string{
		"-y",
		"-nostdin",
		"-i", mp4Path,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-vf", "format=yuv420p",
		"-hls_time", "6", // segment length in seconds
		"-hls_playlist_type", "vod", // VOD playlist (complete)
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg hls error: %v, output: %s", err, string(out))
	}
	return nil
}

// generateThumbnail captures a single frame at `atSecond` from input and writes to outImagePath (jpeg).
func generateThumbnail(ctx context.Context, inputPath, outImagePath string, atSecond int) error {
	// ffmpeg -y -i input -ss 00:00:05 -vframes 1 -q:v 2 out.jpg
	ss := fmt.Sprintf("00:00:%02d", atSecond)
	args := []string{
		"-y",
		"-nostdin",
		"-i", inputPath,
		"-ss", ss,
		"-vframes", "1",
		"-q:v", "2", // quality (lower is better)
		outImagePath,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg thumb error: %v, output: %s", err, string(out))
	}
	return nil
}

/* ----------------------------
   Utilities
   ---------------------------- */

// mimeTypeByExt returns a simple content-type by file extension.
// This is minimal — for production use a proper MIME lookup.
func mimeTypeByExt(ext string) string {
	switch ext {
	case ".m3u8":
		return "application/vnd.apple.mpegurl"
	case ".ts":
		return "video/mp2t"
	case ".mp4":
		return "video/mp4"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
