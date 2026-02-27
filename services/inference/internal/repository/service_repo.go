package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/domain"
)

// ServiceRepository 推理服务仓库接口
type ServiceRepository interface {
	Create(ctx context.Context, service *domain.InferenceService) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.InferenceService, error)
	GetByProjectID(ctx context.Context, projectID uuid.UUID, page, pageSize int) ([]*domain.InferenceService, int64, error)
	List(ctx context.Context, projectID *uuid.UUID, status domain.ServiceStatus, page, pageSize int) ([]*domain.InferenceService, int64, error)
	Update(ctx context.Context, service *domain.InferenceService) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ServiceStatus, message string) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetRunningServices(ctx context.Context) ([]*domain.InferenceService, error)
}

// serviceRepository 推理服务仓库实现
type serviceRepository struct {
	db *gorm.DB
}

// NewServiceRepository 创建推理服务仓库
func NewServiceRepository(db *gorm.DB) ServiceRepository {
	return &serviceRepository{db: db}
}

// Create 创建推理服务
func (r *serviceRepository) Create(ctx context.Context, service *domain.InferenceService) error {
	if service.ID == uuid.Nil {
		service.ID = uuid.New()
	}
	now := time.Now()
	service.CreatedAt = now
	service.UpdatedAt = now

	result := r.db.WithContext(ctx).Create(service)
	if result.Error != nil {
		return fmt.Errorf("failed to create inference service: %w", result.Error)
	}
	return nil
}

// GetByID 根据 ID 获取推理服务
func (r *serviceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.InferenceService, error) {
	var service domain.InferenceService
	result := r.db.WithContext(ctx).First(&service, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("inference service not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get inference service: %w", result.Error)
	}
	return &service, nil
}

// GetByProjectID 根据项目 ID 获取推理服务列表
func (r *serviceRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID, page, pageSize int) ([]*domain.InferenceService, int64, error) {
	var services []*domain.InferenceService
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&domain.InferenceService{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count inference services: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	result := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&services)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("failed to list inference services: %w", result.Error)
	}

	return services, total, nil
}

// List 列出推理服务
func (r *serviceRepository) List(ctx context.Context, projectID *uuid.UUID, status domain.ServiceStatus, page, pageSize int) ([]*domain.InferenceService, int64, error) {
	var services []*domain.InferenceService
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.InferenceService{})

	// 过滤条件
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count inference services: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	result := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&services)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("failed to list inference services: %w", result.Error)
	}

	return services, total, nil
}

// Update 更新推理服务
func (r *serviceRepository) Update(ctx context.Context, service *domain.InferenceService) error {
	service.UpdatedAt = time.Now()
	
	result := r.db.WithContext(ctx).Save(service)
	if result.Error != nil {
		return fmt.Errorf("failed to update inference service: %w", result.Error)
	}
	return nil
}

// UpdateStatus 更新推理服务状态
func (r *serviceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ServiceStatus, message string) error {
	updates := map[string]interface{}{
		"status":         status,
		"status_message": message,
		"updated_at":     time.Now(),
	}

	// 根据状态更新时间戳
	if status == domain.ServiceStatusRunning {
		updates["started_at"] = time.Now()
	}
	if status == domain.ServiceStatusStopped || status == domain.ServiceStatusError {
		updates["stopped_at"] = time.Now()
	}

	result := r.db.WithContext(ctx).Model(&domain.InferenceService{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update inference service status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("inference service not found: %s", id)
	}
	return nil
}

// Delete 删除推理服务
func (r *serviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.InferenceService{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete inference service: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("inference service not found: %s", id)
	}
	return nil
}

// GetRunningServices 获取所有运行中的服务
func (r *serviceRepository) GetRunningServices(ctx context.Context) ([]*domain.InferenceService, error) {
	var services []*domain.InferenceService
	result := r.db.WithContext(ctx).
		Where("status IN ?", []domain.ServiceStatus{
			domain.ServiceStatusDeploying,
			domain.ServiceStatusRunning,
		}).
		Find(&services)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get running services: %w", result.Error)
	}
	return services, nil
}
