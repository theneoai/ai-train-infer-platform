package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard API response type
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MetaInfo struct {
	Page      int   `json:"page,omitempty"`
	PageSize  int   `json:"page_size,omitempty"`
	Total     int64 `json:"total,omitempty"`
	TotalPage int   `json:"total_page,omitempty"`
}

// Success returns a success response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMeta returns a success response with pagination metadata
func SuccessWithMeta(c *gin.Context, data interface{}, meta *MetaInfo) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error returns an error response
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    http.StatusText(status),
			Message: message,
		},
	})
}

// ErrorWithCode returns an error response with custom code
func ErrorWithCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// Created returns a 201 created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent returns a 204 no content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
