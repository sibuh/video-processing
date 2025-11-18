package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"video-processing/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VideoProcessor interface {
	Upload(ctx *gin.Context)
}

type videoHandler struct {
	logger   *slog.Logger
	timeout  time.Duration
	services services.VideoProcessor
}

func NewVideoHandler(logger *slog.Logger, timeout time.Duration, services services.VideoProcessor) VideoProcessor {
	return &videoHandler{
		logger:   logger,
		timeout:  timeout,
		services: services,
	}
}

func (vh videoHandler) Upload(c *gin.Context) {
	// set timeout for request
	ctx, cancel := context.WithTimeout(c.Request.Context(), vh.timeout)
	defer cancel()
	// get user id from context
	uid, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.Request.ParseMultipartForm(100 << 20) // 100 MB

	err := vh.services.Upload(ctx, uid.String(), c.Request.MultipartForm.File)
	if err != nil {
		vh.logger.Error("failed to upload video", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload video"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Video uploaded successfully"})
}
