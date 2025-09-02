package media

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/logger"
)

type MediaService struct {
	client    *minio.Client
	bucket    string
	endpoint  string
	useSSL    bool
	publicURL string
	logger    logger.Logger
}

type MediaInfo struct {
	Path         string    `json:"path"`
	URL          string    `json:"url"`
	FileName     string    `json:"file_name"`
	ContentType  string    `json:"content_type"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ETag         string    `json:"etag"`
}

func NewMediaServiceFromConfig(cfg *config.MinIOConfig) (*MediaService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("minio config is nil")
	}

	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	svc := &MediaService{
		client:    cli,
		bucket:    cfg.BucketName,
		endpoint:  cfg.Endpoint,
		useSSL:    cfg.UseSSL,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
		logger:    logger.GetWithSession("media_service"),
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := cli.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		svc.logger.Warn().Err(err).Str("bucket", cfg.BucketName).Msg("Failed to check if bucket exists")
	} else if !exists {
		if err := cli.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", cfg.BucketName, err)
		}
		svc.logger.Info().Str("bucket", cfg.BucketName).Msg("Created MinIO bucket")
	}

	return svc, nil
}

func (s *MediaService) UploadMedia(ctx context.Context, objectPath string, reader io.Reader, size int64, contentType string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("media service not initialized")
	}
	if objectPath == "" {
		return fmt.Errorf("object path is required")
	}
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	_, err := s.client.PutObject(ctx, s.bucket, objectPath, reader, size, opts)
	if err != nil {
		s.logger.Error().Err(err).Str("bucket", s.bucket).Str("path", objectPath).Msg("Failed to upload media to MinIO")
		return err
	}
	return nil
}

func (s *MediaService) GetMediaURL(ctx context.Context, objectPath string) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("media service not initialized")
	}
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, objectPath), nil
	}
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, s.bucket, objectPath), nil
}

func (s *MediaService) GetMedia(ctx context.Context, objectPath string) (*MediaInfo, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("media service not initialized")
	}
	info, err := s.client.StatObject(ctx, s.bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	url, _ := s.GetMediaURL(ctx, objectPath)
	return &MediaInfo{
		Path:         objectPath,
		URL:          url,
		FileName:     path.Base(objectPath),
		ContentType:  info.ContentType,
		Size:         info.Size,
		LastModified: info.LastModified,
		ETag:         info.ETag,
	}, nil
}

func (s *MediaService) DownloadMedia(ctx context.Context, objectPath string) ([]byte, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("media service not initialized")
	}
	obj, err := s.client.GetObject(ctx, s.bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *MediaService) DeleteMedia(ctx context.Context, objectPath string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("media service not initialized")
	}
	return s.client.RemoveObject(ctx, s.bucket, objectPath, minio.RemoveObjectOptions{})
}