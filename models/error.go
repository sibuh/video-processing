package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrResourceNotFound       = errors.New("resource not found")
	ErrInvalidEmailOrPassword = errors.New("invalid email or password")
	ErrInvalidInputData       = errors.New("invalid input data")
	ErrInvalidUUID            = errors.New("invalid uuid")
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"err"`
}

func (a Error) Error() string {
	return fmt.Sprintf("%d: %s: %s", a.Code, a.Message, a.Err.Error())
}

func NewError(err error) *Error {
	var e *Error
	switch true {
	case errors.Is(err, sql.ErrNoRows):
		e = &Error{
			Code:    http.StatusConflict,
			Message: "resource already exists",
			Err:     err,
		}
	case errors.Is(err, sql.ErrNoRows):
		e = &Error{
			Code:    http.StatusNotFound,
			Message: "resource not found",
			Err:     err,
		}

	case errors.Is(err, sql.ErrNoRows):
		e = &Error{
			Code:    http.StatusNotFound,
			Message: "resource not found",
			Err:     err,
		}

	case errors.Is(err, ErrInvalidInputData):
		e = &Error{
			Code:    http.StatusBadRequest,
			Message: "invalid input data",
			Err:     err,
		}
	case errors.Is(err, ErrInvalidUUID):
		e = &Error{
			Code:    http.StatusBadRequest,
			Message: "failed to parse user id invalid uuid",
			Err:     err,
		}
	default:
		e = &Error{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
			Err:     err,
		}
	}
	return e
}
