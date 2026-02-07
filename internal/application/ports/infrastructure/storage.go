package infrastructure

import (
	"context"
	"fmt"

	pb "leapfor.xyz/esqyma/golang/v1/infrastructure/storage"
)

// StorageProvider defines the contract for storage providers
// This interface uses proto-generated types for data but defines behavioral contracts
// for lifecycle management and operations
//
// The hexagonal architecture adapters (Local, GCS, S3, Azure) implement this interface
// and handle provider-specific translation between proto contracts and provider SDKs.
type StorageProvider interface {
	// Lifecycle Management
	// These methods define behavior that protos don't cover

	// Name returns the provider identifier (e.g., "local", "gcs", "s3", "azure")
	Name() string

	// Initialize sets up the storage provider with configuration
	// Uses proto config for provider-specific settings
	Initialize(config *pb.StorageProviderConfig) error

	// IsHealthy checks if the storage service is accessible
	// Provider-specific health checks (bucket access, credentials, etc.)
	IsHealthy(ctx context.Context) error

	// Close cleans up storage provider resources
	// Handles connection pooling, cleanup, etc.
	Close() error

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool

	// Storage Operations
	// These use proto request/response types for consistency

	// UploadObject uploads an object to storage
	// Handles small uploads with inline content
	UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error)

	// DownloadObject downloads an object from storage
	// Returns object content and metadata
	DownloadObject(ctx context.Context, req *pb.DownloadObjectRequest) (*pb.DownloadObjectResponse, error)

	// GetPresignedUrl generates a temporary URL for direct access
	// Useful for client-side downloads without proxying through the server
	GetPresignedUrl(ctx context.Context, req *pb.GetPresignedUrlRequest) (*pb.GetPresignedUrlResponse, error)

	// Container/Bucket Operations

	// CreateContainer creates a new container/bucket
	// Provider adapters translate to bucket/container creation
	CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*pb.CreateContainerResponse, error)

	// GetContainer retrieves container/bucket information
	// Returns metadata about the container
	GetContainer(ctx context.Context, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error)

	// DeleteContainer deletes a container/bucket
	// Warning: May fail if container is not empty
	DeleteContainer(ctx context.Context, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error)

	// TODO: Future operations (implement when proto contracts are ready)
	// ListObjects(ctx context.Context, req *pb.ListObjectsRequest) (*pb.ListObjectsResponse, error)
	// DeleteObject(ctx context.Context, req *pb.DeleteObjectRequest) (*pb.DeleteObjectResponse, error)
	// GetObjectMetadata(ctx context.Context, req *pb.GetObjectMetadataRequest) (*pb.GetObjectMetadataResponse, error)
	// InitiateMultipartUpload(ctx context.Context, req *pb.InitiateMultipartUploadRequest) (*pb.InitiateMultipartUploadResponse, error)
	// UploadPart(ctx context.Context, req *pb.UploadPartRequest) (*pb.UploadPartResponse, error)
	// CompleteMultipartUpload(ctx context.Context, req *pb.CompleteMultipartUploadRequest) (*pb.CompleteMultipartUploadResponse, error)
	// AbortMultipartUpload(ctx context.Context, req *pb.AbortMultipartUploadRequest) (*pb.AbortMultipartUploadResponse, error)
}

// StorageCapability represents features supported by a storage provider
// This helps use cases determine what operations are available
type StorageCapability string

const (
	// Core capabilities
	StorageCapabilityUpload   StorageCapability = "upload"
	StorageCapabilityDownload StorageCapability = "download"
	StorageCapabilityDelete   StorageCapability = "delete"
	StorageCapabilityList     StorageCapability = "list"
	StorageCapabilityMetadata StorageCapability = "metadata"

	// Advanced capabilities
	StorageCapabilityPresignedUrls   StorageCapability = "presigned_urls"
	StorageCapabilityMultipartUpload StorageCapability = "multipart_upload"
	StorageCapabilityVersioning      StorageCapability = "versioning"
	StorageCapabilityEncryption      StorageCapability = "encryption"
	StorageCapabilityAccessTiers     StorageCapability = "access_tiers"
	StorageCapabilityObjectLock      StorageCapability = "object_lock"
	StorageCapabilityLifecyclePolicy StorageCapability = "lifecycle_policy"
	StorageCapabilityReplication     StorageCapability = "replication"
	StorageCapabilityStreaming       StorageCapability = "streaming"
)

// StorageCapabilityProvider extends StorageProvider with capability discovery
// Implement this to declare what features your adapter supports
type StorageCapabilityProvider interface {
	StorageProvider

	// GetCapabilities returns the list of capabilities this provider supports
	GetCapabilities() []StorageCapability

	// SupportsCapability checks if a specific capability is supported
	SupportsCapability(capability StorageCapability) bool
}

// StorageError represents storage-related errors
type StorageError struct {
	Code    string
	Message string
	Err     error
	Context map[string]any
}

// Error implements the error interface
func (e *StorageError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (details: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewStorageError creates a new storage error
func NewStorageError(code, message string, err error) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]any),
	}
}

// Common storage error codes
const (
	StorageErrorCodeNotFound         = "STORAGE_NOT_FOUND"
	StorageErrorCodeAlreadyExists    = "STORAGE_ALREADY_EXISTS"
	StorageErrorCodeAccessDenied     = "STORAGE_ACCESS_DENIED"
	StorageErrorCodeQuotaExceeded    = "STORAGE_QUOTA_EXCEEDED"
	StorageErrorCodeInvalidPath      = "STORAGE_INVALID_PATH"
	StorageErrorCodeUploadFailed     = "STORAGE_UPLOAD_FAILED"
	StorageErrorCodeDownloadFailed   = "STORAGE_DOWNLOAD_FAILED"
	StorageErrorCodeDeleteFailed     = "STORAGE_DELETE_FAILED"
	StorageErrorCodeProviderError    = "STORAGE_PROVIDER_ERROR"
	StorageErrorCodeConfigError      = "STORAGE_CONFIG_ERROR"
	StorageErrorCodeConnectionFailed = "STORAGE_CONNECTION_FAILED"
)
