package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/repository"
	"go.uber.org/zap"
)

var (
	ErrExperimentNotFound = errors.New("experiment not found")
	ErrRunNotFound        = errors.New("run not found")
	ErrMetricNotFound     = errors.New("metric not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
)

// ExperimentService 实验服务接口
type ExperimentService interface {
	CreateExperiment(ctx context.Context, userID uuid.UUID, req *domain.CreateExperimentRequest) (*domain.ExperimentResponse, error)
	GetExperiment(ctx context.Context, id uuid.UUID) (*domain.ExperimentResponse, error)
	ListExperiments(ctx context.Context, req *domain.ListExperimentsRequest) ([]*domain.ExperimentResponse, int64, error)
	UpdateExperiment(ctx context.Context, id uuid.UUID, req *domain.UpdateExperimentRequest) (*domain.ExperimentResponse, error)
	DeleteExperiment(ctx context.Context, id uuid.UUID) error
	GetExperimentRuns(ctx context.Context, experimentID uuid.UUID) ([]*domain.RunResponse, error)
}

// experimentService 实验服务实现
type experimentService struct {
	expRepo     repository.ExperimentRepository
	runRepo     repository.RunRepository
	metricRepo  repository.MetricRepository
	artifactRepo repository.ArtifactRepository
}

// NewExperimentService 创建实验服务
func NewExperimentService(
	expRepo repository.ExperimentRepository,
	runRepo repository.RunRepository,
	metricRepo repository.MetricRepository,
	artifactRepo repository.ArtifactRepository,
) ExperimentService {
	return &experimentService{
		expRepo:      expRepo,
		runRepo:      runRepo,
		metricRepo:   metricRepo,
		artifactRepo: artifactRepo,
	}
}

func (s *experimentService) CreateExperiment(ctx context.Context, userID uuid.UUID, req *domain.CreateExperimentRequest) (*domain.ExperimentResponse, error) {
	logger.Log.Info("Creating experiment", zap.String("name", req.Name), zap.String("user_id", userID.String()))

	exp := &domain.Experiment{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		ProjectID:   req.ProjectID,
		UserID:      userID,
		Config:      req.Config,
		Tags:        req.Tags,
		Status:      domain.ExperimentStatusRunning,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.expRepo.Create(ctx, exp); err != nil {
		logger.Log.Error("Failed to create experiment", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Experiment created successfully", zap.String("id", exp.ID.String()))
	return exp.ToResponse(), nil
}

func (s *experimentService) GetExperiment(ctx context.Context, id uuid.UUID) (*domain.ExperimentResponse, error) {
	logger.Log.Debug("Getting experiment", zap.String("id", id.String()))

	exp, err := s.expRepo.GetByIDWithRuns(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrExperimentNotFound) {
			return nil, ErrExperimentNotFound
		}
		return nil, err
	}

	return exp.ToResponse(), nil
}

func (s *experimentService) ListExperiments(ctx context.Context, req *domain.ListExperimentsRequest) ([]*domain.ExperimentResponse, int64, error) {
	logger.Log.Debug("Listing experiments", zap.Int("page", req.Page), zap.Int("page_size", req.PageSize))

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	experiments, total, err := s.expRepo.List(ctx, req)
	if err != nil {
		logger.Log.Error("Failed to list experiments", zap.Error(err))
		return nil, 0, err
	}

	responses := make([]*domain.ExperimentResponse, len(experiments))
	for i, exp := range experiments {
		responses[i] = exp.ToResponse()
	}

	return responses, total, nil
}

func (s *experimentService) UpdateExperiment(ctx context.Context, id uuid.UUID, req *domain.UpdateExperimentRequest) (*domain.ExperimentResponse, error) {
	logger.Log.Info("Updating experiment", zap.String("id", id.String()))

	exp, err := s.expRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrExperimentNotFound) {
			return nil, ErrExperimentNotFound
		}
		return nil, err
	}

	if req.Name != "" {
		exp.Name = req.Name
	}
	if req.Description != "" {
		exp.Description = req.Description
	}
	if req.Status != "" {
		exp.Status = domain.ExperimentStatus(req.Status)
	}
	if len(req.Tags) > 0 {
		exp.Tags = req.Tags
	}
	if req.Config.ModelName != "" || req.Config.Framework != "" {
		exp.Config = req.Config
	}

	if err := s.expRepo.Update(ctx, exp); err != nil {
		logger.Log.Error("Failed to update experiment", zap.Error(err))
		return nil, err
	}

	return exp.ToResponse(), nil
}

func (s *experimentService) DeleteExperiment(ctx context.Context, id uuid.UUID) error {
	logger.Log.Info("Deleting experiment", zap.String("id", id.String()))

	if err := s.expRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrExperimentNotFound) {
			return ErrExperimentNotFound
		}
		return err
	}

	return nil
}

func (s *experimentService) GetExperimentRuns(ctx context.Context, experimentID uuid.UUID) ([]*domain.RunResponse, error) {
	logger.Log.Debug("Getting experiment runs", zap.String("experiment_id", experimentID.String()))

	runs, err := s.runRepo.GetByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, err
	}

	responses := make([]*domain.RunResponse, len(runs))
	for i, run := range runs {
		responses[i] = run.ToResponse()
	}

	return responses, nil
}

// RunService 运行记录服务接口
type RunService interface {
	CreateRun(ctx context.Context, experimentID uuid.UUID, runType string, config domain.RunConfig) (*domain.RunResponse, error)
	GetRun(ctx context.Context, id uuid.UUID) (*domain.RunResponse, error)
	UpdateRunStatus(ctx context.Context, id uuid.UUID, status string) error
	CompleteRun(ctx context.Context, id uuid.UUID, metricsSummary map[string]float64) error
}

// runService 运行记录服务实现
type runService struct {
	runRepo    repository.RunRepository
	expRepo    repository.ExperimentRepository
	metricRepo repository.MetricRepository
}

// NewRunService 创建运行记录服务
func NewRunService(
	runRepo repository.RunRepository,
	expRepo repository.ExperimentRepository,
	metricRepo repository.MetricRepository,
) RunService {
	return &runService{
		runRepo:    runRepo,
		expRepo:    expRepo,
		metricRepo: metricRepo,
	}
}

func (s *runService) CreateRun(ctx context.Context, experimentID uuid.UUID, runType string, config domain.RunConfig) (*domain.RunResponse, error) {
	logger.Log.Info("Creating run", zap.String("experiment_id", experimentID.String()), zap.String("type", runType))

	// 验证实验存在
	_, err := s.expRepo.GetByID(ctx, experimentID)
	if err != nil {
		if errors.Is(err, repository.ErrExperimentNotFound) {
			return nil, ErrExperimentNotFound
		}
		return nil, err
	}

	run := &domain.Run{
		ID:           uuid.New(),
		ExperimentID: experimentID,
		RunType:      runType,
		Status:       "pending",
		Config:       config,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.runRepo.Create(ctx, run); err != nil {
		logger.Log.Error("Failed to create run", zap.Error(err))
		return nil, err
	}

	// 更新实验运行计数
	if err := s.expRepo.UpdateRunsCount(ctx, experimentID); err != nil {
		logger.Log.Warn("Failed to update runs count", zap.Error(err))
	}

	return run.ToResponse(), nil
}

func (s *runService) GetRun(ctx context.Context, id uuid.UUID) (*domain.RunResponse, error) {
	run, err := s.runRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrRunNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}

	return run.ToResponse(), nil
}

func (s *runService) UpdateRunStatus(ctx context.Context, id uuid.UUID, status string) error {
	logger.Log.Info("Updating run status", zap.String("id", id.String()), zap.String("status", status))

	if err := s.runRepo.UpdateStatus(ctx, id, status); err != nil {
		if errors.Is(err, repository.ErrRunNotFound) {
			return ErrRunNotFound
		}
		return err
	}

	return nil
}

func (s *runService) CompleteRun(ctx context.Context, id uuid.UUID, metricsSummary map[string]float64) error {
	logger.Log.Info("Completing run", zap.String("id", id.String()))

	run, err := s.runRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrRunNotFound) {
			return ErrRunNotFound
		}
		return err
	}

	now := time.Now()
	run.Status = "completed"
	run.MetricsSummary = metricsSummary
	run.EndedAt = &now
	if run.StartedAt != nil {
		duration := int64(now.Sub(*run.StartedAt).Seconds())
		run.Duration = &duration
	}

	if err := s.runRepo.Update(ctx, run); err != nil {
		return err
	}

	return nil
}

// MetricService 指标服务接口
type MetricService interface {
	RecordMetric(ctx context.Context, req *domain.RecordMetricRequest) error
	BatchRecordMetrics(ctx context.Context, req *domain.BatchRecordMetricsRequest) error
	GetRunMetrics(ctx context.Context, runID uuid.UUID, keys []string) ([]*domain.MetricResponse, error)
	GetMetricSeries(ctx context.Context, runID uuid.UUID, key string) (*domain.MetricSeries, error)
	QueryMetrics(ctx context.Context, req *domain.QueryMetricsRequest) (map[string][]*domain.MetricResponse, error)
}

// metricService 指标服务实现
type metricService struct {
	metricRepo repository.MetricRepository
	runRepo    repository.RunRepository
}

// NewMetricService 创建指标服务
func NewMetricService(metricRepo repository.MetricRepository, runRepo repository.RunRepository) MetricService {
	return &metricService{
		metricRepo: metricRepo,
		runRepo:    runRepo,
	}
}

func (s *metricService) RecordMetric(ctx context.Context, req *domain.RecordMetricRequest) error {
	metric := &domain.Metric{
		ID:        uuid.New(),
		RunID:     req.RunID,
		Key:       req.Key,
		Value:     req.Value,
		Step:      req.Step,
		Timestamp: req.Timestamp,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
	}

	if metric.Timestamp == nil {
		now := time.Now()
		metric.Timestamp = &now
	}

	return s.metricRepo.Create(ctx, metric)
}

func (s *metricService) BatchRecordMetrics(ctx context.Context, req *domain.BatchRecordMetricsRequest) error {
	metrics := make([]*domain.Metric, len(req.Metrics))
	now := time.Now()

	for i, m := range req.Metrics {
		metrics[i] = &domain.Metric{
			ID:        uuid.New(),
			RunID:     m.RunID,
			Key:       m.Key,
			Value:     m.Value,
			Step:      m.Step,
			Timestamp: m.Timestamp,
			Metadata:  m.Metadata,
			CreatedAt: now,
		}
		if metrics[i].Timestamp == nil {
			metrics[i].Timestamp = &now
		}
	}

	return s.metricRepo.BatchCreate(ctx, metrics)
}

func (s *metricService) GetRunMetrics(ctx context.Context, runID uuid.UUID, keys []string) ([]*domain.MetricResponse, error) {
	metrics, err := s.metricRepo.GetByRunID(ctx, runID, keys)
	if err != nil {
		return nil, err
	}

	responses := make([]*domain.MetricResponse, len(metrics))
	for i, m := range metrics {
		responses[i] = &domain.MetricResponse{
			ID:        m.ID,
			RunID:     m.RunID,
			Key:       m.Key,
			Value:     m.Value,
			Step:      m.Step,
			Timestamp: m.Timestamp,
			Metadata:  m.Metadata,
			CreatedAt: m.CreatedAt,
		}
	}

	return responses, nil
}

func (s *metricService) GetMetricSeries(ctx context.Context, runID uuid.UUID, key string) (*domain.MetricSeries, error) {
	metrics, err := s.metricRepo.GetByRunIDAndKey(ctx, runID, key)
	if err != nil {
		return nil, err
	}

	points := make([]domain.MetricPoint, len(metrics))
	for i, m := range metrics {
		var step int64
		if m.Step != nil {
			step = *m.Step
		}
		timestamp := m.CreatedAt
		if m.Timestamp != nil {
			timestamp = *m.Timestamp
		}
		points[i] = domain.MetricPoint{
			Step:      step,
			Value:     m.Value,
			Timestamp: timestamp,
		}
	}

	return &domain.MetricSeries{
		Key:    key,
		Points: points,
	}, nil
}

func (s *metricService) QueryMetrics(ctx context.Context, req *domain.QueryMetricsRequest) (map[string][]*domain.MetricResponse, error) {
	metrics, err := s.metricRepo.QueryMetrics(ctx, req.RunID, req.Keys, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]*domain.MetricResponse)
	for _, m := range metrics {
		resp := &domain.MetricResponse{
			ID:        m.ID,
			RunID:     m.RunID,
			Key:       m.Key,
			Value:     m.Value,
			Step:      m.Step,
			Timestamp: m.Timestamp,
			Metadata:  m.Metadata,
			CreatedAt: m.CreatedAt,
		}
		result[m.Key] = append(result[m.Key], resp)
	}

	return result, nil
}

// VisualizationService 可视化服务接口
type VisualizationService interface {
	GetLossCurve(ctx context.Context, runID uuid.UUID) (*domain.MetricChartData, error)
	GetAccuracyTrend(ctx context.Context, runID uuid.UUID) (*domain.MetricChartData, error)
	CompareExperiments(ctx context.Context, req *domain.CompareExperimentsRequest) (*domain.CompareExperimentsResponse, error)
	CompareHyperparameters(ctx context.Context, req *domain.HyperparameterComparisonRequest) (*domain.HyperparameterComparison, error)
	GetExperimentReport(ctx context.Context, experimentID uuid.UUID) (*domain.ExperimentReport, error)
}

// visualizationService 可视化服务实现
type visualizationService struct {
	expRepo    repository.ExperimentRepository
	runRepo    repository.RunRepository
	metricRepo repository.MetricRepository
}

// NewVisualizationService 创建可视化服务
func NewVisualizationService(
	expRepo repository.ExperimentRepository,
	runRepo repository.RunRepository,
	metricRepo repository.MetricRepository,
) VisualizationService {
	return &visualizationService{
		expRepo:    expRepo,
		runRepo:    runRepo,
		metricRepo: metricRepo,
	}
}

func (s *visualizationService) GetLossCurve(ctx context.Context, runID uuid.UUID) (*domain.MetricChartData, error) {
	// 尝试获取训练损失
	series, err := s.getMetricSeriesForRun(ctx, runID, "loss")
	if err != nil {
		// 尝试其他常见的损失指标名称
		series, err = s.getMetricSeriesForRun(ctx, runID, "train_loss")
		if err != nil {
			series, err = s.getMetricSeriesForRun(ctx, runID, "training_loss")
		}
	}
	return series, err
}

func (s *visualizationService) GetAccuracyTrend(ctx context.Context, runID uuid.UUID) (*domain.MetricChartData, error) {
	// 尝试获取准确率
	series, err := s.getMetricSeriesForRun(ctx, runID, "accuracy")
	if err != nil {
		series, err = s.getMetricSeriesForRun(ctx, runID, "acc")
		if err != nil {
			series, err = s.getMetricSeriesForRun(ctx, runID, "val_accuracy")
		}
	}
	return series, err
}

func (s *visualizationService) getMetricSeriesForRun(ctx context.Context, runID uuid.UUID, key string) (*domain.MetricChartData, error) {
	metrics, err := s.metricRepo.GetByRunIDAndKey(ctx, runID, key)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("metric %s not found", key)
	}

	points := make([]domain.MetricPoint, len(metrics))
	for i, m := range metrics {
		var step int64
		if m.Step != nil {
			step = *m.Step
		}
		timestamp := m.CreatedAt
		if m.Timestamp != nil {
			timestamp = *m.Timestamp
		}
		points[i] = domain.MetricPoint{
			Step:      step,
			Value:     m.Value,
			Timestamp: timestamp,
		}
	}

	return &domain.MetricChartData{
		MetricKey: key,
		Series:    points,
	}, nil
}

func (s *visualizationService) CompareExperiments(ctx context.Context, req *domain.CompareExperimentsRequest) (*domain.CompareExperimentsResponse, error) {
	if len(req.ExperimentIDs) < 2 {
		return nil, errors.New("at least 2 experiments required for comparison")
	}

	comparisons := make([]domain.ExperimentComparison, 0, len(req.ExperimentIDs))
	commonMetrics := make(map[string]int)

	for _, expID := range req.ExperimentIDs {
		exp, err := s.expRepo.GetByIDWithRuns(ctx, expID)
		if err != nil {
			logger.Log.Warn("Failed to get experiment for comparison", zap.String("id", expID.String()), zap.Error(err))
			continue
		}

		// 获取最新运行
		runs, err := s.runRepo.GetByExperimentID(ctx, expID)
		if err != nil {
			logger.Log.Warn("Failed to get runs for comparison", zap.String("experiment_id", expID.String()), zap.Error(err))
		}

		var duration *int64
		var metricsSummary map[string]float64
		if len(runs) > 0 {
			duration = runs[0].Duration
			metricsSummary = runs[0].MetricsSummary
			for key := range metricsSummary {
				commonMetrics[key]++
			}
		}

		comparison := domain.ExperimentComparison{
			ExperimentID:    exp.ID,
			ExperimentName:  exp.Name,
			Status:          string(exp.Status),
			Hyperparameters: exp.Config.Hyperparameters,
			MetricsSummary:  metricsSummary,
			Duration:        duration,
			CreatedAt:       exp.CreatedAt,
		}

		comparisons = append(comparisons, comparison)
	}

	// 找出共同指标
	var commonMetricKeys []string
	for key, count := range commonMetrics {
		if count == len(req.ExperimentIDs) {
			commonMetricKeys = append(commonMetricKeys, key)
		}
	}

	return &domain.CompareExperimentsResponse{
		Experiments:   comparisons,
		CommonMetrics: commonMetricKeys,
	}, nil
}

func (s *visualizationService) CompareHyperparameters(ctx context.Context, req *domain.HyperparameterComparisonRequest) (*domain.HyperparameterComparison, error) {
	// 获取所有实验并提取超参数
	// 简化实现：返回第一个实验的超参数作为示例
	if len(req.ExperimentIDs) == 0 {
		return nil, errors.New("no experiments provided")
	}

	exp, err := s.expRepo.GetByID(ctx, req.ExperimentIDs[0])
	if err != nil {
		return nil, err
	}

	return &domain.HyperparameterComparison{
		ExperimentID:    exp.ID,
		ExperimentName:  exp.Name,
		Hyperparameters: exp.Config.Hyperparameters,
	}, nil
}

func (s *visualizationService) GetExperimentReport(ctx context.Context, experimentID uuid.UUID) (*domain.ExperimentReport, error) {
	exp, err := s.expRepo.GetByID(ctx, experimentID)
	if err != nil {
		return nil, err
	}

	runs, err := s.runRepo.GetByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, err
	}

	// 获取所有运行的指标
	metricsChart := make(map[string][]domain.MetricPoint)
	runResponses := make([]domain.RunResponse, len(runs))

	var totalDuration int64
	var completedRuns, failedRuns int
	var bestRun *domain.Run
	bestMetric := -1.0

	for i, run := range runs {
		runResponses[i] = *run.ToResponse()

		if run.Duration != nil {
			totalDuration += *run.Duration
		}

		switch run.Status {
		case "completed":
			completedRuns++
		case "failed":
			failedRuns++
		}

		// 假设 accuracy 是主要评估指标
		if acc, ok := run.MetricsSummary["accuracy"]; ok {
			if acc > bestMetric {
				bestMetric = acc
				bestRun = run
			}
		}

		// 获取损失曲线数据
		metrics, _ := s.metricRepo.GetByRunIDAndKey(ctx, run.ID, "loss")
		for _, m := range metrics {
			var step int64
			if m.Step != nil {
				step = *m.Step
			}
			metricsChart["loss"] = append(metricsChart["loss"], domain.MetricPoint{
				Step:      step,
				Value:     m.Value,
				Timestamp: m.CreatedAt,
			})
		}
	}

	summary := domain.ReportSummary{
		TotalRuns:     len(runs),
		CompletedRuns: completedRuns,
		FailedRuns:    failedRuns,
		BestMetrics:   make(map[string]float64),
	}

	if len(runs) > 0 {
		summary.AverageDuration = totalDuration / int64(len(runs))
	}
	if bestMetric >= 0 {
		summary.BestMetrics["accuracy"] = bestMetric
	}

	var bestRunResponse *domain.RunResponse
	if bestRun != nil {
		bestRunResponse = bestRun.ToResponse()
	}

	return &domain.ExperimentReport{
		Experiment:   exp.ToResponse(),
		Runs:         runResponses,
		MetricsChart: metricsChart,
		BestRun:      bestRunResponse,
		Summary:      summary,
	}, nil
}

// MLflowService MLflow 集成服务接口
type MLflowService interface {
	LogParams(ctx context.Context, runID uuid.UUID, params map[string]string) error
	LogMetrics(ctx context.Context, runID uuid.UUID, metrics map[string]float64, step int64) error
	LogArtifact(ctx context.Context, runID uuid.UUID, localPath, artifactPath string) error
	RegisterModel(ctx context.Context, modelName string, runID uuid.UUID, artifactPath string) error
}

// mlflowService MLflow 集成服务实现
type mlflowService struct {
	enabled    bool
	trackingURI string
}

// NewMLflowService 创建 MLflow 服务
func NewMLflowService(enabled bool, trackingURI string) MLflowService {
	return &mlflowService{
		enabled:     enabled,
		trackingURI: trackingURI,
	}
}

func (s *mlflowService) LogParams(ctx context.Context, runID uuid.UUID, params map[string]string) error {
	if !s.enabled {
		return nil
	}
	// TODO: 实现 MLflow API 调用
	logger.Log.Debug("MLflow: LogParams", zap.String("run_id", runID.String()), zap.Int("params_count", len(params)))
	return nil
}

func (s *mlflowService) LogMetrics(ctx context.Context, runID uuid.UUID, metrics map[string]float64, step int64) error {
	if !s.enabled {
		return nil
	}
	// TODO: 实现 MLflow API 调用
	logger.Log.Debug("MLflow: LogMetrics", zap.String("run_id", runID.String()), zap.Int("metrics_count", len(metrics)))
	return nil
}

func (s *mlflowService) LogArtifact(ctx context.Context, runID uuid.UUID, localPath, artifactPath string) error {
	if !s.enabled {
		return nil
	}
	// TODO: 实现 MLflow API 调用
	logger.Log.Debug("MLflow: LogArtifact", zap.String("run_id", runID.String()), zap.String("path", localPath))
	return nil
}

func (s *mlflowService) RegisterModel(ctx context.Context, modelName string, runID uuid.UUID, artifactPath string) error {
	if !s.enabled {
		return nil
	}
	// TODO: 实现 MLflow API 调用
	logger.Log.Debug("MLflow: RegisterModel", zap.String("model_name", modelName), zap.String("run_id", runID.String()))
	return nil
}

// MapServiceError 将服务错误映射为 HTTP 状态码
func MapServiceError(err error) int {
	switch {
	case errors.Is(err, ErrExperimentNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrRunNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrMetricNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
