package handler

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/service"
	"go.uber.org/zap"
)

// DatasetHandler 数据集处理器
type DatasetHandler struct {
	service service.DatasetService
	config  UploadConfig
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxSize        int64
	ChunkSize      int64
	TempDir        string
	AllowedFormats []string
}

// NewDatasetHandler 创建数据集处理器
func NewDatasetHandler(svc service.DatasetService, cfg UploadConfig) *DatasetHandler {
	return &DatasetHandler{
		service: svc,
		config:  cfg,
	}
}

// RegisterRoutes 注册路由
func (h *DatasetHandler) RegisterRoutes(router *gin.RouterGroup) {
	datasets := router.Group("/datasets")
	{
		datasets.POST("", h.Create)
		datasets.GET("", h.List)
		datasets.POST("/upload", h.Upload)
		datasets.POST("/upload/init", h.InitUpload)
		datasets.POST("/upload/chunk", h.UploadChunk)
		datasets.POST("/upload/complete", h.CompleteUpload)
		datasets.GET("/upload/:upload_id/progress", h.GetUploadProgress)

		datasets.GET("/:id", h.GetByID)
		datasets.PUT("/:id", h.Update)
		datasets.DELETE("/:id", h.Delete)
		datasets.GET("/:id/download", h.Download)
	}
}

// Create 创建数据集
func (h *DatasetHandler) Create(c *gin.Context) {
	var req domain.DatasetCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	resp, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		logger.Error("failed to create dataset", zap.Error(err))
		response.ErrorWithCode(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	response.Success(c, resp)
}

// GetByID 获取数据集详情
func (h *DatasetHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "dataset id is required")
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid dataset id format")
		return
	}

	resp, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error("failed to get dataset", zap.Error(err), zap.String("id", id))
		if err.Error() == "dataset not found" {
			response.Error(c, http.StatusNotFound, "dataset not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}

// List 获取数据集列表
func (h *DatasetHandler) List(c *gin.Context) {
	query := domain.DefaultListQuery()

	// 解析查询参数
	if projectID := c.Query("project_id"); projectID != "" {
		query.ProjectID = projectID
	}
	if status := c.Query("status"); status != "" {
		query.Status = status
	}
	if format := c.Query("format"); format != "" {
		query.Format = format
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query.Keyword = keyword
	}
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			query.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
			query.PageSize = ps
		}
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		query.SortBy = sortBy
	}
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		query.SortOrder = sortOrder
	}

	resp, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		logger.Error("failed to list datasets", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 计算总页数
	totalPages := int(resp.Total) / query.PageSize
	if int(resp.Total)%query.PageSize > 0 {
		totalPages++
	}

	meta := &response.MetaInfo{
		Page:      query.Page,
		PageSize:  query.PageSize,
		Total:     resp.Total,
		TotalPage: totalPages,
	}

	response.SuccessWithMeta(c, resp.Datasets, meta)
}

// Update 更新数据集
func (h *DatasetHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "dataset id is required")
		return
	}

	var req domain.DatasetUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	resp, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		logger.Error("failed to update dataset", zap.Error(err), zap.String("id", id))
		if err.Error() == "dataset not found" {
			response.Error(c, http.StatusNotFound, "dataset not found")
			return
		}
		response.ErrorWithCode(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	response.Success(c, resp)
}

// Delete 删除数据集
func (h *DatasetHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "dataset id is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		logger.Error("failed to delete dataset", zap.Error(err), zap.String("id", id))
		if err.Error() == "dataset not found" {
			response.Error(c, http.StatusNotFound, "dataset not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "dataset deleted successfully"})
}

// Upload 上传文件（完整文件上传）
func (h *DatasetHandler) Upload(c *gin.Context) {
	// 获取项目 ID
	projectID := c.PostForm("project_id")
	if projectID == "" {
		response.Error(c, http.StatusBadRequest, "project_id is required")
		return
	}

	// 获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("failed to get file: %v", err))
		return
	}
	defer file.Close()

	// 检查文件大小
	if header.Size > h.config.MaxSize {
		response.ErrorWithCode(c, http.StatusBadRequest, "FILE_TOO_LARGE",
			fmt.Sprintf("file size exceeds maximum limit of %d bytes", h.config.MaxSize))
		return
	}

	name := c.PostForm("name")
	if name == "" {
		name = filepath.Base(header.Filename)
	}

	description := c.PostForm("description")
	format := domain.DetectFormat(header.Filename)

	// 创建数据集请求
	createReq := &domain.DatasetCreateRequest{
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Format:      string(format),
	}

	// 先创建数据集记录
	datasetResp, err := h.service.Create(c.Request.Context(), createReq)
	if err != nil {
		logger.Error("failed to create dataset", zap.Error(err))
		response.ErrorWithCode(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	// 上传文件
	if err := h.service.UploadFile(c.Request.Context(), datasetResp.ID.String(), header.Filename, file, header.Size); err != nil {
		logger.Error("failed to upload file", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 获取最新的数据集信息
	resp, err := h.service.GetByID(c.Request.Context(), datasetResp.ID.String())
	if err != nil {
		response.Success(c, datasetResp)
		return
	}

	response.Success(c, resp)
}

// InitUpload 初始化分片上传
func (h *DatasetHandler) InitUpload(c *gin.Context) {
	var req domain.UploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	resp, err := h.service.InitUpload(c.Request.Context(), &req)
	if err != nil {
		logger.Error("failed to init upload", zap.Error(err))
		response.ErrorWithCode(c, http.StatusBadRequest, "INIT_FAILED", err.Error())
		return
	}

	response.Success(c, resp)
}

// UploadChunk 上传分片
func (h *DatasetHandler) UploadChunk(c *gin.Context) {
	uploadID := c.PostForm("upload_id")
	if uploadID == "" {
		response.Error(c, http.StatusBadRequest, "upload_id is required")
		return
	}

	chunkIndexStr := c.PostForm("chunk_index")
	if chunkIndexStr == "" {
		response.Error(c, http.StatusBadRequest, "chunk_index is required")
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		response.Error(c, http.StatusBadRequest, "invalid chunk_index")
		return
	}

	// 获取分片数据
	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("failed to get chunk: %v", err))
		return
	}
	defer file.Close()

	if err := h.service.UploadChunk(c.Request.Context(), uploadID, chunkIndex, file); err != nil {
		logger.Error("failed to upload chunk", zap.Error(err),
			zap.String("upload_id", uploadID),
			zap.Int("chunk_index", chunkIndex))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"upload_id":   uploadID,
		"chunk_index": chunkIndex,
		"message":     "chunk uploaded successfully",
	})
}

// CompleteUpload 完成分片上传
func (h *DatasetHandler) CompleteUpload(c *gin.Context) {
	var req domain.UploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	resp, err := h.service.CompleteUpload(c.Request.Context(), req.UploadID, req.Checksum)
	if err != nil {
		logger.Error("failed to complete upload", zap.Error(err), zap.String("upload_id", req.UploadID))
		response.ErrorWithCode(c, http.StatusBadRequest, "COMPLETE_FAILED", err.Error())
		return
	}

	response.Success(c, resp)
}

// GetUploadProgress 获取上传进度
func (h *DatasetHandler) GetUploadProgress(c *gin.Context) {
	uploadID := c.Param("upload_id")
	if uploadID == "" {
		response.Error(c, http.StatusBadRequest, "upload_id is required")
		return
	}

	progress, err := h.service.GetUploadProgress(c.Request.Context(), uploadID)
	if err != nil {
		logger.Error("failed to get upload progress", zap.Error(err), zap.String("upload_id", uploadID))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if progress == nil {
		response.Error(c, http.StatusNotFound, "upload progress not found")
		return
	}

	response.Success(c, gin.H{
		"upload_id":         progress.UploadID,
		"dataset_id":        progress.DatasetID,
		"filename":          progress.Filename,
		"total_size":        progress.TotalSize,
		"uploaded_size":     progress.UploadedSize,
		"progress_percent":  progress.ProgressPercentage(),
		"total_chunks":      progress.TotalChunks,
		"uploaded_chunks":   len(progress.UploadedChunks),
		"status":            progress.Status,
		"created_at":        progress.CreatedAt,
		"updated_at":        progress.UpdatedAt,
	})
}

// Download 下载数据集
func (h *DatasetHandler) Download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "dataset id is required")
		return
	}

	resp, err := h.service.Download(c.Request.Context(), id)
	if err != nil {
		logger.Error("failed to get download url", zap.Error(err), zap.String("id", id))
		if err.Error() == "dataset not found" {
			response.Error(c, http.StatusNotFound, "dataset not found")
			return
		}
		response.ErrorWithCode(c, http.StatusBadRequest, "DOWNLOAD_FAILED", err.Error())
		return
	}

	// 检查是否为直接下载请求
	direct := c.Query("direct") == "true"
	if direct {
		// 重定向到预签名 URL
		c.Redirect(http.StatusTemporaryRedirect, resp.URL)
		return
	}

	response.Success(c, resp)
}

// HealthCheck 健康检查
func HealthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "data-service",
	})
}
