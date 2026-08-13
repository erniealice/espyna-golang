package gcs

import (
	"context"
	"errors"
	"testing"
)

func TestGCSObjectAccessProbeUsesConfiguredBucket(t *testing.T) {
	var gotBucket string
	provider := &GCSStorageProvider{
		enabled:    true,
		bucketName: "physical-bucket",
		objectAccessProbe: func(_ context.Context, bucket string) error {
			gotBucket = bucket
			return nil
		},
	}
	if err := provider.IsHealthy(context.Background()); err != nil {
		t.Fatalf("IsHealthy() error = %v", err)
	}
	if gotBucket != "physical-bucket" {
		t.Fatalf("probe bucket = %q, want physical-bucket", gotBucket)
	}
}

func TestGCSObjectAccessProbePropagatesAccessDenied(t *testing.T) {
	want := errors.New("storage.objects.list denied")
	provider := &GCSStorageProvider{
		enabled: true,
		objectAccessProbe: func(context.Context, string) error {
			return want
		},
	}
	if err := provider.IsHealthy(context.Background()); !errors.Is(err, want) {
		t.Fatalf("IsHealthy() error = %v, want %v", err, want)
	}
}

func TestGCSDefaultContainerName(t *testing.T) {
	provider := &GCSStorageProvider{bucketName: "physical-bucket"}
	if got := provider.DefaultContainerName(); got != "physical-bucket" {
		t.Fatalf("DefaultContainerName() = %q", got)
	}
}
