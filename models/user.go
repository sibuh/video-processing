package models

import (
	"errors"
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID `json:"id"`
	FirstName         string    `json:"first_name"`
	MiddleName        string    `json:"middle_name"`
	LastName          string    `json:"last_name"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	Phone             string    `json:"phone"`
	Password          string    `json:"password"`
	ProfilePictureURL string    `json:"profile_picture_url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DeletedAt         time.Time `json:"deleted_at"`
}

type UserRegistrationRequest struct {
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	LastName   string `json:"last_name"`
	Username   string `json:"username"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	Email      string `json:"email"`
}

var ValidatePassword validation.RuleFunc = func(value interface{}) error {
	s, _ := value.(string)
	if match, _ := regexp.MatchString(`[A-Za-z]`, s); !match {
		return errors.New("must contain at least one letter")
	}
	if match, _ := regexp.MatchString(`\d`, s); !match {
		return errors.New("must contain at least one digit")
	}
	return nil
}

func (urr UserRegistrationRequest) Validate() error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	err := validation.ValidateStruct(&urr,
		validation.Field(&urr.FirstName, validation.Required.Error("first_name is required"),
			validation.Length(3, 30)),
		validation.Field(&urr.LastName, validation.Required.Error("first_name is required"),
			validation.Length(3, 30)),
		validation.Field(&urr.Username, validation.Required.Error("username is required")),
		validation.Field(&urr.Email, validation.Required.Error("email is required"),
			validation.Match(emailRegex).Error("invalid email format")),
		validation.Field(&urr.Phone, validation.Required.Error("phone is required"),
			validation.Length(9, 12)),
		validation.Field(&urr.Password, validation.Required.Error("password is required"),
			validation.Length(6, 12).Error("password length must be between 6 and 12"), validation.By(ValidatePassword)),
	)
	if err == nil {
		return nil
	}
	return errors.Join(err, ErrInvalidInputData)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func (lr LoginRequest) Validate() error {
	err := validation.ValidateStruct(&lr,
		validation.Field(&lr.Email, validation.Required.Error("email is required")),
		validation.Field(&lr.Password, validation.Required.Error("password is required")),
	)
	if err == nil {
		return nil

	}
	return errors.Join(err, ErrInvalidInputData)
}

type UpdateUserRequest struct {
	FirstName         string `json:"first_name,omitempty"`
	LastName          string `json:"last_name,omitempty"`
	Phone             string `json:"phone,omitempty"`
	Username          string `json:"username,omitempty"`
	Email             string `json:"email,omitempty"`
	ProfilePictureURL string `json:"profile_picture,omitempty"`
}
