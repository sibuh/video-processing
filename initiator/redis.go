package initiator

import (
	"context"
	"log/slog"
	"video-processing/models"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(logger *slog.Logger, config models.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Host + ":" + config.Redis.Port,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	// Ping test
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Error("❌ Redis connection error", "error", err)
	}

	logger.Info("✅ Redis connected successfully")
	return rdb
}
