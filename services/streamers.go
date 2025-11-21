package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"video-processing/models"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

type Streamer interface {
	Stream(ctx context.Context, values map[string]interface{}) error
}

type redisStreamer struct {
	streamName string
	logger     *slog.Logger
	rc         *redis.Client
}

func NewRedisStreamer(streamName string, logger *slog.Logger, rc *redis.Client) Streamer {
	return &redisStreamer{
		streamName: streamName,
		logger:     logger,
		rc:         rc,
	}
}
func (rs *redisStreamer) Stream(ctx context.Context, values map[string]interface{}) error {
	// XAddArgs appends the message to the stream
	cmd := rs.rc.XAdd(ctx, &redis.XAddArgs{
		Stream: rs.streamName,
		ID:     "*", // Let Redis generate a unique timestamp-based ID
		Values: values,
	})

	id, err := cmd.Result()
	if err != nil {
		rs.logger.Error("Failed to publish event", "error", err)
		return models.Error{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
			Params:  fmt.Sprintf("values:%v", values),
			Err:     fmt.Errorf("failed to publish event: %w", err),
		}
	}

	rs.logger.Info("Event published successfully with ID", "id", id)
	return nil
}

type Consumer interface {
	Consume(ctx context.Context) error
}

type redisConsumer struct {
	streamName   string
	groupName    string
	consumerName string
	logger       *slog.Logger
	rc           *redis.Client
	mc           *minio.Client
}

func NewRedisConsumer(streamName, groupName, consumerName string, logger *slog.Logger, rc *redis.Client, mc *minio.Client) Consumer {
	return &redisConsumer{
		streamName:   streamName,
		groupName:    groupName,
		consumerName: consumerName,
		logger:       logger,
		rc:           rc,
		mc:           mc,
	}
}
func (rc *redisConsumer) Consume(ctx context.Context) error {
	// 1. Create Consumer Group
	// 'MKSTREAM' ensures the stream exists if it's currently empty.
	// '$' means "start consuming from the moment this group is created" (ignore old data).
	// Use '0' if you want to process all historical data.
	err := rc.rc.XGroupCreateMkStream(ctx, rc.streamName, rc.groupName, "$").Err()
	if err != nil {
		// Ignore error if group already exists
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return models.Error{
				Code:    http.StatusInternalServerError,
				Message: "internal server error",
				Params:  fmt.Sprintf("streamName:%v, groupName:%v, consumerName:%v", rc.streamName, rc.groupName, rc.consumerName),
				Err:     fmt.Errorf("failed to create group: %w", err),
			}
		}
	}

	// 2. Processing Loop
	for {
		// XReadGroup reads data from the stream
		entries, err := rc.rc.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    rc.groupName,
			Consumer: rc.consumerName,
			Streams:  []string{rc.streamName, ">"}, // ">" means "give me new messages not yet delivered to anyone"
			Count:    10,                           // Batch size
			Block:    2 * time.Second,              // Long polling: block for 2s if no data
		}).Result()

		if err != nil {
			if err == redis.Nil {
				// Timeout (Block time expired), just loop again
				continue
			}
			rc.logger.Error("Error reading stream", "error", err, "params", fmt.Sprintf("streamName:%v, groupName:%v, consumerName:%v", rc.streamName, rc.groupName, rc.consumerName))
			continue
		}

		// Process the batch of entries
		for _, stream := range entries {
			for _, message := range stream.Messages {
				rc.ProcessVideo(context.Background(), message.Values["bucket"].(string), message.Values["key"].(string), "processed/"+uuid.New().String())

				// 3. Acknowledge the message
				// This removes it from the "Pending Entries List" (PEL)
				// ensuring it won't be redelivered.
				err := rc.rc.XAck(ctx, rc.streamName, rc.groupName, message.ID).Err()
				if err != nil {
					rc.logger.Error("Failed to ack message", "error", err, "params", fmt.Sprintf("streamName:%v, groupName:%v, messageID:%v", rc.streamName, rc.groupName, message.ID))
				}
			}
		}
	}
}
