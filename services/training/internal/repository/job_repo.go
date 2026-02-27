package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/domain"
	"gorm.io/gorm"
)

// JobRepository 训练任务仓库接口
type JobRepository interface {
	Create(ctx context.Context, job *domain.TrainingJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.TrainingJob, error)
	List(ctx context.Context, req *domain.ListJobsRequest) ([]*domain.TrainingJob, int64, error)
	Update(ctx context.Context, job *domain.TrainingJob) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, message string) error
	Delete(ctx context.Context, id uuid.UUID) error
	
	// 实验关联
	ListByExperiment(ctx context.Context, experimentID uuid.UUID, page, pageSize int) ([]*domain.TrainingJob, int64, error)
	
	// 运行统计
	GetExperimentMetrics(ctx context.Context, experimentID uuid.UUID) (map[string]interface{}, error)
}

// jobRepository 训练任务仓库实现
type jobRepository struct {
	db *gorm.DB
}

// NewJobRepository 创建仓库实例
func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepository{db: db}
}

// Create 创建训练任务
func (r *jobRepository) Create(ctx context.Context, job *domain.TrainingJob) error {
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	
	if job.Status == "" {
		job.Status = domain.JobStatusPending
	}
	
	now := time.Now()
	job.QueuedAt = &now

	return r.db.WithContext(ctx).Create(job).Error
}

// GetByID 根据 ID 获取任务
func (r *jobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TrainingJob, error) {
	var job domain.TrainingJob
	err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("training job not found: %s", id)
		}
		return nil, err
	}
	return &job, nil
}

// List 列出训练任务
func (r *jobRepository) List(ctx context.Context, req *domain.ListJobsRequest) ([]*domain.TrainingJob, int64, error) {
	var jobs []*domain.TrainingJob
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.TrainingJob{})

	// 应用过滤条件
	if req.ProjectID != "" {
		if projectID, err := uuid.Parse(req.ProjectID); err == nil {
			query = query.Where("project_id = ?", projectID)
		}
	}
	if req.ExperimentID != "" {
		if expID, err := uuid.Parse(req.ExperimentID); err == nil {
			query = query.Where("experiment_id = ?", expID)
		}
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// Update 更新训练任务
func (r *jobRepository) Update(ctx context.Context, job *domain.TrainingJob) error {
	job.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(job).Error
}

// UpdateStatus 更新任务状态
func (r *jobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, message string) error {
	updates := map[string]interface{}{
		"status":         status,
		"status_message": message,
		"updated_at":     time.Now(),
	}

	// 根据状态更新相应的时间戳
	switch status {
	case domain.JobStatusRunning:
		updates["started_at"] = time.Now()
	case domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled:
		updates["completed_at"] = time.Now()
	}

	return r.db.WithContext(ctx).
		Model(&domain.TrainingJob{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Delete 删除训练任务（软删除）
func (r *jobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.TrainingJob{}, "id = ?", id).Error
}

// ListByExperiment 列出实验下的所有任务
func (r *jobRepository) ListByExperiment(ctx context.Context, experimentID uuid.UUID, page, pageSize int) ([]*domain.TrainingJob, int64, error) {
	var jobs []*domain.TrainingJob
	var total int64

	query := r.db.WithContext(ctx).
		Model(&domain.TrainingJob{}).
		Where("experiment_id = ?", experimentID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// GetExperimentMetrics 获取实验级别的指标汇总
func (r *jobRepository) GetExperimentMetrics(ctx context.Context, experimentID uuid.UUID) (map[string]interface{}, error) {
	var results []struct {
		Status string
		Count  int64
	}

	// 按状态统计任务数量
	if err := r.db.WithContext(ctx).
		Model(&domain.TrainingJob{}).
		Select("status, COUNT(*) as count").
		Where("experiment_id = ?", experimentID).
		Group("status").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	metrics := make(map[string]interface{})
	statusCounts := make(map[string]int64)
	var total int64

	for _, r := range results {
		statusCounts[r.Status] = r.Count
		total += r.Count
	}

	metrics["total_jobs"] = total
	metrics["status_distribution"] = statusCounts

	// 统计成功率和失败率
	if total > 0 {
		metrics["completion_rate"] = float64(statusCounts[string(domain.JobStatusCompleted)]) / float64(total) * 100
		metrics["failure_rate"] = float64(statusCounts[string(domain.JobStatusFailed)]) / float64(total) * 100
	}

	return metrics, nil
}
