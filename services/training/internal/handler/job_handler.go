package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/service"
)

// JobHandler 训练任务处理器
type JobHandler struct {
	service service.JobService
}

// NewJobHandler 创建任务处理器
func NewJobHandler(service service.JobService) *JobHandler {
	return &JobHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *JobHandler) RegisterRoutes(router *gin.RouterGroup) {
	jobs := router.Group("/training/jobs")
	{
		jobs.POST("", h.CreateJob)
		jobs.GET("", h.ListJobs)
		jobs.GET("/:id", h.GetJob)
		jobs.PUT("/:id", h.UpdateJob)
		jobs.DELETE("/:id", h.DeleteJob)
		jobs.POST("/:id/stop", h.StopJob)
		jobs.GET("/:id/logs", h.GetLogs)
	}
}

// CreateJob 创建训练任务
// @Summary 创建训练任务
// @Description 创建新的训练任务
// @Tags Training
// @Accept json
// @Produce json
// @Param request body domain.CreateJobRequest true "创建任务请求"
// @Success 200 {object} response.Response{data=domain.JobResponse}
// @Router /api/v1/training/jobs [post]
func (h *JobHandler) CreateJob(c *gin.Context) {
	var req domain.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// 从上下文获取用户 ID（通过中间件设置）
	userIDStr, exists := c.Get("user_id")
	if !exists {
		// 临时使用默认用户 ID
		userIDStr = "00000000-0000-0000-0000-000000000001"
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response.BadRequest(c, "Invalid user_id")
		return
	}

	job, err := h.service.CreateJob(c.Request.Context(), userID, &req)
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInternalServer, err.Error())
		return
	}

	response.Success(c, job.ToResponse())
}

// GetJob 获取任务详情
// @Summary 获取训练任务详情
// @Description 根据 ID 获取训练任务详情
// @Tags Training
// @Produce json
// @Param id path string true "任务 ID"
// @Success 200 {object} response.Response{data=domain.JobResponse}
// @Router /api/v1/training/jobs/{id} [get]
func (h *JobHandler) GetJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid job ID")
		return
	}

	job, err := h.service.GetJob(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Training job")
		return
	}

	response.Success(c, job.ToResponse())
}

// ListJobs 列出训练任务
// @Summary 列出训练任务
// @Description 分页列出训练任务
// @Tags Training
// @Produce json
// @Param project_id query string false "项目 ID"
// @Param experiment_id query string false "实验 ID"
// @Param status query string false "任务状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse{data=[]domain.JobResponse}
// @Router /api/v1/training/jobs [get]
func (h *JobHandler) ListJobs(c *gin.Context) {
	var req domain.ListJobsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid query parameters: %v", err))
		return
	}

	jobs, total, err := h.service.ListJobs(c.Request.Context(), &req)
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInternalServer, err.Error())
		return
	}

	// 转换为响应格式
	responses := make([]*domain.JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToResponse()
	}

	response.PageSuccess(c, responses, req.Page, req.PageSize, total)
}

// UpdateJob 更新训练任务
// @Summary 更新训练任务
// @Description 更新训练任务信息
// @Tags Training
// @Accept json
// @Produce json
// @Param id path string true "任务 ID"
// @Param request body domain.UpdateJobRequest true "更新请求"
// @Success 200 {object} response.Response{data=domain.JobResponse}
// @Router /api/v1/training/jobs/{id} [put]
func (h *JobHandler) UpdateJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid job ID")
		return
	}

	var req domain.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	job, err := h.service.UpdateJob(c.Request.Context(), id, &req)
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInternalServer, err.Error())
		return
	}

	response.Success(c, job.ToResponse())
}

// DeleteJob 删除训练任务
// @Summary 删除训练任务
// @Description 删除训练任务
// @Tags Training
// @Param id path string true "任务 ID"
// @Success 200 {object} response.Response
// @Router /api/v1/training/jobs/{id} [delete]
func (h *JobHandler) DeleteJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid job ID")
		return
	}

	if err := h.service.DeleteJob(c.Request.Context(), id); err != nil {
		response.ErrorWithMessage(c, response.ErrorInternalServer, err.Error())
		return
	}

	response.Success(c, nil)
}

// StopJob 停止训练任务
// @Summary 停止训练任务
// @Description 停止正在运行的训练任务
// @Tags Training
// @Param id path string true "任务 ID"
// @Success 200 {object} response.Response
// @Router /api/v1/training/jobs/{id}/stop [post]
func (h *JobHandler) StopJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid job ID")
		return
	}

	if err := h.service.StopJob(c.Request.Context(), id); err != nil {
		response.ErrorWithMessage(c, response.ErrorInternalServer, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetLogs 获取训练日志（SSE 流式）
// @Summary 获取训练日志
// @Description 获取训练任务的日志（支持 SSE 流式）
// @Tags Training
// @Param id path string true "任务 ID"
// @Param stream query bool false "是否流式输出" default(false)
// @Success 200 {object} response.Response{data=[]domain.LogEntry}
// @Router /api/v1/training/jobs/{id}/logs [get]
func (h *JobHandler) GetLogs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid job ID")
		return
	}

	// 检查是否流式输出
	stream := c.Query("stream") == "true"

	if stream {
		// SSE 流式输出
		h.streamLogsSSE(c, id)
		return
	}

	// 普通查询
	start := c.Query("start")
	countStr := c.DefaultQuery("count", "100")
	count, _ := strconv.ParseInt(countStr, 10, 64)

	logs, err := h.service.GetLogs(c.Request.Context(), id, start, count)
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInternalServer, err.Error())
		return
	}

	response.Success(c, logs)
}

// streamLogsSSE SSE 流式输出日志
func (h *JobHandler) streamLogsSSE(c *gin.Context, jobID uuid.UUID) {
	ctx := c.Request.Context()

	// 设置 SSE 头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// 创建日志通道
	logChan := make(chan domain.LogEntry, 100)

	// 启动日志流
	go func() {
		defer close(logChan)
		h.service.StreamLogs(ctx, jobID, logChan)
	}()

	// 发送日志
	c.Stream(func(w io.Writer) bool {
		select {
		case log, ok := <-logChan:
			if !ok {
				// 通道关闭，发送结束标记
				fmt.Fprintf(w, "event: end\ndata: {}\n\n")
				return false
			}

			// 发送 SSE 事件
			data, _ := json.Marshal(log)
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
			return true
		case <-ctx.Done():
			return false
		}
	})
}
