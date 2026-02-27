// Package minio provides a client wrapper for MinIO object storage operations
package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config MinIO 配置
type Config struct {
	Endpoint        string        // MinIO 服务端点
	AccessKeyID     string        // 访问密钥
	SecretAccessKey string        // 密钥
	UseSSL          bool          // 是否使用 SSL
	Region          string        // 区域
	PresignedExpiry time.Duration // 预签名 URL 过期时间
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		Region:          "us-east-1",
		PresignedExpiry: 15 * time.Minute,
	}
}

// Client MinIO 客户端包装
type Client struct {
	client *minio.Client
	config *Config
}

// New 创建 MinIO 客户端
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &Client{
		client: minioClient,
		config: cfg,
	}, nil
}

// GetClient 获取原始 MinIO 客户端
func (c *Client) GetClient() *minio.Client {
	return c.client
}

// BucketExists 检查 bucket 是否存在
func (c *Client) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	return exists, nil
}

// MakeBucket 创建 bucket
func (c *Client) MakeBucket(ctx context.Context, bucketName string) error {
	exists, err := c.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	err = c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
		Region: c.config.Region,
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
	}
	return nil
}

// RemoveBucket 删除 bucket
func (c *Client) RemoveBucket(ctx context.Context, bucketName string) error {
	err := c.client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to remove bucket %s: %w", bucketName, err)
	}
	return nil
}

// UploadOptions 上传选项
type UploadOptions struct {
	ContentType string
	Metadata    map[string]string
	PartSize    uint64 // 分片大小，默认为 16MB
}

// DefaultUploadOptions 返回默认上传选项
func DefaultUploadOptions() *UploadOptions {
	return &UploadOptions{
		ContentType: "application/octet-stream",
		Metadata:    make(map[string]string),
		PartSize:    16 * 1024 * 1024, // 16MB
	}
}

// UploadResult 上传结果
type UploadResult struct {
	Size        int64
	ETag        string
	VersionID   string
	Location    string
	ContentType string
}

// Upload 上传文件
func (c *Client) Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, opts *UploadOptions) (*UploadResult, error) {
	if opts == nil {
		opts = DefaultUploadOptions()
	}

	// 确保 bucket 存在
	if err := c.MakeBucket(ctx, bucketName); err != nil {
		return nil, err
	}

	// 上传选项
	putOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
		PartSize:     opts.PartSize,
	}

	info, err := c.client.PutObject(ctx, bucketName, objectName, reader, size, putOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object %s: %w", objectName, err)
	}

	return &UploadResult{
		Size:        info.Size,
		ETag:        info.ETag,
		VersionID:   info.VersionID,
		Location:    path.Join(bucketName, objectName),
		ContentType: opts.ContentType,
	}, nil
}

// UploadMultipart 分片上传（用于大文件）
func (c *Client) UploadMultipart(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, opts *UploadOptions) (*UploadResult, error) {
	if opts == nil {
		opts = DefaultUploadOptions()
	}

	// 确保 bucket 存在
	if err := c.MakeBucket(ctx, bucketName); err != nil {
		return nil, err
	}

	// 对于大文件使用分片上传
	putOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
		PartSize:     opts.PartSize,
		NumThreads:   4, // 并发上传线程数
	}

	info, err := c.client.PutObject(ctx, bucketName, objectName, reader, size, putOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object %s: %w", objectName, err)
	}

	return &UploadResult{
		Size:        info.Size,
		ETag:        info.ETag,
		VersionID:   info.VersionID,
		Location:    path.Join(bucketName, objectName),
		ContentType: opts.ContentType,
	}, nil
}

// Download 下载文件
func (c *Client) Download(ctx context.Context, bucketName, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	object, err := c.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object %s: %w", objectName, err)
	}

	stat, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, nil, fmt.Errorf("failed to stat object %s: %w", objectName, err)
	}

	return object, &stat, nil
}

// PresignedGetURL 生成预签名下载 URL
func (c *Client) PresignedGetURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (*url.URL, error) {
	if expiry <= 0 {
		expiry = c.config.PresignedExpiry
	}

	presignedURL, err := c.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL, nil
}

// PresignedPutURL 生成预签名上传 URL
func (c *Client) PresignedPutURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (*url.URL, error) {
	if expiry <= 0 {
		expiry = c.config.PresignedExpiry
	}

	presignedURL, err := c.client.PresignedPutObject(ctx, bucketName, objectName, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL, nil
}

// StatObject 获取对象信息
func (c *Client) StatObject(ctx context.Context, bucketName, objectName string) (*minio.ObjectInfo, error) {
	stat, err := c.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object %s: %w", objectName, err)
	}
	return &stat, nil
}

// RemoveObject 删除对象
func (c *Client) RemoveObject(ctx context.Context, bucketName, objectName string) error {
	err := c.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object %s: %w", objectName, err)
	}
	return nil
}

// ListObjects 列出对象
func (c *Client) ListObjects(ctx context.Context, bucketName, prefix string, recursive bool) <-chan minio.ObjectInfo {
	return c.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	})
}

// CopyObject 复制对象
func (c *Client) CopyObject(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string) (*minio.UploadInfo, error) {
	srcOpts := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcObject,
	}
	dstOpts := minio.CopyDestOptions{
		Bucket: dstBucket,
		Object: dstObject,
	}

	info, err := c.client.CopyObject(ctx, dstOpts, srcOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	return &info, nil
}

// GetProjectBucketName 获取项目 bucket 名称
func GetProjectBucketName(projectID string) string {
	return fmt.Sprintf("project-%s", projectID)
}

// GetDatasetObjectName 获取数据集对象名称
func GetDatasetObjectName(projectID, datasetID, filename string) string {
	return fmt.Sprintf("datasets/%s/%s", datasetID, filename)
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("minio health check failed: %w", err)
	}
	return nil
}
