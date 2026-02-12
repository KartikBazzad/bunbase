package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps MinIO and provides project-scoped object storage (one bucket per project).
type Client struct {
	mc      *minio.Client
	enabled bool
}

// Config holds MinIO connection settings.
type Config struct {
	Endpoint        string // e.g. "minio:9000" or "localhost:9000"
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

// NewClient creates a storage client. If config has empty Endpoint, the client is disabled (all ops return ErrDisabled).
func NewClient(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return &Client{enabled: false}, nil
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &Client{mc: mc, enabled: true}, nil
}

// ErrDisabled is returned when storage is not configured.
var ErrDisabled = fmt.Errorf("storage service not configured")

// BucketForProject returns the bucket name for a project (one bucket per project).
// MinIO/S3: lowercase, digits, hyphens; 3-63 chars.
func BucketForProject(projectID string) string {
	return "project-" + strings.ToLower(projectID)
}

// EnsureBucket creates the project bucket if it does not exist (idempotent).
func (c *Client) EnsureBucket(ctx context.Context, projectID string) error {
	if !c.enabled {
		return ErrDisabled
	}
	bucket := BucketForProject(projectID)
	exists, err := c.mc.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

// PutObject uploads an object to the project bucket.
func (c *Client) PutObject(ctx context.Context, projectID, key string, reader io.Reader, size int64, contentType string) error {
	if !c.enabled {
		return ErrDisabled
	}
	if err := c.EnsureBucket(ctx, projectID); err != nil {
		return err
	}
	bucket := BucketForProject(projectID)
	_, err := c.mc.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

// GetObjectResult holds the reader and metadata for a downloaded object.
type GetObjectResult struct {
	Reader       io.ReadCloser
	ContentType  string
	Size         int64
	LastModified time.Time
}

// GetObject downloads an object from the project bucket.
func (c *Client) GetObject(ctx context.Context, projectID, key string) (*GetObjectResult, error) {
	if !c.enabled {
		return nil, ErrDisabled
	}
	bucket := BucketForProject(projectID)
	obj, err := c.mc.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, err
	}
	return &GetObjectResult{
		Reader:       obj,
		ContentType:  info.ContentType,
		Size:         info.Size,
		LastModified: info.LastModified,
	}, nil
}

// DeleteObject removes an object from the project bucket.
func (c *Client) DeleteObject(ctx context.Context, projectID, key string) error {
	if !c.enabled {
		return ErrDisabled
	}
	bucket := BucketForProject(projectID)
	return c.mc.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// ObjectInfo is a minimal object listing entry.
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

// ListObjects lists objects in the project bucket with optional prefix.
func (c *Client) ListObjects(ctx context.Context, projectID, prefix string) ([]ObjectInfo, error) {
	if !c.enabled {
		return nil, ErrDisabled
	}
	if err := c.EnsureBucket(ctx, projectID); err != nil {
		return nil, err
	}
	bucket := BucketForProject(projectID)
	ch := c.mc.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	var out []ObjectInfo
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, ObjectInfo{Key: obj.Key, Size: obj.Size, LastModified: obj.LastModified})
	}
	return out, nil
}

// Enabled reports whether the storage client is configured.
func (c *Client) Enabled() bool {
	return c.enabled
}
