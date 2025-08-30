package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
)


type S3Config struct {
	Enabled       bool
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PathStyle     bool
	PublicURL     string
	MediaDelivery string
	RetentionDays int
}


type S3Manager struct {
	mu      sync.RWMutex
	clients map[string]*s3.Client
	configs map[string]*S3Config
}


var s3Manager = &S3Manager{
	clients: make(map[string]*s3.Client),
	configs: make(map[string]*S3Config),
}


func GetS3Manager() *S3Manager {
	return s3Manager
}


func (m *S3Manager) InitializeS3Client(userID string, config *S3Config) error {
	if !config.Enabled {
		m.RemoveClient(userID)
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()


	credProvider := credentials.NewStaticCredentialsProvider(
		config.AccessKey,
		config.SecretKey,
		"",
	)


	cfg := aws.Config{
		Region:      config.Region,
		Credentials: credProvider,
	}

	if config.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					URL:               config.Endpoint,
					HostnameImmutable: config.PathStyle,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		cfg.EndpointResolverWithOptions = customResolver
	}


	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = config.PathStyle
	})

	m.clients[userID] = client
	m.configs[userID] = config

	log.Info().Str("userID", userID).Str("bucket", config.Bucket).Msg("S3 client initialized")
	return nil
}


func (m *S3Manager) RemoveClient(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clients, userID)
	delete(m.configs, userID)
}


func (m *S3Manager) GetClient(userID string) (*s3.Client, *S3Config, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, clientOk := m.clients[userID]
	config, configOk := m.configs[userID]

	return client, config, clientOk && configOk
}


func (m *S3Manager) GenerateS3Key(userID, contactJID, messageID string, mimeType string, isIncoming bool) string {

	direction := "outbox"
	if isIncoming {
		direction = "inbox"
	}


	contactJID = strings.ReplaceAll(contactJID, "@", "_")
	contactJID = strings.ReplaceAll(contactJID, ":", "_")


	now := time.Now()
	year := now.Format("2025")
	month := now.Format("05")
	day := now.Format("25")


	mediaType := "documents"
	if strings.HasPrefix(mimeType, "image/") {
		mediaType = "images"
	} else if strings.HasPrefix(mimeType, "video/") {
		mediaType = "videos"
	} else if strings.HasPrefix(mimeType, "audio/") {
		mediaType = "audio"
	}


	ext := ".bin"
	switch {
	case strings.Contains(mimeType, "jpeg"), strings.Contains(mimeType, "jpg"):
		ext = ".jpg"
	case strings.Contains(mimeType, "png"):
		ext = ".png"
	case strings.Contains(mimeType, "gif"):
		ext = ".gif"
	case strings.Contains(mimeType, "webp"):
		ext = ".webp"
	case strings.Contains(mimeType, "mp4"):
		ext = ".mp4"
	case strings.Contains(mimeType, "webm"):
		ext = ".webm"
	case strings.Contains(mimeType, "ogg"):
		ext = ".ogg"
	case strings.Contains(mimeType, "opus"):
		ext = ".opus"
	case strings.Contains(mimeType, "pdf"):
		ext = ".pdf"
	case strings.Contains(mimeType, "doc"):
		if strings.Contains(mimeType, "docx") {
			ext = ".docx"
		} else {
			ext = ".doc"
		}
	}


	key := fmt.Sprintf("users/%s/%s/%s/%s/%s/%s/%s/%s%s",
		userID,
		direction,
		contactJID,
		year,
		month,
		day,
		mediaType,
		messageID,
		ext,
	)

	return key
}


func (m *S3Manager) UploadToS3(ctx context.Context, userID string, key string, data []byte, mimeType string) error {
	client, config, ok := m.GetClient(userID)
	if !ok {
		return fmt.Errorf("S3 client not initialized for user %s", userID)
	}


	contentType := mimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}


	var expires *time.Time
	if config.RetentionDays > 0 {
		expirationTime := time.Now().Add(time.Duration(config.RetentionDays) * 24 * time.Hour)
		expires = &expirationTime
	}

	input := &s3.PutObjectInput{
		Bucket:       aws.String(config.Bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=3600"),
		ACL:          types.ObjectCannedACLPublicRead,
	}

	if expires != nil {
		input.Expires = expires
	}


	if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") || mimeType == "application/pdf" {
		input.ContentDisposition = aws.String("inline")
	}

	_, err := client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}


func (m *S3Manager) GetPublicURL(userID, key string) string {
	_, config, ok := m.GetClient(userID)
	if !ok {
		return ""
	}


	if config.PublicURL != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(config.PublicURL, "/"), config.Bucket, key)
	}


	if config.PathStyle {
		return fmt.Sprintf("%s/%s/%s",
			strings.TrimRight(config.Endpoint, "/"),
			config.Bucket,
			key)
	}


	if strings.Contains(config.Endpoint, "amazonaws.com") {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
			config.Bucket,
			config.Region,
			key)
	}


	endpoint := strings.TrimPrefix(config.Endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return fmt.Sprintf("https://%s.%s/%s", config.Bucket, endpoint, key)
}


func (m *S3Manager) TestConnection(ctx context.Context, userID string) error {
	client, config, ok := m.GetClient(userID)
	if !ok {
		return fmt.Errorf("S3 client not initialized for user %s", userID)
	}


	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(config.Bucket),
		MaxKeys: aws.Int32(1),
	}

	_, err := client.ListObjectsV2(ctx, input)
	return err
}


func (m *S3Manager) ProcessMediaForS3(ctx context.Context, userID, contactJID, messageID string,
	data []byte, mimeType string, fileName string, isIncoming bool) (map[string]interface{}, error) {


	key := m.GenerateS3Key(userID, contactJID, messageID, mimeType, isIncoming)


	err := m.UploadToS3(ctx, userID, key, data, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}


	publicURL := m.GetPublicURL(userID, key)


	s3Data := map[string]interface{}{
		"url":      publicURL,
		"key":      key,
		"bucket":   m.configs[userID].Bucket,
		"size":     len(data),
		"mimeType": mimeType,
		"fileName": fileName,
	}

	return s3Data, nil
}


func (m *S3Manager) DeleteAllUserObjects(ctx context.Context, userID string) error {
	client, config, ok := m.GetClient(userID)
	if !ok {
		return fmt.Errorf("S3 client not initialized for user %s", userID)
	}

	prefix := fmt.Sprintf("users/%s/", userID)
	var toDelete []types.ObjectIdentifier
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            aws.String(config.Bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		}
		output, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to list objects for user %s: %w", userID, err)
		}

		for _, obj := range output.Contents {
			toDelete = append(toDelete, types.ObjectIdentifier{Key: obj.Key})

			if len(toDelete) == 1000 {
				_, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
					Bucket: aws.String(config.Bucket),
					Delete: &types.Delete{Objects: toDelete},
				})
				if err != nil {
					return fmt.Errorf("failed to delete objects for user %s: %w", userID, err)
				}
				toDelete = nil
			}
		}

		if output.IsTruncated != nil && *output.IsTruncated && output.NextContinuationToken != nil {
			continuationToken = output.NextContinuationToken
		} else {
			break
		}
	}


	if len(toDelete) > 0 {
		_, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(config.Bucket),
			Delete: &types.Delete{Objects: toDelete},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects for user %s: %w", userID, err)
		}
	}

	log.Info().Str("userID", userID).Msg("all user files removed from S3")
	return nil
}
