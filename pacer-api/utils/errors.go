package utils

import (
	"errors"
	"fmt"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func ErrNotFound(resource string) *AppError {
	msg := resource + " not found"
	return &AppError{Code: 404, Message: msg, Err: errors.New(msg)}
}

func ErrUnauthorized() *AppError {
	return &AppError{Code: 401, Message: "unauthorized", Err: fmt.Errorf("unauthorized")}
}

func ErrBadRequest(message string) *AppError {
	if message == "" {
		message = "bad request"
	}
	return &AppError{Code: 400, Message: message, Err: errors.New(message)}
}

func ErrInternal(err error) *AppError {
	if err == nil {
		err = fmt.Errorf("internal server error")
	}
	return &AppError{Code: 500, Message: "internal server error", Err: err}
}
