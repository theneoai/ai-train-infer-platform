package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InferenceService 推理服务模型
type InferenceService struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RunID         *uuid.UUID     `json:"run_id" gorm:"type:uuid;index"`
	Name          string         `json:"name" gorm:"not null;size:255"`
	Description   string         `json:"description" gorm:"size:1000"`
	ProjectID     uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`

	// Model configuration
	ModelID       uuid.UUID      `json:"model_id" gorm:"type:uuid;index"`
	ModelVersionID uuid.UUID     `json:"model_version_id" gorm:"type:uuid;index"`

	// Deployment configuration
	Config        JSON           `json:"config" gorm:"type:jsonb;default:'{}'"`
	ResourceConfig JSON          `json:"resource_config" gorm:"type:jsonb;default:'{}'"`
	GPUCount      int            `json:"gpu_count" gorm:"default:0"`

	// Kubernetes/Ray specific
	Namespace     string         `json:"namespace" gorm:"size:100"`
	DeploymentName string        `json:"deployment_name" gorm:"size:255;index"`
	ServiceName   string         `json:"service_name" gorm:"size:255"`
	EndpointURL   string         `json:"endpoint_url" gorm:"size:500"`

	// Autoscaling
	MinReplicas   int            `json:"min_replicas" gorm:"default:1"`
	MaxReplicas   int            `json:"max_replicas" gorm:"default:1"`
	TargetLatency *int           `json:"target_latency"` // ms

	// Status
	Status        string         `json:"status" gorm:"default:'creating';size:50;index:idx_inf_status"`
	StatusMessage string         `json:"status_message" gorm:"size:1000"`
	CurrentReplicas int          `json:"current_replicas" gorm:"default:0"`

	// Traffic split for A/B testing or canary
	TrafficSplit  JSON           `json:"traffic_split" gorm:"type:jsonb;default:'{}'"`

	// Timestamps
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Project       *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User          *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Run           *Run           `json:"run,omitempty" gorm:"foreignKey:RunID"`
	Model         *Model         `json:"model,omitempty" gorm:"foreignKey:ModelID"`
	ModelVersion  *ModelVersion  `json:"model_version,omitempty" gorm:"foreignKey:ModelVersionID"`
}

// BeforeCreate 创建前钩子
func (is *InferenceService) BeforeCreate(tx *gorm.DB) error {
	if is.ID == uuid.Nil {
		is.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (InferenceService) TableName() string {
	return "inference_services"
}

// InferenceServiceStatus 推理服务状态常量
const (
	InferenceStatusCreating   = "creating"
	InferenceStatusDeploying  = "deploying"
	InferenceStatusRunning    = "running"
	InferenceStatusScaling    = "scaling"
	InferenceStatusUpdating   = "updating"
	InferenceStatusStopping   = "stopping"
	InferenceStatusStopped    = "stopped"
	InferenceStatusFailed     = "failed"
	InferenceStatusDeleting   = "deleting"
)

// SimulationEnvironment 仿真环境模型
type SimulationEnvironment struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name          string         `json:"name" gorm:"not null;size:255"`
	Description   string         `json:"description" gorm:"size:1000"`
	ProjectID     uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`

	// Environment configuration
	Type          string         `json:"type" gorm:"not null;size:50"` // llm-sim, agent-sim, etc.
	Config        JSON           `json:"config" gorm:"type:jsonb;default:'{}'"`
	DockerImage   string         `json:"docker_image" gorm:"size:500"`
	Requirements  []string       `json:"requirements" gorm:"type:text[]"`

	// Status
	Status        string         `json:"status" gorm:"default:'pending';size:50;index:idx_sim_status"`
	StatusMessage string         `json:"status_message" gorm:"size:1000"`

	// Kubernetes specific
	Namespace     string         `json:"namespace" gorm:"size:100"`
	PodName       string         `json:"pod_name" gorm:"size:255;index"`

	// Timestamps
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Project       *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User          *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Scenarios     []Scenario     `json:"scenarios,omitempty" gorm:"foreignKey:EnvironmentID"`
}

// BeforeCreate 创建前钩子
func (se *SimulationEnvironment) BeforeCreate(tx *gorm.DB) error {
	if se.ID == uuid.Nil {
		se.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (SimulationEnvironment) TableName() string {
	return "simulation_environments"
}

// Scenario 仿真场景模型
type Scenario struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	EnvironmentID uuid.UUID      `json:"environment_id" gorm:"type:uuid;not null;index"`
	Name          string         `json:"name" gorm:"not null;size:255"`
	Description   string         `json:"description" gorm:"size:1000"`
	Config        JSON           `json:"config" gorm:"type:jsonb;default:'{}'"`
	Status        string         `json:"status" gorm:"default:'ready';size:50"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`

	// Relations
	Environment   *SimulationEnvironment `json:"environment,omitempty" gorm:"foreignKey:EnvironmentID"`
	Results       []SimulationResult     `json:"results,omitempty" gorm:"foreignKey:ScenarioID"`
}

// BeforeCreate 创建前钩子
func (s *Scenario) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Scenario) TableName() string {
	return "scenarios"
}

// SimulationResult 仿真结果模型
type SimulationResult struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ScenarioID   uuid.UUID      `json:"scenario_id" gorm:"type:uuid;not null;index"`
	RunID        *uuid.UUID     `json:"run_id" gorm:"type:uuid;index"`
	Status       string         `json:"status" gorm:"default:'pending';size:50"`
	Config       JSON           `json:"config" gorm:"type:jsonb;default:'{}'"`
	Results      JSON           `json:"results" gorm:"type:jsonb;default:'{}'"`
	Metrics      JSON           `json:"metrics" gorm:"type:jsonb;default:'{}'"`
	StartedAt    *time.Time     `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	CreatedAt    time.Time      `json:"created_at"`

	// Relations
	Scenario     *Scenario      `json:"scenario,omitempty" gorm:"foreignKey:ScenarioID"`
	Run          *Run           `json:"run,omitempty" gorm:"foreignKey:RunID"`
}

// BeforeCreate 创建前钩子
func (sr *SimulationResult) BeforeCreate(tx *gorm.DB) error {
	if sr.ID == uuid.Nil {
		sr.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (SimulationResult) TableName() string {
	return "simulation_results"
}

// SimulationEnvironmentStatus 仿真环境状态常量
const (
	SimulationStatusPending    = "pending"
	SimulationStatusCreating   = "creating"
	SimulationStatusReady      = "ready"
	SimulationStatusRunning    = "running"
	SimulationStatusCompleted  = "completed"
	SimulationStatusFailed     = "failed"
	SimulationStatusDestroying = "destroying"
	SimulationStatusDestroyed  = "destroyed"
)
