package services

import (
	"context"
	"log/slog"
	"time"
	"video-processing/database/db"
	"video-processing/models"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type VideoProcessor interface {
	CreateBucket(ctx context.Context, bucketName string) error
	ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
	Upload(ctx context.Context, userID uuid.UUID, req models.UploadVideoRequest) (string, error)
}

type videoProcessor struct {
	urlExpiry   time.Duration
	logger      *slog.Logger
	minioClient *minio.Client
	db          *db.Queries
	streamer    Streamer
}

func NewVideoProcessor(logger *slog.Logger, minioClient *minio.Client, db *db.Queries, streamer Streamer, urlExpiry time.Duration) VideoProcessor {
	return &videoProcessor{
		urlExpiry:   urlExpiry,
		logger:      logger,
		minioClient: minioClient,
		db:          db,
		streamer:    streamer,
	}
}

func (vp *videoProcessor) CreateBucket(ctx context.Context, bucketName string) error {
	return vp.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
}
func (vp *videoProcessor) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return vp.minioClient.ListBuckets(ctx)
}
func (vp *videoProcessor) Upload(ctx context.Context, userID uuid.UUID, req models.UploadVideoRequest) (string, error) {
	if err := req.Validate(); err != nil {
		vp.logger.Error("Invalid Input", "error", err)
		return "", err
	}
	for _, fileHeader := range req.Videos {
		file, err := fileHeader.Open()
		if err != nil {
			vp.logger.Error("Upload error", "error", err)
			return "", err
		}
		defer file.Close()

		buckets, err := vp.ListBuckets(ctx)
		if err != nil {
			vp.logger.Error("Upload error", "error", err)
			return "", err
		}
		bucketExist := false
		for _, bucket := range buckets {
			if bucket.Name == userID.String() {
				bucketExist = true
			}
		}
		if !bucketExist {
			err := vp.CreateBucket(ctx, userID.String())
			if err != nil {
				vp.logger.Error("Failed to create bucket", "error", err)
				return "", err
			}
		}
		_, err = vp.minioClient.PutObject(ctx, userID.String(), fileHeader.Filename, file, fileHeader.Size, minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		})
		if err != nil {
			vp.logger.Error("Upload error", "error", err)
			return "", err
		}
		// generate url
		url, err := vp.getVideoURL(userID.String(), fileHeader.Filename, vp.urlExpiry)
		if err != nil {
			vp.logger.Error("Upload error", "error", err)
			return "", err
		}
		vp.logger.Info("generated url:", "url", url)
		// save video metadata to database
		_, err = vp.db.CreateVideo(ctx, db.CreateVideoParams{
			UserID:        userID,
			Filename:      fileHeader.Filename,
			Title:         req.Title,
			Description:   req.Description,
			Bucket:        userID.String(),
			Key:           fileHeader.Filename,
			FileSizeBytes: fileHeader.Size,
			ContentType:   fileHeader.Header.Get("Content-Type"),
			Url:           url,
		})
		if err != nil {
			vp.logger.Error("Upload error", "error", err)
			return "", err
		}
		err = vp.streamer.Stream(ctx, map[string]interface{}{
			"bucket": userID.String(),
			"key":    fileHeader.Filename,
		})
		if err != nil {
			vp.logger.Error("Upload error", "error", err)
			return "", err
		}
	}
	return "", nil
}

func (vp *videoProcessor) getVideoURL(bucketName, objectName string, expiry time.Duration) (string, error) {
	// presigned URL, expires in 1 hour
	ctx := context.Background()
	url, err := vp.minioClient.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		vp.logger.Error("GetVideoURL error", "error", err)
		return "", err
	}
	return url.String(), nil
}
