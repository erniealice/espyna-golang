//go:build local_storage || mock_db

package local

import (
	"context"
	"testing"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/storage"
)

// TestLocalStorageProvider tests the local storage provider implementation
func TestLocalStorageProvider(t *testing.T) {
	provider := NewLocalStorageProvider()

	// Test name
	if provider.Name() != "local" {
		t.Errorf("Expected provider name 'local', got '%s'", provider.Name())
	}

	// Test initialization with proto config
	config := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Enabled:  true,
		Config: &pb.StorageProviderConfig_LocalConfig{
			LocalConfig: &pb.LocalStorageConfig{
				BaseDirectory:         "./test_storage",
				AutoCreateDirectories: true,
				FilePermissions:       "0644",
				DirectoryPermissions:  "0755",
			},
		},
	}

	err := provider.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize local provider: %v", err)
	}

	if !provider.IsEnabled() {
		t.Error("Provider should be enabled after initialization")
	}

	ctx := context.Background()

	// Test container creation
	createContainerReq := &pb.CreateContainerRequest{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:     "test-container",
	}

	createResp, err := provider.CreateContainer(ctx, createContainerReq)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	if createResp.Container.Name != "test-container" {
		t.Errorf("Expected container name 'test-container', got '%s'", createResp.Container.Name)
	}

	// Test upload
	testData := []byte("test content")
	uploadReq := &pb.UploadObjectRequest{
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		ContainerName: "test-container",
		ObjectKey:     "test/file.txt",
		Content:       testData,
		ContentType:   "text/plain",
	}

	uploadResp, err := provider.UploadObject(ctx, uploadReq)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	if !uploadResp.Success {
		t.Errorf("Upload should be successful")
	}

	if uploadResp.Object.ObjectKey != "test/file.txt" {
		t.Errorf("Expected object key 'test/file.txt', got '%s'", uploadResp.Object.ObjectKey)
	}

	// Test download
	downloadReq := &pb.DownloadObjectRequest{
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		ContainerName: "test-container",
		ObjectKey:     "test/file.txt",
	}

	downloadResp, err := provider.DownloadObject(ctx, downloadReq)
	if err != nil {
		t.Fatalf("Failed to download file: %v", err)
	}

	if !downloadResp.Success {
		t.Errorf("Download should be successful")
	}

	if string(downloadResp.Content) != string(testData) {
		t.Errorf("Downloaded data doesn't match uploaded data")
	}

	// Test health check
	err = provider.IsHealthy(ctx)
	if err != nil {
		t.Errorf("Provider should be healthy: %v", err)
	}

	// Test get container
	getContainerReq := &pb.GetContainerRequest{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:     "test-container",
	}

	getResp, err := provider.GetContainer(ctx, getContainerReq)
	if err != nil {
		t.Fatalf("Failed to get container: %v", err)
	}

	if getResp.Container.Name != "test-container" {
		t.Errorf("Expected container name 'test-container', got '%s'", getResp.Container.Name)
	}

	// Test delete container (should fail - not empty)
	deleteContainerReq := &pb.DeleteContainerRequest{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:     "test-container",
		Force:    false,
	}

	_, err = provider.DeleteContainer(ctx, deleteContainerReq)
	if err == nil {
		t.Error("Delete should fail when container is not empty and force is false")
	}

	// Test delete container with force
	deleteContainerReq.Force = true
	deleteResp, err := provider.DeleteContainer(ctx, deleteContainerReq)
	if err != nil {
		t.Fatalf("Failed to delete container with force: %v", err)
	}

	if !deleteResp.Success {
		t.Error("Force delete should succeed")
	}

	// Clean up
	err = provider.Close()
	if err != nil {
		t.Errorf("Failed to close provider: %v", err)
	}

	if provider.IsEnabled() {
		t.Error("Provider should be disabled after close")
	}
}

// TestStorageProviderSecurityValidation tests security features of storage providers
func TestStorageProviderSecurityValidation(t *testing.T) {
	provider := NewLocalStorageProvider()

	config := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Enabled:  true,
		Config: &pb.StorageProviderConfig_LocalConfig{
			LocalConfig: &pb.LocalStorageConfig{
				BaseDirectory:         "./test_storage_security",
				AutoCreateDirectories: true,
				FilePermissions:       "0644",
				DirectoryPermissions:  "0755",
			},
		},
	}

	err := provider.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Create test container
	createContainerReq := &pb.CreateContainerRequest{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:     "security-test",
	}

	_, err = provider.CreateContainer(ctx, createContainerReq)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	// Test directory traversal prevention
	testCases := []string{
		"../outside.txt",
		"../../outside.txt",
		"/absolute/path.txt",
		"normal/../traversal.txt",
	}

	for _, path := range testCases {
		uploadReq := &pb.UploadObjectRequest{
			Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
			ContainerName: "security-test",
			ObjectKey:     path,
			Content:       []byte("test"),
		}

		_, err := provider.UploadObject(ctx, uploadReq)
		if err == nil {
			t.Errorf("Upload should fail for potentially dangerous path: %s", path)
		}
	}

	// Clean up
	deleteContainerReq := &pb.DeleteContainerRequest{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:     "security-test",
		Force:    true,
	}
	provider.DeleteContainer(ctx, deleteContainerReq)
}
