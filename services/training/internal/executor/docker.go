package executor

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/domain"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/repository"
)

// Executor 训练执行器接口
type Executor interface {
	Start(ctx context.Context, job *domain.TrainingJob) error
	Stop(ctx context.Context, jobID uuid.UUID) error
	GetStatus(ctx context.Context, jobID uuid.UUID) (domain.JobStatus, error)
	IsRunning(ctx context.Context, jobID uuid.UUID) bool
	GetContainerStats(ctx context.Context, jobID uuid.UUID) (*ContainerStats, error)
}

// ContainerStats 容器统计信息
type ContainerStats struct {
	ContainerID    string    `json:"container_id"`
	Status         string    `json:"status"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    uint64    `json:"memory_usage"`
	MemoryLimit    uint64    `json:"memory_limit"`
	GPUMemoryUsed  uint64    `json:"gpu_memory_used"`
	GPUMemoryTotal uint64    `json:"gpu_memory_total"`
	GPUtilization  float64   `json:"gpu_utilization"`
	StartedAt      time.Time `json:"started_at"`
}

// DockerExecutor Docker 执行器实现
type DockerExecutor struct {
	client       *client.Client
	logRepo      repository.LogRepository
	metricsRepo  MetricsRepository
	jobs         map[uuid.UUID]*JobProcess
	mu           sync.RWMutex
	network      string
	volumeBase   string
	gpuDetector  *GPUDetector
	errorHandler *ErrorHandler
	metricParser *MetricParser
	maxRetries   int
	retryDelay   time.Duration
}

// JobProcess 任务进程信息
type JobProcess struct {
	JobID         uuid.UUID
	ContainerID   string
	ContainerName string
	CancelFunc    context.CancelFunc
	StartedAt     time.Time
	RetryCount    int
	LastError     error
}

// MetricsRepository 指标仓库接口
type MetricsRepository interface {
	SaveMetrics(ctx context.Context, jobID uuid.UUID, metrics *TrainingMetrics) error
	GetMetrics(ctx context.Context, jobID uuid.UUID, metricType string) ([]*TrainingMetrics, error)
}

// TrainingMetrics 训练指标
type TrainingMetrics struct {
	Timestamp    time.Time       `json:"timestamp"`
	Epoch        int             `json:"epoch,omitempty"`
	Step         int             `json:"step,omitempty"`
	Loss         *float64        `json:"loss,omitempty"`
	Accuracy     *float64        `json:"accuracy,omitempty"`
	ValLoss      *float64        `json:"val_loss,omitempty"`
	ValAccuracy  *float64        `json:"val_accuracy,omitempty"`
	LearningRate *float64        `json:"learning_rate,omitempty"`
	Custom       map[string]float64 `json:"custom,omitempty"`
}

// NewDockerExecutor 创建 Docker 执行器
func NewDockerExecutor(logRepo repository.LogRepository, network, volumeBase string) (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to docker daemon: %w", err)
	}

	gpuDetector := NewGPUDetector(cli)
	errorHandler := NewErrorHandler()
	metricParser := NewMetricParser()

	return &DockerExecutor{
		client:       cli,
		logRepo:      logRepo,
		jobs:         make(map[uuid.UUID]*JobProcess),
		network:      network,
		volumeBase:   volumeBase,
		gpuDetector:  gpuDetector,
		errorHandler: errorHandler,
		metricParser: metricParser,
		maxRetries:   3,
		retryDelay:   5 * time.Second,
	}, nil
}

// Start 启动训练任务
func (e *DockerExecutor) Start(ctx context.Context, job *domain.TrainingJob) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.jobs[job.ID]; exists {
		return fmt.Errorf("job %s is already running", job.ID)
	}

	// 拉取镜像
	if err := e.pullImage(ctx, job.Image); err != nil {
		logger.Warn("Failed to pull image, will try to use local", zap.String("image", job.Image), zap.Error(err))
	}

	// 构建容器配置
	config, hostConfig, err := e.buildContainerConfig(job)
	if err != nil {
		return fmt.Errorf("failed to build container config: %w", err)
	}

	// 创建容器
	containerName := fmt.Sprintf("aitip-train-%s", job.ID.String()[:8])
	resp, err := e.client.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	containerID := resp.ID
	logger.Info("Container created", zap.String("job_id", job.ID.String()), zap.String("container_id", containerID))

	// 启动容器
	if err := e.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		e.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
		return fmt.Errorf("failed to start container: %w", err)
	}

	// 获取容器信息
	info, err := e.client.ContainerInspect(ctx, containerID)
	if err != nil {
		logger.Warn("Failed to inspect container", zap.Error(err))
	}

	// 创建任务上下文
	execCtx, cancel := context.WithCancel(context.Background())

	// 保存进程信息
	process := &JobProcess{
		JobID:         job.ID,
		ContainerID:   containerID,
		ContainerName: containerName,
		CancelFunc:    cancel,
		StartedAt:     time.Now(),
	}
	e.jobs[job.ID] = process

	// 启动日志收集
	go e.collectLogs(execCtx, job.ID, containerID)

	// 启动指标收集
	if e.metricsRepo != nil {
		go e.collectMetrics(execCtx, job.ID, containerID)
	}

	// 启动容器监控
	go e.monitorContainer(execCtx, job, containerID, info.State.Pid)

	return nil
}

// Stop 停止训练任务
func (e *DockerExecutor) Stop(ctx context.Context, jobID uuid.UUID) error {
	e.mu.Lock()
	process, exists := e.jobs[jobID]
	e.mu.Unlock()

	if !exists {
		return e.stopOrphanContainer(ctx, jobID)
	}

	if process.CancelFunc != nil {
		process.CancelFunc()
	}

	stopTimeout := 30
	if err := e.client.ContainerStop(ctx, process.ContainerID, container.StopTimeout(&stopTimeout)); err != nil {
		logger.Warn("Failed to stop container gracefully, forcing", zap.String("container_id", process.ContainerID), zap.Error(err))
		if err := e.client.ContainerKill(ctx, process.ContainerID, "SIGKILL"); err != nil {
			logger.Error("Failed to kill container", zap.Error(err))
		}
	}

	done := make(chan struct{})
	go func() {
		e.client.ContainerWait(ctx, process.ContainerID, container.WaitConditionNotRunning)
		close(done)
	}()

	select {
	case <-done:
		e.cleanupJob(jobID, process.ContainerID)
		return nil
	case <-time.After(35 * time.Second):
		e.cleanupJob(jobID, process.ContainerID)
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

	info, err := e.client.ContainerInspect(ctx, process.ContainerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return domain.JobStatusFailed, nil
		}
		return domain.JobStatusFailed, fmt.Errorf("failed to inspect container: %w", err)
	}

	switch info.State.Status {
	case "running":
		return domain.JobStatusRunning, nil
	case "exited":
		if info.State.ExitCode == 0 {
			return domain.JobStatusCompleted, nil
		}
		return domain.JobStatusFailed, nil
	case "dead":
		return domain.JobStatusFailed, nil
	default:
		return domain.JobStatusPending, nil
	}
}

// IsRunning 检查任务是否在运行
func (e *DockerExecutor) IsRunning(ctx context.Context, jobID uuid.UUID) bool {
	status, err := e.GetStatus(ctx, jobID)
	if err != nil {
		return false
	}
	return status == domain.JobStatusRunning
}

// GetContainerStats 获取容器统计信息
func (e *DockerExecutor) GetContainerStats(ctx context.Context, jobID uuid.UUID) (*ContainerStats, error) {
	e.mu.RLock()
	process, exists := e.jobs[jobID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	info, err := e.client.ContainerInspect(ctx, process.ContainerID)
	if err != nil {
		return nil, err
	}

	stats := &ContainerStats{
		ContainerID: process.ContainerID,
		Status:      info.State.Status,
		StartedAt:   process.StartedAt,
	}

	return stats, nil
}

// pullImage 拉取 Docker 镜像
func (e *DockerExecutor) pullImage(ctx context.Context, image string) error {
	logger.Info("Pulling image", zap.String("image", image))
	
	reader, err := e.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	logger.Info("Image pulled successfully", zap.String("image", image))
	return nil
}

// buildContainerConfig 构建容器配置
func (e *DockerExecutor) buildContainerConfig(job *domain.TrainingJob) (*container.Config, *container.HostConfig, error) {
	env := []string{
		fmt.Sprintf("JOB_ID=%s", job.ID.String()),
		fmt.Sprintf("PROJECT_ID=%s", job.ProjectID.String()),
		fmt.Sprintf("MODEL_NAME=%s", job.ModelName),
		fmt.Sprintf("DATASET_PATH=/data"),
		fmt.Sprintf("OUTPUT_PATH=/output"),
	}

	for key, value := range job.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	if job.Hyperparameters != nil {
		for key, value := range job.Hyperparameters {
			env = append(env, fmt.Sprintf("HP_%s=%v", strings.ToUpper(key), value))
		}
	}

	config := &container.Config{
		Image:        job.Image,
		Env:          env,
		WorkingDir:   "/workspace",
		AttachStdout: true,
		AttachStderr: true,
		Labels: map[string]string{
			"aitip.job.id":     job.ID.String(),
			"aitip.job.name":   job.Name,
			"aitip.project.id": job.ProjectID.String(),
			"aitip.framework":  string(job.Framework),
		},
	}

	if len(job.Command) > 0 {
		config.Cmd = job.Command
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Mounts:     []mount.Mount{},
	}

	if job.DatasetPath != "" {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   job.DatasetPath,
			Target:   "/data",
			ReadOnly: true,
		})
	}

	if job.OutputPath != "" {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: job.OutputPath,
			Target: "/output",
		})
	}

	if e.volumeBase != "" {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: e.volumeBase,
			Target: "/workspace",
		})
	}

	if job.CPUCount > 0 {
		hostConfig.Resources.NanoCPUs = int64(job.CPUCount) * 1e9
	}
	if job.MemoryGB > 0 {
		hostConfig.Resources.Memory = int64(job.MemoryGB) * 1024 * 1024 * 1024
		hostConfig.Resources.MemorySwap = hostConfig.Resources.Memory
	}

	hostConfig.ShmSize = 2 * 1024 * 1024 * 1024

	if job.GPUCount > 0 {
		if e.gpuDetector.IsAvailable() {
			hostConfig.DeviceRequests = e.gpuDetector.GetDeviceRequests(job.GPUCount)
			config.Env = append(config.Env, "NVIDIA_VISIBLE_DEVICES=all")
			config.Env = append(config.Env, "CUDA_VISIBLE_DEVICES=0")
		} else {
			logger.Warn("GPU requested but not available", zap.String("job_id", job.ID.String()), zap.Int("gpu_count", job.GPUCount))
		}
	}

	if e.network != "" {
		hostConfig.NetworkMode = container.NetworkMode(e.network)
	}

	if job.Framework == domain.FrameworkTensorFlow {
		config.ExposedPorts = nat.PortSet{
			"6006/tcp": struct{}{},
		}
	}

	return config, hostConfig, nil
}

// collectLogs 收集容器日志
func (e *DockerExecutor) collectLogs(ctx context.Context, jobID uuid.UUID, containerID string) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}

	reader, err := e.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		logger.Error("Failed to get container logs", zap.Error(err))
		return
	}
	defer reader.Close()

	buf := make([]byte, 8*1024)
	var currentStream string

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				logger.Error("Error reading logs", zap.Error(err))
			}
			return
		}

		if n > 8 {
			streamType := buf[0]
			if streamType == 1 {
				currentStream = "stdout"
			} else if streamType == 2 {
				currentStream = "stderr"
			}

			data := string(buf[8:n])
			lines := strings.Split(data, "\n")

			for _, line := range lines {
				if line = strings.TrimSpace(line); line == "" {
					continue
				}

				level := "INFO"
				if currentStream == "stderr" {
					level = "ERROR"
				} else if strings.Contains(line, "WARN") || strings.Contains(line, "warning") {
					level = "WARN"
				} else if strings.Contains(line, "ERROR") || strings.Contains(line, "error") {
					level = "ERROR"
				}

				entry := &domain.LogEntry{
					Level:     level,
					Source:    currentStream,
					Message:   line,
					Timestamp: time.Now(),
				}

				if metrics := e.metricParser.Parse(line); metrics != nil {
					entry.Message = fmt.Sprintf("[METRICS] %s", line)
					if e.metricsRepo != nil {
						go e.saveMetrics(context.Background(), jobID, metrics)
					}
				}

				logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				e.logRepo.AppendLog(logCtx, jobID, entry)
				cancel()
			}
		}
	}
}

// collectMetrics 收集容器指标
func (e *DockerExecutor) collectMetrics(ctx context.Context, jobID uuid.UUID, containerID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := e.client.ContainerStats(ctx, containerID, false)
			if err != nil {
				continue
			}
			_ = stats.Body.Close()

			if e.gpuDetector.IsAvailable() {
				_ = e.gpuDetector.GetGPUStats(containerID)
			}
		}
	}
}

// monitorContainer 监控容器状态
func (e *DockerExecutor) monitorContainer(ctx context.Context, job *domain.TrainingJob, containerID string, pid int) {
	statusCh, errCh := e.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("Container wait error", zap.String("job_id", job.ID.String()), zap.Error(err))
			e.handleJobFailure(job.ID, containerID, err)
		}
	case status := <-statusCh:
		e.handleContainerExit(job.ID, containerID, status)
	case <-ctx.Done():
		return
	}
}

// handleContainerExit 处理容器退出
func (e *DockerExecutor) handleContainerExit(jobID uuid.UUID, containerID string, status container.WaitResponse) {
	exitCode := status.StatusCode
	logger.Info("Container exited", zap.String("job_id", jobID.String()), zap.Int64("exit_code", exitCode))

	var finalStatus domain.JobStatus
	var message string

	if exitCode == 0 {
		finalStatus = domain.JobStatusCompleted
		message = "Training completed successfully"
	} else {
		if e.errorHandler.IsOOM(containerID, exitCode) {
			finalStatus = domain.JobStatusFailed
			message = "Training failed: Out of Memory (OOM)"
		} else {
			finalStatus = domain.JobStatusFailed
			message = fmt.Sprintf("Training failed with exit code %d", exitCode)
		}

		process := e.jobs[jobID]
		if process != nil && process.RetryCount < e.maxRetries {
			if e.errorHandler.ShouldRetry(exitCode) {
				logger.Info("Retrying job", zap.String("job_id", jobID.String()), zap.Int("attempt", process.RetryCount+1))
			}
		}
	}

	level := "INFO"
	if finalStatus == domain.JobStatusFailed {
		level = "ERROR"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	e.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
		Level:     level,
		Source:    "system",
		Message:   message,
		Timestamp: time.Now(),
	})
	cancel()

	e.cleanupJob(jobID, containerID)
}

// handleJobFailure 处理任务失败
func (e *DockerExecutor) handleJobFailure(jobID uuid.UUID, containerID string, err error) {
	logger.Error("Job failed", zap.String("job_id", jobID.String()), zap.Error(err))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	e.logRepo.AppendLog(ctx, jobID, &domain.LogEntry{
		Level:     "ERROR",
		Source:    "system",
		Message:   fmt.Sprintf("Job failed: %v", err),
		Timestamp: time.Now(),
	})
	cancel()

	e.cleanupJob(jobID, containerID)
}

// cleanupJob 清理任务资源
func (e *DockerExecutor) cleanupJob(jobID uuid.UUID, containerID string) {
	e.mu.Lock()
	delete(e.jobs, jobID)
	e.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	e.client.ContainerStop(ctx, containerID, container.StopTimeout(nil))
}

// stopOrphanContainer 停止孤儿容器
func (e *DockerExecutor) stopOrphanContainer(ctx context.Context, jobID uuid.UUID) error {
	containerName := fmt.Sprintf("aitip-train-%s", jobID.String()[:8])
	
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", containerName)

	containers, err := e.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return err
	}

	for _, c := range containers {
		e.client.ContainerStop(ctx, c.ID, container.StopTimeout(nil))
	}

	return nil
}

// SetMetricsRepository 设置指标仓库
func (e *DockerExecutor) SetMetricsRepository(repo MetricsRepository) {
	e.metricsRepo = repo
}

// saveMetrics 保存训练指标
func (e *DockerExecutor) saveMetrics(ctx context.Context, jobID uuid.UUID, metrics *TrainingMetrics) {
	if e.metricsRepo == nil {
		return
	}

	if err := e.metricsRepo.SaveMetrics(ctx, jobID, metrics); err != nil {
		logger.Error("Failed to save metrics", zap.Error(err))
	}
}

// Close 关闭执行器
func (e *DockerExecutor) Close() error {
	e.mu.RLock()
	jobs := make(map[uuid.UUID]*JobProcess)
	for k, v := range e.jobs {
		jobs[k] = v
	}
	e.mu.RUnlock()

	for jobID := range jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		e.Stop(ctx, jobID)
		cancel()
	}

	return e.client.Close()
}

// MetricParser 指标解析器
type MetricParser struct {
	pytorchLossPattern *regexp.Regexp
	pytorchAccPattern  *regexp.Regexp
	tfLossPattern      *regexp.Regexp
	tfAccPattern       *regexp.Regexp
	epochPattern       *regexp.Regexp
	stepPattern        *regexp.Regexp
}

// NewMetricParser 创建指标解析器
func NewMetricParser() *MetricParser {
	return &MetricParser{
		pytorchLossPattern: regexp.MustCompile(`(?i)loss[:=\s]+([0-9.]+(?:e[+-]?[0-9]+)?)`),
		pytorchAccPattern:  regexp.MustCompile(`(?i)accuracy|acc[:=\s]+([0-9.]+)`),
		tfLossPattern:      regexp.MustCompile(`(?i)loss:\s*([0-9.]+)`),
		tfAccPattern:       regexp.MustCompile(`(?i)accuracy:\s*([0-9.]+)`),
		epochPattern:       regexp.MustCompile(`(?i)epoch[:/\s]+(\d+)`),
		stepPattern:        regexp.MustCompile(`(?i)step|batch[:/\s]+(\d+)`),
	}
}

// Parse 解析日志行中的指标
func (p *MetricParser) Parse(line string) *TrainingMetrics {
	metrics := &TrainingMetrics{
		Timestamp: time.Now(),
		Custom:    make(map[string]float64),
	}

	hasMetrics := false

	if matches := p.epochPattern.FindStringSubmatch(line); len(matches) > 1 {
		if epoch, err := strconv.Atoi(matches[1]); err == nil {
			metrics.Epoch = epoch
			hasMetrics = true
		}
	}

	if matches := p.stepPattern.FindStringSubmatch(line); len(matches) > 1 {
		if step, err := strconv.Atoi(matches[1]); err == nil {
			metrics.Step = step
			hasMetrics = true
		}
	}

	if matches := p.pytorchLossPattern.FindStringSubmatch(line); len(matches) > 1 {
		if loss, err := strconv.ParseFloat(matches[1], 64); err == nil {
			metrics.Loss = &loss
			hasMetrics = true
		}
	}

	if matches := p.pytorchAccPattern.FindStringSubmatch(line); len(matches) > 1 {
		if acc, err := strconv.ParseFloat(matches[1], 64); err == nil {
			metrics.Accuracy = &acc
			hasMetrics = true
		}
	}

	if hasMetrics {
		return metrics
	}
	return nil
}
