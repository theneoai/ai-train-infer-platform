package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/models"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrExperimentNotFound = errors.New("experiment not found")
	ErrRunNotFound        = errors.New("run not found")
	ErrMetricNotFound     = errors.New("metric not found")
)

// ExperimentRepository 实验仓库接口
type ExperimentRepository interface {
	Create(ctx context.Context, exp *domain.Experiment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error)
	GetByIDWithRuns(ctx context.Context, id uuid.UUID) (*domain.Experiment, error)
	Update(ctx context.Context, exp *domain.Experiment) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, req *domain.ListExperimentsRequest) ([]*domain.Experiment, int64, error)
	CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	UpdateRunsCount(ctx context.Context, experimentID uuid.UUID) error
}

// experimentRepository 实验仓库实现
type experimentRepository struct {
	db *gorm.DB
}

// NewExperimentRepository 创建实验仓库
func NewExperimentRepository(db *gorm.DB) ExperimentRepository {
	return &experimentRepository{db: db}
}

func (r *experimentRepository) Create(ctx context.Context, exp *domain.Experiment) error {
	logger.Log.Debug("Creating experiment", zap.String("name", exp.Name))

	model := exp.ToModel()
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Log.Error("Failed to create experiment", zap.Error(err))
		return err
	}

	exp.ID = model.ID
	exp.CreatedAt = model.CreatedAt
	exp.UpdatedAt = model.UpdatedAt

	logger.Log.Info("Experiment created", zap.String("id", exp.ID.String()))
	return nil
}

func (r *experimentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error) {
	var model models.Experiment
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExperimentNotFound
		}
		logger.Log.Error("Failed to get experiment", zap.String("id", id.String()), zap.Error(err))
		return nil, err
	}

	exp := &domain.Experiment{}
	exp.FromModel(&model)
	return exp, nil
}

func (r *experimentRepository) GetByIDWithRuns(ctx context.Context, id uuid.UUID) (*domain.Experiment, error) {
	var model models.Experiment
	if err := r.db.WithContext(ctx).Preload("Runs").First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExperimentNotFound
		}
		logger.Log.Error("Failed to get experiment with runs", zap.String("id", id.String()), zap.Error(err))
		return nil, err
	}

	exp := &domain.Experiment{}
	exp.FromModel(&model)
	exp.RunsCount = len(model.Runs)
	return exp, nil
}

func (r *experimentRepository) Update(ctx context.Context, exp *domain.Experiment) error {
	logger.Log.Debug("Updating experiment", zap.String("id", exp.ID.String()))

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if exp.Name != "" {
		updates["name"] = exp.Name
	}
	if exp.Description != "" {
		updates["description"] = exp.Description
	}
	if len(exp.Tags) > 0 {
		updates["tags"] = exp.Tags
	}
	if exp.Status != "" {
		updates["status"] = string(exp.Status)
	}
	if exp.Config.ModelName != "" || exp.Config.Framework != "" {
		updates["config"] = models.JSON(exp.Config)
	}

	result := r.db.WithContext(ctx).Model(&models.Experiment{}).Where("id = ?", exp.ID).Updates(updates)
	if result.Error != nil {
		logger.Log.Error("Failed to update experiment", zap.String("id", exp.ID.String()), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrExperimentNotFound
	}

	logger.Log.Info("Experiment updated", zap.String("id", exp.ID.String()))
	return nil
}

func (r *experimentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	logger.Log.Debug("Deleting experiment", zap.String("id", id.String()))

	result := r.db.WithContext(ctx).Delete(&models.Experiment{}, "id = ?", id)
	if result.Error != nil {
		logger.Log.Error("Failed to delete experiment", zap.String("id", id.String()), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrExperimentNotFound
	}

	logger.Log.Info("Experiment deleted", zap.String("id", id.String()))
	return nil
}

func (r *experimentRepository) List(ctx context.Context, req *domain.ListExperimentsRequest) ([]*domain.Experiment, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&models.Experiment{})

	if req.ProjectID != uuid.Nil {
		query = query.Where("project_id = ?", req.ProjectID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", req.Search)
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		logger.Log.Error("Failed to count experiments", zap.Error(err))
		return nil, 0, err
	}

	var models []models.Experiment
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&models).Error; err != nil {
		logger.Log.Error("Failed to list experiments", zap.Error(err))
		return nil, 0, err
	}

	experiments := make([]*domain.Experiment, len(models))
	for i, m := range models {
		exp := &domain.Experiment{}
		exp.FromModel(&m)
		experiments[i] = exp
	}

	return experiments, total, nil
}

func (r *experimentRepository) CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Experiment{}).Where("project_id = ?", projectID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *experimentRepository) UpdateRunsCount(ctx context.Context, experimentID uuid.UUID) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Run{}).Where("experiment_id = ?", experimentID).Count(&count).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&models.Experiment{}).Where("id = ?", experimentID).Update("runs_count", count).Error
}

// RunRepository 运行记录仓库接口
type RunRepository interface {
	Create(ctx context.Context, run *domain.Run) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	GetByExperimentID(ctx context.Context, experimentID uuid.UUID) ([]*domain.Run, error)
	Update(ctx context.Context, run *domain.Run) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateMetricsSummary(ctx context.Context, id uuid.UUID, summary map[string]float64) error
}

// runRepository 运行记录仓库实现
type runRepository struct {
	db *gorm.DB
}

// NewRunRepository 创建运行记录仓库
func NewRunRepository(db *gorm.DB) RunRepository {
	return &runRepository{db: db}
}

func (r *runRepository) Create(ctx context.Context, run *domain.Run) error {
	logger.Log.Debug("Creating run", zap.String("experiment_id", run.ExperimentID.String()))

	model := run.ToModel()
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Log.Error("Failed to create run", zap.Error(err))
		return err
	}

	run.ID = model.ID
	run.CreatedAt = model.CreatedAt
	run.UpdatedAt = model.UpdatedAt

	logger.Log.Info("Run created", zap.String("id", run.ID.String()))
	return nil
}

func (r *runRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	var model models.Run
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRunNotFound
		}
		logger.Log.Error("Failed to get run", zap.String("id", id.String()), zap.Error(err))
		return nil, err
	}

	run := &domain.Run{}
	run.FromModel(&model)
	return run, nil
}

func (r *runRepository) GetByExperimentID(ctx context.Context, experimentID uuid.UUID) ([]*domain.Run, error) {
	var models []models.Run
	if err := r.db.WithContext(ctx).Where("experiment_id = ?", experimentID).
		Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Log.Error("Failed to get runs", zap.String("experiment_id", experimentID.String()), zap.Error(err))
		return nil, err
	}

	runs := make([]*domain.Run, len(models))
	for i, m := range models {
		run := &domain.Run{}
		run.FromModel(&m)
		runs[i] = run
	}

	return runs, nil
}

func (r *runRepository) Update(ctx context.Context, run *domain.Run) error {
	logger.Log.Debug("Updating run", zap.String("id", run.ID.String()))

	result := r.db.WithContext(ctx).Model(&models.Run{}).Where("id = ?", run.ID).Updates(map[string]interface{}{
		"status":          run.Status,
		"metrics_summary": models.JSON(run.MetricsSummary),
		"started_at":      run.StartedAt,
		"ended_at":        run.EndedAt,
		"duration":        run.Duration,
		"updated_at":      time.Now(),
	})

	if result.Error != nil {
		logger.Log.Error("Failed to update run", zap.String("id", run.ID.String()), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (r *runRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "running" {
		now := time.Now()
		updates["started_at"] = &now
	}
	if status == "completed" || status == "failed" || status == "stopped" {
		now := time.Now()
		updates["ended_at"] = &now
	}

	result := r.db.WithContext(ctx).Model(&models.Run{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (r *runRepository) UpdateMetricsSummary(ctx context.Context, id uuid.UUID, summary map[string]float64) error {
	result := r.db.WithContext(ctx).Model(&models.Run{}).Where("id = ?", id).Updates(map[string]interface{}{
		"metrics_summary": models.JSON(summary),
		"updated_at":      time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrRunNotFound
	}

	return nil
}

// MetricRepository 指标仓库接口
type MetricRepository interface {
	Create(ctx context.Context, metric *domain.Metric) error
	BatchCreate(ctx context.Context, metrics []*domain.Metric) error
	GetByRunID(ctx context.Context, runID uuid.UUID, keys []string) ([]*domain.Metric, error)
	GetByRunIDAndKey(ctx context.Context, runID uuid.UUID, key string) ([]*domain.Metric, error)
	GetLatestByRunID(ctx context.Context, runID uuid.UUID, limit int) ([]*domain.Metric, error)
	GetMetricKeysByRunID(ctx context.Context, runID uuid.UUID) ([]string, error)
	QueryMetrics(ctx context.Context, runID uuid.UUID, keys []string, startTime, endTime *time.Time) ([]*domain.Metric, error)
}

// metricRepository 指标仓库实现
type metricRepository struct {
	db *gorm.DB
}

// NewMetricRepository 创建指标仓库
func NewMetricRepository(db *gorm.DB) MetricRepository {
	return &metricRepository{db: db}
}

func (r *metricRepository) Create(ctx context.Context, metric *domain.Metric) error {
	model := metric.ToModel()
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Log.Error("Failed to create metric", zap.Error(err))
		return err
	}
	metric.ID = model.ID
	metric.CreatedAt = model.CreatedAt
	return nil
}

func (r *metricRepository) BatchCreate(ctx context.Context, metrics []*domain.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	models := make([]*models.Metric, len(metrics))
	for i, m := range metrics {
		models[i] = m.ToModel()
	}

	if err := r.db.WithContext(ctx).CreateInBatches(models, 100).Error; err != nil {
		logger.Log.Error("Failed to batch create metrics", zap.Int("count", len(metrics)), zap.Error(err))
		return err
	}

	for i, m := range models {
		metrics[i].ID = m.ID
		metrics[i].CreatedAt = m.CreatedAt
	}

	return nil
}

func (r *metricRepository) GetByRunID(ctx context.Context, runID uuid.UUID, keys []string) ([]*domain.Metric, error) {
	query := r.db.WithContext(ctx).Where("run_id = ?", runID)
	if len(keys) > 0 {
		query = query.Where("key IN ?", keys)
	}

	var models []models.Metric
	if err := query.Order("step ASC, timestamp ASC").Find(&models).Error; err != nil {
		logger.Log.Error("Failed to get metrics", zap.String("run_id", runID.String()), zap.Error(err))
		return nil, err
	}

	metrics := make([]*domain.Metric, len(models))
	for i, m := range models {
		metrics[i] = &domain.Metric{
			ID:        m.ID,
			RunID:     m.RunID,
			Key:       m.Key,
			Value:     m.Value,
			Step:      m.Step,
			Timestamp: m.Timestamp,
			CreatedAt: m.CreatedAt,
		}
		if m.Metadata != nil {
			metrics[i].Metadata, _ = m.Metadata.(map[string]interface{})
		}
	}

	return metrics, nil
}

func (r *metricRepository) GetByRunIDAndKey(ctx context.Context, runID uuid.UUID, key string) ([]*domain.Metric, error) {
	var models []models.Metric
	if err := r.db.WithContext(ctx).Where("run_id = ? AND key = ?", runID, key).
		Order("step ASC, timestamp ASC").Find(&models).Error; err != nil {
		logger.Log.Error("Failed to get metrics by key", zap.String("run_id", runID.String()), zap.String("key", key), zap.Error(err))
		return nil, err
	}

	metrics := make([]*domain.Metric, len(models))
	for i, m := range models {
		metrics[i] = &domain.Metric{
			ID:        m.ID,
			RunID:     m.RunID,
			Key:       m.Key,
			Value:     m.Value,
			Step:      m.Step,
			Timestamp: m.Timestamp,
			CreatedAt: m.CreatedAt,
		}
	}

	return metrics, nil
}

func (r *metricRepository) GetLatestByRunID(ctx context.Context, runID uuid.UUID, limit int) ([]*domain.Metric, error) {
	var models []models.Metric
	if err := r.db.WithContext(ctx).Where("run_id = ?", runID).
		Order("created_at DESC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	metrics := make([]*domain.Metric, len(models))
	for i, m := range models {
		metrics[i] = &domain.Metric{
			ID:        m.ID,
			RunID:     m.RunID,
			Key:       m.Key,
			Value:     m.Value,
			Step:      m.Step,
			Timestamp: m.Timestamp,
			CreatedAt: m.CreatedAt,
		}
	}

	return metrics, nil
}

func (r *metricRepository) GetMetricKeysByRunID(ctx context.Context, runID uuid.UUID) ([]string, error) {
	var keys []string
	if err := r.db.WithContext(ctx).Model(&models.Metric{}).
		Where("run_id = ?", runID).
		Distinct("key").
		Pluck("key", &keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *metricRepository) QueryMetrics(ctx context.Context, runID uuid.UUID, keys []string, startTime, endTime *time.Time) ([]*domain.Metric, error) {
	query := r.db.WithContext(ctx).Where("run_id = ?", runID)

	if len(keys) > 0 {
		query = query.Where("key IN ?", keys)
	}
	if startTime != nil {
		query = query.Where("timestamp >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("timestamp <= ?", *endTime)
	}

	var models []models.Metric
	if err := query.Order("timestamp ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	metrics := make([]*domain.Metric, len(models))
	for i, m := range models {
		metrics[i] = &domain.Metric{
			ID:        m.ID,
			RunID:     m.RunID,
			Key:       m.Key,
			Value:     m.Value,
			Step:      m.Step,
			Timestamp: m.Timestamp,
			CreatedAt: m.CreatedAt,
		}
	}

	return metrics, nil
}

// ArtifactRepository 工件仓库接口
type ArtifactRepository interface {
	Create(ctx context.Context, artifact *domain.Artifact) error
	GetByRunID(ctx context.Context, runID uuid.UUID) ([]*domain.Artifact, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error)
}

// artifactRepository 工件仓库实现
type artifactRepository struct {
	db *gorm.DB
}

// NewArtifactRepository 创建工件仓库
func NewArtifactRepository(db *gorm.DB) ArtifactRepository {
	return &artifactRepository{db: db}
}

func (r *artifactRepository) Create(ctx context.Context, artifact *domain.Artifact) error {
	model := &models.Artifact{
		ID:          artifact.ID,
		RunID:       artifact.RunID,
		Name:        artifact.Name,
		Type:        artifact.Type,
		StoragePath: artifact.StoragePath,
		Size:        artifact.Size,
		Metadata:    models.JSON(artifact.Metadata),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Log.Error("Failed to create artifact", zap.Error(err))
		return err
	}

	artifact.ID = model.ID
	artifact.CreatedAt = model.CreatedAt
	return nil
}

func (r *artifactRepository) GetByRunID(ctx context.Context, runID uuid.UUID) ([]*domain.Artifact, error) {
	var models []models.Artifact
	if err := r.db.WithContext(ctx).Where("run_id = ?", runID).Find(&models).Error; err != nil {
		return nil, err
	}

	artifacts := make([]*domain.Artifact, len(models))
	for i, m := range models {
		artifacts[i] = &domain.Artifact{
			ID:          m.ID,
			RunID:       m.RunID,
			Name:        m.Name,
			Type:        m.Type,
			StoragePath: m.StoragePath,
			Size:        m.Size,
			CreatedAt:   m.CreatedAt,
		}
		if m.Metadata != nil {
			artifacts[i].Metadata, _ = m.Metadata.(map[string]interface{})
		}
	}

	return artifacts, nil
}

func (r *artifactRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error) {
	var model models.Artifact
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("artifact not found")
		}
		return nil, err
	}

	artifact := &domain.Artifact{
		ID:          model.ID,
		RunID:       model.RunID,
		Name:        model.Name,
		Type:        model.Type,
		StoragePath: model.StoragePath,
		Size:        model.Size,
		CreatedAt:   model.CreatedAt,
	}
	if model.Metadata != nil {
		artifact.Metadata, _ = model.Metadata.(map[string]interface{})
	}

	return artifact, nil
}
