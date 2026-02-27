package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/models"
)

// ExperimentStatus 实验状态
type ExperimentStatus string

const (
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusFailed    ExperimentStatus = "failed"
	ExperimentStatusStopped   ExperimentStatus = "stopped"
)

// Experiment 实验领域模型
type Experiment struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	ProjectID   uuid.UUID        `json:"project_id"`
	UserID      uuid.UUID        `json:"user_id"`
	Config      ExperimentConfig `json:"config"`
	Tags        []string         `json:"tags"`
	Status      ExperimentStatus `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	RunsCount   int              `json:"runs_count"`
}

// ExperimentConfig 实验配置
type ExperimentConfig struct {
	ModelName       string                 `json:"model_name,omitempty"`
	DatasetPath     string                 `json:"dataset_path,omitempty"`
	Hyperparameters map[string]interface{} `json:"hyperparameters,omitempty"`
	Framework       string                 `json:"framework,omitempty"` // pytorch, tensorflow, etc.
	TaskType        string                 `json:"task_type,omitempty"` // classification, generation, etc.
}

// ToModel 转换为数据库模型
func (e *Experiment) ToModel() *models.Experiment {
	config := models.JSON{}
	if e.Config.ModelName != "" {
		config["model_name"] = e.Config.ModelName
	}
	if e.Config.DatasetPath != "" {
		config["dataset_path"] = e.Config.DatasetPath
	}
	if e.Config.Framework != "" {
		config["framework"] = e.Config.Framework
	}
	if e.Config.TaskType != "" {
		config["task_type"] = e.Config.TaskType
	}
	if e.Config.Hyperparameters != nil {
		config["hyperparameters"] = e.Config.Hyperparameters
	}

	return &models.Experiment{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		ProjectID:   e.ProjectID,
		UserID:      e.UserID,
		Config:      config,
		Tags:        e.Tags,
		Status:      string(e.Status),
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// FromModel 从数据库模型转换
func (e *Experiment) FromModel(m *models.Experiment) {
	e.ID = m.ID
	e.Name = m.Name
	e.Description = m.Description
	e.ProjectID = m.ProjectID
	e.UserID = m.UserID
	e.Tags = m.Tags
	e.Status = ExperimentStatus(m.Status)
	e.CreatedAt = m.CreatedAt
	e.UpdatedAt = m.UpdatedAt
	if m.Config != nil {
		if v, ok := m.Config["model_name"].(string); ok {
			e.Config.ModelName = v
		}
		if v, ok := m.Config["dataset_path"].(string); ok {
			e.Config.DatasetPath = v
		}
		if v, ok := m.Config["framework"].(string); ok {
			e.Config.Framework = v
		}
		if v, ok := m.Config["task_type"].(string); ok {
			e.Config.TaskType = v
		}
		if v, ok := m.Config["hyperparameters"].(map[string]interface{}); ok {
			e.Config.Hyperparameters = v
		}
	}
}

// Run 运行记录领域模型
type Run struct {
	ID             uuid.UUID      `json:"id"`
	ExperimentID   uuid.UUID      `json:"experiment_id"`
	RunType        string         `json:"run_type"` // training, inference, simulation
	Status         string         `json:"status"`
	Config         RunConfig      `json:"config"`
	MetricsSummary map[string]float64 `json:"metrics_summary"`
	StartedAt      *time.Time     `json:"started_at"`
	EndedAt        *time.Time     `json:"ended_at"`
	Duration       *int64         `json:"duration"` // seconds
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// RunConfig 运行配置
type RunConfig struct {
	Hyperparameters map[string]interface{} `json:"hyperparameters,omitempty"`
	Resources       map[string]interface{} `json:"resources,omitempty"`
	Environment     map[string]string      `json:"environment,omitempty"`
}

// ToModel 转换为数据库模型
func (r *Run) ToModel() *models.Run {
	config := models.JSON{}
	if r.Config.Hyperparameters != nil {
		config["hyperparameters"] = r.Config.Hyperparameters
	}
	if r.Config.Resources != nil {
		config["resources"] = r.Config.Resources
	}
	if r.Config.Environment != nil {
		config["environment"] = r.Config.Environment
	}

	metricsSummary := models.JSON{}
	for k, v := range r.MetricsSummary {
		metricsSummary[k] = v
	}

	return &models.Run{
		ID:             r.ID,
		ExperimentID:   r.ExperimentID,
		RunType:        r.RunType,
		Status:         r.Status,
		Config:         config,
		MetricsSummary: metricsSummary,
		StartedAt:      r.StartedAt,
		EndedAt:        r.EndedAt,
		Duration:       r.Duration,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// FromModel 从数据库模型转换
func (r *Run) FromModel(m *models.Run) {
	r.ID = m.ID
	r.ExperimentID = m.ExperimentID
	r.RunType = m.RunType
	r.Status = m.Status
	r.StartedAt = m.StartedAt
	r.EndedAt = m.EndedAt
	r.Duration = m.Duration
	r.CreatedAt = m.CreatedAt
	r.UpdatedAt = m.UpdatedAt

	if m.Config != nil {
		r.Config.Hyperparameters, _ = m.Config["hyperparameters"].(map[string]interface{})
		r.Config.Resources, _ = m.Config["resources"].(map[string]interface{})
		r.Config.Environment, _ = m.Config["environment"].(map[string]string)
	}
	if m.MetricsSummary != nil {
		r.MetricsSummary = make(map[string]float64)
		for k, v := range m.MetricsSummary {
			if fv, ok := v.(float64); ok {
				r.MetricsSummary[k] = fv
			}
		}
	}
}

// Metric 指标领域模型
type Metric struct {
	ID        uuid.UUID              `json:"id"`
	RunID     uuid.UUID              `json:"run_id"`
	Key       string                 `json:"key"`
	Value     float64                `json:"value"`
	Step      *int64                 `json:"step"`
	Timestamp *time.Time             `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

// ToModel 转换为数据库模型
func (m *Metric) ToModel() *models.Metric {
	return &models.Metric{
		ID:        m.ID,
		RunID:     m.RunID,
		Key:       m.Key,
		Value:     m.Value,
		Step:      m.Step,
		Timestamp: m.Timestamp,
		Metadata:  models.JSON(m.Metadata),
		CreatedAt: m.CreatedAt,
	}
}

// MetricPoint 指标数据点（用于图表）
type MetricPoint struct {
	Step      int64     `json:"step"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// MetricSeries 指标序列（用于图表）
type MetricSeries struct {
	Key    string        `json:"key"`
	Points []MetricPoint `json:"points"`
}

// Artifact 工件领域模型
type Artifact struct {
	ID          uuid.UUID              `json:"id"`
	RunID       uuid.UUID              `json:"run_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	StoragePath string                 `json:"storage_path"`
	Size        int64                  `json:"size"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// CreateExperimentRequest 创建实验请求
type CreateExperimentRequest struct {
	Name        string           `json:"name" binding:"required,max=255"`
	Description string           `json:"description" binding:"max=1000"`
	ProjectID   uuid.UUID        `json:"project_id" binding:"required"`
	Config      ExperimentConfig `json:"config"`
	Tags        []string         `json:"tags"`
}

// UpdateExperimentRequest 更新实验请求
type UpdateExperimentRequest struct {
	Name        string           `json:"name" binding:"omitempty,max=255"`
	Description string           `json:"description" binding:"max=1000"`
	Config      ExperimentConfig `json:"config"`
	Tags        []string         `json:"tags"`
	Status      string           `json:"status" binding:"omitempty,oneof=running completed failed stopped"`
}

// ListExperimentsRequest 列实验请求
type ListExperimentsRequest struct {
	ProjectID uuid.UUID `form:"project_id"`
	Status    string    `form:"status"`
	Search    string    `form:"search"`
	Page      int       `form:"page,default=1"`
	PageSize  int       `form:"page_size,default=20"`
}

// ExperimentResponse 实验响应
type ExperimentResponse struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	ProjectID   uuid.UUID        `json:"project_id"`
	UserID      uuid.UUID        `json:"user_id"`
	Config      ExperimentConfig `json:"config"`
	Tags        []string         `json:"tags"`
	Status      string           `json:"status"`
	RunsCount   int              `json:"runs_count"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// ToResponse 转换为响应
func (e *Experiment) ToResponse() *ExperimentResponse {
	return &ExperimentResponse{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		ProjectID:   e.ProjectID,
		UserID:      e.UserID,
		Config:      e.Config,
		Tags:        e.Tags,
		Status:      string(e.Status),
		RunsCount:   e.RunsCount,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// RunResponse 运行响应
type RunResponse struct {
	ID             uuid.UUID          `json:"id"`
	ExperimentID   uuid.UUID          `json:"experiment_id"`
	RunType        string             `json:"run_type"`
	Status         string             `json:"status"`
	Config         RunConfig          `json:"config"`
	MetricsSummary map[string]float64 `json:"metrics_summary"`
	StartedAt      *time.Time         `json:"started_at"`
	EndedAt        *time.Time         `json:"ended_at"`
	Duration       *int64             `json:"duration"`
	CreatedAt      time.Time          `json:"created_at"`
}

// ToResponse 转换为响应
func (r *Run) ToResponse() *RunResponse {
	return &RunResponse{
		ID:             r.ID,
		ExperimentID:   r.ExperimentID,
		RunType:        r.RunType,
		Status:         r.Status,
		Config:         r.Config,
		MetricsSummary: r.MetricsSummary,
		StartedAt:      r.StartedAt,
		EndedAt:        r.EndedAt,
		Duration:       r.Duration,
		CreatedAt:      r.CreatedAt,
	}
}

// MetricResponse 指标响应
type MetricResponse struct {
	ID        uuid.UUID              `json:"id"`
	RunID     uuid.UUID              `json:"run_id"`
	Key       string                 `json:"key"`
	Value     float64                `json:"value"`
	Step      *int64                 `json:"step"`
	Timestamp *time.Time             `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

// RecordMetricRequest 记录指标请求
type RecordMetricRequest struct {
	RunID     uuid.UUID              `json:"run_id" binding:"required"`
	Key       string                 `json:"key" binding:"required"`
	Value     float64                `json:"value" binding:"required"`
	Step      *int64                 `json:"step"`
	Timestamp *time.Time             `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// BatchRecordMetricsRequest 批量记录指标请求
type BatchRecordMetricsRequest struct {
	Metrics []RecordMetricRequest `json:"metrics" binding:"required,min=1"`
}

// QueryMetricsRequest 查询指标请求
type QueryMetricsRequest struct {
	RunID     uuid.UUID `form:"run_id" binding:"required"`
	Keys      []string  `form:"keys"`
	StartTime *time.Time `form:"start_time"`
	EndTime   *time.Time `form:"end_time"`
}

// CompareExperimentsRequest 实验对比请求
type CompareExperimentsRequest struct {
	ExperimentIDs []uuid.UUID `json:"experiment_ids" binding:"required,min=2,max=10"`
	MetricKeys    []string    `json:"metric_keys"`
}

// ExperimentComparison 实验对比结果
type ExperimentComparison struct {
	ExperimentID   uuid.UUID              `json:"experiment_id"`
	ExperimentName string                 `json:"experiment_name"`
	Status         string                 `json:"status"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	MetricsSummary map[string]float64     `json:"metrics_summary"`
	Duration       *int64                 `json:"duration"`
	CreatedAt      time.Time              `json:"created_at"`
}

// CompareExperimentsResponse 实验对比响应
type CompareExperimentsResponse struct {
	Experiments []ExperimentComparison `json:"experiments"`
	CommonMetrics []string             `json:"common_metrics"`
}

// MetricChartDataRequest 指标图表数据请求
type MetricChartDataRequest struct {
	ExperimentID uuid.UUID `form:"experiment_id"`
	RunID        uuid.UUID `form:"run_id"`
	MetricKey    string    `form:"metric_key" binding:"required"`
}

// MetricChartData 指标图表数据
type MetricChartData struct {
	MetricKey string        `json:"metric_key"`
	Series    []MetricPoint `json:"series"`
}

// ExperimentReportRequest 实验报表请求
type ExperimentReportRequest struct {
	ExperimentID uuid.UUID `json:"experiment_id" binding:"required"`
}

// ExperimentReport 实验报表
type ExperimentReport struct {
	Experiment   *ExperimentResponse    `json:"experiment"`
	Runs         []RunResponse          `json:"runs"`
	MetricsChart map[string][]MetricPoint `json:"metrics_chart"`
	BestRun      *RunResponse           `json:"best_run,omitempty"`
	Summary      ReportSummary          `json:"summary"`
}

// ReportSummary 报表摘要
type ReportSummary struct {
	TotalRuns       int                `json:"total_runs"`
	CompletedRuns   int                `json:"completed_runs"`
	FailedRuns      int                `json:"failed_runs"`
	AverageDuration int64              `json:"average_duration"`
	BestMetrics     map[string]float64 `json:"best_metrics"`
}

// HyperparameterSearchSpace 超参数搜索空间
type HyperparameterSearchSpace struct {
	LearningRate *FloatRange   `json:"learning_rate,omitempty"`
	BatchSize    []int         `json:"batch_size,omitempty"`
	Epochs       *IntRange     `json:"epochs,omitempty"`
}

// FloatRange 浮点范围
type FloatRange struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Step float64 `json:"step,omitempty"`
}

// IntRange 整数范围
type IntRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
	Step int `json:"step,omitempty"`
}

// HyperparameterComparisonRequest 超参数对比请求
type HyperparameterComparisonRequest struct {
	ExperimentIDs []uuid.UUID `json:"experiment_ids" binding:"required,min=2,max=20"`
}

// HyperparameterComparison 超参数对比结果
type HyperparameterComparison struct {
	ExperimentID   uuid.UUID              `json:"experiment_id"`
	ExperimentName string                 `json:"experiment_name"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	BestMetric     float64                `json:"best_metric"`
	MetricName     string                 `json:"metric_name"`
}
