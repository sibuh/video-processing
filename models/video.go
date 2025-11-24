package models

import (
	"mime/multipart"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type UploadVideoRequest struct {
	Title       string                  `form:"title" binding:"required"`
	Description string                  `form:"description" binding:"required"`
	Videos      []*multipart.FileHeader `form:"videos" binding:"required"`
}

func (u *UploadVideoRequest) Validate() error {

	return validation.ValidateStruct(u,
		validation.Field(&u.Title, validation.Required.Error("title is required")),
		validation.Field(&u.Description, validation.Required.Error("description is required")),
		validation.Field(&u.Videos, validation.Required.Error("at least one video is required")),
	)
}
