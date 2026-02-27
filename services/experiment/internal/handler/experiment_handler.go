package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/service"
	"go.uber.org/zap"
)

// ExperimentHandler 实验处理器
type ExperimentHandler struct {
	expService service.ExperimentService
	runService service.RunService
	metricService service.MetricService
	vizService service.VisualizationService
}

// NewExperimentHandler 创建实验处理器
func NewExperimentHandler(
	expService service.ExperimentService,
	runService service.RunService,
	metricService service.MetricService,
	vizService service.VisualizationService,
) *ExperimentHandler {
	return &ExperimentHandler{
		expService: expService,
		runService: runService,
		metricService: metricService,
		vizService: vizService,
	}
}

// CreateExperiment 创建实验
// POST /api/v1/experiments
func (h *ExperimentHandler) CreateExperiment(c *gin.Context) {
	var req domain.CreateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Invalid create experiment request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 从上下文获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	exp, err := h.expService.CreateExperiment(c.Request.Context(), uid, &req)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Created(c, exp)
}

// ListExperiments 列实验
// GET /api/v1/experiments
func (h *ExperimentHandler) ListExperiments(c *gin.Context) {
	var req domain.ListExperimentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Log.Warn("Invalid list experiments request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 解析 project_id 查询参数
	if projectID := c.Query("project_id"); projectID != "" {
		pid, err := uuid.Parse(projectID)
		if err == nil {
			req.ProjectID = pid
		}
	}

	experiments, total, err := h.expService.ListExperiments(c.Request.Context(), &req)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	// 计算总页数
	totalPage := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPage++
	}

	response.SuccessWithMeta(c, experiments, &response.MetaInfo{
		Page:      req.Page,
		PageSize:  req.PageSize,
		Total:     total,
		TotalPage: totalPage,
	})
}

// GetExperiment 获取实验详情
// GET /api/v1/experiments/:id
func (h *ExperimentHandler) GetExperiment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid experiment id")
		return
	}

	exp, err := h.expService.GetExperiment(c.Request.Context(), id)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, exp)
}

// UpdateExperiment 更新实验
// PUT /api/v1/experiments/:id
func (h *ExperimentHandler) UpdateExperiment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid experiment id")
		return
	}

	var req domain.UpdateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Invalid update experiment request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	exp, err := h.expService.UpdateExperiment(c.Request.Context(), id, &req)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, exp)
}

// DeleteExperiment 删除实验
// DELETE /api/v1/experiments/:id
func (h *ExperimentHandler) DeleteExperiment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid experiment id")
		return
	}

	if err := h.expService.DeleteExperiment(c.Request.Context(), id); err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.NoContent(c)
}

// GetExperimentRuns 获取实验运行列表
// GET /api/v1/experiments/:id/runs
func (h *ExperimentHandler) GetExperimentRuns(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid experiment id")
		return
	}

	runs, err := h.expService.GetExperimentRuns(c.Request.Context(), experimentID)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, runs)
}

// RunHandler 运行记录处理器
type RunHandler struct {
	runService service.RunService
	metricService service.MetricService
}

// NewRunHandler 创建运行记录处理器
func NewRunHandler(runService service.RunService, metricService service.MetricService) *RunHandler {
	return &RunHandler{
		runService: runService,
		metricService: metricService,
	}
}

// CreateRun 创建运行记录
// POST /api/v1/runs
func (h *RunHandler) CreateRun(c *gin.Context) {
	var req struct {
		ExperimentID uuid.UUID       `json:"experiment_id" binding:"required"`
		RunType      string          `json:"run_type" binding:"required,oneof=training inference simulation"`
		Config       domain.RunConfig `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	run, err := h.runService.CreateRun(c.Request.Context(), req.ExperimentID, req.RunType, req.Config)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Created(c, run)
}

// GetRun 获取运行记录详情
// GET /api/v1/runs/:id
func (h *RunHandler) GetRun(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	run, err := h.runService.GetRun(c.Request.Context(), id)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, run)
}

// UpdateRunStatus 更新运行状态
// PUT /api/v1/runs/:id/status
func (h *RunHandler) UpdateRunStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=pending running completed failed stopped"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.runService.UpdateRunStatus(c.Request.Context(), id, req.Status); err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "status updated"})
}

// CompleteRun 完成运行
// POST /api/v1/runs/:id/complete
func (h *RunHandler) CompleteRun(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	var req struct {
		MetricsSummary map[string]float64 `json:"metrics_summary"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.runService.CompleteRun(c.Request.Context(), id, req.MetricsSummary); err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "run completed"})
}

// MetricHandler 指标处理器
type MetricHandler struct {
	metricService service.MetricService
}

// NewMetricHandler 创建指标处理器
func NewMetricHandler(metricService service.MetricService) *MetricHandler {
	return &MetricHandler{metricService: metricService}
}

// RecordMetric 记录单个指标
// POST /api/v1/metrics
func (h *MetricHandler) RecordMetric(c *gin.Context) {
	var req domain.RecordMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.metricService.RecordMetric(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Created(c, gin.H{"message": "metric recorded"})
}

// BatchRecordMetrics 批量记录指标
// POST /api/v1/metrics/batch
func (h *MetricHandler) BatchRecordMetrics(c *gin.Context) {
	var req domain.BatchRecordMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.metricService.BatchRecordMetrics(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Created(c, gin.H{"message": "metrics recorded", "count": len(req.Metrics)})
}

// GetRunMetrics 获取运行的指标
// GET /api/v1/runs/:id/metrics
func (h *MetricHandler) GetRunMetrics(c *gin.Context) {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	// 获取指定的指标键
	keys := c.QueryArray("keys")

	metrics, err := h.metricService.GetRunMetrics(c.Request.Context(), runID, keys)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, metrics)
}

// GetMetricSeries 获取指标序列（用于图表）
// GET /api/v1/runs/:id/metrics/:key/series
func (h *MetricHandler) GetMetricSeries(c *gin.Context) {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	key := c.Param("key")
	if key == "" {
		response.Error(c, http.StatusBadRequest, "metric key is required")
		return
	}

	series, err := h.metricService.GetMetricSeries(c.Request.Context(), runID, key)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, series)
}

// QueryMetrics 查询指标
// GET /api/v1/metrics/query
func (h *MetricHandler) QueryMetrics(c *gin.Context) {
	var req domain.QueryMetricsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.RunID == uuid.Nil {
		response.Error(c, http.StatusBadRequest, "run_id is required")
		return
	}

	metrics, err := h.metricService.QueryMetrics(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, metrics)
}

// VisualizationHandler 可视化处理器
type VisualizationHandler struct {
	vizService service.VisualizationService
}

// NewVisualizationHandler 创建可视化处理器
func NewVisualizationHandler(vizService service.VisualizationService) *VisualizationHandler {
	return &VisualizationHandler{vizService: vizService}
}

// GetLossCurve 获取损失曲线
// GET /api/v1/runs/:id/loss-curve
func (h *VisualizationHandler) GetLossCurve(c *gin.Context) {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	data, err := h.vizService.GetLossCurve(c.Request.Context(), runID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, data)
}

// GetAccuracyTrend 获取准确率趋势
// GET /api/v1/runs/:id/accuracy-trend
func (h *VisualizationHandler) GetAccuracyTrend(c *gin.Context) {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	data, err := h.vizService.GetAccuracyTrend(c.Request.Context(), runID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, data)
}

// CompareExperiments 对比实验
// POST /api/v1/experiments/compare
func (h *VisualizationHandler) CompareExperiments(c *gin.Context) {
	var req domain.CompareExperimentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	comparison, err := h.vizService.CompareExperiments(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, comparison)
}

// GetExperimentReport 获取实验报表
// GET /api/v1/experiments/:id/report
func (h *VisualizationHandler) GetExperimentReport(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid experiment id")
		return
	}

	report, err := h.vizService.GetExperimentReport(c.Request.Context(), experimentID)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, report)
}

// GetHyperparameterComparison 超参数对比
// POST /api/v1/experiments/hyperparameters/compare
func (h *VisualizationHandler) GetHyperparameterComparison(c *gin.Context) {
	var req domain.HyperparameterComparisonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	comparison, err := h.vizService.CompareHyperparameters(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, comparison)
}
