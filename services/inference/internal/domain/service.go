package domain

import (
	"time"

	"github.com/google/uuid"
)

// ServiceStatus 推理服务状态
type ServiceStatus string

const (
	ServiceStatusPending    ServiceStatus = "pending"    // 等待中
	ServiceStatusDeploying  ServiceStatus = "deploying"  // 部署中
	ServiceStatusRunning    ServiceStatus = "running"    // 运行中
	ServiceStatusStopping   ServiceStatus = "stopping"   // 停止中
	ServiceStatusStopped    ServiceStatus = "stopped"    // 已停止
	ServiceStatusError      ServiceStatus = "error"      // 错误
)

// InferenceType 推理类型
type InferenceType string

const (
	InferenceTypeTriton InferenceType = "triton" // Triton Inference Server
	InferenceTypeVLLM   InferenceType = "vllm"   // vLLM (大模型推理)
)

// InferenceService 推理服务领域模型
type InferenceService struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	ProjectID   uuid.UUID       `json:"project_id"`
	ModelID     uuid.UUID       `json:"model_id"`
	UserID      uuid.UUID       `json:"user_id"`

	// 推理配置
	Type        InferenceType           `json:"type"`        // triton, vllm
	Config      map[string]interface{}  `json:"config"`      // 推理配置
	Environment map[string]string       `json:"environment"` // 环境变量

	// 资源配置
	GPUCount    int             `json:"gpu_count"`
	GPUType     string          `json:"gpu_type"`
	CPUCount    int             `json:"cpu_count"`
	MemoryGB    int             `json:"memory_gb"`

	// 端口配置
	ContainerPort int           `json:"container_port"` // 容器内部端口
	HostPort      int           `json:"host_port"`      // 主机映射端口

	// 状态信息
	Status        ServiceStatus `json:"status"`
	StatusMessage string        `json:"status_message"`
	HealthStatus  string        `json:"health_status"` // healthy, unhealthy, unknown

	// Docker 执行信息
	ContainerID   string        `json:"container_id"`
	ContainerName string        `json:"container_name"`
	Image         string        `json:"image"`

	// 端点信息
	EndpointURL   string        `json:"endpoint_url"`
	InternalURL   string        `json:"internal_url"`

	// 时间戳
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	StartedAt     *time.Time    `json:"started_at,omitempty"`
	StoppedAt     *time.Time    `json:"stopped_at,omitempty"`
}

// IsTerminal 检查状态是否为终止状态
func (s *InferenceService) IsTerminal() bool {
	return s.Status == ServiceStatusStopped || s.Status == ServiceStatusError
}

// CanStart 检查服务是否可以启动
func (s *InferenceService) CanStart() bool {
	return s.Status == ServiceStatusPending || 
		   s.Status == ServiceStatusStopped || 
		   s.Status == ServiceStatusError
}

// CanStop 检查服务是否可以停止
func (s *InferenceService) CanStop() bool {
	return s.Status == ServiceStatusDeploying || 
		   s.Status == ServiceStatusRunning
}

// UpdateStatus 更新状态
func (s *InferenceService) UpdateStatus(status ServiceStatus, message string) {
	s.Status = status
	s.StatusMessage = message
	s.UpdatedAt = time.Now()

	switch status {
	case ServiceStatusRunning:
		now := time.Now()
		s.StartedAt = &now
	case ServiceStatusStopped, ServiceStatusError:
		now := time.Now()
		s.StoppedAt = &now
	}
}

// SetHealthStatus 设置健康状态
func (s *InferenceService) SetHealthStatus(healthy bool) {
	if healthy {
		s.HealthStatus = "healthy"
	} else {
		s.HealthStatus = "unhealthy"
	}
}

// CreateServiceRequest 创建推理服务请求
type CreateServiceRequest struct {
	Name        string                 `json:"name" binding:"required,max=255"`
	Description string                 `json:"description" binding:"max=1000"`
	ProjectID   string                 `json:"project_id" binding:"required,uuid"`
	ModelID     string                 `json:"model_id" binding:"required,uuid"`
	Type        InferenceType          `json:"type" binding:"required,oneof=triton vllm"`
	Config      map[string]interface{} `json:"config"`
	Environment map[string]string      `json:"environment"`
	GPUCount    int                    `json:"gpu_count" binding:"min=0,max=8"`
	GPUType     string                 `json:"gpu_type" binding:"max=50"`
	CPUCount    int                    `json:"cpu_count" binding:"min=1,max=64"`
	MemoryGB    int                    `json:"memory_gb" binding:"min=1,max=256"`
}

// UpdateServiceRequest 更新推理服务请求
type UpdateServiceRequest struct {
	Name        string                 `json:"name" binding:"omitempty,max=255"`
	Description string                 `json:"description" binding:"omitempty,max=1000"`
	Config      map[string]interface{} `json:"config"`
	Environment map[string]string      `json:"environment"`
}

// ListServicesRequest 列出推理服务请求
type ListServicesRequest struct {
	ProjectID string        `form:"project_id" binding:"omitempty,uuid"`
	Status    ServiceStatus `form:"status" binding:"omitempty,oneof=pending deploying running stopping stopped error"`
	Page      int           `form:"page,default=1" binding:"min=1"`
	PageSize  int           `form:"page_size,default=20" binding:"min=1,max=100"`
}

// ServiceResponse 推理服务响应
type ServiceResponse struct {
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	ProjectID     uuid.UUID              `json:"project_id"`
	ModelID       uuid.UUID              `json:"model_id"`
	UserID        uuid.UUID              `json:"user_id"`
	Type          InferenceType          `json:"type"`
	Status        ServiceStatus          `json:"status"`
	StatusMessage string                 `json:"status_message,omitempty"`
	HealthStatus  string                 `json:"health_status"`
	GPUCount      int                    `json:"gpu_count"`
	ContainerID   string                 `json:"container_id,omitempty"`
	ContainerName string                 `json:"container_name,omitempty"`
	Image         string                 `json:"image,omitempty"`
	EndpointURL   string                 `json:"endpoint_url,omitempty"`
	InternalURL   string                 `json:"internal_url,omitempty"`
	HostPort      int                    `json:"host_port,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	StoppedAt     *time.Time             `json:"stopped_at,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// ToResponse 转换为响应
func (s *InferenceService) ToResponse() *ServiceResponse {
	return &ServiceResponse{
		ID:            s.ID,
		Name:          s.Name,
		Description:   s.Description,
		ProjectID:     s.ProjectID,
		ModelID:       s.ModelID,
		UserID:        s.UserID,
		Type:          s.Type,
		Status:        s.Status,
		StatusMessage: s.StatusMessage,
		HealthStatus:  s.HealthStatus,
		GPUCount:      s.GPUCount,
		ContainerID:   s.ContainerID,
		ContainerName: s.ContainerName,
		Image:         s.Image,
		EndpointURL:   s.EndpointURL,
		InternalURL:   s.InternalURL,
		HostPort:      s.HostPort,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		StartedAt:     s.StartedAt,
		StoppedAt:     s.StoppedAt,
		Config:        s.Config,
	}
}

// StartServiceRequest 启动服务请求
type StartServiceRequest struct {
	WaitForHealthy bool `json:"wait_for_healthy"` // 是否等待健康检查通过
}

// StopServiceRequest 停止服务请求
type StopServiceRequest struct {
	Force bool `json:"force"` // 是否强制停止
}

// ServiceEvent 推理服务事件
type ServiceEvent struct {
	Type      string        `json:"type"` // created, started, stopped, error
	ServiceID uuid.UUID     `json:"service_id"`
	Status    ServiceStatus `json:"status"`
	Message   string        `json:"message"`
	Timestamp time.Time     `json:"timestamp"`
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	ContainerID   string            `json:"container_id"`
	ContainerName string            `json:"container_name"`
	Image         string            `json:"image"`
	Status        string            `json:"status"`
	State         string            `json:"state"`
	ExitCode      int               `json:"exit_code"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	FinishedAt    *time.Time        `json:"finished_at,omitempty"`
	Ports         map[string][]PortBinding `json:"ports,omitempty"`
	Health        string            `json:"health"` // healthy, unhealthy, starting, none
}

// PortBinding 端口绑定
type PortBinding struct {
	HostIP   string `json:"host_ip"`
	HostPort string `json:"host_port"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	StoragePath string    `json:"storage_path"`
	Format      string    `json:"format"` // triton, pytorch, safetensors, etc.
}
