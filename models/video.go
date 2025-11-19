package models

import "mime/multipart"

type UploadVideoRequest struct {
	Title       string                  `form:"title" binding:"required"`
	Description string                  `form:"description" binding:"required"`
	Videos      []*multipart.FileHeader `form:"videos" binding:"required"`
}
