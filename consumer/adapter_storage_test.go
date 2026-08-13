package consumer

import (
	"context"
	"testing"

	storagepb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/storage"
)

type fallbackStorageOperations struct{}

func (*fallbackStorageOperations) Name() string                    { return "fallback" }
func (*fallbackStorageOperations) IsEnabled() bool                 { return true }
func (*fallbackStorageOperations) IsHealthy(context.Context) error { return nil }
func (*fallbackStorageOperations) Close() error                    { return nil }
func (*fallbackStorageOperations) UploadObject(context.Context, *storagepb.UploadObjectRequest) (*storagepb.UploadObjectResponse, error) {
	return &storagepb.UploadObjectResponse{}, nil
}
func (*fallbackStorageOperations) DownloadObject(context.Context, *storagepb.DownloadObjectRequest) (*storagepb.DownloadObjectResponse, error) {
	return &storagepb.DownloadObjectResponse{}, nil
}
func (*fallbackStorageOperations) DeleteObject(context.Context, *storagepb.DeleteObjectRequest) (*storagepb.DeleteObjectResponse, error) {
	return &storagepb.DeleteObjectResponse{Success: true}, nil
}
func (*fallbackStorageOperations) GetPresignedUrl(context.Context, *storagepb.GetPresignedUrlRequest) (*storagepb.GetPresignedUrlResponse, error) {
	return &storagepb.GetPresignedUrlResponse{}, nil
}
func (*fallbackStorageOperations) CreateContainer(context.Context, *storagepb.CreateContainerRequest) (*storagepb.CreateContainerResponse, error) {
	return &storagepb.CreateContainerResponse{}, nil
}
func (*fallbackStorageOperations) GetContainer(context.Context, *storagepb.GetContainerRequest) (*storagepb.GetContainerResponse, error) {
	return &storagepb.GetContainerResponse{}, nil
}
func (*fallbackStorageOperations) DeleteContainer(context.Context, *storagepb.DeleteContainerRequest) (*storagepb.DeleteContainerResponse, error) {
	return &storagepb.DeleteContainerResponse{}, nil
}

type configuredStorageOperations struct {
	fallbackStorageOperations
	container string
}

func (p *configuredStorageOperations) DefaultContainerName() string { return p.container }

func TestResolveContainerNameUsesConfiguredCloudDefault(t *testing.T) {
	adapter := &StorageAdapter{provider: &configuredStorageOperations{container: "physical-bucket"}}
	if got := adapter.ResolveContainerName("templates"); got != "physical-bucket" {
		t.Fatalf("ResolveContainerName() = %q, want physical-bucket", got)
	}
}

func TestResolveContainerNamePreservesFallbackWithoutCapability(t *testing.T) {
	adapter := &StorageAdapter{provider: &fallbackStorageOperations{}}
	if got := adapter.ResolveContainerName("templates"); got != "templates" {
		t.Fatalf("ResolveContainerName() = %q, want templates", got)
	}
}

func TestResolveContainerNamePreservesFallbackForEmptyConfiguredDefault(t *testing.T) {
	adapter := &StorageAdapter{provider: &configuredStorageOperations{}}
	if got := adapter.ResolveContainerName("attachments"); got != "attachments" {
		t.Fatalf("ResolveContainerName() = %q, want attachments", got)
	}
}

func TestResolveContainerNameNilAdapterPreservesFallback(t *testing.T) {
	var adapter *StorageAdapter
	if got := adapter.ResolveContainerName("templates"); got != "templates" {
		t.Fatalf("ResolveContainerName() = %q, want templates", got)
	}
}

func TestStorageAdapterDeleteObjectPassesOneExactTarget(t *testing.T) {
	provider := &recordingStorageOperations{}
	adapter := &StorageAdapter{provider: provider}

	resp, err := adapter.DeleteObject(context.Background(), "templates", "a/../target.docx")
	if err != nil {
		t.Fatalf("DeleteObject() error = %v", err)
	}
	if resp == nil || !resp.Success {
		t.Fatalf("DeleteObject() response = %#v, want success", resp)
	}
	if provider.req == nil || provider.req.ContainerName != "templates" || provider.req.ObjectKey != "a/../target.docx" {
		t.Fatalf("provider request = %#v, want exact container/key", provider.req)
	}
}

func TestStorageAdapterDeleteObjectRejectsEmptyInputs(t *testing.T) {
	provider := &recordingStorageOperations{}
	adapter := &StorageAdapter{provider: provider}
	for _, tc := range []struct {
		name      string
		container string
		key       string
	}{
		{name: "empty container", key: "file"},
		{name: "empty key", container: "templates"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := adapter.DeleteObject(context.Background(), tc.container, tc.key); err == nil {
				t.Fatal("DeleteObject() error = nil, want validation error")
			}
			if provider.req != nil {
				t.Fatal("provider called for invalid delete target")
			}
		})
	}
}

type recordingStorageOperations struct {
	fallbackStorageOperations
	req *storagepb.DeleteObjectRequest
}

func (p *recordingStorageOperations) DeleteObject(_ context.Context, req *storagepb.DeleteObjectRequest) (*storagepb.DeleteObjectResponse, error) {
	p.req = req
	return &storagepb.DeleteObjectResponse{Success: true}, nil
}
