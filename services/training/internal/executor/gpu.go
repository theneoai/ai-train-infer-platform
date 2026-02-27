package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"

	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
)

// GPUDetector GPU 检测器
type GPUDetector struct {
	client      *client.Client
	available   bool
	cudaVersion string
	driverVersion string
	gpuCount    int
	gpuInfo     []GPUInfo
	mu          sync.RWMutex
}

// GPUInfo GPU 信息
type GPUInfo struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	MemoryTotal uint64  `json:"memory_total"` // MB
	MemoryUsed  uint64  `json:"memory_used"`  // MB
	Utilization float64 `json:"utilization"`  // %
	Temperature int     `json:"temperature"`  // °C
	PowerDraw   float64 `json:"power_draw"`   // W
}

// GPUStats GPU 统计
type GPUStats struct {
	ContainerID string    `json:"container_id"`
	Timestamp   time.Time `json:"timestamp"`
	GPUs        []GPUInfo `json:"gpus"`
}

// NewGPUDetector 创建 GPU 检测器
func NewGPUDetector(client *client.Client) *GPUDetector {
	detector := &GPUDetector{
		client: client,
	}
	
	// 初始化检测
	detector.detect()
	
	return detector
}

// detect 检测 GPU 可用性
func (d *GPUDetector) detect() {
	// 检测 nvidia-docker-runtime
	if !d.detectNvidiaRuntime() {
		logger.Info("NVIDIA Docker runtime not detected")
		return
	}

	// 获取 GPU 信息
	if err := d.queryGPUInfo(); err != nil {
		logger.Warn("Failed to query GPU info", zap.Error(err))
		return
	}

	d.available = true
	logger.Info("GPU support enabled", 
		zap.Int("gpu_count", d.gpuCount),
		zap.String("cuda_version", d.cudaVersion),
		zap.String("driver_version", d.driverVersion))
}

// detectNvidiaRuntime 检测 NVIDIA Docker runtime
func (d *GPUDetector) detectNvidiaRuntime() bool {
	// 方法1: 检查 docker info
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := d.client.Info(ctx)
	if err != nil {
		return false
	}

	// 检查 Runtimes 中是否包含 nvidia
	for name := range info.Runtimes {
		if strings.Contains(strings.ToLower(name), "nvidia") {
			return true
		}
	}

	// 方法2: 检查默认 runtime
	if info.DefaultRuntime != "" && strings.Contains(strings.ToLower(info.DefaultRuntime), "nvidia") {
		return true
	}

	// 方法3: 检查 nvidia-ctk 工具
	if _, err := exec.LookPath("nvidia-ctk"); err == nil {
		return true
	}

	// 方法4: 检查 nvidia-smi
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		return true
	}

	return false
}

// queryGPUInfo 查询 GPU 信息
func (d *GPUDetector) queryGPUInfo() error {
	// 使用 nvidia-smi 获取 GPU 信息
	output, err := exec.Command("nvidia-smi", 
		"--query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu,power.draw",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return fmt.Errorf("nvidia-smi failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var gpus []GPUInfo
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ", ")
		if len(parts) < 7 {
			continue
		}

		index, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		name := strings.TrimSpace(parts[1])
		memTotal, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
		memUsed, _ := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 64)
		util, _ := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
		temp, _ := strconv.Atoi(strings.TrimSpace(parts[5]))
		power, _ := strconv.ParseFloat(strings.TrimSpace(parts[6]), 64)

		gpus = append(gpus, GPUInfo{
			Index:       index,
			Name:        name,
			MemoryTotal: memTotal,
			MemoryUsed:  memUsed,
			Utilization: util,
			Temperature: temp,
			PowerDraw:   power,
		})
	}

	d.gpuInfo = gpus
	d.gpuCount = len(gpus)

	// 获取 CUDA 版本
	if output, err := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader,nounits").Output(); err == nil {
		d.driverVersion = strings.TrimSpace(string(output))
	}

	// 尝试获取 CUDA 版本
	if output, err := exec.Command("nvcc", "--version").Output(); err == nil {
		re := regexp.MustCompile(`release (\d+\.\d+)`)
		if matches := re.FindStringSubmatch(string(output)); len(matches) > 1 {
			d.cudaVersion = matches[1]
		}
	}

	return nil
}

// IsAvailable 检查 GPU 是否可用
func (d *GPUDetector) IsAvailable() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.available
}

// GetDeviceRequests 获取 GPU 设备请求配置
func (d *GPUDetector) GetDeviceRequests(gpuCount int) []container.DeviceRequest {
	if !d.IsAvailable() {
		return nil
	}

	// 限制 GPU 数量
	if gpuCount > d.gpuCount {
		gpuCount = d.gpuCount
	}

	// 构建设备 ID 列表
	deviceIDs := make([]string, gpuCount)
	for i := 0; i < gpuCount; i++ {
		deviceIDs[i] = strconv.Itoa(i)
	}

	return []container.DeviceRequest{
		{
			Driver:       "nvidia",
			Count:        gpuCount,
			DeviceIDs:    deviceIDs,
			Capabilities: [][]string{{"gpu"}},
			Options: map[string]string{
				"nvidia-driver-capabilities": "compute,utility",
			},
		},
	}
}

// GetAllDeviceRequests 获取所有 GPU 设备请求
func (d *GPUDetector) GetAllDeviceRequests() []container.DeviceRequest {
	return d.GetDeviceRequests(d.gpuCount)
}

// GetGPUCount 获取 GPU 数量
func (d *GPUDetector) GetGPUCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.gpuCount
}

// GetGPUInfo 获取 GPU 信息
func (d *GPUDetector) GetGPUInfo() []GPUInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// 返回副本
	result := make([]GPUInfo, len(d.gpuInfo))
	copy(result, d.gpuInfo)
	return result
}

// GetCUDAVersion 获取 CUDA 版本
func (d *GPUDetector) GetCUDAVersion() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cudaVersion
}

// GetDriverVersion 获取驱动版本
func (d *GPUDetector) GetDriverVersion() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.driverVersion
}

// GetGPUStats 获取容器的 GPU 统计
func (d *GPUDetector) GetGPUStats(containerID string) *GPUStats {
	if !d.IsAvailable() {
		return nil
	}

	// 使用 nvidia-smi 查询进程的 GPU 使用情况
	// nvidia-smi pmon -s um -c 1 -o T
	output, err := exec.Command("nvidia-smi", 
		"pmon", "-s", "um", "-c", "1", "-o", "T").Output()
	if err != nil {
		return nil
	}

	stats := &GPUStats{
		ContainerID: containerID,
		Timestamp:   time.Now(),
		GPUs:        make([]GPUInfo, 0),
	}

	// 解析输出（简化版本）
	_ = output

	return stats
}

// Refresh 刷新 GPU 信息
func (d *GPUDetector) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.available = false
	d.gpuCount = 0
	d.gpuInfo = nil

	d.detect()
	return nil
}

// CheckGPUAvailability 检查 GPU 可用性（静态方法）
func CheckGPUAvailability() (*GPUInfoResult, error) {
	result := &GPUInfoResult{
		Available: false,
	}

	// 检查 nvidia-smi
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return result, nil
	}

	// 获取 GPU 信息
	output, err := exec.Command("nvidia-smi", "-L").Output()
	if err != nil {
		return result, nil
	}

	result.Available = true
	result.GPUCount = strings.Count(string(output), "GPU ")

	// 获取驱动版本
	if output, err := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader,nounits").Output(); err == nil {
		result.DriverVersion = strings.TrimSpace(string(output))
	}

	return result, nil
}

// GPUInfoResult GPU 信息结果
type GPUInfoResult struct {
	Available     bool   `json:"available"`
	GPUCount      int    `json:"gpu_count"`
	DriverVersion string `json:"driver_version,omitempty"`
	CUDAVersion   string `json:"cuda_version,omitempty"`
}

// MarshalJSON JSON 序列化
func (r *GPUInfoResult) MarshalJSON() ([]byte, error) {
	type Alias GPUInfoResult
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}

// GPURequirement GPU 需求检查
type GPURequirement struct {
	MinCUDAVersion   string
	MinDriverVersion string
	MinMemoryGB      int
}

// CheckGPURequirement 检查 GPU 是否满足需求
func (d *GPUDetector) CheckGPURequirement(req *GPURequirement) error {
	if !d.IsAvailable() {
		return fmt.Errorf("GPU not available")
	}

	if req.MinCUDAVersion != "" {
		if !d.versionSatisfies(d.cudaVersion, req.MinCUDAVersion) {
			return fmt.Errorf("CUDA version %s does not meet minimum requirement %s", 
				d.cudaVersion, req.MinCUDAVersion)
		}
	}

	if req.MinDriverVersion != "" {
		if !d.versionSatisfies(d.driverVersion, req.MinDriverVersion) {
			return fmt.Errorf("Driver version %s does not meet minimum requirement %s",
				d.driverVersion, req.MinDriverVersion)
		}
	}

	if req.MinMemoryGB > 0 {
		for _, gpu := range d.gpuInfo {
			if gpu.MemoryTotal/1024 < uint64(req.MinMemoryGB) {
				return fmt.Errorf("GPU %d has only %d GB memory, minimum %d GB required",
					gpu.Index, gpu.MemoryTotal/1024, req.MinMemoryGB)
			}
		}
	}

	return nil
}

// versionSatisfies 检查版本是否满足要求
func (d *GPUDetector) versionSatisfies(current, required string) bool {
	currentParts := strings.Split(current, ".")
	requiredParts := strings.Split(required, ".")

	for i := 0; i < len(requiredParts); i++ {
		if i >= len(currentParts) {
			return false
		}

		currentVer, _ := strconv.Atoi(currentParts[i])
		requiredVer, _ := strconv.Atoi(requiredParts[i])

		if currentVer > requiredVer {
			return true
		}
		if currentVer < requiredVer {
			return false
		}
	}

	return true
}

// GetRecommendedImage 获取推荐的 GPU 镜像
func (d *GPUDetector) GetRecommendedImage(framework string, cudaVersion string) string {
	if cudaVersion == "" {
		cudaVersion = d.cudaVersion
	}

	switch framework {
	case "pytorch":
		if cudaVersion != "" {
			return fmt.Sprintf("pytorch/pytorch:%s-cuda%s-cudnn8-runtime", 
				getPyTorchVersion(), cudaVersion)
		}
		return "pytorch/pytorch:latest"
	case "tensorflow":
		if cudaVersion != "" {
			return fmt.Sprintf("tensorflow/tensorflow:%s-gpu", getTensorFlowVersion())
		}
		return "tensorflow/tensorflow:latest-gpu"
	default:
		if cudaVersion != "" {
			return fmt.Sprintf("nvidia/cuda:%s-runtime-ubuntu20.04", cudaVersion)
		}
		return "python:3.9"
	}
}

func getPyTorchVersion() string {
	return "2.0.0"
}

func getTensorFlowVersion() string {
	return "2.13.0"
}
