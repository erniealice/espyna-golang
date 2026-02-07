//go:build google && gcp_storage

package gcs

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/protobuf/types/known/timestamppb"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/common/google"
	storagecommon "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/common"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	pb "leapfor.xyz/esqyma/golang/v1/infrastructure/storage"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterStorageProvider(
		"gcs",
		func() ports.StorageProvider {
			return NewGCSStorageProvider()
		},
		transformConfig,
	)
	registry.RegisterStorageBuildFromEnv("gcs", buildFromEnv)
}

// buildFromEnv creates and initializes a GCS storage provider from environment variables.
func buildFromEnv() (ports.StorageProvider, error) {
	bucketName := os.Getenv("STORAGE_BUCKET_NAME")
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")

	protoConfig := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_GCP,
		Config: &pb.StorageProviderConfig_GcsConfig{
			GcsConfig: &pb.GcsStorageConfig{
				DefaultBucket: bucketName,
				ProjectId:     projectId,
			},
		},
	}
	p := NewGCSStorageProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("gcs: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to GCS storage proto config.
func transformConfig(rawConfig map[string]any) (*pb.StorageProviderConfig, error) {
	return nil, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// GCSStorageProvider implements Google Cloud Storage provider
// This adapter translates proto contracts to GCS SDK operations
type GCSStorageProvider struct {
	config        *pb.StorageProviderConfig
	bucketName    string
	projectID     string
	enabled       bool
	timeout       time.Duration
	clientManager *google.GoogleClientManager
}

// NewGCSStorageProvider creates a new Google Cloud Storage provider
func NewGCSStorageProvider() ports.StorageProvider {
	return &GCSStorageProvider{
		enabled: false,
		timeout: 30 * time.Second,
	}
}

// Name returns the name of this storage provider
func (p *GCSStorageProvider) Name() string {
	return "gcs"
}

// Initialize sets up the GCS storage provider with proto configuration
func (p *GCSStorageProvider) Initialize(config *pb.StorageProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Verify provider type
	if config.Provider != pb.StorageProvider_STORAGE_PROVIDER_GCP {
		return fmt.Errorf("invalid provider type: expected GCP, got %v", config.Provider)
	}

	// Extract GCS-specific configuration
	gcsConfig := config.GetGcsConfig()
	if gcsConfig == nil {
		return fmt.Errorf("GCS storage configuration is missing")
	}

	p.projectID = gcsConfig.ProjectId
	p.bucketName = gcsConfig.DefaultBucket

	if p.bucketName == "" {
		return fmt.Errorf("default_bucket cannot be empty")
	}

	// Initialize Google client manager
	googleConfig := &google.GoogleConfig{
		StorageTimeout: p.timeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	// Create new client manager (replaces singleton pattern)
	manager, err := google.NewGoogleClientManager(ctx, googleConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Google storage client: %w", err)
	}

	p.clientManager = manager
	p.config = config
	p.enabled = true

	// Test bucket accessibility
	client := p.clientManager.GetStorageClient()
	bucket := client.Bucket(p.bucketName)
	_, err = bucket.Attrs(ctx)
	if err != nil {
		p.enabled = false
		return fmt.Errorf("GCS bucket access test failed: %w", err)
	}

	return nil
}

// UploadObject stores an object in GCS
func (p *GCSStorageProvider) UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "GCS storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	// Validate request
	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	// Use container name as bucket (or default bucket)
	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}

	// Sanitize object key
	objectKey := strings.Trim(req.ObjectKey, "/")

	// Create context with timeout
	uploadCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Get Google storage client from manager
	client := p.clientManager.GetStorageClient()
	obj := client.Bucket(bucketName).Object(objectKey)

	// Check if exists and handle overwrite
	if !req.Overwrite {
		_, err := obj.Attrs(uploadCtx)
		if err == nil {
			return &pb.UploadObjectResponse{
				Success: false,
				Message: "object already exists and overwrite is false",
			}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "object exists", nil)
		}
	}

	// Create writer
	writer := obj.NewWriter(uploadCtx)
	defer writer.Close()

	// Set content type
	contentType := req.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(req.ObjectKey))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	writer.ContentType = contentType

	// Set metadata
	if len(req.Metadata) > 0 {
		writer.Metadata = req.Metadata
	}

	// Write data
	if _, err := writer.Write(req.Content); err != nil {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to write data: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeUploadFailed, "write failed", err)
	}

	// Finalize upload
	if err := writer.Close(); err != nil {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to finalize upload: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeUploadFailed, "finalize failed", err)
	}

	// Get object attributes
	attrs, _ := obj.Attrs(uploadCtx)

	// Build storage object
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(bucketName, objectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_GCP,
		ContainerName: bucketName,
		ObjectKey:     objectKey,
		Size:          int64(len(req.Content)),
		ContentType:   contentType,
		StorageClass:  "STANDARD",
		IsEncrypted:   false,
	}

	if attrs != nil {
		storageObject.Etag = attrs.Etag
		storageObject.LastModified = timestamppb.New(attrs.Updated)
		storageObject.CreatedAt = timestamppb.New(attrs.Created)
		storageObject.StorageClass = string(attrs.StorageClass)
		storageObject.Url = attrs.MediaLink
	}

	duration := time.Since(startTime)

	return &pb.UploadObjectResponse{
		Success:          true,
		Object:           storageObject,
		UploadDurationMs: duration.Milliseconds(),
		Message:          "upload successful",
	}, nil
}

// DownloadObject retrieves an object from GCS
func (p *GCSStorageProvider) DownloadObject(ctx context.Context, req *pb.DownloadObjectRequest) (*pb.DownloadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "GCS storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}

	objectKey := strings.Trim(req.ObjectKey, "/")

	// Create context with timeout
	downloadCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Get Google storage client from manager
	client := p.clientManager.GetStorageClient()
	obj := client.Bucket(bucketName).Object(objectKey)

	// Create reader
	reader, err := obj.NewReader(downloadCtx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return &pb.DownloadObjectResponse{
				Success: false,
				Message: fmt.Sprintf("file not found: %s/%s", bucketName, objectKey),
			}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", err)
		}
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to open object: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "open failed", err)
	}
	defer reader.Close()

	// Read data
	data, err := io.ReadAll(reader)
	if err != nil {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to read data: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "read failed", err)
	}

	// Get object attributes
	attrs, _ := obj.Attrs(downloadCtx)

	// Build storage object
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(bucketName, objectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_GCP,
		ContainerName: bucketName,
		ObjectKey:     objectKey,
		Size:          int64(len(data)),
		ContentType:   reader.Attrs.ContentType,
		StorageClass:  "STANDARD",
	}

	if attrs != nil {
		storageObject.Etag = attrs.Etag
		storageObject.LastModified = timestamppb.New(attrs.Updated)
		storageObject.CreatedAt = timestamppb.New(attrs.Created)
		storageObject.StorageClass = string(attrs.StorageClass)
	}

	duration := time.Since(startTime)

	return &pb.DownloadObjectResponse{
		Success:            true,
		Object:             storageObject,
		Content:            data,
		DownloadDurationMs: duration.Milliseconds(),
		Message:            "download successful",
	}, nil
}

// GetPresignedUrl generates a signed URL for GCS object
func (p *GCSStorageProvider) GetPresignedUrl(ctx context.Context, req *pb.GetPresignedUrlRequest) (*pb.GetPresignedUrlResponse, error) {
	if !p.enabled {
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: "GCS storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	// Note: Generating signed URLs requires service account credentials
	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}

	objectKey := strings.Trim(req.ObjectKey, "/")
	expiresIn := time.Duration(req.ExpiresInSeconds) * time.Second
	expiresAt := time.Now().Add(expiresIn)

	// Determine HTTP method based on operation
	method := "GET"
	if req.Operation == pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_UPLOAD {
		method = "PUT"
	} else if req.Operation == pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_DELETE {
		method = "DELETE"
	}

	// Create SignedURLOptions for GCS
	opts := &storage.SignedURLOptions{
		Method:  method,
		Expires: expiresAt,
	}

	// Add content type for uploads
	if req.Operation == pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_UPLOAD && req.ContentType != "" {
		opts.ContentType = req.ContentType
	}

	// Note: SignedURL requires credentials to be set up properly
	// For now, return not supported error if we can't generate signed URLs
	// This typically requires service account credentials
	url, err := storage.SignedURL(bucketName, objectKey, opts)
	if err != nil {
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: fmt.Sprintf("failed to generate signed URL: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "signed URL failed", err)
	}

	return &pb.GetPresignedUrlResponse{
		Success:    true,
		Url:        url,
		ExpiresAt:  timestamppb.New(expiresAt),
		HttpMethod: method,
		Message:    "signed URL generated successfully",
	}, nil
}

// CreateContainer creates a new GCS bucket
func (p *GCSStorageProvider) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*pb.CreateContainerResponse, error) {
	if !p.enabled {
		return &pb.CreateContainerResponse{
			Message: "GCS storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.CreateContainerResponse{
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	createCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	client := p.clientManager.GetStorageClient()
	bucket := client.Bucket(req.Name)

	// Create bucket
	if err := bucket.Create(createCtx, p.projectID, nil); err != nil {
		return &pb.CreateContainerResponse{
			Message: fmt.Sprintf("failed to create bucket: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "creation failed", err)
	}

	attrs, _ := bucket.Attrs(createCtx)

	container := &pb.StorageContainer{
		Id:                req.Name,
		Provider:          pb.StorageProvider_STORAGE_PROVIDER_GCP,
		Name:              req.Name,
		Description:       req.Description,
		Location:          attrs.Location,
		CreatedAt:         timestamppb.New(attrs.Created),
		IsPublic:          false,
		VersioningEnabled: attrs.VersioningEnabled,
		EncryptionEnabled: false,
	}

	return &pb.CreateContainerResponse{
		Container: container,
		Message:   "bucket created successfully",
	}, nil
}

// GetContainer retrieves GCS bucket information
func (p *GCSStorageProvider) GetContainer(ctx context.Context, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error) {
	if !p.enabled {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	getCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	client := p.clientManager.GetStorageClient()
	bucket := client.Bucket(req.Name)

	attrs, err := bucket.Attrs(getCtx)
	if err != nil {
		if err == storage.ErrBucketNotExist {
			return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "bucket not found", err)
		}
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "failed to get bucket", err)
	}

	container := &pb.StorageContainer{
		Id:                  req.Name,
		Provider:            pb.StorageProvider_STORAGE_PROVIDER_GCP,
		Name:                req.Name,
		Location:            attrs.Location,
		CreatedAt:           timestamppb.New(attrs.Created),
		IsPublic:            false,
		VersioningEnabled:   attrs.VersioningEnabled,
		DefaultStorageClass: string(attrs.StorageClass),
		EncryptionEnabled:   false,
	}

	return &pb.GetContainerResponse{
		Container: container,
	}, nil
}

// DeleteContainer deletes a GCS bucket
func (p *GCSStorageProvider) DeleteContainer(ctx context.Context, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	if !p.enabled {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "GCS storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	deleteCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	client := p.clientManager.GetStorageClient()
	bucket := client.Bucket(req.Name)

	// Delete bucket
	if err := bucket.Delete(deleteCtx); err != nil {
		if err == storage.ErrBucketNotExist {
			return &pb.DeleteContainerResponse{
				Success: false,
				Message: "bucket not found",
			}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", err)
		}
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: fmt.Sprintf("failed to delete bucket: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDeleteFailed, "deletion failed", err)
	}

	return &pb.DeleteContainerResponse{
		Success: true,
		Message: "bucket deleted successfully",
	}, nil
}

// IsHealthy checks if the GCS storage service is available
func (p *GCSStorageProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("GCS storage provider is not initialized")
	}

	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := p.clientManager.GetStorageClient()
	bucket := client.Bucket(p.bucketName)
	_, err := bucket.Attrs(healthCtx)
	return err
}

// Close cleans up GCS client resources
func (p *GCSStorageProvider) Close() error {
	p.enabled = false
	if p.clientManager != nil {
		return p.clientManager.Close()
	}
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *GCSStorageProvider) IsEnabled() bool {
	return p.enabled
}
