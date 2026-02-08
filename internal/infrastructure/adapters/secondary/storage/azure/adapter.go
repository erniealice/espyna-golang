//go:build azure && azure_blob

package azure

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
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
		"azure",
		func() ports.StorageProvider {
			return NewAzureStorageProvider()
		},
		transformConfig,
	)
	registry.RegisterStorageBuildFromEnv("azure", buildFromEnv)
}

// buildFromEnv creates and initializes an Azure storage provider from environment variables.
func buildFromEnv() (ports.StorageProvider, error) {
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT")
	containerName := os.Getenv("AZURE_CONTAINER_NAME")

	protoConfig := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_AZURE,
		Config: &pb.StorageProviderConfig_AzureConfig{
			AzureConfig: &pb.AzureStorageConfig{
				StorageAccount:   accountName,
				DefaultContainer: containerName,
			},
		},
	}
	p := NewAzureStorageProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("azure: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to Azure storage proto config.
func transformConfig(rawConfig map[string]any) (*pb.StorageProviderConfig, error) {
	return nil, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// AzureStorageProvider implements Azure Blob Storage provider
// This adapter translates proto contracts to Azure Blob SDK operations
type AzureStorageProvider struct {
	config         *pb.StorageProviderConfig
	client         *azblob.Client
	storageAccount string
	containerName  string
	enabled        bool
	timeout        time.Duration
	serviceURL     string
}

// NewAzureStorageProvider creates a new Azure Blob Storage provider
func NewAzureStorageProvider() ports.StorageProvider {
	return &AzureStorageProvider{
		enabled: false,
		timeout: 30 * time.Second,
	}
}

// Name returns the name of this storage provider
func (p *AzureStorageProvider) Name() string {
	return "azure"
}

// Initialize sets up the Azure Blob Storage provider with proto configuration
func (p *AzureStorageProvider) Initialize(config *pb.StorageProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Verify provider type
	if config.Provider != pb.StorageProvider_STORAGE_PROVIDER_AZURE {
		return fmt.Errorf("invalid provider type: expected AZURE, got %v", config.Provider)
	}

	// Extract Azure-specific configuration
	azureConfig := config.GetAzureConfig()
	if azureConfig == nil {
		return fmt.Errorf("Azure storage configuration is missing")
	}

	p.storageAccount = azureConfig.StorageAccount
	p.containerName = azureConfig.DefaultContainer

	if p.storageAccount == "" {
		return fmt.Errorf("storage_account cannot be empty")
	}
	if p.containerName == "" {
		return fmt.Errorf("default_container cannot be empty")
	}

	// Build service URL
	if azureConfig.EndpointUrl != "" {
		p.serviceURL = azureConfig.EndpointUrl
	} else {
		p.serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net/", p.storageAccount)
	}

	// Initialize Azure Blob client with appropriate credentials
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	var client *azblob.Client
	var err error

	// Determine authentication method
	if azureConfig.UseManagedIdentity {
		// Use Managed Identity
		var cred *azidentity.ManagedIdentityCredential
		if azureConfig.ManagedIdentityClientId != "" {
			// User-assigned managed identity
			cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
				ID: azidentity.ClientID(azureConfig.ManagedIdentityClientId),
			})
		} else {
			// System-assigned managed identity
			cred, err = azidentity.NewManagedIdentityCredential(nil)
		}

		if err != nil {
			return fmt.Errorf("failed to create managed identity credential: %w", err)
		}

		client, err = azblob.NewClient(p.serviceURL, cred, nil)
		if err != nil {
			return fmt.Errorf("failed to create Azure Blob client with managed identity: %w", err)
		}

	} else if azureConfig.ConnectionString != "" {
		// Use connection string
		client, err = azblob.NewClientFromConnectionString(azureConfig.ConnectionString, nil)
		if err != nil {
			return fmt.Errorf("failed to create Azure Blob client from connection string: %w", err)
		}

	} else if azureConfig.AccountKey != "" {
		// Use shared key credential
		cred, err := azblob.NewSharedKeyCredential(p.storageAccount, azureConfig.AccountKey)
		if err != nil {
			return fmt.Errorf("failed to create shared key credential: %w", err)
		}

		client, err = azblob.NewClientWithSharedKeyCredential(p.serviceURL, cred, nil)
		if err != nil {
			return fmt.Errorf("failed to create Azure Blob client with shared key: %w", err)
		}

	} else {
		// Use default Azure credential (tries environment variables, managed identity, etc.)
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return fmt.Errorf("failed to create default Azure credential: %w", err)
		}

		client, err = azblob.NewClient(p.serviceURL, cred, nil)
		if err != nil {
			return fmt.Errorf("failed to create Azure Blob client with default credential: %w", err)
		}
	}

	p.client = client
	p.config = config
	p.enabled = true

	// Test container accessibility
	if err := p.testContainerAccess(ctx); err != nil {
		p.enabled = false
		return fmt.Errorf("Azure Blob container access test failed: %w", err)
	}

	return nil
}

// testContainerAccess tests if we can access the configured container
func (p *AzureStorageProvider) testContainerAccess(ctx context.Context) error {
	containerClient := p.client.ServiceClient().NewContainerClient(p.containerName)
	_, err := containerClient.GetProperties(ctx, nil)

	if err != nil {
		// Container might not exist yet, which is okay
		if !bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return fmt.Errorf("cannot access container %s: %w", p.containerName, err)
		}
	}

	return nil
}

// UploadObject stores an object in Azure Blob Storage
func (p *AzureStorageProvider) UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "Azure Blob storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	// Validate request
	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	// Use container name from request or default
	containerName := req.ContainerName
	if containerName == "" {
		containerName = p.containerName
	}

	// Sanitize object key (blob name)
	blobName := strings.Trim(req.ObjectKey, "/")

	// Create context with timeout
	uploadCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Get blob client
	blobClient := p.client.ServiceClient().NewContainerClient(containerName).NewBlockBlobClient(blobName)

	// Check if exists and handle overwrite
	if !req.Overwrite {
		_, err := blobClient.GetProperties(uploadCtx, nil)
		if err == nil {
			return &pb.UploadObjectResponse{
				Success: false,
				Message: "blob already exists and overwrite is false",
			}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "blob exists", nil)
		}
	}

	// Set content type
	contentType := req.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(req.ObjectKey))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// Prepare upload options
	uploadOptions := &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: to.Ptr(contentType),
		},
		Metadata: req.Metadata,
	}

	// Set cache control
	if req.CacheControl != "" {
		uploadOptions.HTTPHeaders.BlobCacheControl = to.Ptr(req.CacheControl)
	}

	// Set content disposition
	if req.ContentDisposition != "" {
		uploadOptions.HTTPHeaders.BlobContentDisposition = to.Ptr(req.ContentDisposition)
	}

	// Apply Azure-specific options if provided
	if azureOpts := req.GetAzureOptions(); azureOpts != nil {
		if azureOpts.AccessTier != "" {
			uploadOptions.AccessTier = (*blob.AccessTier)(to.Ptr(azureOpts.AccessTier))
		}
		if azureOpts.EncryptionScope != "" {
			uploadOptions.CPKScopeInfo = &blob.CPKScopeInfo{
				EncryptionScope: to.Ptr(azureOpts.EncryptionScope),
			}
		}
	}

	// Upload blob
	_, err := blobClient.UploadBuffer(uploadCtx, req.Content, uploadOptions)
	if err != nil {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to upload blob: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeUploadFailed, "upload failed", err)
	}

	// Get blob properties
	props, _ := blobClient.GetProperties(uploadCtx, nil)

	// Build storage object
	now := time.Now()
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(containerName, blobName),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_AZURE,
		ContainerName: containerName,
		ObjectKey:     blobName,
		Size:          int64(len(req.Content)),
		ContentType:   contentType,
		LastModified:  timestamppb.New(now),
		CreatedAt:     timestamppb.New(now),
		StorageClass:  "standard",
		IsEncrypted:   req.EnableEncryption,
		Metadata:      req.Metadata,
	}

	if props != nil {
		if props.ETag != nil {
			storageObject.Etag = string(*props.ETag)
		}
		if props.LastModified != nil {
			storageObject.LastModified = timestamppb.New(*props.LastModified)
		}
		if props.CreationTime != nil {
			storageObject.CreatedAt = timestamppb.New(*props.CreationTime)
		}
		if props.ContentLength != nil {
			storageObject.Size = *props.ContentLength
		}
	}

	duration := time.Since(startTime)

	return &pb.UploadObjectResponse{
		Success:          true,
		Object:           storageObject,
		UploadDurationMs: duration.Milliseconds(),
		Message:          "upload successful",
	}, nil
}

// DownloadObject retrieves an object from Azure Blob Storage
func (p *AzureStorageProvider) DownloadObject(ctx context.Context, req *pb.DownloadObjectRequest) (*pb.DownloadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "Azure Blob storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	containerName := req.ContainerName
	if containerName == "" {
		containerName = p.containerName
	}

	blobName := strings.Trim(req.ObjectKey, "/")

	// Create context with timeout
	downloadCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Get blob client
	blobClient := p.client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)

	// Prepare download options
	downloadOptions := &azblob.DownloadStreamOptions{}

	if req.Range != "" {
		// Parse range header (e.g., "bytes=0-1023")
		downloadOptions.Range = azblob.HTTPRange{}
		// Note: Full range parsing would be needed here
	}

	// Download blob
	response, err := blobClient.DownloadStream(downloadCtx, downloadOptions)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return &pb.DownloadObjectResponse{
				Success: false,
				Message: fmt.Sprintf("blob not found: %s/%s", containerName, blobName),
			}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", err)
		}
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to download blob: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "download failed", err)
	}

	// Read data
	data, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to read blob data: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "read failed", err)
	}

	// Build storage object
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(containerName, blobName),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_AZURE,
		ContainerName: containerName,
		ObjectKey:     blobName,
		Size:          int64(len(data)), // Use actual data length
		StorageClass:  "standard",
	}

	// Set optional fields from response
	if response.ContentLength != nil {
		storageObject.Size = *response.ContentLength
	}
	if response.ContentType != nil {
		storageObject.ContentType = *response.ContentType
	}
	if response.ETag != nil {
		storageObject.Etag = string(*response.ETag)
	}
	if response.LastModified != nil {
		storageObject.LastModified = timestamppb.New(*response.LastModified)
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

// GetPresignedUrl generates a SAS URL for Azure Blob Storage
func (p *AzureStorageProvider) GetPresignedUrl(ctx context.Context, req *pb.GetPresignedUrlRequest) (*pb.GetPresignedUrlResponse, error) {
	if !p.enabled {
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: "Azure Blob storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	containerName := req.ContainerName
	if containerName == "" {
		containerName = p.containerName
	}

	blobName := strings.Trim(req.ObjectKey, "/")
	expiresIn := time.Duration(req.ExpiresInSeconds) * time.Second
	expiresAt := time.Now().Add(expiresIn)

	// Get blob client
	blobClient := p.client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)

	// Determine permissions based on operation
	permissions := sas.BlobPermissions{}
	httpMethod := "GET"

	switch req.Operation {
	case pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_DOWNLOAD:
		permissions.Read = true
		httpMethod = "GET"

	case pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_UPLOAD:
		permissions.Write = true
		permissions.Create = true
		httpMethod = "PUT"

	case pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_DELETE:
		permissions.Delete = true
		httpMethod = "DELETE"

	default:
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: "unsupported operation",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "unsupported operation", nil)
	}

	// Create SAS URL
	sasURL, err := blobClient.GetSASURL(permissions, expiresAt, nil)
	if err != nil {
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: fmt.Sprintf("failed to generate SAS URL: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "SAS URL failed", err)
	}

	return &pb.GetPresignedUrlResponse{
		Success:    true,
		Url:        sasURL,
		ExpiresAt:  timestamppb.New(expiresAt),
		HttpMethod: httpMethod,
		Message:    "SAS URL generated successfully",
	}, nil
}

// CreateContainer creates a new Azure Blob container
func (p *AzureStorageProvider) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*pb.CreateContainerResponse, error) {
	if !p.enabled {
		return &pb.CreateContainerResponse{
			Message: "Azure Blob storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.CreateContainerResponse{
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	createCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Get container client
	containerClient := p.client.ServiceClient().NewContainerClient(req.Name)

	// Create container options
	createOptions := &azblob.CreateContainerOptions{
		Metadata: req.Metadata,
	}

	// Set public access level
	if req.IsPublic {
		createOptions.Access = to.Ptr(container.PublicAccessTypeBlob)
	} else {
		createOptions.Access = to.Ptr(container.PublicAccessTypeNone)
	}

	// Create container
	_, err := containerClient.Create(createCtx, createOptions)
	if err != nil {
		if bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
			return &pb.CreateContainerResponse{
				Message: "container already exists",
			}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "already exists", err)
		}
		return &pb.CreateContainerResponse{
			Message: fmt.Sprintf("failed to create container: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "creation failed", err)
	}

	// Get container properties
	props, _ := containerClient.GetProperties(createCtx, nil)

	// Build container response
	now := time.Now()
	pbContainer := &pb.StorageContainer{
		Id:                req.Name,
		Provider:          pb.StorageProvider_STORAGE_PROVIDER_AZURE,
		Name:              req.Name,
		Description:       req.Description,
		Location:          p.storageAccount,
		CreatedAt:         timestamppb.New(now),
		IsPublic:          req.IsPublic,
		VersioningEnabled: false,
		EncryptionEnabled: true, // Azure encrypts by default
		Metadata:          req.Metadata,
	}

	if props != nil && props.LastModified != nil {
		pbContainer.CreatedAt = timestamppb.New(*props.LastModified)
	}

	return &pb.CreateContainerResponse{
		Container: pbContainer,
		Message:   "container created successfully",
	}, nil
}

// GetContainer retrieves Azure Blob container information
func (p *AzureStorageProvider) GetContainer(ctx context.Context, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error) {
	if !p.enabled {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	getCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Get container client
	containerClient := p.client.ServiceClient().NewContainerClient(req.Name)

	// Get container properties
	props, err := containerClient.GetProperties(getCtx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "container not found", err)
		}
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "failed to get container", err)
	}

	// Determine if public
	isPublic := false
	if props.PublicAccess != nil {
		isPublic = *props.PublicAccess != container.PublicAccessTypeNone
	}

	pbContainer := &pb.StorageContainer{
		Id:                req.Name,
		Provider:          pb.StorageProvider_STORAGE_PROVIDER_AZURE,
		Name:              req.Name,
		Location:          p.storageAccount,
		IsPublic:          isPublic,
		VersioningEnabled: false,
		EncryptionEnabled: true,
		Metadata:          props.Metadata,
	}

	if props.LastModified != nil {
		pbContainer.CreatedAt = timestamppb.New(*props.LastModified)
	}

	return &pb.GetContainerResponse{
		Container: pbContainer,
	}, nil
}

// DeleteContainer deletes an Azure Blob container
func (p *AzureStorageProvider) DeleteContainer(ctx context.Context, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	if !p.enabled {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "Azure Blob storage provider is not initialized",
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

	// Get container client
	containerClient := p.client.ServiceClient().NewContainerClient(req.Name)

	// If force delete, we need to delete all blobs first
	if req.Force {
		// List and delete all blobs
		pager := containerClient.NewListBlobsFlatPager(nil)

		for pager.More() {
			page, err := pager.NextPage(deleteCtx)
			if err != nil {
				return &pb.DeleteContainerResponse{
					Success: false,
					Message: fmt.Sprintf("failed to list blobs: %v", err),
				}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "list failed", err)
			}

			// Delete each blob
			for _, blobItem := range page.Segment.BlobItems {
				blobClient := containerClient.NewBlobClient(*blobItem.Name)
				_, err := blobClient.Delete(deleteCtx, nil)
				if err != nil {
					return &pb.DeleteContainerResponse{
						Success: false,
						Message: fmt.Sprintf("failed to delete blob %s: %v", *blobItem.Name, err),
					}, ports.NewStorageError(ports.StorageErrorCodeDeleteFailed, "blob deletion failed", err)
				}
			}
		}
	}

	// Delete container
	_, err := containerClient.Delete(deleteCtx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return &pb.DeleteContainerResponse{
				Success: false,
				Message: "container not found",
			}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", err)
		}
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

// IsHealthy checks if the Azure Blob Storage service is available
func (p *AzureStorageProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("Azure Blob storage provider is not initialized")
	}

	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return p.testContainerAccess(healthCtx)
}

// Close cleans up Azure Blob Storage client resources
func (p *AzureStorageProvider) Close() error {
	p.enabled = false
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *AzureStorageProvider) IsEnabled() bool {
	return p.enabled
}
