package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/repository"
)

// Executor 训练执行器接口
type Executor interface {
	Start(ctx context.Context, job *domain.TrainingJob) error
	Stop(ctx context.Context, jobID uuid.UUID) error
	GetStatus(ctx context.Context, jobID uuid.UUID) (domain.JobStatus, error)
	IsRunning(ctx context.Context, jobID uuid.UUID) bool
}

// DockerExecutor Docker 执行器实现
type DockerExecutor struct {
	logRepo    repository.LogRepository
	jobs       map[uuid.UUID]*JobProcess
	mu         sync.RWMutex
	network    string
	volumeBase string
}

// JobProcess 任务进程信息
type JobProcess struct {
	JobID       uuid.UUID
	Cmd         *exec.Cmd
	CancelFunc  context.CancelFunc
	ContainerID string
	StartedAt   time.Time
}

// NewDockerExecutor 创建 Docker 执行器
func NewDockerExecutor(logRepo repository.LogRepository, network, volumeBase string) *DockerExecutor {
	return &DockerExecutor{
		logRepo:    logRepo,
		jobs:       make(map[uuid.UUID]*JobProcess),
		network:    network,
		volumeBase: volumeBase,
	}
}

// Start 启动训练任务
func (e *DockerExecutor) Start(ctx context.Context, job *domain.TrainingJob) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.jobs[job.ID]; exists {
		return fmt.Errorf("job %s is already running", job.ID)
	}

	args := e.buildDockerArgs(job)
	execCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(execCtx, "docker", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start docker: %w", err)
	}

	process := &JobProcess{
		JobID:       job.ID,
		Cmd:         cmd,
		CancelFunc:  cancel,
		StartedAt:   time.Now(),
	}
	e.jobs[job.ID] = process

	go e.collectLogs(job.ID, stdout, "stdout")
	go e.collectLogs(job.ID, stderr, "stderr")
	go e.monitorJob(job.ID, cmd, cancel)

	return nil
}

// Stop 停止训练任务
func (e *DockerExecutor) Stop(ctx context.Context, jobID uuid.UUID) error {
	e.mu.Lock()
	process, exists := e.jobs[jobID]
	e.mu.Unlock()

	if !exists {
		return nil
	}

	if process.CancelFunc != nil {
		process.CancelFunc()
	}

	done := make(chan error, 1)
	go func() {
		done <- process.Cmd.Wait()
	}()

	select {
	case <-done:
		e.mu.Lock()
		delete(e.jobs, jobID)
		e.mu.Unlock()
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for job %s to stop", jobID)
	}
}

// GetStatus 获取任务状态
func (e *DockerExecutor) GetStatus(ctx context.Context, jobID uuid.UUID) (domain.JobStatus, error) {
	e.mu.RLock()
	process, exists := e.jobs[jobID]
	e.mu.RUnlock()

	if !exists {
		return domain.JobStatusPending, nil
	}

	if process.Cmd.ProcessState != nil && process.Cmd.ProcessState.Exited() {
		if process.Cmd.ProcessState.Success() {
			return domain.JobStatusCompleted, nil
		}
		return domain.JobStatusFailed, nil
	}

	return domain.JobStatusRunning, nil
}

// IsRunning 检查任务是否在运行
func (e *DockerExecutor) IsRunning(ctx context.Context, jobID uuid.UUID) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	process, exists := e.jobs[jobID]
	if !exists {
		return false
	}

	return process.Cmd.ProcessState == nil || !process.Cmd.ProcessState.Exited()
}

// buildDockerArgs 构建 Docker 命令参数
func (e *DockerExecutor) buildDockerArgs(job *domain.TrainingJob) []string {
	args := []string{
		"run",
		"--rm",
		"--name", fmt.Sprintf("aitip-train-%s", job.ID.String()[:8]),
	}

	if job.GPUCount > 0 {
		args = append(args, "--gpus", fmt.Sprintf("nvidia=%d", job.GPUCount))
	}

	if job.CPUCount > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%d", job.CPUCount))
	}

	if job.MemoryGB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dg", job.MemoryGB))
	}

	if e.network != "" {
		args = append(args, "--network", e.network)
	}

	if e.volumeBase != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/workspace", e.volumeBase))
	}
	if job.DatasetPath != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/data:ro", job.DatasetPath))
	}
	if job.OutputPath != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/output", job.OutputPath))
	}

	for key, value := range job.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, "-w", "/workspace")
	args = append(args, job.Image)

	if len(job.Command) > 0 {
		args = append(args, job.Command...)
	}

	return args
}

// collectLogs 收集日志
func (e *DockerExecutor) collectLogs(jobID uuid.UUID, reader io.Reader, source string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		level := "INFO"
		if strings.Contains(line, "ERROR") || strings.Contains(line, "error") || strings.Contains(line, "Error") {
			level = "ERROR"
		} else if strings.Contains(line, "WARN") || strings.Contains(line, "warning") {
			level = "WARN"
		}

		entry := &domain.LogEntry{
			Level:     level,
			Source:    source,
			Message:   line,
			Timestamp: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		e.logRepo.AppendLog(ctx, jobID, entry)
		cancel()
	}
}

// monitorJob 监控任务执行
func (e *DockerExecutor) monitorJob(jobID uuid.UUID, cmd *exec.Cmd, cancel context.CancelFunc) {
	err := cmd.Wait()

	e.mu.Lock()
	delete(e.jobs, jobID)
	e.mu.Unlock()

	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCtx()

	if err != nil {
		e.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
			Level:     "ERROR",
			Source:    "system",
			Message:   fmt.Sprintf("Job failed: %v", err),
			Timestamp: time.Now(),
		})
	} else {
		e.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
			Level:     "INFO",
			Source:    "system",
			Message:   "Job completed successfully",
			Timestamp: time.Now(),
		})
	}
}
