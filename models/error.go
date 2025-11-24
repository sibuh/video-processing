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
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
	Params      string `json:"params"`
	Err         error  `json:"err"`
}

func (a Error) Error() string {
	return fmt.Sprintf("%d: %s: %s: %s: %+v", a.Code, a.Message, a.Description, a.Params, a.Err)
}

func IndentifyDbError(err error) Error {
	var e Error
	switch true {
	case errors.Is(err, sql.ErrNoRows):
		e = Error{
			Code:    http.StatusConflict,
			Message: "resource already exists",
			Err:     err,
		}
	case errors.Is(err, sql.ErrNoRows):
		e = Error{
			Code:    http.StatusNotFound,
			Message: "resource not found",
			Err:     err,
		}

	default:
		e = Error{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
			Err:     err,
		}
	}
	return e
}
func (e Error) AddParams(params string) Error {
	e.Params += params
	return e
}
