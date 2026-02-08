package consumer

import (
	"context"
	"fmt"

	storagepb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/storage"
)

// storageOperations defines the operations interface for storage without Initialize
// This avoids conflicts between contracts.Provider and ports.StorageProvider
type storageOperations interface {
	Name() string
	IsEnabled() bool
	IsHealthy(ctx context.Context) error
	Close() error
	UploadObject(ctx context.Context, req *storagepb.UploadObjectRequest) (*storagepb.UploadObjectResponse, error)
	DownloadObject(ctx context.Context, req *storagepb.DownloadObjectRequest) (*storagepb.DownloadObjectResponse, error)
	GetPresignedUrl(ctx context.Context, req *storagepb.GetPresignedUrlRequest) (*storagepb.GetPresignedUrlResponse, error)
	CreateContainer(ctx context.Context, req *storagepb.CreateContainerRequest) (*storagepb.CreateContainerResponse, error)
	GetContainer(ctx context.Context, req *storagepb.GetContainerRequest) (*storagepb.GetContainerResponse, error)
	DeleteContainer(ctx context.Context, req *storagepb.DeleteContainerRequest) (*storagepb.DeleteContainerResponse, error)
}

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Storage Adapter

Provides direct access to storage operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY storage provider (Local, GCS, S3, Azure, Mock)
based on your CONFIG_STORAGE_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewStorageAdapterFromContainer(container)

	// Upload object
	resp, err := adapter.UploadObject(ctx, "my-bucket", "path/to/file.txt", []byte("content"))

	// Download object
	data, err := adapter.DownloadObject(ctx, "my-bucket", "path/to/file.txt")

	// Get presigned URL for direct download
	url, err := adapter.GetPresignedUrl(ctx, "my-bucket", "path/to/file.txt", 3600)
*/

// StorageAdapter provides technology-agnostic access to storage services.
// It wraps the StorageProvider interface and works with Local, GCS, S3, Azure, etc.
type StorageAdapter struct {
	provider  storageOperations
	container *Container
}

// NewStorageAdapterFromContainer creates a StorageAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's provider.
func NewStorageAdapterFromContainer(container *Container) *StorageAdapter {
	if container == nil {
		return nil
	}

	// Get storage provider from container
	providerContract := container.GetStorageProvider()
	if providerContract == nil {
		return nil
	}

	// Cast to storageOperations interface (avoids Initialize method conflict)
	provider, ok := providerContract.(storageOperations)
	if !ok {
		return nil
	}

	return &StorageAdapter{
		provider:  provider,
		container: container,
	}
}

// Close closes the storage adapter.
// Note: If created from container, this does NOT close the container.
func (a *StorageAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetProvider returns the underlying storage provider for advanced operations.
func (a *StorageAdapter) GetProvider() storageOperations {
	return a.provider
}

// Name returns the name of the underlying storage provider (e.g., "local", "gcs", "s3", "mock")
func (a *StorageAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the storage provider is enabled
func (a *StorageAdapter) IsEnabled() bool {
	return a.provider != nil && a.provider.IsEnabled()
}

// --- Storage Operations ---

// UploadObject uploads an object to storage.
func (a *StorageAdapter) UploadObject(ctx context.Context, containerName, objectKey string, content []byte) (*storagepb.UploadObjectResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.UploadObjectRequest{
		ContainerName: containerName,
		ObjectKey:     objectKey,
		Content:       content,
	}

	return a.provider.UploadObject(ctx, req)
}

// UploadObjectWithContentType uploads an object with a specific content type.
func (a *StorageAdapter) UploadObjectWithContentType(ctx context.Context, containerName, objectKey string, content []byte, contentType string) (*storagepb.UploadObjectResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.UploadObjectRequest{
		ContainerName: containerName,
		ObjectKey:     objectKey,
		Content:       content,
		ContentType:   contentType,
	}

	return a.provider.UploadObject(ctx, req)
}

// UploadObjectProto uploads an object using the protobuf request type directly.
// Use this for full control over all upload parameters.
func (a *StorageAdapter) UploadObjectProto(ctx context.Context, req *storagepb.UploadObjectRequest) (*storagepb.UploadObjectResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}
	return a.provider.UploadObject(ctx, req)
}

// DownloadObject downloads an object from storage and returns its content.
func (a *StorageAdapter) DownloadObject(ctx context.Context, containerName, objectKey string) ([]byte, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.DownloadObjectRequest{
		ContainerName: containerName,
		ObjectKey:     objectKey,
	}

	resp, err := a.provider.DownloadObject(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Content, nil
}

// DownloadObjectFull downloads an object and returns the full response with metadata.
func (a *StorageAdapter) DownloadObjectFull(ctx context.Context, containerName, objectKey string) (*storagepb.DownloadObjectResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.DownloadObjectRequest{
		ContainerName: containerName,
		ObjectKey:     objectKey,
	}

	return a.provider.DownloadObject(ctx, req)
}

// GetPresignedUrl generates a temporary URL for direct access to an object.
// expiresInSeconds specifies how long the URL should be valid.
func (a *StorageAdapter) GetPresignedUrl(ctx context.Context, containerName, objectKey string, expiresInSeconds int64) (string, error) {
	if a.provider == nil {
		return "", fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.GetPresignedUrlRequest{
		ContainerName:    containerName,
		ObjectKey:        objectKey,
		ExpiresInSeconds: expiresInSeconds,
	}

	resp, err := a.provider.GetPresignedUrl(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Url, nil
}

// GetPresignedUrlForUpload generates a presigned URL for direct upload.
func (a *StorageAdapter) GetPresignedUrlForUpload(ctx context.Context, containerName, objectKey string, expiresInSeconds int64) (string, error) {
	if a.provider == nil {
		return "", fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.GetPresignedUrlRequest{
		ContainerName:    containerName,
		ObjectKey:        objectKey,
		ExpiresInSeconds: expiresInSeconds,
		Operation:        storagepb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_UPLOAD,
	}

	resp, err := a.provider.GetPresignedUrl(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Url, nil
}

// --- Container/Bucket Operations ---

// CreateContainer creates a new storage container/bucket.
func (a *StorageAdapter) CreateContainer(ctx context.Context, name string) (*storagepb.CreateContainerResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.CreateContainerRequest{
		Name: name,
	}

	return a.provider.CreateContainer(ctx, req)
}

// GetContainer retrieves information about a container/bucket.
func (a *StorageAdapter) GetContainer(ctx context.Context, name string) (*storagepb.GetContainerResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.GetContainerRequest{
		Name: name,
	}

	return a.provider.GetContainer(ctx, req)
}

// DeleteContainer deletes a container/bucket.
// Warning: May fail if container is not empty.
func (a *StorageAdapter) DeleteContainer(ctx context.Context, name string) (*storagepb.DeleteContainerResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("storage provider not initialized")
	}

	req := &storagepb.DeleteContainerRequest{
		Name: name,
	}

	return a.provider.DeleteContainer(ctx, req)
}

// IsHealthy checks if the storage provider is healthy and available.
func (a *StorageAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("storage provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// --- Storage capability type for consumer convenience ---

// StorageCapability represents storage operation capabilities
type StorageCapability string

// Storage capability constants
const (
	StorageCapabilityUpload          StorageCapability = "upload"
	StorageCapabilityDownload        StorageCapability = "download"
	StorageCapabilityDelete          StorageCapability = "delete"
	StorageCapabilityList            StorageCapability = "list"
	StorageCapabilityMetadata        StorageCapability = "metadata"
	StorageCapabilityPresignedUrls   StorageCapability = "presigned_urls"
	StorageCapabilityMultipartUpload StorageCapability = "multipart_upload"
	StorageCapabilityVersioning      StorageCapability = "versioning"
	StorageCapabilityEncryption      StorageCapability = "encryption"
)
