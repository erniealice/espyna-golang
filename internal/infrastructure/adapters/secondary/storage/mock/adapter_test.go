//go:build mock_storage

package mock

import (
	"context"
	"testing"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/storage"
)

// TestMockStorageProvider tests the mock storage provider implementation
func TestMockStorageProvider(t *testing.T) {
	provider := NewMockStorageProvider()

	// Test name
	if provider.Name() != "mock" {
		t.Errorf("Expected provider name 'mock', got '%s'", provider.Name())
	}

	// Test initialization with proto config
	config := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL, // Mock uses local type
		Enabled:  true,
	}

	err := provider.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize mock provider: %v", err)
	}

	if !provider.IsEnabled() {
		t.Error("Provider should be enabled after initialization")
	}

	ctx := context.Background()

	// Test container creation
	createContainerReq := &pb.CreateContainerRequest{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		Name:     "mock-container",
	}

	createResp, err := provider.CreateContainer(ctx, createContainerReq)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	if createResp.Container.Name != "mock-container" {
		t.Errorf("Expected container name 'mock-container', got '%s'", createResp.Container.Name)
	}

	// Test upload new file
	testData := []byte("new test content")
	uploadReq := &pb.UploadObjectRequest{
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		ContainerName: "mock-container",
		ObjectKey:     "new/file.txt",
		Content:       testData,
		ContentType:   "text/plain",
	}

	uploadResp, err := provider.UploadObject(ctx, uploadReq)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	if !uploadResp.Success {
		t.Error("Upload should be successful")
	}

	if uploadResp.Object.ObjectKey != "new/file.txt" {
		t.Errorf("Expected object key 'new/file.txt', got '%s'", uploadResp.Object.ObjectKey)
	}

	// Test data count
	mockProvider := provider.(*MockStorageProvider)
	if mockProvider.GetObjectCount() != 1 {
		t.Errorf("Expected 1 file in storage, got %d", mockProvider.GetObjectCount())
	}

	// Test download
	downloadReq := &pb.DownloadObjectRequest{
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_LOCAL,
		ContainerName: "mock-container",
		ObjectKey:     "new/file.txt",
	}

	downloadResp, err := provider.DownloadObject(ctx, downloadReq)
	if err != nil {
		t.Fatalf("Failed to download file: %v", err)
	}

	if string(downloadResp.Content) != string(testData) {
		t.Errorf("Downloaded data doesn't match uploaded data")
	}

	// Test clear data
	mockProvider.ClearAll()
	if mockProvider.GetObjectCount() != 0 {
		t.Errorf("Expected 0 files after clear, got %d", mockProvider.GetObjectCount())
	}

	// Test health check
	err = provider.IsHealthy(ctx)
	if err != nil {
		t.Errorf("Provider should be healthy: %v", err)
	}

	// Clean up
	err = provider.Close()
	if err != nil {
		t.Errorf("Failed to close provider: %v", err)
	}
}
