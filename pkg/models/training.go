package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TrainingJob 训练任务模型
type TrainingJob struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RunID           *uuid.UUID     `json:"run_id" gorm:"type:uuid;index"`
	Name            string         `json:"name" gorm:"not null;size:255"`
	Description     string         `json:"description" gorm:"size:1000"`
	ProjectID       uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	UserID          uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`

	// Model configuration
	ModelName       string         `json:"model_name" gorm:"size:255"`
	ModelVersion    string         `json:"model_version" gorm:"size:50"`
	DatasetPath     string         `json:"dataset_path" gorm:"size:500"`

	// Hyperparameters
	Hyperparameters JSON           `json:"hyperparameters" gorm:"type:jsonb;default:'{}'"`

	// Resource configuration
	Resources       JSON           `json:"resources" gorm:"type:jsonb;default:'{}'"`
	GPUCount        int            `json:"gpu_count" gorm:"default:0"`
	GPUType         string         `json:"gpu_type" gorm:"size:50"`

	// Ray/Kubernetes specific
	ClusterConfig   JSON           `json:"cluster_config" gorm:"type:jsonb;default:'{}'"`
	Namespace       string         `json:"namespace" gorm:"size:100"`
	JobID           string         `json:"job_id" gorm:"size:255;index"` // Ray Job ID or K8s Job name
	WorkerCount     int            `json:"worker_count" gorm:"default:1"`

	// Status
	Status          string         `json:"status" gorm:"default:'pending';size:50;index:idx_train_status"`
	StatusMessage   string         `json:"status_message" gorm:"size:1000"`
	Progress        float64        `json:"progress" gorm:"default:0"` // 0-100

	// Timestamps
	QueuedAt        *time.Time     `json:"queued_at"`
	StartedAt       *time.Time     `json:"started_at"`
	CompletedAt     *time.Time     `json:"completed_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Project    *Project    `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User       *User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Run        *Run        `json:"run,omitempty" gorm:"foreignKey:RunID"`
	Checkpoints []Checkpoint `json:"checkpoints,omitempty" gorm:"foreignKey:TrainingJobID"`
}

// BeforeCreate 创建前钩子
func (tj *TrainingJob) BeforeCreate(tx *gorm.DB) error {
	if tj.ID == uuid.Nil {
		tj.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (TrainingJob) TableName() string {
	return "training_jobs"
}

// Checkpoint 模型检查点
type Checkpoint struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TrainingJobID  uuid.UUID      `json:"training_job_id" gorm:"type:uuid;not null;index"`
	Step           int64          `json:"step" gorm:"not null"`
	Epoch          *int64         `json:"epoch"`
	StoragePath    string         `json:"storage_path" gorm:"not null;size:500"`
	Metrics        JSON           `json:"metrics" gorm:"type:jsonb;default:'{}'"`
	Size           int64          `json:"size" gorm:"default:0"`
	IsBest         bool           `json:"is_best" gorm:"default:false"`
	CreatedAt      time.Time      `json:"created_at"`

	// Relations
	TrainingJob *TrainingJob `json:"training_job,omitempty" gorm:"foreignKey:TrainingJobID"`
}

// BeforeCreate 创建前钩子
func (c *Checkpoint) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Checkpoint) TableName() string {
	return "checkpoints"
}

// TrainingJobStatus 训练任务状态常量
const (
	TrainingStatusPending    = "pending"
	TrainingStatusQueued     = "queued"
	TrainingStatusRunning    = "running"
	TrainingStatusPaused     = "paused"
	TrainingStatusCompleted  = "completed"
	TrainingStatusFailed     = "failed"
	TrainingStatusCancelled  = "cancelled"
	TrainingStatusStopping   = "stopping"
)

// Model 模型注册表
type Model struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name          string         `json:"name" gorm:"not null;size:255"`
	Description   string         `json:"description" gorm:"size:1000"`
	ProjectID     uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	LatestVersion *string        `json:"latest_version" gorm:"size:50"`
	Framework     string         `json:"framework" gorm:"size:50"` // pytorch, tensorflow, etc.
	TaskType      string         `json:"task_type" gorm:"size:50"`  // classification, generation, etc.
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Project  *Project        `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Versions []ModelVersion  `json:"versions,omitempty" gorm:"foreignKey:ModelID"`
}

// BeforeCreate 创建前钩子
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Model) TableName() string {
	return "models"
}

// ModelVersion 模型版本
type ModelVersion struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ModelID       uuid.UUID      `json:"model_id" gorm:"type:uuid;not null;index"`
	Version       string         `json:"version" gorm:"not null;size:50"`
	RunID         *uuid.UUID     `json:"run_id" gorm:"type:uuid;index"`
	TrainingJobID *uuid.UUID     `json:"training_job_id" gorm:"type:uuid;index"`
	StoragePath   string         `json:"storage_path" gorm:"size:500"`
	Metadata      JSON           `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	Status        string         `json:"status" gorm:"default:'active';size:50"`
	CreatedAt     time.Time      `json:"created_at"`

	// Relations
	Model       *Model       `json:"model,omitempty" gorm:"foreignKey:ModelID"`
	Run         *Run         `json:"run,omitempty" gorm:"foreignKey:RunID"`
	TrainingJob *TrainingJob `json:"training_job,omitempty" gorm:"foreignKey:TrainingJobID"`
}

// BeforeCreate 创建前钩子
func (mv *ModelVersion) BeforeCreate(tx *gorm.DB) error {
	if mv.ID == uuid.Nil {
		mv.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (ModelVersion) TableName() string {
	return "model_versions"
}
