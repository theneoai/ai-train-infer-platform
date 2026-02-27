package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResponseCode 响应码
type ResponseCode int

const (
	// Success 成功
	Success ResponseCode = 0
	// ErrorUnknown 未知错误
	ErrorUnknown ResponseCode = 100001
	// ErrorInvalidParams 参数错误
	ErrorInvalidParams ResponseCode = 100002
	// ErrorUnauthorized 未授权
	ErrorUnauthorized ResponseCode = 100003
	// ErrorForbidden 禁止访问
	ErrorForbidden ResponseCode = 100004
	// ErrorNotFound 资源不存在
	ErrorNotFound ResponseCode = 100005
	// ErrorInternalServer 服务器内部错误
	ErrorInternalServer ResponseCode = 100006
	// ErrorTooManyRequests 请求过于频繁
	ErrorTooManyRequests ResponseCode = 100007
	// ErrorServiceUnavailable 服务不可用
	ErrorServiceUnavailable ResponseCode = 100008
)

// codeMessageMap 响应码对应的错误信息
var codeMessageMap = map[ResponseCode]string{
	Success:                 "success",
	ErrorUnknown:            "unknown error",
	ErrorInvalidParams:      "invalid parameters",
	ErrorUnauthorized:       "unauthorized",
	ErrorForbidden:          "forbidden",
	ErrorNotFound:           "resource not found",
	ErrorInternalServer:     "internal server error",
	ErrorTooManyRequests:    "too many requests",
	ErrorServiceUnavailable: "service unavailable",
}

// Message 获取响应码对应的错误信息
func (c ResponseCode) Message() string {
	if msg, ok := codeMessageMap[c]; ok {
		return msg
	}
	return codeMessageMap[ErrorUnknown]
}

// HTTPStatus 获取响应码对应的 HTTP 状态码
func (c ResponseCode) HTTPStatus() int {
	switch c {
	case Success:
		return http.StatusOK
	case ErrorInvalidParams:
		return http.StatusBadRequest
	case ErrorUnauthorized:
		return http.StatusUnauthorized
	case ErrorForbidden:
		return http.StatusForbidden
	case ErrorNotFound:
		return http.StatusNotFound
	case ErrorTooManyRequests:
		return http.StatusTooManyRequests
	case ErrorInternalServer, ErrorUnknown:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Response 标准响应结构
type Response struct {
	Code    ResponseCode `json:"code"`
	Message string       `json:"message"`
	Data    interface{}  `json:"data,omitempty"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PageResponse 分页响应
type PageResponse struct {
	Code       ResponseCode `json:"code"`
	Message    string       `json:"message"`
	Data       interface{}  `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// NewResponse 创建新响应
func NewResponse(code ResponseCode, data interface{}) *Response {
	return &Response{
		Code:    code,
		Message: code.Message(),
		Data:    data,
	}
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, NewResponse(Success, data))
}

// SuccessWithMessage 成功响应（带自定义消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, &Response{
		Code:    Success,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code ResponseCode) {
	c.JSON(code.HTTPStatus(), NewResponse(code, nil))
}

// ErrorWithMessage 错误响应（带自定义消息）
func ErrorWithMessage(c *gin.Context, code ResponseCode, message string) {
	c.JSON(code.HTTPStatus(), &Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ErrorWithData 错误响应（带数据）
func ErrorWithData(c *gin.Context, code ResponseCode, message string, data interface{}) {
	if message == "" {
		message = code.Message()
	}
	c.JSON(code.HTTPStatus(), &Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, &PageResponse{
		Code:    Success,
		Message: Success.Message(),
		Data:    data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// BadRequest 参数错误
func BadRequest(c *gin.Context, message string) {
	ErrorWithMessage(c, ErrorInvalidParams, message)
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "authentication required"
	}
	ErrorWithMessage(c, ErrorUnauthorized, message)
}

// Forbidden 禁止访问
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "access denied"
	}
	ErrorWithMessage(c, ErrorForbidden, message)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, resource string) {
	message := "resource not found"
	if resource != "" {
		message = resource + " not found"
	}
	ErrorWithMessage(c, ErrorNotFound, message)
}

// InternalServerError 服务器内部错误
func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	ErrorWithMessage(c, ErrorInternalServer, message)
}

// TooManyRequests 请求过于频繁
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "too many requests, please try again later"
	}
	ErrorWithMessage(c, ErrorTooManyRequests, message)
}
