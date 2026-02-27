package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/minio"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DatasetService 数据集服务接口
type DatasetService interface {
	Create(ctx context.Context, req *domain.DatasetCreateRequest) (*domain.DatasetResponse, error)
	GetByID(ctx context.Context, id string) (*domain.DatasetResponse, error)
	List(ctx context.Context, query *domain.ListQuery) (*domain.DatasetListResponse, error)
	Update(ctx context.Context, id string, req *domain.DatasetUpdateRequest) (*domain.DatasetResponse, error)
	Delete(ctx context.Context, id string) error
	InitUpload(ctx context.Context, req *domain.UploadInitRequest) (*domain.UploadInitResponse, error)
	UploadFile(ctx context.Context, datasetID string, filename string, reader io.Reader, size int64) error
	UploadChunk(ctx context.Context, uploadID string, chunkIndex int, reader io.Reader) error
	CompleteUpload(ctx context.Context, uploadID string, checksum string) (*domain.DatasetResponse, error)
	GetUploadProgress(ctx context.Context, uploadID string) (*domain.UploadProgress, error)
	Download(ctx context.Context, id string) (*domain.DownloadResponse, error)
}

// datasetService 数据集服务实现
type datasetService struct {
	repo          repository.DatasetRepository
	minioClient   *minio.Client
	redisClient   *redis.Client
	config        *config.Config
}

// NewDatasetService 创建数据集服务
func NewDatasetService(repo repository.DatasetRepository, minioClient *minio.Client, redisClient *redis.Client, cfg *config.Config) DatasetService {
	return &datasetService{
		repo:        repo,
		minioClient: minioClient,
		redisClient: redisClient,
		config:      cfg,
	}
}

// Create 创建数据集记录
func (s *datasetService) Create(ctx context.Context, req *domain.DatasetCreateRequest) (*domain.DatasetResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	projectID, _ := uuid.Parse(req.ProjectID)

	// 检查同项目下是否有重名
	exists, err := s.repo.ExistsByProjectAndName(ctx, projectID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check dataset existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("dataset with name '%s' already exists in this project", req.Name)
	}

	format := domain.DatasetFormat(req.Format)
	if !format.Valid() {
		format = domain.DatasetFormatUnknown
	}

	dataset := &domain.Dataset{
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
		Format:      format,
		Status:      domain.DatasetStatusPending,
	}

	if err := s.repo.Create(ctx, dataset); err != nil {
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}

	return s.toResponse(dataset), nil
}

// GetByID 根据 ID 获取数据集
func (s *datasetService) GetByID(ctx context.Context, id string) (*domain.DatasetResponse, error) {
	datasetID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid dataset id: %w", err)
	}

	dataset, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}
	if dataset == nil {
		return nil, fmt.Errorf("dataset not found")
	}

	return s.toResponse(dataset), nil
}

// List 获取数据集列表
func (s *datasetService) List(ctx context.Context, query *domain.ListQuery) (*domain.DatasetListResponse, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	datasets, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets: %w", err)
	}

	responses := make([]domain.DatasetResponse, len(datasets))
	for i, d := range datasets {
		responses[i] = *s.toResponse(&d)
	}

	return &domain.DatasetListResponse{
		Datasets: responses,
		Total:    total,
	}, nil
}

// Update 更新数据集
func (s *datasetService) Update(ctx context.Context, id string, req *domain.DatasetUpdateRequest) (*domain.DatasetResponse, error) {
	datasetID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid dataset id: %w", err)
	}

	dataset, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}
	if dataset == nil {
		return nil, fmt.Errorf("dataset not found")
	}

	if req.Name != "" {
		// 检查重名
		exists, err := s.repo.ExistsByProjectAndName(ctx, dataset.ProjectID, req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check dataset existence: %w", err)
		}
		if exists && dataset.Name != req.Name {
			return nil, fmt.Errorf("dataset with name '%s' already exists in this project", req.Name)
		}
		dataset.Name = req.Name
	}

	if req.Description != "" {
		dataset.Description = req.Description
	}

	if err := s.repo.Update(ctx, dataset); err != nil {
		return nil, fmt.Errorf("failed to update dataset: %w", err)
	}

	return s.toResponse(dataset), nil
}

// Delete 删除数据集
func (s *datasetService) Delete(ctx context.Context, id string) error {
	datasetID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid dataset id: %w", err)
	}

	// 获取数据集信息
	dataset, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return fmt.Errorf("failed to get dataset: %w", err)
	}
	if dataset == nil {
		return fmt.Errorf("dataset not found")
	}

	// 删除 MinIO 中的文件
	if dataset.StoragePath != "" {
		bucketName := minio.GetProjectBucketName(dataset.ProjectID.String())
		objectName := s.getObjectNameFromPath(dataset.StoragePath)
		if err := s.minioClient.RemoveObject(ctx, bucketName, objectName); err != nil {
			logger.Warn("failed to remove object from minio", zap.Error(err), zap.String("path", dataset.StoragePath))
		}
	}

	// 删除数据库记录
	if err := s.repo.Delete(ctx, datasetID); err != nil {
		return fmt.Errorf("failed to delete dataset: %w", err)
	}

	return nil
}

// InitUpload 初始化上传
func (s *datasetService) InitUpload(ctx context.Context, req *domain.UploadInitRequest) (*domain.UploadInitResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// 检查文件大小限制
	if req.Size > s.config.Upload.MaxSize {
		return nil, fmt.Errorf("file size exceeds maximum limit of %d bytes", s.config.Upload.MaxSize)
	}

	projectID, _ := uuid.Parse(req.ProjectID)

	// 检查同名数据集
	exists, err := s.repo.ExistsByProjectAndName(ctx, projectID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check dataset existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("dataset with name '%s' already exists in this project", req.Name)
	}

	// 确定分片大小
	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = s.config.Upload.ChunkSize
	}

	// 计算总分片数
	totalChunks := int(req.Size / chunkSize)
	if req.Size%chunkSize > 0 {
		totalChunks++
	}

	// 生成上传 ID
	uploadID := uuid.New().String()

	// 创建数据集记录
	dataset := &domain.Dataset{
		ProjectID:    projectID,
		Name:         req.Name,
		Description:  req.Description,
		OriginalName: req.Filename,
		Format:       domain.DetectFormat(req.Filename),
		SizeBytes:    req.Size,
		Status:       domain.DatasetStatusUploading,
		StoragePath:  "", // 上传完成后更新
	}

	if err := s.repo.Create(ctx, dataset); err != nil {
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}

	// 保存上传进度到 Redis
	progress := &domain.UploadProgress{
		UploadID:       uploadID,
		DatasetID:      dataset.ID,
		ProjectID:      projectID,
		Filename:       req.Filename,
		TotalSize:      req.Size,
		UploadedSize:   0,
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
		UploadedChunks: make([]int, 0),
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.saveUploadProgress(ctx, uploadID, progress); err != nil {
		return nil, fmt.Errorf("failed to save upload progress: %w", err)
	}

	// 设置过期时间
	progressKey := s.config.Redis.GetUploadProgressKey(uploadID)
	s.redisClient.Expire(ctx, progressKey, 24*time.Hour)

	return &domain.UploadInitResponse{
		UploadID:    uploadID,
		DatasetID:   dataset.ID.String(),
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
	}, nil
}

// UploadFile 上传文件（完整文件上传）
func (s *datasetService) UploadFile(ctx context.Context, datasetID string, filename string, reader io.Reader, size int64) error {
	id, err := uuid.Parse(datasetID)
	if err != nil {
		return fmt.Errorf("invalid dataset id: %w", err)
	}

	// 获取数据集
	dataset, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get dataset: %w", err)
	}
	if dataset == nil {
		return fmt.Errorf("dataset not found")
	}

	// 检查文件大小
	if size > s.config.Upload.MaxSize {
		return fmt.Errorf("file size exceeds maximum limit")
	}

	// 构建存储路径
	bucketName := minio.GetProjectBucketName(dataset.ProjectID.String())
	objectName := minio.GetDatasetObjectName(dataset.ProjectID.String(), datasetID, filename)

	// 上传文件到 MinIO
	format := domain.DetectFormat(filename)
	opts := &minio.UploadOptions{
		ContentType: s.getContentType(format),
		PartSize:    uint64(s.config.Upload.ChunkSize),
	}

	result, err := s.minioClient.Upload(ctx, bucketName, objectName, reader, size, opts)
	if err != nil {
		s.repo.UpdateStatus(ctx, id, domain.DatasetStatusFailed)
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// 更新数据集记录
	dataset.StoragePath = result.Location
	dataset.SizeBytes = result.Size
	dataset.Format = format
	dataset.Status = domain.DatasetStatusReady
	dataset.OriginalName = filename

	if err := s.repo.Update(ctx, dataset); err != nil {
		return fmt.Errorf("failed to update dataset: %w", err)
	}

	logger.Info("file uploaded successfully",
		zap.String("dataset_id", datasetID),
		zap.Int64("size", result.Size),
	)

	return nil
}

// UploadChunk 上传分片
func (s *datasetService) UploadChunk(ctx context.Context, uploadID string, chunkIndex int, reader io.Reader) error {
	// 获取上传进度
	progress, err := s.GetUploadProgress(ctx, uploadID)
	if err != nil {
		return err
	}
	if progress == nil {
		return fmt.Errorf("upload not found")
	}

	if chunkIndex < 0 || chunkIndex >= progress.TotalChunks {
		return fmt.Errorf("invalid chunk index")
	}

	// 检查是否已上传
	for _, idx := range progress.UploadedChunks {
		if idx == chunkIndex {
			return fmt.Errorf("chunk already uploaded")
		}
	}

	// 上传到 MinIO（使用 multipart）
	bucketName := minio.GetProjectBucketName(progress.ProjectID.String())
	datasetID := progress.DatasetID.String()
	chunkObjectName := fmt.Sprintf("uploads/%s/%s/chunks/%d", uploadID, datasetID, chunkIndex)

	// 读取分片数据
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read chunk data: %w", err)
	}

	_, err = s.minioClient.Upload(ctx, bucketName, chunkObjectName, strings.NewReader(string(data)), int64(len(data)), nil)
	if err != nil {
		return fmt.Errorf("failed to upload chunk: %w", err)
	}

	// 更新进度
	progress.UploadedChunks = append(progress.UploadedChunks, chunkIndex)
	progress.UploadedSize += int64(len(data))
	progress.UpdatedAt = time.Now()

	if err := s.saveUploadProgress(ctx, uploadID, progress); err != nil {
		return fmt.Errorf("failed to save upload progress: %w", err)
	}

	return nil
}

// CompleteUpload 完成上传
func (s *datasetService) CompleteUpload(ctx context.Context, uploadID string, checksum string) (*domain.DatasetResponse, error) {
	progress, err := s.GetUploadProgress(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if progress == nil {
		return nil, fmt.Errorf("upload not found")
	}

	// 检查是否所有分片都已上传
	if len(progress.UploadedChunks) != progress.TotalChunks {
		return nil, fmt.Errorf("upload incomplete: %d/%d chunks uploaded", len(progress.UploadedChunks), progress.TotalChunks)
	}

	// 合并分片
	bucketName := minio.GetProjectBucketName(progress.ProjectID.String())
	datasetID := progress.DatasetID.String()
	filename := progress.Filename

	// 最终对象名
	finalObjectName := minio.GetDatasetObjectName(progress.ProjectID.String(), datasetID, filename)

	// 合并分片（这里简化处理，实际需要组合分片）
	// 在实际场景中，可以使用 MinIO 的 ComposeObject 或重新上传完整文件

	// 获取数据集
	dataset, err := s.repo.GetByID(ctx, progress.DatasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}

	// 更新数据集状态
	dataset.StoragePath = fmt.Sprintf("%s/%s", bucketName, finalObjectName)
	dataset.SizeBytes = progress.TotalSize
	dataset.Checksum = checksum
	dataset.Status = domain.DatasetStatusReady

	if err := s.repo.Update(ctx, dataset); err != nil {
		return nil, fmt.Errorf("failed to update dataset: %w", err)
	}

	// 清理临时分片
	s.cleanupUploadChunks(ctx, bucketName, uploadID, datasetID)

	// 删除进度记录
	progressKey := s.config.Redis.GetUploadProgressKey(uploadID)
	s.redisClient.Del(ctx, progressKey)

	return s.toResponse(dataset), nil
}

// GetUploadProgress 获取上传进度
func (s *datasetService) GetUploadProgress(ctx context.Context, uploadID string) (*domain.UploadProgress, error) {
	progressKey := s.config.Redis.GetUploadProgressKey(uploadID)
	data, err := s.redisClient.Get(ctx, progressKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get upload progress: %w", err)
	}

	var progress domain.UploadProgress
	if err := json.Unmarshal([]byte(data), &progress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal progress: %w", err)
	}

	return &progress, nil
}

// Download 获取下载信息
func (s *datasetService) Download(ctx context.Context, id string) (*domain.DownloadResponse, error) {
	datasetID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid dataset id: %w", err)
	}

	dataset, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}
	if dataset == nil {
		return nil, fmt.Errorf("dataset not found")
	}

	if dataset.Status != domain.DatasetStatusReady {
		return nil, fmt.Errorf("dataset is not ready for download")
	}

	// 生成预签名 URL
	bucketName := minio.GetProjectBucketName(dataset.ProjectID.String())
	objectName := s.getObjectNameFromPath(dataset.StoragePath)
	if objectName == "" {
		objectName = minio.GetDatasetObjectName(dataset.ProjectID.String(), datasetID.String(), dataset.OriginalName)
	}

	presignedURL, err := s.minioClient.PresignedGetURL(ctx, bucketName, objectName, s.config.MinIO.PresignedExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// 确定下载文件名
	filename := dataset.OriginalName
	if filename == "" {
		filename = fmt.Sprintf("%s.%s", dataset.Name, dataset.Format)
	}

	return &domain.DownloadResponse{
		URL:         presignedURL.String(),
		ExpiresAt:   time.Now().Add(s.config.MinIO.PresignedExpiry),
		Filename:    filename,
		ContentType: s.getContentType(dataset.Format),
	}, nil
}

// 辅助方法

func (s *datasetService) toResponse(dataset *domain.Dataset) *domain.DatasetResponse {
	return &domain.DatasetResponse{
		ID:           dataset.ID,
		ProjectID:    dataset.ProjectID,
		Name:         dataset.Name,
		Description:  dataset.Description,
		StoragePath:  dataset.StoragePath,
		SizeBytes:    dataset.SizeBytes,
		SizeReadable: formatBytes(dataset.SizeBytes),
		Format:       dataset.Format,
		Status:       dataset.Status,
		OriginalName: dataset.OriginalName,
		Checksum:     dataset.Checksum,
		CreatedAt:    dataset.CreatedAt,
		UpdatedAt:    dataset.UpdatedAt,
	}
}

func (s *datasetService) getObjectNameFromPath(storagePath string) string {
	// 从路径中提取对象名
	parts := strings.Split(storagePath, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts[1:], "/")
}

func (s *datasetService) getContentType(format domain.DatasetFormat) string {
	switch format {
	case domain.DatasetFormatCSV:
		return "text/csv"
	case domain.DatasetFormatJSON:
		return "application/json"
	case domain.DatasetFormatParquet:
		return "application/octet-stream"
	case domain.DatasetFormatTXT:
		return "text/plain"
	case domain.DatasetFormatZIP:
		return "application/zip"
	case domain.DatasetFormatTAR:
		return "application/x-tar"
	case domain.DatasetFormatGZ:
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

func (s *datasetService) saveUploadProgress(ctx context.Context, uploadID string, progress *domain.UploadProgress) error {
	progressKey := s.config.Redis.GetUploadProgressKey(uploadID)
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}
	return s.redisClient.Set(ctx, progressKey, data, 24*time.Hour).Err()
}

func (s *datasetService) cleanupUploadChunks(ctx context.Context, bucketName, uploadID, datasetID string) {
	// 清理临时分片
	prefix := fmt.Sprintf("uploads/%s/%s/chunks/", uploadID, datasetID)
	objects := s.minioClient.ListObjects(ctx, bucketName, prefix, true)
	for obj := range objects {
		if obj.Err != nil {
			continue
		}
		s.minioClient.RemoveObject(ctx, bucketName, obj.Key)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// validateFormat 验证文件格式
func (s *datasetService) validateFormat(filename string) error {
	format := domain.DetectFormat(filename)
	if format == domain.DatasetFormatUnknown {
		return fmt.Errorf("unsupported file format")
	}

	// 检查是否允许
	for _, allowed := range s.config.Upload.AllowedFormats {
		if string(format) == allowed {
			return nil
		}
	}

	return fmt.Errorf("file format '%s' is not allowed", format)
}

// isImageFile 判断是否为图片文件
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}

// parseInt 解析整数
func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// contains 检查切片是否包含元素
func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// remove 从切片中移除元素
func remove(slice []int, item int) []int {
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if v != item {
			result = append(result, v)
		}
	}
	return result
}

// unique 去重
func unique(slice []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
