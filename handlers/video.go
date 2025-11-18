package handlers

import (
	"log/slog"
	"net/http"

	"video-processing/services"

	"github.com/gin-gonic/gin"
)

type VideoProcessor interface {
	Upload(ctx *gin.Context)
}

type videoHandler struct {
	logger   *slog.Logger
	services services.VideoProcessor
}

func (vh videoHandler) Upload(ctx *gin.Context) {
	ctx.Request.ParseMultipartForm(100 << 20) // 100 MB
	file, _, err := ctx.Request.FormFile("video")
	if err != nil {
		ctx.Error(err)
		return
	}
	defer file.Close()

	// // Save locally
	// tempDir := "./uploads"
	// os.MkdirAll(tempDir, 0755)
	// fileID := uuid.New().String()
	// localPath := filepath.Join(tempDir, fileID+filepath.Ext(header.Filename))
	// outFile, _ := os.Create(localPath)
	// defer outFile.Close()
	// _, _ = file.Seek(0, 0)
	// _, _ = outFile.ReadFrom(file)

	// // Enqueue job
	// job := queue.VideoJob{
	// 	ID:        fileID,
	// 	FilePath:  localPath,
	// 	OutputDir: "./processed",
	// }
	// err = videoQueue.Enqueue(job)
	// if err != nil {
	// 	ctx.Error(err)
	// 	return
	// }
	ctx.JSON(http.StatusOK, gin.H{"message": "Video uploaded successfully"})
}
