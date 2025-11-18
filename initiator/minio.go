package initiator

import (
	"log/slog"
	"video-processing/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func InitMinio(logger *slog.Logger, config models.Config) *minio.Client {

	client, err := minio.New(config.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Minio.AccessKey, config.Minio.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		logger.Error("❌ MinIO init error", "error", err)
	}

	logger.Info("✅ MinIO connected successfully")
	return client

}
