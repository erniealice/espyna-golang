//go:build local_storage || mock_db

package local

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
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
		"local",
		func() ports.StorageProvider {
			return NewLocalStorageProvider()
		},
		transformConfig,
	)
	registry.RegisterStorageBuildFromEnv("local", buildFromEnv)
}

// buildFromEnv creates and initializes a local storage provider from environment variables.
func buildFromEnv() (ports.StorageProvider, error) {
	basePath := os.Getenv("STORAGE_BASE_PATH")
	if basePath == "" {
		basePath = "./storage"
	}

	protoConfig := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Config: &pb.StorageProviderConfig_LocalConfig{
			LocalConfig: &pb.LocalStorageConfig{
				BaseDirectory:         basePath,
				AutoCreateDirectories: true,
			},
		},
	}
	p := NewLocalStorageProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("local: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to local storage proto config.
func transformConfig(rawConfig map[string]any) (*pb.StorageProviderConfig, error) {
	return nil, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// LocalStorageProvider implements file storage on the local filesystem
// This adapter translates proto contracts to filesystem operations
type LocalStorageProvider struct {
	config   *pb.StorageProviderConfig
	basePath string
	enabled  bool
}

// NewLocalStorageProvider creates a new local storage provider
func NewLocalStorageProvider() ports.StorageProvider {
	return &LocalStorageProvider{
		enabled: false, // Disabled by default until initialized
	}
}

// Name returns the name of this storage provider
func (p *LocalStorageProvider) Name() string {
	return "local"
}

// Initialize sets up the local storage provider with proto configuration
func (p *LocalStorageProvider) Initialize(config *pb.StorageProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required for local storage provider")
	}

	// Verify provider type
	if config.Provider != pb.StorageProvider_STORAGE_PROVIDER_LOCAL {
		return fmt.Errorf("invalid provider type: expected LOCAL, got %v", config.Provider)
	}

	// Extract local-specific configuration
	localConfig := config.GetLocalConfig()
	if localConfig == nil {
		return fmt.Errorf("local storage configuration is missing")
	}

	baseDir := localConfig.BaseDirectory
	if baseDir == "" {
		return fmt.Errorf("base_directory cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Create directory if configured to do so
	if localConfig.AutoCreateDirectories {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	// Verify directory exists
	if stat, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("storage directory does not exist: %w", err)
	} else if !stat.IsDir() {
		return fmt.Errorf("base_directory is not a directory: %s", absPath)
	}

	// Verify directory is writable by creating a temporary file
	testFile := filepath.Join(absPath, ".espyna_storage_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("storage directory is not writable: %w", err)
	}
	os.Remove(testFile)

	p.config = config
	p.basePath = absPath
	p.enabled = true

	return nil
}

// UploadObject stores an object using proto request/response
func (p *LocalStorageProvider) UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "local storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "provider not initialized", nil)
	}

	// Validate request
	if req.ContainerName == "" {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "container_name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "container name required", nil)
	}

	if req.ObjectKey == "" {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "object_key is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "object key required", nil)
	}

	// Security: validate paths BEFORE sanitizing
	if !isValidPath(req.ContainerName) || !isValidPath(req.ObjectKey) {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "path traversal attempt detected",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "invalid path", nil)
	}

	// Sanitize and build path
	containerPath := filepath.Join(p.basePath, sanitizePath(req.ContainerName))
	objectPath := filepath.Join(containerPath, sanitizePath(req.ObjectKey))

	// Double-check: ensure final path is within base directory
	if !isPathWithinBase(objectPath, p.basePath) {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "path traversal attempt detected",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "invalid path", nil)
	}

	// Create directory structure if needed
	dir := filepath.Dir(objectPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to create directory: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeUploadFailed, "directory creation failed", err)
	}

	// Check if file exists and handle overwrite setting
	if _, err := os.Stat(objectPath); err == nil && !req.Overwrite {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "object already exists and overwrite is false",
		}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "object exists", nil)
	}

	// Write file
	if err := os.WriteFile(objectPath, req.Content, 0644); err != nil {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to write file: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeUploadFailed, "write failed", err)
	}

	// Get file info for response
	fileInfo, _ := os.Stat(objectPath)
	contentType := req.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(req.ObjectKey))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// Build storage object
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(req.ContainerName, req.ObjectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		ContainerName: req.ContainerName,
		ObjectKey:     req.ObjectKey,
		Size:          fileInfo.Size(),
		ContentType:   contentType,
		Etag:          fmt.Sprintf("%d-%d", fileInfo.Size(), fileInfo.ModTime().Unix()),
		LastModified:  timestamppb.New(fileInfo.ModTime()),
		CreatedAt:     timestamppb.New(fileInfo.ModTime()),
		StorageClass:  "local",
		IsEncrypted:   false,
		Url:           fmt.Sprintf("file://%s", objectPath),
	}

	duration := time.Since(startTime)

	return &pb.UploadObjectResponse{
		Success:          true,
		Object:           storageObject,
		UploadDurationMs: duration.Milliseconds(),
		Message:          "upload successful",
	}, nil
}

// DownloadObject retrieves an object using proto request/response
func (p *LocalStorageProvider) DownloadObject(ctx context.Context, req *pb.DownloadObjectRequest) (*pb.DownloadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "local storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "provider not initialized", nil)
	}

	// Validate request
	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	// Build path
	containerPath := filepath.Join(p.basePath, sanitizePath(req.ContainerName))
	objectPath := filepath.Join(containerPath, sanitizePath(req.ObjectKey))

	// Security: ensure path is within base directory
	if !isPathWithinBase(objectPath, p.basePath) {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "path traversal attempt detected",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "invalid path", nil)
	}

	// Read file
	data, err := os.ReadFile(objectPath)
	if os.IsNotExist(err) {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("file not found: %s/%s", req.ContainerName, req.ObjectKey),
		}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "file not found", err)
	}
	if err != nil {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to read file: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "read failed", err)
	}

	// Get file info
	fileInfo, _ := os.Stat(objectPath)
	contentType := mime.TypeByExtension(filepath.Ext(req.ObjectKey))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Build storage object
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(req.ContainerName, req.ObjectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		ContainerName: req.ContainerName,
		ObjectKey:     req.ObjectKey,
		Size:          fileInfo.Size(),
		ContentType:   contentType,
		Etag:          fmt.Sprintf("%d-%d", fileInfo.Size(), fileInfo.ModTime().Unix()),
		LastModified:  timestamppb.New(fileInfo.ModTime()),
		CreatedAt:     timestamppb.New(fileInfo.ModTime()),
		StorageClass:  "local",
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

// GetPresignedUrl generates a presigned URL (not applicable for local storage)
func (p *LocalStorageProvider) GetPresignedUrl(ctx context.Context, req *pb.GetPresignedUrlRequest) (*pb.GetPresignedUrlResponse, error) {
	return &pb.GetPresignedUrlResponse{
		Success: false,
		Message: "presigned URLs are not supported for local storage",
	}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "presigned URLs not supported", nil)
}

// CreateContainer creates a new container (directory)
func (p *LocalStorageProvider) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*pb.CreateContainerResponse, error) {
	if !p.enabled {
		return &pb.CreateContainerResponse{
			Message: "local storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "provider not initialized", nil)
	}

	if req.Name == "" {
		return &pb.CreateContainerResponse{
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	// Build container path
	containerPath := filepath.Join(p.basePath, sanitizePath(req.Name))

	// Security: ensure path is within base directory
	if !isPathWithinBase(containerPath, p.basePath) {
		return &pb.CreateContainerResponse{
			Message: "invalid container name",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "invalid name", nil)
	}

	// Check if container already exists
	if stat, err := os.Stat(containerPath); err == nil {
		if !stat.IsDir() {
			return &pb.CreateContainerResponse{
				Message: "path exists but is not a directory",
			}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "not a directory", nil)
		}
		// Container already exists - this is okay
	}

	// Create directory
	if err := os.MkdirAll(containerPath, 0755); err != nil {
		return &pb.CreateContainerResponse{
			Message: fmt.Sprintf("failed to create container: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "creation failed", err)
	}

	// Get directory info
	dirInfo, _ := os.Stat(containerPath)

	container := &pb.StorageContainer{
		Id:                req.Name,
		Provider:          pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:              req.Name,
		Description:       req.Description,
		Location:          p.basePath,
		CreatedAt:         timestamppb.New(dirInfo.ModTime()),
		IsPublic:          false,
		VersioningEnabled: false,
		EncryptionEnabled: false,
	}

	return &pb.CreateContainerResponse{
		Container: container,
		Message:   "container created successfully",
	}, nil
}

// GetContainer retrieves container information
func (p *LocalStorageProvider) GetContainer(ctx context.Context, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error) {
	if !p.enabled {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "provider not initialized", nil)
	}

	if req.Name == "" {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	// Build container path
	containerPath := filepath.Join(p.basePath, sanitizePath(req.Name))

	// Check if container exists
	stat, err := os.Stat(containerPath)
	if os.IsNotExist(err) {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "container not found", err)
	}
	if err != nil {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "stat failed", err)
	}

	if !stat.IsDir() {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not a directory", nil)
	}

	container := &pb.StorageContainer{
		Id:                req.Name,
		Provider:          pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:              req.Name,
		Location:          p.basePath,
		CreatedAt:         timestamppb.New(stat.ModTime()),
		IsPublic:          false,
		VersioningEnabled: false,
		EncryptionEnabled: false,
	}

	return &pb.GetContainerResponse{
		Container: container,
	}, nil
}

// DeleteContainer deletes a container (directory)
func (p *LocalStorageProvider) DeleteContainer(ctx context.Context, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	if !p.enabled {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "local storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "provider not initialized", nil)
	}

	if req.Name == "" {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	// Build container path
	containerPath := filepath.Join(p.basePath, sanitizePath(req.Name))

	// Security: ensure path is within base directory
	if !isPathWithinBase(containerPath, p.basePath) {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "invalid container name",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "invalid name", nil)
	}

	// Check if directory is empty (unless force delete)
	if !req.Force {
		entries, err := os.ReadDir(containerPath)
		if err == nil && len(entries) > 0 {
			return &pb.DeleteContainerResponse{
				Success: false,
				Message: "container is not empty (use force=true to delete)",
			}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "container not empty", nil)
		}
	}

	// Delete directory
	var err error
	if req.Force {
		err = os.RemoveAll(containerPath) // Remove all contents
	} else {
		err = os.Remove(containerPath) // Only works if empty
	}

	if os.IsNotExist(err) {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "container not found",
		}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "container not found", err)
	}

	if err != nil {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: fmt.Sprintf("failed to delete container: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDeleteFailed, "deletion failed", err)
	}

	return &pb.DeleteContainerResponse{
		Success: true,
		Message: "container deleted successfully",
	}, nil
}

// IsHealthy checks if the storage service is available
func (p *LocalStorageProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("local storage provider is not initialized")
	}

	// Test directory accessibility
	if _, err := os.Stat(p.basePath); err != nil {
		return fmt.Errorf("storage directory is not accessible: %w", err)
	}

	// Test write permission with a temporary file
	testFile := filepath.Join(p.basePath, ".espyna_health_check")
	if err := os.WriteFile(testFile, []byte("health_check"), 0644); err != nil {
		return fmt.Errorf("storage directory is not writable: %w", err)
	}
	os.Remove(testFile)

	return nil
}

// Close cleans up storage provider resources
func (p *LocalStorageProvider) Close() error {
	p.enabled = false
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *LocalStorageProvider) IsEnabled() bool {
	return p.enabled
}

// Upload provides backward compatibility with simple interface
// This method wraps UploadObject for vanilla HTTP adapter
func (p *LocalStorageProvider) Upload(ctx context.Context, path string, data []byte) (string, error) {
	// Parse path (format: "container/key")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid path format, expected 'container/key', got: %s", path)
	}

	container := parts[0]
	key := parts[1]

	// Create proto request
	req := &pb.UploadObjectRequest{
		ContainerName: container,
		ObjectKey:     key,
		Content:       data,
		ContentType:   "application/octet-stream",
	}

	// Upload using proto method
	resp, err := p.UploadObject(ctx, req)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("upload failed: %s", resp.Message)
	}

	// Return full path
	return fmt.Sprintf("%s/%s", container, key), nil
}

// Download provides backward compatibility with simple interface
// This method wraps DownloadObject for vanilla HTTP adapter
func (p *LocalStorageProvider) Download(ctx context.Context, path string) ([]byte, error) {
	// Parse path (format: "container/key")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid path format, expected 'container/key', got: %s", path)
	}

	container := parts[0]
	key := parts[1]

	// Create proto request
	req := &pb.DownloadObjectRequest{
		ContainerName: container,
		ObjectKey:     key,
	}

	// Download using proto method
	resp, err := p.DownloadObject(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("download failed: %s", resp.Message)
	}

	return resp.Content, nil
}

// Helper functions

// isValidPath checks if a path is safe (doesn't contain traversal attempts or absolute paths)
func isValidPath(path string) bool {
	// Reject empty paths
	if path == "" {
		return false
	}

	// Reject absolute paths (both Unix-style and Windows-style)
	if filepath.IsAbs(path) {
		return false
	}

	// Reject Unix-style absolute paths that might not be caught on Windows
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return false
	}

	// Reject paths with .. components
	if strings.Contains(path, "..") {
		return false
	}

	// Clean and check if it changed (indicates traversal attempts)
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return false
	}

	return true
}

// sanitizePath removes potentially dangerous characters from paths
func sanitizePath(path string) string {
	// Clean the path
	clean := filepath.Clean(path)

	// Remove any absolute path indicators
	clean = strings.TrimPrefix(clean, string(filepath.Separator))

	// Remove leading dots to prevent hidden files
	clean = strings.TrimPrefix(clean, ".")

	return clean
}

// isPathWithinBase checks if the resolved path is within the base directory
func isPathWithinBase(path, base string) bool {
	// Convert both paths to absolute and clean them
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}

	// Use filepath.Rel to check if path is within base
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return false
	}

	// If the relative path starts with ".." or is absolute, it's outside the base
	return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}
