package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/domain"
	"gorm.io/gorm"
)

// DatasetRepository 数据集仓库接口
type DatasetRepository interface {
	Create(ctx context.Context, dataset *domain.Dataset) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Dataset, error)
	GetByIDWithMetadata(ctx context.Context, id uuid.UUID) (*domain.Dataset, error)
	List(ctx context.Context, query *domain.ListQuery) ([]domain.Dataset, int64, error)
	Update(ctx context.Context, dataset *domain.Dataset) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DatasetStatus) error
	UpdateSize(ctx context.Context, id uuid.UUID, size int64) error
	ExistsByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (bool, error)
	CreateMetadata(ctx context.Context, metadata *domain.DatasetMetadata) error
	GetMetadata(ctx context.Context, datasetID uuid.UUID) (*domain.DatasetMetadata, error)
	UpdateMetadata(ctx context.Context, metadata *domain.DatasetMetadata) error
	CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
}

// datasetRepository 数据集仓库实现
type datasetRepository struct {
	db *gorm.DB
}

// NewDatasetRepository 创建数据集仓库
func NewDatasetRepository(db *gorm.DB) DatasetRepository {
	return &datasetRepository{db: db}
}

// Create 创建数据集
func (r *datasetRepository) Create(ctx context.Context, dataset *domain.Dataset) error {
	return r.db.WithContext(ctx).Create(dataset).Error
}

// GetByID 根据 ID 获取数据集
func (r *datasetRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Dataset, error) {
	var dataset domain.Dataset
	err := r.db.WithContext(ctx).First(&dataset, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}
	return &dataset, nil
}

// GetByIDWithMetadata 根据 ID 获取数据集（包含元数据）
func (r *datasetRepository) GetByIDWithMetadata(ctx context.Context, id uuid.UUID) (*domain.Dataset, error) {
	var dataset domain.Dataset
	err := r.db.WithContext(ctx).
		Preload("Metadata").
		First(&dataset, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get dataset with metadata: %w", err)
	}
	return &dataset, nil
}

// List 获取数据集列表
func (r *datasetRepository) List(ctx context.Context, query *domain.ListQuery) ([]domain.Dataset, int64, error) {
	var datasets []domain.Dataset
	var total int64

	db := r.db.WithContext(ctx).Model(&domain.Dataset{})

	// 应用过滤条件
	if query.ProjectID != "" {
		projectID, err := uuid.Parse(query.ProjectID)
		if err == nil {
			db = db.Where("project_id = ?", projectID)
		}
	}

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.Format != "" {
		db = db.Where("format = ?", query.Format)
	}

	if query.Keyword != "" {
		keyword := fmt.Sprintf("%%%s%%", query.Keyword)
		db = db.Where("name ILIKE ? OR description ILIKE ?", keyword, keyword)
	}

	// 排除已删除的
	db = db.Where("status != ?", domain.DatasetStatusDeleted)

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count datasets: %w", err)
	}

	// 排序
	order := fmt.Sprintf("%s %s", query.SortBy, query.SortOrder)
	db = db.Order(order)

	// 分页
	db = db.Offset(query.GetOffset()).Limit(query.GetLimit())

	// 查询
	if err := db.Find(&datasets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list datasets: %w", err)
	}

	return datasets, total, nil
}

// Update 更新数据集
func (r *datasetRepository) Update(ctx context.Context, dataset *domain.Dataset) error {
	return r.db.WithContext(ctx).Save(dataset).Error
}

// Delete 删除数据集（软删除）
func (r *datasetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.Dataset{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete dataset: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateStatus 更新数据集状态
func (r *datasetRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DatasetStatus) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Dataset{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update dataset status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateSize 更新数据集大小
func (r *datasetRepository) UpdateSize(ctx context.Context, id uuid.UUID, size int64) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Dataset{}).
		Where("id = ?", id).
		Update("size_bytes", size)
	if result.Error != nil {
		return fmt.Errorf("failed to update dataset size: %w", result.Error)
	}
	return nil
}

// ExistsByProjectAndName 检查项目名称组合是否存在
func (r *datasetRepository) ExistsByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Dataset{}).
		Where("project_id = ? AND name = ? AND status != ?", projectID, name, domain.DatasetStatusDeleted).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check dataset existence: %w", err)
	}
	return count > 0, nil
}

// CreateMetadata 创建元数据
func (r *datasetRepository) CreateMetadata(ctx context.Context, metadata *domain.DatasetMetadata) error {
	return r.db.WithContext(ctx).Create(metadata).Error
}

// GetMetadata 获取元数据
func (r *datasetRepository) GetMetadata(ctx context.Context, datasetID uuid.UUID) (*domain.DatasetMetadata, error) {
	var metadata domain.DatasetMetadata
	err := r.db.WithContext(ctx).
		Where("dataset_id = ?", datasetID).
		First(&metadata).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	return &metadata, nil
}

// UpdateMetadata 更新元数据
func (r *datasetRepository) UpdateMetadata(ctx context.Context, metadata *domain.DatasetMetadata) error {
	return r.db.WithContext(ctx).Save(metadata).Error
}

// CountByProject 统计项目下的数据集数量
func (r *datasetRepository) CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Dataset{}).
		Where("project_id = ? AND status != ?", projectID, domain.DatasetStatusDeleted).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count datasets by project: %w", err)
	}
	return count, nil
}

// UploadProgressRepository 上传进度仓库接口
type UploadProgressRepository interface {
	Get(ctx context.Context, uploadID string) (*domain.UploadProgress, error)
	Save(ctx context.Context, progress *domain.UploadProgress) error
	UpdateChunk(ctx context.Context, uploadID string, chunkIndex int, uploadedSize int64) error
	Delete(ctx context.Context, uploadID string) error
	SetExpiry(ctx context.Context, uploadID string, expiration time.Duration) error
}

// uploadProgressRepository 上传进度仓库实现（基于 Redis）
type uploadProgressRepository struct {
	redis RedisClient
}

// RedisClient Redis 客户端接口
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HIncrBy(ctx context.Context, key, field string, incr int64) error
	Del(ctx context.Context, keys ...string) error
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
}

// NewUploadProgressRepository 创建上传进度仓库
func NewUploadProgressRepository(redis RedisClient) UploadProgressRepository {
	return &uploadProgressRepository{redis: redis}
}

// Get 获取上传进度
func (r *uploadProgressRepository) Get(ctx context.Context, uploadID string) (*domain.UploadProgress, error) {
	// 从 Redis 获取
	// 这里简化实现，实际需要序列化/反序列化
	return nil, nil
}

// Save 保存上传进度
func (r *uploadProgressRepository) Save(ctx context.Context, progress *domain.UploadProgress) error {
	// 保存到 Redis
	return nil
}

// UpdateChunk 更新分片上传进度
func (r *uploadProgressRepository) UpdateChunk(ctx context.Context, uploadID string, chunkIndex int, uploadedSize int64) error {
	// 更新 Redis
	return nil
}

// Delete 删除上传进度
func (r *uploadProgressRepository) Delete(ctx context.Context, uploadID string) error {
	return nil
}

// SetExpiry 设置过期时间
func (r *uploadProgressRepository) SetExpiry(ctx context.Context, uploadID string, expiration time.Duration) error {
	return nil
}
