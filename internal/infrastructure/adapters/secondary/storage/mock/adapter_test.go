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

func TestMockDeleteObjectDeletesOnlyExactTarget(t *testing.T) {
	provider := NewMockStorageProvider()
	if err := provider.Initialize(&pb.StorageProviderConfig{Provider: pb.StorageProvider_STORAGE_PROVIDER_LOCAL}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	defer provider.Close()
	ctx := context.Background()
	if _, err := provider.CreateContainer(ctx, &pb.CreateContainerRequest{Name: "templates"}); err != nil {
		t.Fatalf("CreateContainer() error = %v", err)
	}
	for _, key := range []string{"target.docx", "sibling.docx"} {
		resp, err := provider.UploadObject(ctx, &pb.UploadObjectRequest{ContainerName: "templates", ObjectKey: key, Content: []byte(key), Overwrite: true})
		if err != nil || !resp.Success {
			t.Fatalf("UploadObject(%q) = %#v, %v", key, resp, err)
		}
	}

	resp, err := provider.DeleteObject(ctx, &pb.DeleteObjectRequest{ContainerName: "templates", ObjectKey: "target.docx"})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("DeleteObject(target) = %#v, %v", resp, err)
	}
	mockProvider := provider.(*MockStorageProvider)
	if mockProvider.GetObjectCount() != 1 {
		t.Fatalf("GetObjectCount() = %d, want sibling retained", mockProvider.GetObjectCount())
	}
	if _, err := provider.DownloadObject(ctx, &pb.DownloadObjectRequest{ContainerName: "templates", ObjectKey: "sibling.docx"}); err != nil {
		t.Fatalf("sibling DownloadObject() error = %v", err)
	}

	resp, err = provider.DeleteObject(ctx, &pb.DeleteObjectRequest{ContainerName: "templates", ObjectKey: "missing.docx"})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("missing DeleteObject() = %#v, %v; want idempotent success", resp, err)
	}
}
