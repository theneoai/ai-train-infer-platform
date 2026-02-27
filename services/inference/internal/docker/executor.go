package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/domain"
)

// Executor Docker 执行器
type Executor struct {
	client       *client.Client
	network      string
	modelCache   string
}

// NewExecutor 创建 Docker 执行器
func NewExecutor(dockerHost, network, modelCache string) (*Executor, error) {
	var cli *client.Client
	var err error

	if dockerHost != "" {
		cli, err = client.NewClientWithOpts(
			client.WithHost(dockerHost),
			client.WithAPIVersionNegotiation(),
		)
	} else {
		cli, err = client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping docker daemon: %w", err)
	}

	return &Executor{
		client:     cli,
		network:    network,
		modelCache: modelCache,
	}, nil
}

// Close 关闭 Docker 客户端
func (e *Executor) Close() error {
	return e.client.Close()
}

// CreateContainer 创建推理容器
func (e *Executor) CreateContainer(ctx context.Context, service *domain.InferenceService, modelPath string) (string, string, error) {
	containerName := fmt.Sprintf("inference-%s", service.ID.String()[:8])
	
	// 获取镜像
	image := e.getImageForType(service.Type)
	
	// 拉取镜像
	logger.Info("Pulling image", zap.String("image", image))
	if err := e.pullImage(ctx, image); err != nil {
		return "", "", fmt.Errorf("failed to pull image: %w", err)
	}

	// 准备端口映射
	hostPort := strconv.Itoa(service.HostPort)
	containerPort := strconv.Itoa(service.ContainerPort)
	
	portBindings := nat.PortMap{
		nat.Port(containerPort + "/tcp"): []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: hostPort},
		},
	}
	exposedPorts := nat.PortSet{
		nat.Port(containerPort + "/tcp"): struct{}{},
	}

	// 准备环境变量
	env := e.buildEnvironment(service, modelPath)

	// 准备挂载
	mounts := e.buildMounts(service, modelPath)

	// 准备资源限制
	resources := container.Resources{
		Memory:    int64(service.MemoryGB) * 1024 * 1024 * 1024,
		CPUQuota:  int64(service.CPUCount) * 100000,
		CPUPeriod: 100000,
	}

	// 如果有 GPU
	if service.GPUCount > 0 {
		resources.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       "nvidia",
				Count:        service.GPUCount,
				Capabilities: [][]string{{"gpu"}},
			},
		}
	}

	// 创建容器配置
	config := &container.Config{
		Image:        image,
		Env:          env,
		ExposedPorts: exposedPorts,
		Hostname:     containerName,
		Labels: map[string]string{
			"app":           "inference",
			"service_id":    service.ID.String(),
			"project_id":    service.ProjectID.String(),
			"model_id":      service.ModelID.String(),
		},
	}

	// 主机配置
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		Resources:    resources,
		NetworkMode:  container.NetworkMode(e.network),
		RestartPolicy: container.RestartPolicy{
			Name:              "unless-stopped",
			MaximumRetryCount: 3,
		},
	}

	// 创建容器
	resp, err := e.client.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", "", fmt.Errorf("failed to create container: %w", err)
	}

	logger.Info("Container created",
		zap.String("container_id", resp.ID[:12]),
		zap.String("container_name", containerName),
	)

	return resp.ID, containerName, nil
}

// StartContainer 启动容器
func (e *Executor) StartContainer(ctx context.Context, containerID string) error {
	if err := e.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// StopContainer 停止容器
func (e *Executor) StopContainer(ctx context.Context, containerID string, timeout int) error {
	timeoutDuration := time.Duration(timeout) * time.Second
	if err := e.client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

// RemoveContainer 删除容器
func (e *Executor) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	if err := e.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}

// GetContainerInfo 获取容器信息
func (e *Executor) GetContainerInfo(ctx context.Context, containerID string) (*domain.ContainerInfo, error) {
	info, err := e.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// 解析端口映射
	ports := make(map[string][]domain.PortBinding)
	for port, bindings := range info.NetworkSettings.Ports {
		portStr := string(port)
		for _, binding := range bindings {
			ports[portStr] = append(ports[portStr], domain.PortBinding{
				HostIP:   binding.HostIP,
				HostPort: binding.HostPort,
			})
		}
	}

	// 获取健康状态
	health := "none"
	if info.State.Health != nil {
		health = info.State.Health.Status
	}

	return &domain.ContainerInfo{
		ContainerID:   info.ID,
		ContainerName: strings.TrimPrefix(info.Name, "/"),
		Image:         info.Config.Image,
		Status:        info.State.Status,
		State:         info.State.State,
		ExitCode:      info.State.ExitCode,
		StartedAt:     parseTime(info.State.StartedAt),
		FinishedAt:    parseTime(info.State.FinishedAt),
		Ports:         ports,
		Health:        health,
	}, nil
}

// HealthCheck 健康检查
func (e *Executor) HealthCheck(ctx context.Context, containerID string) (bool, error) {
	info, err := e.GetContainerInfo(ctx, containerID)
	if err != nil {
		return false, err
	}

	// 检查容器状态
	if info.State != "running" {
		return false, fmt.Errorf("container is not running: %s", info.State)
	}

	// 检查健康状态
	if info.Health != "" && info.Health != "none" {
		return info.Health == "healthy", nil
	}

	// 如果没有健康检查配置，假设健康
	return true, nil
}

// GetContainerLogs 获取容器日志
func (e *Executor) GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	}

	reader, err := e.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// 读取日志
	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return string(logs), nil
}

// ListContainers 列出推理容器
func (e *Executor) ListContainers(ctx context.Context, serviceID *uuid.UUID) ([]types.Container, error) {
	filters := filters.NewArgs()
	filters.Add("label", "app=inference")
	if serviceID != nil {
		filters.Add("label", fmt.Sprintf("service_id=%s", serviceID.String()))
	}

	containers, err := e.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

// getImageForType 根据推理类型获取镜像
func (e *Executor) getImageForType(inferenceType domain.InferenceType) string {
	switch inferenceType {
	case domain.InferenceTypeTriton:
		return "nvcr.io/nvidia/tritonserver:24.01-py3"
	case domain.InferenceTypeVLLM:
		return "vllm/vllm-openai:latest"
	default:
		return "nvcr.io/nvidia/tritonserver:24.01-py3"
	}
}

// pullImage 拉取镜像
func (e *Executor) pullImage(ctx context.Context, image string) error {
	reader, err := e.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// 等待拉取完成
	_, err = io.Copy(io.Discard, reader)
	return err
}

// buildEnvironment 构建环境变量
func (e *Executor) buildEnvironment(service *domain.InferenceService, modelPath string) []string {
	env := []string{
		fmt.Sprintf("INFERENCE_TYPE=%s", service.Type),
		fmt.Sprintf("MODEL_PATH=%s", modelPath),
		fmt.Sprintf("SERVICE_ID=%s", service.ID),
		"CUDA_VISIBLE_DEVICES=0",
	}

	// 添加自定义环境变量
	for k, v := range service.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// buildMounts 构建挂载
func (e *Executor) buildMounts(service *domain.InferenceService, modelPath string) []mount.Mount {
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: modelPath,
			Target: "/models",
			ReadOnly: true,
		},
	}

	// 添加缓存目录
	if e.modelCache != "" {
		cachePath := filepath.Join(e.modelCache, service.ID.String())
		os.MkdirAll(cachePath, 0755)
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: cachePath,
			Target: "/cache",
		})
	}

	return mounts
}

// parseTime 解析时间字符串
func parseTime(timeStr string) *time.Time {
	if timeStr == "" || timeStr == "0001-01-01T00:00:00Z" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return nil
	}
	return &t
}
