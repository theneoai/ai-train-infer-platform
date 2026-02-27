package domain

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus 训练任务状态
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
	JobStatusStopping   JobStatus = "stopping"
)

// FrameworkType 框架类型
type FrameworkType string

const (
	FrameworkPyTorch    FrameworkType = "pytorch"
	FrameworkTensorFlow FrameworkType = "tensorflow"
	FrameworkOther      FrameworkType = "other"
)

// TrainingJob 训练任务领域模型
type TrainingJob struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	ProjectID       uuid.UUID       `json:"project_id"`
	ExperimentID    *uuid.UUID      `json:"experiment_id,omitempty"`
	UserID          uuid.UUID       `json:"user_id"`
	RunID           *uuid.UUID      `json:"run_id,omitempty"`

	// 模型配置
	ModelName       string          `json:"model_name"`
	ModelVersion    string          `json:"model_version"`
	DatasetPath     string          `json:"dataset_path"`
	OutputPath      string          `json:"output_path"`  // 模型输出路径

	// 训练配置
	Framework       FrameworkType   `json:"framework"`
	Image           string          `json:"image"`        // Docker 镜像
	Command         []string        `json:"command"`      // 训练命令
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	Environment     map[string]string `json:"environment"` // 环境变量

	// 资源配置
	GPUCount        int             `json:"gpu_count"`
	GPUType         string          `json:"gpu_type"`
	CPUCount        int             `json:"cpu_count"`
	MemoryGB        int             `json:"memory_gb"`
	TimeoutHours    int             `json:"timeout_hours"`

	// 状态信息
	Status          JobStatus       `json:"status"`
	StatusMessage   string          `json:"status_message"`
	Progress        float64         `json:"progress"`
	ExitCode        *int            `json:"exit_code,omitempty"`

	// Docker 执行信息
	ContainerID     string          `json:"container_id"`
	ContainerName   string          `json:"container_name"`
	PID             int             `json:"pid"`

	// 关联的模型
	ModelID         *uuid.UUID      `json:"model_id,omitempty"`

	// 时间戳
	QueuedAt        *time.Time      `json:"queued_at"`
	StartedAt       *time.Time      `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// IsTerminal 检查状态是否为终止状态
func (j *TrainingJob) IsTerminal() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed || j.Status == JobStatusCancelled
}

// CanStart 检查任务是否可以开始
func (j *TrainingJob) CanStart() bool {
	return j.Status == JobStatusPending
}

// CanStop 检查任务是否可以停止
func (j *TrainingJob) CanStop() bool {
	return j.Status == JobStatusPending || j.Status == JobStatusRunning
}

// UpdateStatus 更新状态
func (j *TrainingJob) UpdateStatus(status JobStatus, message string) {
	j.Status = status
	j.StatusMessage = message
	j.UpdatedAt = time.Now()

	switch status {
	case JobStatusRunning:
		now := time.Now()
		j.StartedAt = &now
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		now := time.Now()
		j.CompletedAt = &now
	}
}

// SetProgress 设置进度
func (j *TrainingJob) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	j.Progress = progress
	j.UpdatedAt = time.Now()
}

// JobLog 训练日志
type JobLog struct {
	ID        uuid.UUID `json:"id"`
	JobID     uuid.UUID `json:"job_id"`
	Level     string    `json:"level"`     // INFO, WARN, ERROR
	Source    string    `json:"source"`    // stdout, stderr
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateJobRequest 创建任务请求
type CreateJobRequest struct {
	Name            string                 `json:"name" binding:"required,max=255"`
	Description     string                 `json:"description" binding:"max=1000"`
	ProjectID       string                 `json:"project_id" binding:"required,uuid"`
	ExperimentID    string                 `json:"experiment_id" binding:"omitempty,uuid"`
	ModelName       string                 `json:"model_name" binding:"required,max=255"`
	ModelVersion    string                 `json:"model_version" binding:"max=50"`
	DatasetPath     string                 `json:"dataset_path" binding:"required,max=500"`
	OutputPath      string                 `json:"output_path" binding:"required,max=500"`
	Framework       FrameworkType          `json:"framework" binding:"required,oneof=pytorch tensorflow other"`
	Image           string                 `json:"image" binding:"required,max=500"`
	Command         []string               `json:"command"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	Environment     map[string]string      `json:"environment"`
	GPUCount        int                    `json:"gpu_count" binding:"min=0,max=8"`
	GPUType         string                 `json:"gpu_type" binding:"max=50"`
	CPUCount        int                    `json:"cpu_count" binding:"min=1,max=64"`
	MemoryGB        int                    `json:"memory_gb" binding:"min=1,max=256"`
	TimeoutHours    int                    `json:"timeout_hours" binding:"min=1,max=168"`
}

// UpdateJobRequest 更新任务请求
type UpdateJobRequest struct {
	Name        string `json:"name" binding:"omitempty,max=255"`
	Description string `json:"description" binding:"omitempty,max=1000"`
}

// ListJobsRequest 列出任务请求
type ListJobsRequest struct {
	ProjectID    string     `form:"project_id" binding:"omitempty,uuid"`
	ExperimentID string     `form:"experiment_id" binding:"omitempty,uuid"`
	Status       JobStatus  `form:"status" binding:"omitempty,oneof=pending running completed failed cancelled stopping"`
	Page         int        `form:"page,default=1" binding:"min=1"`
	PageSize     int        `form:"page_size,default=20" binding:"min=1,max=100"`
}

// JobResponse 任务响应
type JobResponse struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	ProjectID       uuid.UUID              `json:"project_id"`
	ExperimentID    *uuid.UUID             `json:"experiment_id,omitempty"`
	UserID          uuid.UUID              `json:"user_id"`
	ModelName       string                 `json:"model_name"`
	ModelVersion    string                 `json:"model_version"`
	Framework       FrameworkType          `json:"framework"`
	Status          JobStatus              `json:"status"`
	StatusMessage   string                 `json:"status_message"`
	Progress        float64                `json:"progress"`
	GPUCount        int                    `json:"gpu_count"`
	ContainerID     string                 `json:"container_id,omitempty"`
	ModelID         *uuid.UUID             `json:"model_id,omitempty"`
	QueuedAt        *time.Time             `json:"queued_at"`
	StartedAt       *time.Time             `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Hyperparameters map[string]interface{} `json:"hyperparameters,omitempty"`
}

// ToResponse 转换为响应
func (j *TrainingJob) ToResponse() *JobResponse {
	return &JobResponse{
		ID:              j.ID,
		Name:            j.Name,
		Description:     j.Description,
		ProjectID:       j.ProjectID,
		ExperimentID:    j.ExperimentID,
		UserID:          j.UserID,
		ModelName:       j.ModelName,
		ModelVersion:    j.ModelVersion,
		Framework:       j.Framework,
		Status:          j.Status,
		StatusMessage:   j.StatusMessage,
		Progress:        j.Progress,
		GPUCount:        j.GPUCount,
		ContainerID:     j.ContainerID,
		ModelID:         j.ModelID,
		QueuedAt:        j.QueuedAt,
		StartedAt:       j.StartedAt,
		CompletedAt:     j.CompletedAt,
		CreatedAt:       j.CreatedAt,
		UpdatedAt:       j.UpdatedAt,
		Hyperparameters: j.Hyperparameters,
	}
}

// LogEntry 日志条目（用于 SSE 流）
type LogEntry struct {
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
