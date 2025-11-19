package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"video-processing/models"
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

// @Summary Upload video
// @Tags video
// @Accept multipart/form-data
// @Produce json
// @Param videos formData file true "Video file"
// @Param title formData string true "Video title"
// @Param description formData string true "Video description"
// @Success 200 {object} string "Video uploaded successfully"
// @Failure 400 {object} string "Bad request"
// @Failure 500 {object} string "Internal server error"
// @Router /v1/upload [post]
// @Security BearerAuth
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
	var req models.UploadVideoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Request.ParseMultipartForm(100 << 20) // 100 MB

	err := vh.services.Upload(ctx, uid, req)
	if err != nil {
		vh.logger.Error("failed to upload video", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload video"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Video uploaded successfully"})
}
