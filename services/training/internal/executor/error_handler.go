package executor

import (
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
)

// ErrorHandler 错误处理器
type ErrorHandler struct {
	oomPatterns       []string
	retryablePatterns []string
	exitCodeMap       map[int]string
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		oomPatterns: []string{
			"out of memory",
			"out-of-memory",
			"oom",
			"cannot allocate memory",
			"cuda out of memory",
			"runtimeerror: cuda error: out of memory",
			"killed",
			"signal 9",
		},
		retryablePatterns: []string{
			"connection refused",
			"connection timeout",
			"temporary failure",
			"no such host",
		},
		exitCodeMap: map[int]string{
			0:   "success",
			1:   "general error",
			126: "command not executable",
			127: "command not found",
			128: "invalid exit argument",
			130: "interrupted (Ctrl+C)",
			137: "killed by SIGKILL (OOM likely)",
			139: "segmentation fault",
			143: "terminated by SIGTERM",
			255: "exit status out of range",
		},
	}
}

// IsOOM 检查是否为 OOM 错误
func (h *ErrorHandler) IsOOM(containerID string, exitCode int64) bool {
	if exitCode == 137 || exitCode == 9 {
		return true
	}
	return false
}

// IsOOMLog 从日志中检测 OOM
func (h *ErrorHandler) IsOOMLog(log string) bool {
	lowerLog := strings.ToLower(log)
	for _, pattern := range h.oomPatterns {
		if strings.Contains(lowerLog, pattern) {
			return true
		}
	}
	return false
}

// ShouldRetry 检查是否应该重试
func (h *ErrorHandler) ShouldRetry(exitCode int64) bool {
	retryableCodes := []int64{1, 128, 255}
	for _, code := range retryableCodes {
		if exitCode == code {
			return true
		}
	}
	return false
}

// GetExitCodeDescription 获取退出码描述
func (h *ErrorHandler) GetExitCodeDescription(exitCode int64) string {
	if desc, ok := h.exitCodeMap[int(exitCode)]; ok {
		return desc
	}
	return "unknown error"
}

// ErrorType 错误类型
type ErrorType string

const (
	ErrorTypeNone              ErrorType = "none"
	ErrorTypeOOM               ErrorType = "oom"
	ErrorTypeGeneral           ErrorType = "general"
	ErrorTypeCommand           ErrorType = "command"
	ErrorTypeInterrupted       ErrorType = "interrupted"
	ErrorTypeSegmentationFault ErrorType = "segmentation_fault"
)

// ErrorClassification 错误分类
type ErrorClassification struct {
	ExitCode       int64     `json:"exit_code"`
	ErrorType      ErrorType `json:"error_type"`
	Description    string    `json:"description"`
	IsOOM          bool      `json:"is_oom"`
	Recoverable    bool      `json:"recoverable"`
	Recommendation string    `json:"recommendation"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries  int
	RetryDelay  time.Duration
	BackoffMult float64
	MaxDelay    time.Duration
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:  3,
		RetryDelay:  5 * time.Second,
		BackoffMult: 2.0,
		MaxDelay:    60 * time.Second,
	}
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	StartupTimeout  time.Duration
	HealthTimeout   time.Duration
	ShutdownTimeout time.Duration
	OverallTimeout  time.Duration
}

// DefaultTimeoutConfig 默认超时配置
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		StartupTimeout:  5 * time.Minute,
		HealthTimeout:   30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		OverallTimeout:  24 * time.Hour,
	}
}
