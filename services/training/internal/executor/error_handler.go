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
			"tensorflow.python.framework.errors_impl.resourceexhaustederror",
			"killed",
			"signal 9",
			"signal killed",
		},
		retryablePatterns: []string{
			"connection refused",
			"connection timeout",
			"temporary failure",
			"no such host",
			"network is unreachable",
			"i/o timeout",
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
	// 检查退出码
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
	// 可以被重试的退出码
	retryableCodes := []int64{1, 128, 255}
	for _, code := range retryableCodes {
		if exitCode == code {
			return true
		}
	}
	return false
}

// ShouldRetryError 检查错误是否应该重试
func (h *ErrorHandler) ShouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	for _, pattern := range h.retryablePatterns {
		if strings.Contains(errStr, pattern) {
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

// ClassifyError 分类错误
func (h *ErrorHandler) ClassifyError(exitCode int64, log string) ErrorClassification {
	classification := ErrorClassification{
		ExitCode:    exitCode,
		Description: h.GetExitCodeDescription(exitCode),
	}

	// 判断是否为 OOM
	if h.IsOOM("", exitCode) || h.IsOOMLog(log) {
		classification.IsOOM = true
		classification.ErrorType = ErrorTypeOOM
		classification.Recoverable = false
		classification.Recommendation = "Increase memory limit or reduce batch size"
		return classification
	}

	// 根据退出码分类
	switch exitCode {
	case 0:
		classification.ErrorType = ErrorTypeNone
		classification.Recoverable = false
	case 137, 9:
		classification.ErrorType = ErrorTypeOOM
		classification.Recoverable = false
		classification.Recommendation = "Increase memory limit or reduce batch size"
	case 1:
		classification.ErrorType = ErrorTypeGeneral
		classification.Recoverable = true
		classification.Recommendation = "Check application logs for details"
	case 126, 127:
		classification.ErrorType = ErrorTypeCommand
		classification.Recoverable = false
		classification.Recommendation = "Check command syntax and executable permissions"
	case 130:
		classification.ErrorType = ErrorTypeInterrupted
		classification.Recoverable = true
		classification.Recommendation = "Job was interrupted, can be resumed"
	case 139:
		classification.ErrorType = ErrorTypeSegmentationFault
		classification.Recoverable = false
		classification.Recommendation = "Check for memory access errors in code"
	default:
		if exitCode >= 128 {
			classification.ErrorType = ErrorTypeSignal
			classification.Recoverable = h.ShouldRetry(exitCode)
		} else {
			classification.ErrorType = ErrorTypeUnknown
			classification.Recoverable = h.ShouldRetry(exitCode)
		}
	}

	return classification
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
	ErrorTypeSignal            ErrorType = "signal"
	ErrorTypeUnknown           ErrorType = "unknown"
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

// CalculateDelay 计算重试延迟
func (p *RetryPolicy) CalculateDelay(attempt int) time.Duration {
	delay := p.RetryDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * p.BackoffMult)
		if delay > p.MaxDelay {
			delay = p.MaxDelay
			break
		}
	}
	return delay
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	StartupTimeout  time.Duration // 容器启动超时
	HealthTimeout   time.Duration // 健康检查超时
	ShutdownTimeout time.Duration // 关闭超时
	OverallTimeout  time.Duration // 整体任务超时
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

// ContainerHealthChecker 容器健康检查器
type ContainerHealthChecker struct {
	client interface{}
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// CheckContainerHealth 检查容器健康状态
func (h *ContainerHealthChecker) CheckContainerHealth(containerID string) *HealthStatus {
	// 这里可以实现具体的健康检查逻辑
	// 例如：检查容器状态、进程是否存在等
	return &HealthStatus{
		Healthy:   true,
		CheckedAt: time.Now(),
	}
}

// RecoveryStrategy 恢复策略接口
type RecoveryStrategy interface {
	CanRecover(classification ErrorClassification) bool
	Recover(ctx interface{}, jobID string, containerID string) error
}

// SimpleRecoveryStrategy 简单恢复策略
type SimpleRecoveryStrategy struct{}

// CanRecover 检查是否可以恢复
func (s *SimpleRecoveryStrategy) CanRecover(classification ErrorClassification) bool {
	return classification.Recoverable
}

// Recover 执行恢复
func (s *SimpleRecoveryStrategy) Recover(ctx interface{}, jobID string, containerID string) error {
	// 简单恢复策略：重新启动容器
	// 实际实现中可以根据错误类型采取不同的恢复措施
	return nil
}
