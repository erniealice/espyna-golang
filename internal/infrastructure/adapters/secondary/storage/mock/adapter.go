//go:build mock_storage

package mock

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	storagecommon "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/storage/common"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/storage"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterStorageProvider(
		"mock_storage",
		func() ports.StorageProvider {
			return NewMockStorageProvider()
		},
		transformConfig,
	)
	registry.RegisterStorageBuildFromEnv("mock_storage", buildFromEnv)
}

// buildFromEnv creates and initializes a mock storage provider.
func buildFromEnv() (ports.StorageProvider, error) {
	protoConfig := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Config: &pb.StorageProviderConfig_LocalConfig{
			LocalConfig: &pb.LocalStorageConfig{
				BaseDirectory:         "./mock_storage",
				AutoCreateDirectories: true,
			},
		},
	}
	p := NewMockStorageProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("mock_storage: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to mock storage proto config.
func transformConfig(rawConfig map[string]any) (*pb.StorageProviderConfig, error) {
	return &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Config: &pb.StorageProviderConfig_LocalConfig{
			LocalConfig: &pb.LocalStorageConfig{
				BaseDirectory:         "./mock_storage",
				AutoCreateDirectories: true,
			},
		},
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// MockStorageProvider implements an in-memory storage provider for testing
// This adapter is perfect for unit tests and development without external dependencies
type MockStorageProvider struct {
	config     *pb.StorageProviderConfig
	objects    map[string]*mockObject    // containerName/objectKey -> object
	containers map[string]*mockContainer // containerName -> container
	enabled    bool
	mutex      sync.RWMutex
}

type mockObject struct {
	data      []byte
	metadata  *pb.StorageObject
	createdAt time.Time
	updatedAt time.Time
}

type mockContainer struct {
	metadata  *pb.StorageContainer
	createdAt time.Time
}

// NewMockStorageProvider creates a new mock storage provider
func NewMockStorageProvider() ports.StorageProvider {
	return &MockStorageProvider{
		objects:    make(map[string]*mockObject),
		containers: make(map[string]*mockContainer),
		enabled:    false,
	}
}

// Name returns the name of this storage provider
func (p *MockStorageProvider) Name() string {
	return "mock"
}

// Initialize sets up the mock storage provider with proto configuration
func (p *MockStorageProvider) Initialize(config *pb.StorageProviderConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Mock provider accepts any provider type for testing flexibility
	p.config = config
	p.enabled = true

	return nil
}

// UploadObject stores an object in memory
func (p *MockStorageProvider) UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "mock storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	// Validate request
	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Ensure container exists
	if _, exists := p.containers[req.ContainerName]; !exists {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("container not found: %s", req.ContainerName),
		}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "container not found", nil)
	}

	key := objectKey(req.ContainerName, req.ObjectKey)

	// Check if exists and handle overwrite
	if _, exists := p.objects[key]; exists && !req.Overwrite {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "object already exists and overwrite is false",
		}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "object exists", nil)
	}

	// Store object
	dataCopy := make([]byte, len(req.Content))
	copy(dataCopy, req.Content)

	now := time.Now()
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(req.ContainerName, req.ObjectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL, // Mock uses local type
		ContainerName: req.ContainerName,
		ObjectKey:     req.ObjectKey,
		Size:          int64(len(req.Content)),
		ContentType:   req.ContentType,
		Etag:          fmt.Sprintf("mock-etag-%d", now.Unix()),
		LastModified:  timestamppb.New(now),
		CreatedAt:     timestamppb.New(now),
		StorageClass:  "mock",
		IsEncrypted:   req.EnableEncryption,
		Metadata:      req.Metadata,
	}

	p.objects[key] = &mockObject{
		data:      dataCopy,
		metadata:  storageObject,
		createdAt: now,
		updatedAt: now,
	}

	duration := time.Since(startTime)

	return &pb.UploadObjectResponse{
		Success:          true,
		Object:           storageObject,
		UploadDurationMs: duration.Milliseconds(),
		Message:          "upload successful",
	}, nil
}

// DownloadObject retrieves an object from memory
func (p *MockStorageProvider) DownloadObject(ctx context.Context, req *pb.DownloadObjectRequest) (*pb.DownloadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "mock storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing fields", nil)
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	key := objectKey(req.ContainerName, req.ObjectKey)
	obj, exists := p.objects[key]
	if !exists {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("object not found: %s/%s", req.ContainerName, req.ObjectKey),
		}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", nil)
	}

	// Return copy of data
	dataCopy := make([]byte, len(obj.data))
	copy(dataCopy, obj.data)

	duration := time.Since(startTime)

	return &pb.DownloadObjectResponse{
		Success:            true,
		Object:             obj.metadata,
		Content:            dataCopy,
		DownloadDurationMs: duration.Milliseconds(),
		Message:            "download successful",
	}, nil
}

// GetPresignedUrl generates a mock presigned URL
func (p *MockStorageProvider) GetPresignedUrl(ctx context.Context, req *pb.GetPresignedUrlRequest) (*pb.GetPresignedUrlResponse, error) {
	if !p.enabled {
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: "mock storage provider is not initialized",
		}, nil
	}

	// Generate mock URL
	mockUrl := fmt.Sprintf("mock://%s/%s?expires=%d", req.ContainerName, req.ObjectKey, req.ExpiresInSeconds)
	expiresAt := time.Now().Add(time.Duration(req.ExpiresInSeconds) * time.Second)

	return &pb.GetPresignedUrlResponse{
		Success:    true,
		Url:        mockUrl,
		ExpiresAt:  timestamppb.New(expiresAt),
		HttpMethod: "GET",
		Message:    "mock presigned URL generated",
	}, nil
}

// CreateContainer creates a new container in memory
func (p *MockStorageProvider) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*pb.CreateContainerResponse, error) {
	if !p.enabled {
		return &pb.CreateContainerResponse{
			Message: "mock storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.CreateContainerResponse{
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if container already exists
	if _, exists := p.containers[req.Name]; exists {
		return &pb.CreateContainerResponse{
			Message: "container already exists",
		}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "exists", nil)
	}

	now := time.Now()
	container := &pb.StorageContainer{
		Id:                  req.Name,
		Provider:            pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:                req.Name,
		Description:         req.Description,
		Location:            "mock",
		CreatedAt:           timestamppb.New(now),
		IsPublic:            req.IsPublic,
		VersioningEnabled:   req.VersioningEnabled,
		DefaultStorageClass: req.DefaultStorageClass,
		EncryptionEnabled:   false,
		Metadata:            req.Metadata,
	}

	p.containers[req.Name] = &mockContainer{
		metadata:  container,
		createdAt: now,
	}

	return &pb.CreateContainerResponse{
		Container: container,
		Message:   "container created successfully",
	}, nil
}

// GetContainer retrieves container information
func (p *MockStorageProvider) GetContainer(ctx context.Context, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error) {
	if !p.enabled {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	container, exists := p.containers[req.Name]
	if !exists {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "container not found", nil)
	}

	return &pb.GetContainerResponse{
		Container: container.metadata,
	}, nil
}

// DeleteContainer deletes a container from memory
func (p *MockStorageProvider) DeleteContainer(ctx context.Context, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	if !p.enabled {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "mock storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if container exists
	if _, exists := p.containers[req.Name]; !exists {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "container not found",
		}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", nil)
	}

	// Check if container has objects (unless force)
	if !req.Force {
		for key := range p.objects {
			if strings.HasPrefix(key, req.Name+"/") {
				return &pb.DeleteContainerResponse{
					Success: false,
					Message: "container is not empty (use force=true to delete)",
				}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not empty", nil)
			}
		}
	}

	// Delete container and all objects if force
	if req.Force {
		// Delete all objects in container
		for key := range p.objects {
			if strings.HasPrefix(key, req.Name+"/") {
				delete(p.objects, key)
			}
		}
	}

	delete(p.containers, req.Name)

	return &pb.DeleteContainerResponse{
		Success: true,
		Message: "container deleted successfully",
	}, nil
}

// IsHealthy checks if the mock storage service is available
func (p *MockStorageProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("mock storage provider is not initialized")
	}
	return nil
}

// Close cleans up mock storage resources
func (p *MockStorageProvider) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.objects = make(map[string]*mockObject)
	p.containers = make(map[string]*mockContainer)
	p.enabled = false

	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *MockStorageProvider) IsEnabled() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.enabled
}

// Helper functions for testing

// GetObjectCount returns the number of stored objects (for testing)
func (p *MockStorageProvider) GetObjectCount() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.objects)
}

// GetContainerCount returns the number of containers (for testing)
func (p *MockStorageProvider) GetContainerCount() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.containers)
}

// ClearAll clears all data (for testing)
func (p *MockStorageProvider) ClearAll() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.objects = make(map[string]*mockObject)
	p.containers = make(map[string]*mockContainer)
}

// Helper functions

func objectKey(containerName, objectKey string) string {
	return filepath.Join(containerName, objectKey)
}
