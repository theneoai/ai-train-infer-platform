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
	db         *gorm.DB
	upgrader   websocket.Upgrader
	clients    map[uuid.UUID]map[*MetricClient]bool
	clientsMu  sync.RWMutex
	broadcast  chan *MetricMessage
	register   chan *MetricClient
	unregister chan *MetricClient
	stopCh     chan struct{}
}

// MetricClient WebSocket 客户端
type MetricClient struct {
	JobID     uuid.UUID
	Conn      *websocket.Conn
	Send      chan []byte
	Collector *MetricCollector
}

// MetricMessage 指标消息
type MetricMessage struct {
	JobID     uuid.UUID       `json:"job_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// MetricRecord TimescaleDB 指标记录
type MetricRecord struct {
	ID         uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	JobID      uuid.UUID       `gorm:"type:uuid;index:idx_job_time" json:"job_id"`
	Timestamp  time.Time       `gorm:"index:idx_job_time" json:"timestamp"`
	MetricType string          `json:"metric_type"`
	Epoch      *int            `json:"epoch,omitempty"`
	Step       *int            `json:"step,omitempty"`
	Value      float64         `json:"value"`
	Tags       json.RawMessage `gorm:"type:jsonb" json:"tags,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
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
				return true
			},
		},
		clients:    make(map[uuid.UUID]map[*MetricClient]bool),
		broadcast:  make(chan *MetricMessage, 256),
		register:   make(chan *MetricClient),
		unregister: make(chan *MetricClient),
		stopCh:     make(chan struct{}),
	}

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

		case message := <-c.broadcast:
			c.clientsMu.RLock()
			clients := c.clients[message.JobID]
			c.clientsMu.RUnlock()

			for client := range clients {
				select {
				case client.Send <- messageToBytes(message):
				default:
					c.unregister <- client
				}
			}

		case <-c.stopCh:
			return
		}
	}
}

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

	go client.writePump()
	go client.readPump()
}

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
	}
}

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
		logger.Warn("Broadcast channel full, dropping message", zap.String("job_id", jobID.String()))
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

	c.BroadcastMetric(jobID, "metric", metrics)
	return nil
}

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

	return records
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// AutoMigrate 自动迁移数据库表
func (c *MetricCollector) AutoMigrate() error {
	return c.db.AutoMigrate(&MetricRecord{})
}

// Stop 停止收集器
func (c *MetricCollector) Stop() {
	close(c.stopCh)

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
