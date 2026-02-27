package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Experiment 实验模型
type Experiment struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"not null;size:255"`
	Description string         `json:"description" gorm:"size:1000"`
	ProjectID   uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index:idx_exp_project"`
	UserID      uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Config      JSON           `json:"config" gorm:"type:jsonb;default:'{}'"`
	Tags        []string       `json:"tags" gorm:"type:text[]"`
	Status      string         `json:"status" gorm:"default:'running';size:50;index:idx_exp_status"`
	CreatedAt   time.Time      `json:"created_at" gorm:"index:idx_exp_created"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Runs    []Run    `json:"runs,omitempty" gorm:"foreignKey:ExperimentID"`
}

// BeforeCreate 创建前钩子
func (e *Experiment) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Experiment) TableName() string {
	return "experiments"
}

// Run 运行记录模型
type Run struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ExperimentID   uuid.UUID      `json:"experiment_id" gorm:"type:uuid;not null;index:idx_run_exp"`
	RunType        string         `json:"run_type" gorm:"not null;size:50;index:idx_run_type"` // training, inference, simulation
	Status         string         `json:"status" gorm:"default:'pending';size:50;index:idx_run_status"`
	Config         JSON           `json:"config" gorm:"type:jsonb;default:'{}'"`
	MetricsSummary JSON           `json:"metrics_summary" gorm:"type:jsonb;default:'{}'"`
	StartedAt      *time.Time     `json:"started_at" gorm:"index"`
	EndedAt        *time.Time     `json:"ended_at"`
	Duration       *int64         `json:"duration"` // seconds
	CreatedAt      time.Time      `json:"created_at" gorm:"index:idx_run_created"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Experiment *Experiment `json:"experiment,omitempty" gorm:"foreignKey:ExperimentID"`
	Metrics    []Metric    `json:"metrics,omitempty" gorm:"foreignKey:RunID"`
	Artifacts  []Artifact  `json:"artifacts,omitempty" gorm:"foreignKey:RunID"`
	Logs       []LogEntry  `json:"logs,omitempty" gorm:"foreignKey:RunID"`
}

// BeforeCreate 创建前钩子
func (r *Run) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Run) TableName() string {
	return "runs"
}

// Metric 指标模型
type Metric struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RunID     uuid.UUID      `json:"run_id" gorm:"type:uuid;not null;index:idx_metric_run"`
	Key       string         `json:"key" gorm:"not null;size:255;index:idx_metric_key"`
	Value     float64        `json:"value" gorm:"not null"`
	Step      *int64         `json:"step" gorm:"index"`
	Timestamp *time.Time     `json:"timestamp" gorm:"index:idx_metric_time"`
	Metadata  JSON           `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time      `json:"created_at"`

	// Relations
	Run *Run `json:"run,omitempty" gorm:"foreignKey:RunID"`
}

// BeforeCreate 创建前钩子
func (m *Metric) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Metric) TableName() string {
	return "metrics"
}

// Artifact 工件模型
type Artifact struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RunID       uuid.UUID      `json:"run_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"not null;size:255"`
	Type        string         `json:"type" gorm:"not null;size:50"` // model, checkpoint, dataset, etc.
	StoragePath string         `json:"storage_path" gorm:"not null;size:500"`
	Size        int64          `json:"size" gorm:"default:0"`
	Metadata    JSON           `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt   time.Time      `json:"created_at"`

	// Relations
	Run *Run `json:"run,omitempty" gorm:"foreignKey:RunID"`
}

// BeforeCreate 创建前钩子
func (a *Artifact) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Artifact) TableName() string {
	return "artifacts"
}

// LogEntry 日志条目模型
type LogEntry struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RunID     uuid.UUID `json:"run_id" gorm:"type:uuid;not null;index:idx_log_run"`
	Level     string    `json:"level" gorm:"size:20;index:idx_log_level"` // INFO, WARN, ERROR, etc.
	Message   string    `json:"message" gorm:"type:text"`
	Source    string    `json:"source" gorm:"size:255"` // stdout, stderr, etc.
	Timestamp time.Time `json:"timestamp" gorm:"index:idx_log_time"`
}

// BeforeCreate 创建前钩子
func (l *LogEntry) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (LogEntry) TableName() string {
	return "log_entries"
}
