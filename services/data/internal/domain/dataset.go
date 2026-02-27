package domain

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DatasetStatus 数据集状态
type DatasetStatus string

const (
	// DatasetStatusPending 待处理
	DatasetStatusPending DatasetStatus = "pending"
	// DatasetStatusUploading 上传中
	DatasetStatusUploading DatasetStatus = "uploading"
	// DatasetStatusProcessing 处理中
	DatasetStatusProcessing DatasetStatus = "processing"
	// DatasetStatusReady 可用
	DatasetStatusReady DatasetStatus = "ready"
	// DatasetStatusFailed 失败
	DatasetStatusFailed DatasetStatus = "failed"
	// DatasetStatusDeleted 已删除
	DatasetStatusDeleted DatasetStatus = "deleted"
)

// Valid 验证状态是否有效
func (s DatasetStatus) Valid() bool {
	switch s {
	case DatasetStatusPending, DatasetStatusUploading, DatasetStatusProcessing,
		DatasetStatusReady, DatasetStatusFailed, DatasetStatusDeleted:
		return true
	}
	return false
}

// DatasetFormat 数据集格式
type DatasetFormat string

const (
	// DatasetFormatCSV CSV 格式
	DatasetFormatCSV DatasetFormat = "csv"
	// DatasetFormatJSON JSON 格式
	DatasetFormatJSON DatasetFormat = "json"
	// DatasetFormatParquet Parquet 格式
	DatasetFormatParquet DatasetFormat = "parquet"
	// DatasetFormatTXT 文本格式
	DatasetFormatTXT DatasetFormat = "txt"
	// DatasetFormatZIP ZIP 压缩格式
	DatasetFormatZIP DatasetFormat = "zip"
	// DatasetFormatTAR TAR 压缩格式
	DatasetFormatTAR DatasetFormat = "tar"
	// DatasetFormatGZ GZIP 压缩格式
	DatasetFormatGZ DatasetFormat = "gz"
	// DatasetFormatUnknown 未知格式
	DatasetFormatUnknown DatasetFormat = "unknown"
)

// DetectFormat 从文件名检测格式
func DetectFormat(filename string) DatasetFormat {
	ext := strings.ToLower(filepath.Ext(filename))
	// 移除点号
	ext = strings.TrimPrefix(ext, ".")

	switch ext {
	case "csv":
		return DatasetFormatCSV
	case "json":
		return DatasetFormatJSON
	case "parquet":
		return DatasetFormatParquet
	case "txt":
		return DatasetFormatTXT
	case "zip":
		return DatasetFormatZIP
	case "tar":
		return DatasetFormatTAR
	case "gz", "gzip":
		return DatasetFormatGZ
	default:
		return DatasetFormatUnknown
	}
}

// Dataset 数据集领域模型
type Dataset struct {
	ID           uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID    uuid.UUID     `json:"project_id" gorm:"type:uuid;not null;index"`
	Name         string        `json:"name" gorm:"not null"`
	Description  string        `json:"description"`
	StoragePath  string        `json:"storage_path" gorm:"not null"`
	SizeBytes    int64         `json:"size_bytes"`
	Format       DatasetFormat `json:"format"`
	Status       DatasetStatus `json:"status" gorm:"default:'pending'"`
	OriginalName string        `json:"original_name"` // 原始文件名
	Checksum     string        `json:"checksum"`      // 文件校验和
	Metadata     *DatasetMetadata `json:"metadata,omitempty" gorm:"foreignKey:DatasetID"`
	CreatedAt    time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Dataset) TableName() string {
	return "datasets"
}

// DatasetMetadata 数据集元数据
type DatasetMetadata struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	DatasetID    uuid.UUID `json:"dataset_id" gorm:"type:uuid;not null;uniqueIndex"`
	RowCount     int64     `json:"row_count"`
	ColumnCount  int       `json:"column_count"`
	Columns      string    `json:"columns"` // JSON 格式存储列信息
	SampleData   string    `json:"sample_data"` // JSON 格式存储样本数据
	CustomAttrs  string    `json:"custom_attrs"` // JSON 格式存储自定义属性
}

// TableName 指定表名
func (DatasetMetadata) TableName() string {
	return "dataset_metadata"
}

// UploadProgress 上传进度
type UploadProgress struct {
	UploadID      string    `json:"upload_id"`
	DatasetID     uuid.UUID `json:"dataset_id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Filename      string    `json:"filename"`
	TotalSize     int64     `json:"total_size"`
	UploadedSize  int64     `json:"uploaded_size"`
	ChunkSize     int64     `json:"chunk_size"`
	TotalChunks   int       `json:"total_chunks"`
	UploadedChunks []int    `json:"uploaded_chunks"`
	Status        string    `json:"status"` // pending, uploading, completed, failed
	Error         string    `json:"error,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProgressPercentage 计算进度百分比
func (p *UploadProgress) ProgressPercentage() float64 {
	if p.TotalSize == 0 {
		return 0
	}
	return float64(p.UploadedSize) / float64(p.TotalSize) * 100
}

// IsComplete 检查是否上传完成
func (p *UploadProgress) IsComplete() bool {
	return p.UploadedSize >= p.TotalSize
}

// DatasetCreateRequest 创建数据集请求
type DatasetCreateRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description" binding:"max=1000"`
	Format      string `json:"format"`
}

// Validate 验证请求
func (r *DatasetCreateRequest) Validate() error {
	if r.ProjectID == "" {
		return errors.New("project_id is required")
	}
	if _, err := uuid.Parse(r.ProjectID); err != nil {
		return errors.New("invalid project_id format")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// DatasetUpdateRequest 更新数据集请求
type DatasetUpdateRequest struct {
	Name        string `json:"name" binding:"max=255"`
	Description string `json:"description" binding:"max=1000"`
}

// DatasetResponse 数据集响应
type DatasetResponse struct {
	ID           uuid.UUID     `json:"id"`
	ProjectID    uuid.UUID     `json:"project_id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	StoragePath  string        `json:"storage_path"`
	SizeBytes    int64         `json:"size_bytes"`
	SizeReadable string        `json:"size_readable"`
	Format       DatasetFormat `json:"format"`
	Status       DatasetStatus `json:"status"`
	OriginalName string        `json:"original_name"`
	Checksum     string        `json:"checksum"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// DatasetListResponse 数据集列表响应
type DatasetListResponse struct {
	Datasets []DatasetResponse `json:"datasets"`
	Total    int64             `json:"total"`
}

// UploadInitRequest 初始化上传请求
type UploadInitRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Filename    string `json:"filename" binding:"required"`
	Size        int64  `json:"size" binding:"required,min=1"`
	ChunkSize   int64  `json:"chunk_size"` // 可选，默认使用配置
}

// Validate 验证请求
func (r *UploadInitRequest) Validate() error {
	if r.ProjectID == "" {
		return errors.New("project_id is required")
	}
	if _, err := uuid.Parse(r.ProjectID); err != nil {
		return errors.New("invalid project_id format")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Filename == "" {
		return errors.New("filename is required")
	}
	if r.Size <= 0 {
		return errors.New("size must be greater than 0")
	}
	return nil
}

// UploadInitResponse 初始化上传响应
type UploadInitResponse struct {
	UploadID    string `json:"upload_id"`
	DatasetID   string `json:"dataset_id"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
}

// UploadChunkRequest 上传分片请求
type UploadChunkRequest struct {
	UploadID   string `json:"upload_id" binding:"required"`
	ChunkIndex int    `json:"chunk_index" binding:"required,min=0"`
	ChunkData  []byte `json:"chunk_data"`
}

// UploadCompleteRequest 完成上传请求
type UploadCompleteRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
	Checksum string `json:"checksum"` // 可选的文件校验和
}

// DownloadResponse 下载响应
type DownloadResponse struct {
	URL         string    `json:"url"`          // 预签名 URL
	ExpiresAt   time.Time `json:"expires_at"`   // URL 过期时间
	Filename    string    `json:"filename"`     // 下载文件名
	ContentType string    `json:"content_type"` // Content-Type
}

// ListQuery 列表查询参数
type ListQuery struct {
	ProjectID  string
	Status     string
	Format     string
	Keyword    string
	Page       int
	PageSize   int
	SortBy     string
	SortOrder  string
}

// DefaultListQuery 返回默认列表查询
func DefaultListQuery() *ListQuery {
	return &ListQuery{
		Page:      1,
		PageSize:  20,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}

// Validate 验证查询参数
func (q *ListQuery) Validate() error {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	return nil
}

// GetOffset 获取偏移量
func (q *ListQuery) GetOffset() int {
	return (q.Page - 1) * q.PageSize
}

// GetLimit 获取限制数量
func (q *ListQuery) GetLimit() int {
	return q.PageSize
}
