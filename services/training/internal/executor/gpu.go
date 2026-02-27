package executor

import (
	"context"
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
	client        *client.Client
	available     bool
	cudaVersion   string
	driverVersion string
	gpuCount      int
	gpuInfo       []GPUInfo
	mu            sync.RWMutex
}

// GPUInfo GPU 信息
type GPUInfo struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	Utilization float64 `json:"utilization"`
	Temperature int     `json:"temperature"`
	PowerDraw   float64 `json:"power_draw"`
}

// NewGPUDetector 创建 GPU 检测器
func NewGPUDetector(client *client.Client) *GPUDetector {
	detector := &GPUDetector{client: client}
	detector.detect()
	return detector
}

// detect 检测 GPU 可用性
func (d *GPUDetector) detect() {
	if !d.detectNvidiaRuntime() {
		logger.Info("NVIDIA Docker runtime not detected")
		return
	}

	if err := d.queryGPUInfo(); err != nil {
		logger.Warn("Failed to query GPU info", zap.Error(err))
		return
	}

	d.available = true
	logger.Info("GPU support enabled", zap.Int("gpu_count", d.gpuCount), zap.String("cuda_version", d.cudaVersion), zap.String("driver_version", d.driverVersion))
}

// detectNvidiaRuntime 检测 NVIDIA Docker runtime
func (d *GPUDetector) detectNvidiaRuntime() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := d.client.Info(ctx)
	if err != nil {
		return false
	}

	for name := range info.Runtimes {
		if strings.Contains(strings.ToLower(name), "nvidia") {
			return true
		}
	}

	if info.DefaultRuntime != "" && strings.Contains(strings.ToLower(info.DefaultRuntime), "nvidia") {
		return true
	}

	if _, err := exec.LookPath("nvidia-ctk"); err == nil {
		return true
	}

	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		return true
	}

	return false
}

// queryGPUInfo 查询 GPU 信息
func (d *GPUDetector) queryGPUInfo() error {
	output, err := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu,power.draw", "--format=csv,noheader,nounits").Output()
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

	if output, err := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader,nounits").Output(); err == nil {
		d.driverVersion = strings.TrimSpace(string(output))
	}

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

	if gpuCount > d.gpuCount {
		gpuCount = d.gpuCount
	}

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

	result := make([]GPUInfo, len(d.gpuInfo))
	copy(result, d.gpuInfo)
	return result
}

// GetGPUStats 获取容器的 GPU 统计
func (d *GPUDetector) GetGPUStats(containerID string) map[string]interface{} {
	if !d.IsAvailable() {
		return nil
	}

	output, err := exec.Command("nvidia-smi", "pmon", "-s", "um", "-c", "1", "-o", "T").Output()
	if err != nil {
		return nil
	}

	return map[string]interface{}{
		"container_id": containerID,
		"timestamp":    time.Now(),
		"raw_output":   string(output),
	}
}

// GetRecommendedImage 获取推荐的 GPU 镜像
func (d *GPUDetector) GetRecommendedImage(framework string) string {
	cudaVersion := d.cudaVersion
	if cudaVersion == "" {
		cudaVersion = "11.7"
	}

	switch framework {
	case "pytorch":
		return fmt.Sprintf("pytorch/pytorch:2.0.0-cuda%s-cudnn8-runtime", cudaVersion)
	case "tensorflow":
		return "tensorflow/tensorflow:2.13.0-gpu"
	default:
		return fmt.Sprintf("nvidia/cuda:%s-runtime-ubuntu20.04", cudaVersion)
	}
}
