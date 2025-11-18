package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

type VideoProcessor interface {
	UploadFile(ctx context.Context, objectName, filePath, contentType string) error
	GetFileURL(objectName string) string
}

type videoProcessor struct {
	BucketName  string
	logger      *slog.Logger
	minioClient *minio.Client
}

func NewVideoProcessor(endpoint, accessKey, secretKey, bucket string) VideoProcessor {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // set true if using HTTPS
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, errBucket := client.BucketExists(ctx, bucket)
	if errBucket != nil {
		log.Fatalln(errBucket)
	}
	if !exists {
		client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	}

	return &videoProcessor{
		minioClient: client,
		BucketName:  bucket,
		logger:      slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (vp *videoProcessor) UploadFile(ctx context.Context, objectName, filePath, contentType string) error {
	_, err := vp.minioClient.FPutObject(ctx, vp.BucketName, objectName, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		vp.logger.Error("UploadFile error", "error", err)
		return err
	}
	return nil
}

func (vp *videoProcessor) GetFileURL(objectName string) string {
	// presigned URL, expires in 1 hour
	ctx := context.Background()
	url, err := vp.minioClient.PresignedGetObject(ctx, vp.BucketName, objectName, 3600, nil)
	if err != nil {
		vp.logger.Error("GetFileURL error", "error", err)
		return ""
	}
	return url.String()
}

func Transcode(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath,
		"-c:v", "libx264", "-preset", "fast", "-crf", "23",
		"-c:a", "aac", outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %s", out)
	}
	return nil
}

func GenerateThumbnail(inputPath, outputPath string, seconds int) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-ss", fmt.Sprintf("00:00:%02d", seconds),
		"-vframes", "1", outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("thumbnail error: %s", out)
	}
	return nil
}

func ConvertToHLS(inputPath, outputDir string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath,
		"-profile:v", "baseline", "-level", "3.0",
		"-start_number", "0", "-hls_time", "10",
		"-hls_list_size", "0", "-f", "hls", fmt.Sprintf("%s/index.m3u8", outputDir))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("HLS error: %s", out)
	}
	return nil
}

type VideoJob struct {
	ID        string `json:"id"`
	FilePath  string `json:"file_path"`
	OutputDir string `json:"output_dir"`
}

type Queue struct {
	Client *redis.Client
	Key    string
}

func NewQueue(addr string) *Queue {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &Queue{Client: rdb, Key: "video_jobs"}
}

func (q *Queue) Enqueue(job VideoJob) error {
	ctx := context.Background()
	data, _ := json.Marshal(job)
	return q.Client.LPush(ctx, q.Key, data).Err()
}

func (q *Queue) Dequeue() (*VideoJob, error) {
	ctx := context.Background()
	val, err := q.Client.BRPop(ctx, 0, q.Key).Result()
	if err != nil {
		return nil, err
	}
	var job VideoJob
	_ = json.Unmarshal([]byte(val[1]), &job)
	return &job, nil
}

// func processJob(job *queue.VideoJob) {
// 	fmt.Println("Processing job:", job.ID)
// 	outputFile := fmt.Sprintf("%s/%s.mp4", job.OutputDir, job.ID)
// 	thumbFile := fmt.Sprintf("%s/%s.jpg", job.OutputDir, job.ID)

// 	os.MkdirAll(job.OutputDir, 0755)
// 	if err := ffmpeg.Transcode(job.FilePath, outputFile); err != nil {
// 		log.Println("Transcode error:", err)
// 		return
// 	}
// 	if err := ffmpeg.GenerateThumbnail(outputFile, thumbFile, 5); err != nil {
// 		log.Println("Thumbnail error:", err)
// 		return
// 	}
// 	log.Println("Job completed:", job.ID)
// }
