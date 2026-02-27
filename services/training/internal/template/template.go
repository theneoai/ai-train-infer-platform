package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/domain"
)

// TemplateType 模板类型
type TemplateType string

const (
	TemplatePyTorch     TemplateType = "pytorch"
	TemplateTensorFlow  TemplateType = "tensorflow"
	TemplateHuggingFace TemplateType = "huggingface"
	TemplateCustom      TemplateType = "custom"
)

// TrainingTemplate 训练模板接口
type TrainingTemplate interface {
	Generate(config *TrainingConfig) (string, error)
	GetDefaultImage(cudaVersion string) string
	GetDefaultCommand() []string
}

// TrainingConfig 训练配置
type TrainingConfig struct {
	Framework       domain.FrameworkType   `json:"framework"`
	ModelName       string                 `json:"model_name"`
	DatasetPath     string                 `json:"dataset_path"`
	OutputPath      string                 `json:"output_path"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	Environment     map[string]string      `json:"environment"`
	Script          string                 `json:"script,omitempty"` // 自定义脚本
	CustomArgs      []string               `json:"custom_args,omitempty"`
}

// TemplateManager 模板管理器
type TemplateManager struct {
	templates map[TemplateType]TrainingTemplate
}

// NewTemplateManager 创建模板管理器
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: map[TemplateType]TrainingTemplate{
			TemplatePyTorch:     NewPyTorchTemplate(),
			TemplateTensorFlow:  NewTensorFlowTemplate(),
			TemplateHuggingFace: NewHuggingFaceTemplate(),
			TemplateCustom:      NewCustomTemplate(),
		},
	}
}

// GetTemplate 获取模板
func (m *TemplateManager) GetTemplate(t TemplateType) (TrainingTemplate, error) {
	if template, ok := m.templates[t]; ok {
		return template, nil
	}
	return nil, fmt.Errorf("template not found: %s", t)
}

// GenerateScript 生成训练脚本
func (m *TemplateManager) GenerateScript(config *TrainingConfig) (string, error) {
	var templateType TemplateType
	
	switch config.Framework {
	case domain.FrameworkPyTorch:
		templateType = TemplatePyTorch
	case domain.FrameworkTensorFlow:
		templateType = TemplateTensorFlow
	default:
		templateType = TemplateCustom
	}

	template, err := m.GetTemplate(templateType)
	if err != nil {
		return "", err
	}

	return template.Generate(config)
}

// GetDefaultImage 获取默认镜像
func (m *TemplateManager) GetDefaultImage(framework domain.FrameworkType, cudaVersion string) string {
	var templateType TemplateType
	
	switch framework {
	case domain.FrameworkPyTorch:
		templateType = TemplatePyTorch
	case domain.FrameworkTensorFlow:
		templateType = TemplateTensorFlow
	default:
		return "python:3.9"
	}

	if template, err := m.GetTemplate(templateType); err == nil {
		return template.GetDefaultImage(cudaVersion)
	}
	return "python:3.9"
}

// PyTorchTemplate PyTorch 训练模板
type PyTorchTemplate struct {
	baseTemplate
}

// NewPyTorchTemplate 创建 PyTorch 模板
func NewPyTorchTemplate() *PyTorchTemplate {
	return &PyTorchTemplate{}
}

// Generate 生成训练脚本
func (t *PyTorchTemplate) Generate(config *TrainingConfig) (string, error) {
	tmpl := `#!/usr/bin/env python3
"""
Auto-generated PyTorch Training Script
Framework: PyTorch
Model: {{.ModelName}}
"""

import os
import sys
import json
import time
import logging
from datetime import datetime

import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import DataLoader, Dataset
from torch.utils.tensorboard import SummaryWriter

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('/output/training.log')
    ]
)
logger = logging.getLogger(__name__)

# 训练配置
class Config:
    MODEL_NAME = "{{.ModelName}}"
    DATASET_PATH = os.environ.get('DATASET_PATH', '{{.DatasetPath}}')
    OUTPUT_PATH = os.environ.get('OUTPUT_PATH', '{{.OutputPath}}')
    
    # 超参数
    {{range $key, $value := .Hyperparameters}}
    {{$key | upper}} = {{$value | formatValue}}
    {{end}}
    
    # 默认值
    EPOCHS = int(os.environ.get('EPOCHS', 10))
    BATCH_SIZE = int(os.environ.get('BATCH_SIZE', 32))
    LEARNING_RATE = float(os.environ.get('LEARNING_RATE', 0.001))
    DEVICE = torch.device('cuda' if torch.cuda.is_available() else 'cpu')

def log_metrics(epoch, step, loss, accuracy=None, val_loss=None, val_accuracy=None):
    """记录训练指标"""
    metrics = {
        'epoch': epoch,
        'step': step,
        'loss': f'{loss:.6f}',
        'timestamp': datetime.now().isoformat()
    }
    if accuracy is not None:
        metrics['accuracy'] = f'{accuracy:.6f}'
    if val_loss is not None:
        metrics['val_loss'] = f'{val_loss:.6f}'
    if val_accuracy is not None:
        metrics['val_accuracy'] = f'{val_accuracy:.6f}'
    
    logger.info(f"METRICS: {json.dumps(metrics)}")

def save_checkpoint(model, optimizer, epoch, path):
    """保存检查点"""
    checkpoint = {
        'epoch': epoch,
        'model_state_dict': model.state_dict(),
        'optimizer_state_dict': optimizer.state_dict(),
    }
    torch.save(checkpoint, path)
    logger.info(f"Checkpoint saved: {path}")

def train():
    """主训练函数"""
    logger.info(f"Starting training with PyTorch {torch.__version__}")
    logger.info(f"Device: {Config.DEVICE}")
    logger.info(f"CUDA available: {torch.cuda.is_available()}")
    
    if torch.cuda.is_available():
        logger.info(f"CUDA device: {torch.cuda.get_device_name(0)}")
        logger.info(f"CUDA memory: {torch.cuda.get_device_properties(0).total_memory / 1e9:.2f} GB")
    
    # 创建输出目录
    os.makedirs(Config.OUTPUT_PATH, exist_ok=True)
    
    # TensorBoard 写入器
    writer = SummaryWriter(os.path.join(Config.OUTPUT_PATH, 'runs'))
    
    try:
        # 训练循环占位符
        # 实际使用时需要替换为具体的模型和数据集加载代码
        logger.info("Training configuration loaded successfully")
        logger.info(f"Model: {Config.MODEL_NAME}")
        logger.info(f"Dataset: {Config.DATASET_PATH}")
        
        # 模拟训练循环（实际使用时替换）
        for epoch in range(Config.EPOCHS):
            logger.info(f"Epoch {epoch + 1}/{Config.EPOCHS}")
            
            # 模拟训练步骤
            for step in range(100):
                # 这里应该是实际的前向/反向传播
                loss = 1.0 / (epoch + step + 1)  # 模拟损失
                
                if step % 10 == 0:
                    log_metrics(epoch + 1, step, loss)
                    writer.add_scalar('Loss/train', loss, epoch * 100 + step)
            
            # 保存检查点
            checkpoint_path = os.path.join(Config.OUTPUT_PATH, f'checkpoint_epoch_{epoch + 1}.pth')
            # save_checkpoint(model, optimizer, epoch, checkpoint_path)
        
        logger.info("Training completed successfully!")
        
    except Exception as e:
        logger.error(f"Training failed: {str(e)}", exc_info=True)
        sys.exit(1)
    finally:
        writer.close()

if __name__ == '__main__':
    train()
`

	funcMap := template.FuncMap{
		"upper":       strings.ToUpper,
		"formatValue": formatPythonValue,
	}

	return t.execute(tmpl, config, funcMap)
}

// GetDefaultImage 获取默认镜像
func (t *PyTorchTemplate) GetDefaultImage(cudaVersion string) string {
	if cudaVersion != "" {
		return fmt.Sprintf("pytorch/pytorch:2.0.0-cuda%s-cudnn8-runtime", cudaVersion)
	}
	return "pytorch/pytorch:2.0.0-cuda11.7-cudnn8-runtime"
}

// GetDefaultCommand 获取默认命令
func (t *PyTorchTemplate) GetDefaultCommand() []string {
	return []string{"python", "train.py"}
}

// TensorFlowTemplate TensorFlow 训练模板
type TensorFlowTemplate struct {
	baseTemplate
}

// NewTensorFlowTemplate 创建 TensorFlow 模板
func NewTensorFlowTemplate() *TensorFlowTemplate {
	return &TensorFlowTemplate{}
}

// Generate 生成训练脚本
func (t *TensorFlowTemplate) Generate(config *TrainingConfig) (string, error) {
	tmpl := `#!/usr/bin/env python3
"""
Auto-generated TensorFlow Training Script
Framework: TensorFlow
Model: {{.ModelName}}
"""

import os
import sys
import json
import time
import logging
from datetime import datetime

import tensorflow as tf
from tensorflow import keras
from tensorflow.keras import layers, Model
from tensorflow.keras.callbacks import ModelCheckpoint, TensorBoard, EarlyStopping

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('/output/training.log')
    ]
)
logger = logging.getLogger(__name__)

# 训练配置
class Config:
    MODEL_NAME = "{{.ModelName}}"
    DATASET_PATH = os.environ.get('DATASET_PATH', '{{.DatasetPath}}')
    OUTPUT_PATH = os.environ.get('OUTPUT_PATH', '{{.OutputPath}}')
    
    # 超参数
    {{range $key, $value := .Hyperparameters}}
    {{$key | upper}} = {{$value | formatValue}}
    {{end}}
    
    # 默认值
    EPOCHS = int(os.environ.get('EPOCHS', 10))
    BATCH_SIZE = int(os.environ.get('BATCH_SIZE', 32))
    LEARNING_RATE = float(os.environ.get('LEARNING_RATE', 0.001))
    
    # GPU 配置
    GPUS = tf.config.experimental.list_physical_devices('GPU')
    if GPUS:
        try:
            for gpu in GPUS:
                tf.config.experimental.set_memory_growth(gpu, True)
            logger.info(f"Using {len(GPUS)} GPU(s)")
        except RuntimeError as e:
            logger.error(f"GPU configuration error: {e}")

class MetricsLogger(keras.callbacks.Callback):
    """自定义指标记录回调"""
    
    def on_epoch_end(self, epoch, logs=None):
        logs = logs or {}
        metrics = {
            'epoch': epoch + 1,
            'timestamp': datetime.now().isoformat()
        }
        for key, value in logs.items():
            metrics[key] = f'{value:.6f}'
        logger.info(f"METRICS: {json.dumps(metrics)}")
    
    def on_batch_end(self, batch, logs=None):
        logs = logs or {}
        if batch % 100 == 0:
            metrics = {
                'step': batch,
                'timestamp': datetime.now().isoformat()
            }
            if 'loss' in logs:
                metrics['loss'] = f'{logs["loss"]:.6f}'
            logger.info(f"METRICS: {json.dumps(metrics)}")

def train():
    """主训练函数"""
    logger.info(f"TensorFlow version: {tf.__version__}")
    logger.info(f"Keras version: {keras.__version__}")
    logger.info(f"GPU available: {len(Config.GPUS) > 0}")
    
    # 创建输出目录
    os.makedirs(Config.OUTPUT_PATH, exist_ok=True)
    
    try:
        logger.info("Training configuration loaded successfully")
        logger.info(f"Model: {Config.MODEL_NAME}")
        logger.info(f"Dataset: {Config.DATASET_PATH}")
        logger.info(f"Epochs: {Config.EPOCHS}")
        logger.info(f"Batch size: {Config.BATCH_SIZE}")
        
        # 回调函数
        callbacks = [
            MetricsLogger(),
            ModelCheckpoint(
                filepath=os.path.join(Config.OUTPUT_PATH, 'model_{epoch:02d}.h5'),
                save_best_only=True,
                monitor='val_loss',
                mode='min'
            ),
            TensorBoard(
                log_dir=os.path.join(Config.OUTPUT_PATH, 'logs'),
                histogram_freq=1
            ),
            EarlyStopping(
                monitor='val_loss',
                patience=5,
                restore_best_weights=True
            )
        ]
        
        # 训练循环占位符
        # 实际使用时需要替换为具体的模型和数据集加载代码
        
        # 示例：创建简单的模型（实际使用时替换）
        # model = create_model()
        # model.compile(
        #     optimizer=keras.optimizers.Adam(learning_rate=Config.LEARNING_RATE),
        #     loss='sparse_categorical_crossentropy',
        #     metrics=['accuracy']
        # )
        
        # # 加载数据
        # train_dataset, val_dataset = load_data(Config.DATASET_PATH, Config.BATCH_SIZE)
        
        # # 训练
        # history = model.fit(
        #     train_dataset,
        #     validation_data=val_dataset,
        #     epochs=Config.EPOCHS,
        #     callbacks=callbacks
        # )
        
        # # 保存最终模型
        # model.save(os.path.join(Config.OUTPUT_PATH, 'final_model.h5'))
        
        # 模拟训练完成
        logger.info("Training completed successfully!")
        
    except Exception as e:
        logger.error(f"Training failed: {str(e)}", exc_info=True)
        sys.exit(1)

if __name__ == '__main__':
    train()
`

	funcMap := template.FuncMap{
		"upper":       strings.ToUpper,
		"formatValue": formatPythonValue,
	}

	return t.execute(tmpl, config, funcMap)
}

// GetDefaultImage 获取默认镜像
func (t *TensorFlowTemplate) GetDefaultImage(cudaVersion string) string {
	if cudaVersion != "" {
		return fmt.Sprintf("tensorflow/tensorflow:2.13.0-gpu")
	}
	return "tensorflow/tensorflow:2.13.0-gpu"
}

// GetDefaultCommand 获取默认命令
func (t *TensorFlowTemplate) GetDefaultCommand() []string {
	return []string{"python", "train.py"}
}

// HuggingFaceTemplate HuggingFace 训练模板
type HuggingFaceTemplate struct {
	baseTemplate
}

// NewHuggingFaceTemplate 创建 HuggingFace 模板
func NewHuggingFaceTemplate() *HuggingFaceTemplate {
	return &HuggingFaceTemplate{}
}

// Generate 生成训练脚本
func (t *HuggingFaceTemplate) Generate(config *TrainingConfig) (string, error) {
	tmpl := `#!/usr/bin/env python3
"""
Auto-generated HuggingFace Training Script
Framework: Transformers
Model: {{.ModelName}}
"""

import os
import sys
import json
import logging
from datetime import datetime

from transformers import (
    AutoModel, AutoTokenizer, AutoConfig,
    TrainingArguments, Trainer,
    DataCollatorWithPadding
)
from datasets import load_dataset
import torch

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('/output/training.log')
    ]
)
logger = logging.getLogger(__name__)

class Config:
    MODEL_NAME = "{{.ModelName}}"
    DATASET_PATH = os.environ.get('DATASET_PATH', '{{.DatasetPath}}')
    OUTPUT_PATH = os.environ.get('OUTPUT_PATH', '{{.OutputPath}}')
    
    # 超参数
    {{range $key, $value := .Hyperparameters}}
    {{$key | upper}} = {{$value | formatValue}}
    {{end}}
    
    # 默认值
    EPOCHS = int(os.environ.get('EPOCHS', 3))
    BATCH_SIZE = int(os.environ.get('BATCH_SIZE', 8))
    LEARNING_RATE = float(os.environ.get('LEARNING_RATE', 5e-5))
    MAX_LENGTH = int(os.environ.get('MAX_LENGTH', 512))

def compute_metrics(eval_pred):
    """计算评估指标"""
    predictions, labels = eval_pred
    # 添加具体的指标计算
    return {"accuracy": 0.0}

class MetricsCallback:
    """自定义指标回调"""
    
    def on_epoch_end(self, args, state, control, **kwargs):
        metrics = state.log_history[-1] if state.log_history else {}
        metrics['epoch'] = state.epoch
        metrics['timestamp'] = datetime.now().isoformat()
        logger.info(f"METRICS: {json.dumps(metrics, default=str)}")

def train():
    """主训练函数"""
    logger.info(f"PyTorch version: {torch.__version__}")
    logger.info(f"CUDA available: {torch.cuda.is_available()}")
    
    os.makedirs(Config.OUTPUT_PATH, exist_ok=True)
    
    try:
        logger.info(f"Loading model: {Config.MODEL_NAME}")
        
        # 加载模型和 tokenizer
        tokenizer = AutoTokenizer.from_pretrained(Config.MODEL_NAME)
        model = AutoModel.from_pretrained(Config.MODEL_NAME)
        
        # 训练参数
        training_args = TrainingArguments(
            output_dir=Config.OUTPUT_PATH,
            num_train_epochs=Config.EPOCHS,
            per_device_train_batch_size=Config.BATCH_SIZE,
            per_device_eval_batch_size=Config.BATCH_SIZE,
            learning_rate=Config.LEARNING_RATE,
            logging_dir=os.path.join(Config.OUTPUT_PATH, 'logs'),
            logging_steps=10,
            save_steps=500,
            evaluation_strategy="epoch",
            save_strategy="epoch",
            load_best_model_at_end=True,
        )
        
        logger.info("Training configuration loaded successfully")
        logger.info("Training completed!")
        
    except Exception as e:
        logger.error(f"Training failed: {str(e)}", exc_info=True)
        sys.exit(1)

if __name__ == '__main__':
    train()
`

	funcMap := template.FuncMap{
		"upper":       strings.ToUpper,
		"formatValue": formatPythonValue,
	}

	return t.execute(tmpl, config, funcMap)
}

// GetDefaultImage 获取默认镜像
func (t *HuggingFaceTemplate) GetDefaultImage(cudaVersion string) string {
	return "huggingface/transformers-pytorch-gpu:latest"
}

// GetDefaultCommand 获取默认命令
func (t *HuggingFaceTemplate) GetDefaultCommand() []string {
	return []string{"python", "train.py"}
}

// CustomTemplate 自定义脚本模板
type CustomTemplate struct {
	baseTemplate
}

// NewCustomTemplate 创建自定义模板
func NewCustomTemplate() *CustomTemplate {
	return &CustomTemplate{}
}

// Generate 生成训练脚本
func (t *CustomTemplate) Generate(config *TrainingConfig) (string, error) {
	if config.Script != "" {
		return config.Script, nil
	}
	return "# Custom training script\n# Please provide your own script", nil
}

// GetDefaultImage 获取默认镜像
func (t *CustomTemplate) GetDefaultImage(cudaVersion string) string {
	return "python:3.9"
}

// GetDefaultCommand 获取默认命令
func (t *CustomTemplate) GetDefaultCommand() []string {
	return []string{"python", "train.py"}
}

// baseTemplate 基础模板
type baseTemplate struct{}

// execute 执行模板
func (t *baseTemplate) execute(tmpl string, config *TrainingConfig, funcMap template.FuncMap) (string, error) {
	template, err := template.New("training").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := template.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// formatPythonValue 格式化 Python 值
func formatPythonValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case int, int32, int64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case []interface{}:
		var parts []string
		for _, item := range val {
			parts = append(parts, formatPythonValue(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
		return fmt.Sprintf("%q", val)
	default:
		return fmt.Sprintf("%q", val)
	}
}

// InjectHyperparameters 将超参数注入命令
func InjectHyperparameters(command []string, hyperparams map[string]interface{}) []string {
	if len(hyperparams) == 0 {
		return command
	}

	// 将超参数转换为命令行参数
	for key, value := range hyperparams {
		paramName := "--" + strings.ToLower(strings.ReplaceAll(key, "_", "-"))
		command = append(command, paramName, fmt.Sprintf("%v", value))
	}

	return command
}

// BuildTrainingCommand 构建训练命令
func BuildTrainingCommand(framework domain.FrameworkType, scriptPath string, hyperparams map[string]interface{}) []string {
	var command []string

	switch framework {
	case domain.FrameworkPyTorch:
		command = []string{"python", scriptPath}
	case domain.FrameworkTensorFlow:
		command = []string{"python", scriptPath}
	default:
		command = []string{"python", scriptPath}
	}

	return InjectHyperparameters(command, hyperparams)
}
