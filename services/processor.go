package services

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

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

type Variant struct {
	Name    string // logical name like "1080p"
	Width   int
	Height  int
	Bitrate string // e.g., "4000k"
}

var variants = []Variant{
	{Name: "1080p", Width: 1920, Height: 1080, Bitrate: "4000k"},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: "2000k"},
	{Name: "480p", Width: 854, Height: 480, Bitrate: "1000k"},
	{Name: "360p", Width: 640, Height: 360, Bitrate: "500k"},
	{Name: "240p", Width: 426, Height: 240, Bitrate: "250k"},
	{Name: "144p", Width: 256, Height: 144, Bitrate: "100k"},
}

func Process(ctx context.Context, logger *slog.Logger, bucket, sourceObj, resultsPrefix string, client *minio.Client) {

	// Create a temp working dir for the job; cleaned up on exit.
	workDir, err := os.MkdirTemp("", "video-job-*")
	if err != nil {
		logger.Error("failed create temp dir", "error", err)
	}
	defer os.RemoveAll(workDir) // cleanup everything at the end
	logger.Info("workdir", "workDir", workDir)

	// Step 1: download source video from MinIO to local file
	localSourcePath := filepath.Join(workDir, "source"+filepath.Ext(sourceObj))
	logger.Info("downloading s3://%s/%s -> %s", "bucket", bucket, "sourceObj", sourceObj, "localSourcePath", localSourcePath)
	if err := downloadFromMinio(ctx, client, bucket, sourceObj, localSourcePath); err != nil {
		logger.Error("download failed", "error", err)
	}
	logger.Info("download complete")

	// For each variant: transcode -> generate HLS -> thumbnail -> upload
	for _, v := range variants {
		logger.Info("processing variant", "name", v.Name, "width", v.Width, "height", v.Height, "bitrate", v.Bitrate)

		// create variant output dir inside workDir
		varDir := filepath.Join(workDir, v.Name)
		if err := os.MkdirAll(varDir, 0o755); err != nil {
			logger.Error("mkdir", "error", err)
		}

		// 2.a Transcode to MP4 (local)
		mp4Path := filepath.Join(varDir, fmt.Sprintf("%s.mp4", v.Name))
		if err := transcodeToMP4(ctx, localSourcePath, mp4Path, v); err != nil {
			logger.Error("transcode failed", "error", err)
		}
		logger.Info("transcoded mp4", "mp4Path", mp4Path)

		// 2.b Generate HLS (creates index.m3u8 and segment files in varDir/hls/)
		hlsDir := filepath.Join(varDir, "hls")
		if err := os.MkdirAll(hlsDir, 0o755); err != nil {
			logger.Error("mkdir hls", "error", err)
		}
		if err := generateHLS(ctx, mp4Path, hlsDir); err != nil {
			logger.Error("hls generation failed", "error", err)
		}
		logger.Info("hls generated at", "hlsDir", hlsDir)

		// 2.c Generate thumbnail (we capture at 5 seconds)
		thumbPath := filepath.Join(varDir, fmt.Sprintf("%s-thumb.jpg", v.Name))
		if err := generateThumbnail(ctx, mp4Path, thumbPath, 5); err != nil {
			logger.Error("thumbnail failed", "error", err)
		}
		logger.Info("thumbnail generated", "thumbPath", thumbPath)

		// 2.d Upload mp4 + hls files + thumbnail to MinIO under resultsPrefix/<variant>/
		destPrefix := filepath.Join(resultsPrefix, v.Name) // e.g., processed/uuid/1080p
		// Normalize to use forward slashes (MinIO object keys use /)
		destPrefix = filepath.ToSlash(destPrefix)
		logger.Info("uploading files to s3://", "bucket", bucket, "destPrefix", destPrefix)
		if err := uploadDirToMinio(ctx, client, bucket, destPrefix, varDir); err != nil {
			logger.Error("upload failed", "error", err)
		}
		logger.Info("upload complete for variant", "name", v.Name)
	}

	log.Println("All variants processed and uploaded successfully")
}

/* ----------------------------
   MinIO: download and upload helpers
   ---------------------------- */

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
func uploadDirToMinio(ctx context.Context, client *minio.Client, bucket, destPrefix, dir string) error {
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
