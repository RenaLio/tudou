package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ListResponse[T any] struct {
	Total    int64 `json:"total"`
	Items    []T   `json:"items"`
	Page     int64 `json:"page"`
	PageSize int64 `json:"pageSize"`
}

func Success(ctx *gin.Context, data any) {
	if data == nil {
		data = map[string]any{}
	}
	resp := Response{Code: 0, Message: "success", Data: data}
	ctx.JSON(http.StatusOK, resp)
}

type AppError struct {
	Code     int
	HTTPCode int
	Message  string
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code int, httpCode int, message string) *AppError {
	return &AppError{
		Code:     code,
		HTTPCode: httpCode,
		Message:  message,
	}
}

func (e *AppError) WithMessage(message string) *AppError {
	if e == nil {
		return nil
	}
	return &AppError{
		Code:     e.Code,
		HTTPCode: e.HTTPCode,
		Message:  message,
	}
}

func Fail(ctx *gin.Context, err error, data any) {
	if data == nil {
		data = map[string]any{}
	}
	if err == nil {
		Success(ctx, data)
		return
	}
	if appErr, ok := errors.AsType[*AppError](err); ok {
		resp := Response{Code: appErr.Code, Message: appErr.Error(), Data: data}
		ctx.JSON(appErr.HTTPCode, resp)
		return
	}
	ctx.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "unknown error", Data: data})
}
