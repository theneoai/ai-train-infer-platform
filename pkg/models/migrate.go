package models

import "gorm.io/gorm"

// AutoMigrate 自动迁移所有模型
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// 用户相关
		&User{},
		&Organization{},
		&Project{},

		// 实验相关
		&Experiment{},
		&Run{},
		&Metric{},
		&Artifact{},
		&LogEntry{},

		// 训练相关
		&TrainingJob{},
		&Checkpoint{},
		&Model{},
		&ModelVersion{},

		// 推理和仿真
		&InferenceService{},
		&SimulationEnvironment{},
		&Scenario{},
		&SimulationResult{},
	)
}

// GetAllModels 获取所有模型（用于测试或文档生成）
func GetAllModels() []interface{} {
	return []interface{}{
		&User{},
		&Organization{},
		&Project{},
		&Experiment{},
		&Run{},
		&Metric{},
		&Artifact{},
		&LogEntry{},
		&TrainingJob{},
		&Checkpoint{},
		&Model{},
		&ModelVersion{},
		&InferenceService{},
		&SimulationEnvironment{},
		&Scenario{},
		&SimulationResult{},
	}
}
