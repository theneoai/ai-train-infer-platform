package errors

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	ErrInvalidParams      = errors.New("invalid parameters")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("resource not found")
	ErrInternalServer     = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTooManyRequests    = errors.New("too many requests")
	ErrTimeout            = errors.New("request timeout")
	ErrDuplicateEntry     = errors.New("duplicate entry")
	ErrDatabase           = errors.New("database error")
	ErrCache              = errors.New("cache error")
	ErrValidation         = errors.New("validation error")
	ErrRateLimit          = errors.New("rate limit exceeded")
)

// AppError 应用错误
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(err error, code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapWithCode 使用已有错误码包装错误
func WrapWithCode(err error, appErr *AppError) *AppError {
	return &AppError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Err:     err,
	}
}

// Is 检查错误是否匹配
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As 转换错误类型
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// CodeFromError 从错误获取错误码
func CodeFromError(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return 500
}

// MessageFromError 从错误获取错误消息
func MessageFromError(err error) string {
	if err == nil {
		return ""
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return err.Error()
}
