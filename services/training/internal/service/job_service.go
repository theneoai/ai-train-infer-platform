package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/executor"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/repository"
)

// JobService 训练任务服务接口
type JobService interface {
	CreateJob(ctx context.Context, userID uuid.UUID, req *domain.CreateJobRequest) (*domain.TrainingJob, error)
	GetJob(ctx context.Context, jobID uuid.UUID) (*domain.TrainingJob, error)
	ListJobs(ctx context.Context, req *domain.ListJobsRequest) ([]*domain.TrainingJob, int64, error)
	UpdateJob(ctx context.Context, jobID uuid.UUID, req *domain.UpdateJobRequest) (*domain.TrainingJob, error)
	DeleteJob(ctx context.Context, jobID uuid.UUID) error
	StopJob(ctx context.Context, jobID uuid.UUID) error
	
	// 实验关联
	ListJobsByExperiment(ctx context.Context, experimentID uuid.UUID, page, pageSize int) ([]*domain.TrainingJob, int64, error)
	GetExperimentMetrics(ctx context.Context, experimentID uuid.UUID) (map[string]interface{}, error)
	
	// 日志
	GetLogs(ctx context.Context, jobID uuid.UUID, start string, count int64) ([]domain.LogEntry, error)
	StreamLogs(ctx context.Context, jobID uuid.UUID, logChan chan<- domain.LogEntry) error
}

// jobService 训练任务服务实现
type jobService struct {
	cfg       *config.Config
	jobRepo   repository.JobRepository
	logRepo   repository.LogRepository
	executor  *executor.DockerExecutor
}

// NewJobService 创建任务服务实例
func NewJobService(cfg *config.Config, jobRepo repository.JobRepository, logRepo repository.LogRepository, exec *executor.DockerExecutor) JobService {
	return &jobService{
		cfg:      cfg,
		jobRepo:  jobRepo,
		logRepo:  logRepo,
		executor: exec,
	}
}

// CreateJob 创建训练任务
func (s *jobService) CreateJob(ctx context.Context, userID uuid.UUID, req *domain.CreateJobRequest) (*domain.TrainingJob, error) {
	// 解析项目 ID
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project_id: %w", err)
	}

	// 构建任务对象
	job := &domain.TrainingJob{
		ID:              uuid.New(),
		Name:            req.Name,
		Description:     req.Description,
		ProjectID:       projectID,
		UserID:          userID,
		ModelName:       req.ModelName,
		ModelVersion:    req.ModelVersion,
		DatasetPath:     req.DatasetPath,
		OutputPath:      req.OutputPath,
		Framework:       req.Framework,
		Image:           req.Image,
		Command:         req.Command,
		Hyperparameters: req.Hyperparameters,
		Environment:     req.Environment,
		GPUCount:        req.GPUCount,
		GPUType:         req.GPUType,
		CPUCount:        req.CPUCount,
		MemoryGB:        req.MemoryGB,
		TimeoutHours:    req.TimeoutHours,
		Status:          domain.JobStatusPending,
		Progress:        0,
	}

	// 解析实验 ID（如果提供）
	if req.ExperimentID != "" {
		expID, err := uuid.Parse(req.ExperimentID)
		if err != nil {
			return nil, fmt.Errorf("invalid experiment_id: %w", err)
		}
		job.ExperimentID = &expID
	}

	// 设置默认值
	if job.CPUCount == 0 {
		job.CPUCount = 4
	}
	if job.MemoryGB == 0 {
		job.MemoryGB = 16
	}
	if job.TimeoutHours == 0 {
		job.TimeoutHours = int(s.cfg.DefaultTimeout.Hours())
	}
	if job.TimeoutHours == 0 {
		job.TimeoutHours = 24
	}

	// 设置默认镜像
	if job.Image == "" {
		switch job.Framework {
		case domain.FrameworkPyTorch:
			job.Image = "pytorch/pytorch:latest"
		case domain.FrameworkTensorFlow:
			job.Image = "tensorflow/tensorflow:latest-gpu"
		default:
			job.Image = "python:3.9"
		}
	}

	// 设置环境变量
	if job.Environment == nil {
		job.Environment = make(map[string]string)
	}
	job.Environment["JOB_ID"] = job.ID.String()
	job.Environment["PROJECT_ID"] = job.ProjectID.String()
	job.Environment["MODEL_NAME"] = job.ModelName
	job.Environment["DATASET_PATH"] = "/data"
	job.Environment["OUTPUT_PATH"] = "/output"

	// 序列化超参数
	if job.Hyperparameters != nil {
		hpJSON, _ := json.Marshal(job.Hyperparameters)
		job.Environment["HYPERPARAMETERS"] = string(hpJSON)
	}

	// 保存到数据库
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// 记录创建日志
	s.logRepo.AppendLog(ctx, job.ID, &domain.LogEntry{
		Level:     "INFO",
		Source:    "system",
		Message:   fmt.Sprintf("Training job created: %s", job.Name),
		Timestamp: time.Now(),
	})

	// 异步启动任务
	go s.startJobAsync(job.ID)

	return job, nil
}

// GetJob 获取任务详情
func (s *jobService) GetJob(ctx context.Context, jobID uuid.UUID) (*domain.TrainingJob, error) {
	return s.jobRepo.GetByID(ctx, jobID)
}

// ListJobs 列出训练任务
func (s *jobService) ListJobs(ctx context.Context, req *domain.ListJobsRequest) ([]*domain.TrainingJob, int64, error) {
	return s.jobRepo.List(ctx, req)
}

// UpdateJob 更新任务
func (s *jobService) UpdateJob(ctx context.Context, jobID uuid.UUID, req *domain.UpdateJobRequest) (*domain.TrainingJob, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	// 只能更新非运行中的任务
	if job.Status == domain.JobStatusRunning {
		return nil, fmt.Errorf("cannot update running job")
	}

	if req.Name != "" {
		job.Name = req.Name
	}
	if req.Description != "" {
		job.Description = req.Description
	}

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

// DeleteJob 删除任务
func (s *jobService) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	// 如果任务在运行，先停止
	if job.Status == domain.JobStatusRunning {
		if err := s.StopJob(ctx, jobID); err != nil {
			return fmt.Errorf("failed to stop running job: %w", err)
		}
	}

	return s.jobRepo.Delete(ctx, jobID)
}

// StopJob 停止任务
func (s *jobService) StopJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	if !job.CanStop() {
		return fmt.Errorf("job cannot be stopped in status: %s", job.Status)
	}

	// 更新状态为 stopping
	if err := s.jobRepo.UpdateStatus(ctx, jobID, domain.JobStatusStopping, "User requested stop"); err != nil {
		return err
	}

	// 停止执行器
	if err := s.executor.Stop(ctx, jobID); err != nil {
		// 即使停止失败，也继续更新状态
		s.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
			Level:     "WARN",
			Source:    "system",
			Message:   fmt.Sprintf("Failed to stop executor: %v", err),
			Timestamp: time.Now(),
		})
	}

	// 更新状态为 cancelled
	return s.jobRepo.UpdateStatus(ctx, jobID, domain.JobStatusCancelled, "Stopped by user")
}

// startJobAsync 异步启动任务
func (s *jobService) startJobAsync(jobID uuid.UUID) {
	// 给数据库事务一些时间完成
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取任务详情
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		s.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
			Level:     "ERROR",
			Source:    "system",
			Message:   fmt.Sprintf("Failed to get job for starting: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// 更新状态为 running
	if err := s.jobRepo.UpdateStatus(ctx, jobID, domain.JobStatusRunning, "Starting training..."); err != nil {
		s.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
			Level:     "ERROR",
			Source:    "system",
			Message:   fmt.Sprintf("Failed to update job status: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// 记录启动日志
	s.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
		Level:     "INFO",
		Source:    "system",
		Message:   fmt.Sprintf("Starting training with image: %s", job.Image),
		Timestamp: time.Now(),
	})

	// 启动执行器
	execCtx, execCancel := context.WithTimeout(context.Background(), time.Duration(job.TimeoutHours)*time.Hour)
	defer execCancel()

	if err := s.executor.Start(execCtx, job); err != nil {
		s.jobRepo.UpdateStatus(ctx, jobID, domain.JobStatusFailed, fmt.Sprintf("Failed to start: %v", err))
		s.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
			Level:     "ERROR",
			Source:    "system",
			Message:   fmt.Sprintf("Failed to start executor: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// 监控任务完成
	go s.monitorJobCompletion(jobID)
}

// monitorJobCompletion 监控任务完成状态
func (s *jobService) monitorJobCompletion(jobID uuid.UUID) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		// 检查执行器状态
		isRunning := s.executor.IsRunning(ctx, jobID)
		
		if !isRunning {
			// 任务已完成，更新状态
			status, _ := s.executor.GetStatus(ctx, jobID)
			
			var finalStatus domain.JobStatus
			var message string
			
			switch status {
			case domain.JobStatusCompleted:
				finalStatus = domain.JobStatusCompleted
				message = "Training completed successfully"
			case domain.JobStatusFailed:
				finalStatus = domain.JobStatusFailed
				message = "Training failed"
			case domain.JobStatusCancelled:
				finalStatus = domain.JobStatusCancelled
				message = "Training cancelled"
			default:
				finalStatus = domain.JobStatusFailed
				message = "Training ended with unknown status"
			}
			
			s.jobRepo.UpdateStatus(ctx, jobID, finalStatus, message)
			s.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
				Level:     "INFO",
				Source:    "system",
				Message:   message,
				Timestamp: time.Now(),
			})
			
			cancel()
			break
		}
		
		cancel()
	}
}

// ListJobsByExperiment 列出实验下的任务
func (s *jobService) ListJobsByExperiment(ctx context.Context, experimentID uuid.UUID, page, pageSize int) ([]*domain.TrainingJob, int64, error) {
	return s.jobRepo.ListByExperiment(ctx, experimentID, page, pageSize)
}

// GetExperimentMetrics 获取实验指标
func (s *jobService) GetExperimentMetrics(ctx context.Context, experimentID uuid.UUID) (map[string]interface{}, error) {
	return s.jobRepo.GetExperimentMetrics(ctx, experimentID)
}

// GetLogs 获取日志
func (s *jobService) GetLogs(ctx context.Context, jobID uuid.UUID, start string, count int64) ([]domain.LogEntry, error) {
	// 先验证任务存在
	if _, err := s.jobRepo.GetByID(ctx, jobID); err != nil {
		return nil, err
	}

	return s.logRepo.ReadLogs(ctx, jobID, start, count)
}

// StreamLogs 流式获取日志
func (s *jobService) StreamLogs(ctx context.Context, jobID uuid.UUID, logChan chan<- domain.LogEntry) error {
	// 先验证任务存在
	if _, err := s.jobRepo.GetByID(ctx, jobID); err != nil {
		return err
	}

	// 先发送历史日志
	historyLogs, err := s.logRepo.ReadLogs(ctx, jobID, "0", 100)
	if err == nil {
		for _, log := range historyLogs {
			select {
			case logChan <- log:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// 然后实时推送新日志
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logs, err := s.logRepo.ReadLogsRealtime(ctx, jobID, 2*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}

		for _, log := range logs {
			select {
			case logChan <- log:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// 检查任务是否还在运行
		if !s.executor.IsRunning(ctx, jobID) {
			// 最后再读一次确保没有遗漏
			logs, _ = s.logRepo.ReadLogsRealtime(ctx, jobID, 500*time.Millisecond)
			for _, log := range logs {
				select {
				case logChan <- log:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		}
	}
}
