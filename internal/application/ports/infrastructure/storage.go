package infrastructure

import (
	"context"
	"fmt"
	"io"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/storage"
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

// StreamingStorageProvider is an OPTIONAL capability sub-interface that extends
// StorageProvider with bounded-memory, stream-through upload/download. It mirrors
// the StorageCapabilityProvider pattern above: the base StorageProvider does NOT
// declare these methods, so adapters that have not yet opted in keep compiling
// unchanged (mock/local/non-streaming adapters are not forced to implement the
// stream tier in lockstep — that would break the build across all 5 adapters at
// once).
//
// Q-ST-STREAM (LOCKED, B+C): stream-through (io.Reader on the way in,
// io.ReadCloser on the way out, copied via io.Copy) is the UNIVERSAL default; the
// presigned-direct tier (StorageCapabilityPresignedUrls) is the capability-gated
// cloud add-on. Streaming bypasses the in-memory []byte Content field on the proto
// request/response so a multi-hundred-MB object never lands fully in RAM; the proto
// req still carries the container/key/content-type/metadata envelope.
//
// CALLERS MUST type-assert and fall back. Streaming is a capability-gated default,
// not a hard requirement:
//
//	if s, ok := provider.(StreamingStorageProvider); ok {
//	    // bounded-memory path: pump req.Body through io.Copy
//	    resp, err := s.UploadStream(ctx, req, body)        // body is an io.Reader
//	    rc, dlResp, err := s.DownloadStream(ctx, dlReq)    // rc is an io.ReadCloser — caller MUST Close()
//	} else {
//	    // buffered fallback: read the whole object into req.Content / resp.Content
//	    resp, err := provider.UploadObject(ctx, reqWithContent)
//	    dlResp, err := provider.DownloadObject(ctx, dlReq)
//	}
//
// The io.Reader handed to UploadStream is the place to wrap http.MaxBytesReader so
// the byte ceiling is enforced as the stream is consumed (no pre-buffering needed).
type StreamingStorageProvider interface {
	StorageProvider

	// UploadStream uploads an object by streaming body directly to the backend.
	// The proto req carries the container/key/content-type/metadata envelope; its
	// []byte Content field is IGNORED — the streamed body is the payload. The
	// returned response mirrors UploadObject (object metadata, no Content echo).
	// Implementations must NOT buffer the whole body in memory when the backend
	// supports a native streaming writer (S3 PutObject Body, GCS writer, Azure
	// upload-stream, local os.Create+io.Copy).
	UploadStream(ctx context.Context, req *pb.UploadObjectRequest, body io.Reader) (*pb.UploadObjectResponse, error)

	// DownloadStream opens the object as a stream and returns the body as an
	// io.ReadCloser the caller MUST Close, alongside the metadata response (whose
	// []byte Content field is left nil — the bytes flow through the ReadCloser, not
	// the proto). Returns a not-found-shaped error when the object is absent.
	DownloadStream(ctx context.Context, req *pb.DownloadObjectRequest) (io.ReadCloser, *pb.DownloadObjectResponse, error)
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
