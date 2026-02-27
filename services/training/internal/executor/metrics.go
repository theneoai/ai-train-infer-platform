package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
)

// MetricCollector 指标收集器
type MetricCollector struct {
	db           *gorm.DB
	upgrader     websocket.Upgrader
	clients      map[uuid.UUID]map[*MetricClient]bool
	clientsMu    sync.RWMutex
	broadcast    chan *MetricMessage
	register     chan *MetricClient
	unregister   chan *MetricClient
	stopCh       chan struct{}
}

// MetricClient WebSocket 客户端
type MetricClient struct {
	JobID    uuid.UUID
	Conn     *websocket.Conn
	Send     chan []byte
	Collector *MetricCollector
}

// MetricMessage 指标消息
type MetricMessage struct {
	JobID    uuid.UUID       `json:"job_id"`
	Type     string          `json:"type"` // "metric", "status", "log"
	Payload  json.RawMessage `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
}

// MetricRecord TimescaleDB 指标记录
type MetricRecord struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	JobID       uuid.UUID       `gorm:"type:uuid;index:idx_job_time" json:"job_id"`
	Timestamp   time.Time       `gorm:"index:idx_job_time" json:"timestamp"`
	MetricType  string          `json:"metric_type"` // "loss", "accuracy", "val_loss", "val_accuracy", "custom"
	Epoch       *int            `json:"epoch,omitempty"`
	Step        *int            `json:"step,omitempty"`
	Value       float64         `json:"value"`
	Tags        json.RawMessage `gorm:"type:jsonb" json:"tags,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// TableName 表名
func (MetricRecord) TableName() string {
	return "training_metrics"
}

// NewMetricCollector 创建指标收集器
func NewMetricCollector(db *gorm.DB) *MetricCollector {
	collector := &MetricCollector{
		db: db,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 生产环境应该检查来源
			},
		},
		clients:    make(map[uuid.UUID]map[*MetricClient]bool),
		broadcast:  make(chan *MetricMessage, 256),
		register:   make(chan *MetricClient),
		unregister: make(chan *MetricClient),
		stopCh:     make(chan struct{}),
	}

	// 启动 hub
	go collector.runHub()

	return collector
}

// runHub 运行 WebSocket hub
func (c *MetricCollector) runHub() {
	for {
		select {
		case client := <-c.register:
			c.clientsMu.Lock()
			if c.clients[client.JobID] == nil {
				c.clients[client.JobID] = make(map[*MetricClient]bool)
			}
			c.clients[client.JobID][client] = true
			c.clientsMu.Unlock()
			logger.Info("Metric client registered", zap.String("job_id", client.JobID.String()))

		case client := <-c.unregister:
			c.clientsMu.Lock()
			if clients, ok := c.clients[client.JobID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(c.clients, client.JobID)
					}
				}
			}
			c.clientsMu.Unlock()
			logger.Info("Metric client unregistered", zap.String("job_id", client.JobID.String()))

		case message := <-c.broadcast:
			c.clientsMu.RLock()
			clients := c.clients[message.JobID]
			c.clientsMu.RUnlock()

			for client := range clients {
				select {
				case client.Send <- messageToBytes(message):
				default:
					// 客户端发送缓冲区满，关闭连接
					c.unregister <- client
				}
			}

		case <-c.stopCh:
			return
		}
	}
}

// messageToBytes 消息转字节
func messageToBytes(msg *MetricMessage) []byte {
	data, _ := json.Marshal(msg)
	return data
}

// HandleWebSocket 处理 WebSocket 连接
func (c *MetricCollector) HandleWebSocket(w http.ResponseWriter, r *http.Request, jobID uuid.UUID) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	client := &MetricClient{
		JobID:     jobID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Collector: c,
	}

	c.register <- client

	// 启动读写 goroutine
	go client.writePump()
	go client.readPump()
}

// readPump 读取 WebSocket 消息
func (c *MetricClient) readPump() {
	defer func() {
		c.Collector.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}
		// 客户端消息处理（如果需要）
	}
}

// writePump 写入 WebSocket 消息
func (c *MetricClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加等待的消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastMetric 广播指标
func (c *MetricCollector) BroadcastMetric(jobID uuid.UUID, metricType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	msg := &MetricMessage{
		JobID:     jobID,
		Type:      metricType,
		Payload:   data,
		Timestamp: time.Now(),
	}

	select {
	case c.broadcast <- msg:
	default:
		// 广播缓冲区满，丢弃消息
		logger.Warn("Broadcast channel full, dropping message", 
			zap.String("job_id", jobID.String()))
	}
}

// SaveMetric 保存指标到 TimescaleDB
func (c *MetricCollector) SaveMetric(ctx context.Context, jobID uuid.UUID, metrics *TrainingMetrics) error {
	records := c.metricsToRecords(jobID, metrics)
	
	for _, record := range records {
		if err := c.db.WithContext(ctx).Create(record).Error; err != nil {
			return fmt.Errorf("failed to save metric: %w", err)
		}
	}

	// 广播到 WebSocket 客户端
	c.BroadcastMetric(jobID, "metric", metrics)

	return nil
}

// SaveBatchMetrics 批量保存指标
func (c *MetricCollector) SaveBatchMetrics(ctx context.Context, jobID uuid.UUID, metricsList []*TrainingMetrics) error {
	if len(metricsList) == 0 {
		return nil
	}

	var records []*MetricRecord
	for _, metrics := range metricsList {
		records = append(records, c.metricsToRecords(jobID, metrics)...)
	}

	if err := c.db.WithContext(ctx).CreateInBatches(records, 100).Error; err != nil {
		return fmt.Errorf("failed to batch save metrics: %w", err)
	}

	// 广播最新指标
	c.BroadcastMetric(jobID, "metric", metricsList[len(metricsList)-1])

	return nil
}

// metricsToRecords 转换指标到记录
func (c *MetricCollector) metricsToRecords(jobID uuid.UUID, metrics *TrainingMetrics) []*MetricRecord {
	var records []*MetricRecord

	if metrics.Loss != nil {
		records = append(records, &MetricRecord{
			JobID:      jobID,
			Timestamp:  metrics.Timestamp,
			MetricType: "loss",
			Epoch:      intPtr(metrics.Epoch),
			Step:       intPtr(metrics.Step),
			Value:      *metrics.Loss,
		})
	}

	if metrics.Accuracy != nil {
		records = append(records, &MetricRecord{
			JobID:      jobID,
			Timestamp:  metrics.Timestamp,
			MetricType: "accuracy",
			Epoch:      intPtr(metrics.Epoch),
			Step:       intPtr(metrics.Step),
			Value:      *metrics.Accuracy,
		})
	}

	if metrics.ValLoss != nil {
		records = append(records, &MetricRecord{
			JobID:      jobID,
			Timestamp:  metrics.Timestamp,
			MetricType: "val_loss",
			Epoch:      intPtr(metrics.Epoch),
			Step:       intPtr(metrics.Step),
			Value:      *metrics.ValLoss,
		})
	}

	if metrics.ValAccuracy != nil {
		records = append(records, &MetricRecord{
			JobID:      jobID,
			Timestamp:  metrics.Timestamp,
			MetricType: "val_accuracy",
			Epoch:      intPtr(metrics.Epoch),
			Step:       intPtr(metrics.Step),
			Value:      *metrics.ValAccuracy,
		})
	}

	if metrics.LearningRate != nil {
		records = append(records, &MetricRecord{
			JobID:      jobID,
			Timestamp:  metrics.Timestamp,
			MetricType: "learning_rate",
			Epoch:      intPtr(metrics.Epoch),
			Step:       intPtr(metrics.Step),
			Value:      *metrics.LearningRate,
		})
	}

	// 自定义指标
	for name, value := range metrics.Custom {
		tags, _ := json.Marshal(map[string]string{"name": name})
		records = append(records, &MetricRecord{
			JobID:      jobID,
			Timestamp:  metrics.Timestamp,
			MetricType: "custom",
			Epoch:      intPtr(metrics.Epoch),
			Step:       intPtr(metrics.Step),
			Value:      value,
			Tags:       tags,
		})
	}

	return records
}

// intPtr int 转指针
func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// GetMetrics 获取任务指标
func (c *MetricCollector) GetMetrics(ctx context.Context, jobID uuid.UUID, metricType string, start, end time.Time, limit int) ([]*MetricRecord, error) {
	var records []*MetricRecord
	
	query := c.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Where("timestamp BETWEEN ? AND ?", start, end).
		Order("timestamp DESC")

	if metricType != "" {
		query = query.Where("metric_type = ?", metricType)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

// GetLatestMetrics 获取最新指标
func (c *MetricCollector) GetLatestMetrics(ctx context.Context, jobID uuid.UUID, metricTypes []string) (map[string]*MetricRecord, error) {
	result := make(map[string]*MetricRecord)

	for _, metricType := range metricTypes {
		var record MetricRecord
		err := c.db.WithContext(ctx).
			Where("job_id = ? AND metric_type = ?", jobID, metricType).
			Order("timestamp DESC").
			First(&record).Error
		
		if err == nil {
			result[metricType] = &record
		}
	}

	return result, nil
}

// GetMetricSummary 获取指标摘要
func (c *MetricCollector) GetMetricSummary(ctx context.Context, jobID uuid.UUID) (*MetricSummary, error) {
	summary := &MetricSummary{JobID: jobID}

	// 获取各项指标的最新值和统计
	metricTypes := []string{"loss", "accuracy", "val_loss", "val_accuracy"}
	
	for _, metricType := range metricTypes {
		var stats struct {
			Latest float64
			Min    float64
			Max    float64
			Avg    float64
			Count  int64
		}

		err := c.db.WithContext(ctx).Raw(`
			SELECT 
				value as latest,
				MIN(value) as min,
				MAX(value) as max,
				AVG(value) as avg,
				COUNT(*) as count
			FROM training_metrics
			WHERE job_id = ? AND metric_type = ?
			GROUP BY metric_type, value
			ORDER BY timestamp DESC
			LIMIT 1
		`, jobID, metricType).Scan(&stats).Error

		if err == nil {
			summary.Stats[metricType] = map[string]interface{}{
				"latest": stats.Latest,
				"min":    stats.Min,
				"max":    stats.Max,
				"avg":    stats.Avg,
				"count":  stats.Count,
			}
		}
	}

	return summary, nil
}

// MetricSummary 指标摘要
type MetricSummary struct {
	JobID uuid.UUID                `json:"job_id"`
	Stats map[string]interface{}   `json:"stats"`
}

// AutoMigrate 自动迁移数据库表
func (c *MetricCollector) AutoMigrate() error {
	return c.db.AutoMigrate(&MetricRecord{})
}

// Stop 停止收集器
func (c *MetricCollector) Stop() {
	close(c.stopCh)
	
	// 关闭所有客户端连接
	c.clientsMu.Lock()
	for _, clients := range c.clients {
		for client := range clients {
			close(client.Send)
			client.Conn.Close()
		}
	}
	c.clients = make(map[uuid.UUID]map[*MetricClient]bool)
	c.clientsMu.Unlock()
}

// Ensure Hypertable 确保 TimescaleDB hypertable
func (c *MetricCollector) EnsureHypertable() error {
	// 创建 hypertable（如果 TimescaleDB 已安装）
	err := c.db.Exec(`
		SELECT create_hypertable('training_metrics', 'timestamp', 
			if_not_exists => TRUE,
			migrate_data => TRUE
		)
	`).Error

	if err != nil {
		// TimescaleDB 可能未安装，忽略错误
		logger.Warn("Failed to create hypertable, TimescaleDB may not be installed", zap.Error(err))
	}

	return nil
}
