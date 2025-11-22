package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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
	err := vp.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return models.Error{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
			Params:  fmt.Sprintf("bucketName: %v", bucketName),
			Err:     fmt.Errorf("failed to create bucket: %w", err),
		}
	}
	return nil
}
func (vp *videoProcessor) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	buckets, err := vp.minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, models.Error{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
			Err:     fmt.Errorf("failed to list buckets: %w", err),
		}
	}
	return buckets, nil
}
func (vp *videoProcessor) Upload(ctx context.Context, userID uuid.UUID, req models.UploadVideoRequest) (string, error) {
	paramsInString := fmt.Sprintf("userID: %v, req: %v", userID, req)
	if err := req.Validate(); err != nil {
		return "", models.Error{
			Code:    http.StatusBadRequest,
			Message: "invalid input data",
			Params:  paramsInString,
			Err:     err,
		}
	}
	for _, fileHeader := range req.Videos {
		file, err := fileHeader.Open()
		if err != nil {
			return "", models.Error{
				Code:    http.StatusInternalServerError,
				Message: "internal server error",
				Params:  paramsInString,
				Err:     fmt.Errorf("failed to open file: %w", err),
			}
		}
		defer file.Close()

		buckets, err := vp.ListBuckets(ctx)
		if err != nil {
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
				return "", err
			}
		}
		_, err = vp.minioClient.PutObject(ctx, userID.String(), fileHeader.Filename, file, fileHeader.Size, minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		})
		if err != nil {
			return "", models.Error{
				Code:    http.StatusInternalServerError,
				Message: "internal server error",
				Params:  paramsInString,
				Err:     fmt.Errorf("failed to upload file: %w", err),
			}
		}
		// generate url
		url, err := vp.getVideoURL(userID.String(), fileHeader.Filename, vp.urlExpiry)
		if err != nil {
			return "", err
		}
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
			return "", models.Error{
				Code:    http.StatusInternalServerError,
				Message: "internal server error",
				Params:  paramsInString,
				Err:     fmt.Errorf("failed to save video metadata: %w", err),
			}
		}
		err = vp.streamer.Stream(ctx, map[string]interface{}{
			"bucket": userID.String(),
			"key":    fileHeader.Filename,
		})
		if err != nil {
			return "", models.Error{
				Code:    http.StatusInternalServerError,
				Message: "internal server error",
				Params:  paramsInString,
				Err:     fmt.Errorf("failed to stream video: %w", err),
			}
		}
	}
	return "", nil
}

func (vp *videoProcessor) getVideoURL(bucketName, objectName string, expiry time.Duration) (string, error) {
	// presigned URL, expires in 1 hour
	ctx := context.Background()
	url, err := vp.minioClient.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", models.Error{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
			Params:  fmt.Sprintf("bucketName: %v, objectName: %v, expiry: %v", bucketName, objectName, expiry),
			Err:     fmt.Errorf("failed to generate video url: %w", err),
		}
	}
	return url.String(), nil
}
