package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/domain"
)

// ModelRepository 模型仓库接口
type ModelRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ModelInfo, error)
	GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ModelInfo, error)
}

// modelRepository 模型仓库实现
type modelRepository struct {
	db *gorm.DB
}

// NewModelRepository 创建模型仓库
func NewModelRepository(db *gorm.DB) ModelRepository {
	return &modelRepository{db: db}
}

// Model 数据库模型结构
type Model struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"not null;size:255"`
	Version     string    `gorm:"size:50;default:'v1'"`
	StoragePath string    `gorm:"size:500"`
	TrainingJobID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   string    `gorm:"autoCreateTime"`
}

// TableName 表名
func (Model) TableName() string {
	return "models"
}

// GetByID 根据 ID 获取模型
func (r *modelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ModelInfo, error) {
	var model Model
	result := r.db.WithContext(ctx).First(&model, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("model not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get model: %w", result.Error)
	}

	return &domain.ModelInfo{
		ID:          model.ID,
		Name:        model.Name,
		Version:     model.Version,
		StoragePath: model.StoragePath,
		Format:      r.detectFormat(model.StoragePath),
	}, nil
}

// GetByProjectID 根据项目 ID 获取模型列表
func (r *modelRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ModelInfo, error) {
	var models []Model
	result := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list models: %w", result.Error)
	}

	modelInfos := make([]*domain.ModelInfo, len(models))
	for i, m := range models {
		modelInfos[i] = &domain.ModelInfo{
			ID:          m.ID,
			Name:        m.Name,
			Version:     m.Version,
			StoragePath: m.StoragePath,
			Format:      r.detectFormat(m.StoragePath),
		}
	}

	return modelInfos, nil
}

// detectFormat 检测模型格式
func (r *modelRepository) detectFormat(storagePath string) string {
	// 简单检测模型格式
	// 实际实现可以更复杂，根据文件扩展名或目录结构
	if len(storagePath) < 4 {
		return "unknown"
	}
	ext := storagePath[len(storagePath)-4:]
	switch ext {
	case ".pt", "pth":
		return "pytorch"
	case "onnx":
		return "onnx"
	case "on":
		return "tensorflow"
	default:
		return "triton" // 默认假设为 Triton 格式
	}
}
