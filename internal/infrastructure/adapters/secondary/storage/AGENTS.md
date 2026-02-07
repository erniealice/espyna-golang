# Storage Adapter - Connection Guide

## Overview
This directory contains storage provider implementations for the Espyna hexagonal architecture. Storage adapters handle file upload, download, and deletion operations across different storage backends.

## Directory Structure (Reorganized 2025-12-20)

```
storage/
├── AGENTS.md                    # This documentation file
├── azure/                       # Azure Blob Storage
│   └── adapter.go               # AzureStorageProvider implementation
├── gcs/                         # Google Cloud Storage
│   └── adapter.go               # GCSStorageProvider implementation
├── local/                       # Local filesystem storage
│   ├── adapter.go               # LocalStorageProvider implementation
│   └── adapter_test.go          # Local storage tests
├── mock/                        # In-memory mock storage (testing)
│   ├── adapter.go               # MockStorageProvider implementation
│   └── adapter_test.go          # Mock storage tests
├── s3/                          # AWS S3 storage
│   └── adapter.go               # S3StorageProvider implementation
└── common/                      # Shared utilities
    └── helpers.go               # GenerateObjectID, DetectContentType
```

## Architecture Context

### Hexagonal Architecture Position
```
┌─────────────────────────────────────────────┐
│       Application Layer (Use Cases)        │
│         Domain Logic & Business Rules       │
└─────────────────────────────────────────────┘
                    ↓ Ports (Interfaces)
┌─────────────────────────────────────────────┐
│     Secondary Adapters (Infrastructure)     │
│  ┌─────────────────────────────────────┐   │
│  │     Storage Adapter (This Layer)    │   │
│  │  • local/adapter.go                  │   │
│  │  • gcs/adapter.go (GCP)              │   │
│  │  • s3/adapter.go (AWS)               │   │
│  │  • azure/adapter.go (Azure)          │   │
│  │  • mock/adapter.go (Testing)         │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
                    ↓
      ┌────────────────────────────────┐
      │  Local FS / GCS / S3 / Azure   │
      └────────────────────────────────┘
```

## Available Storage Providers

The Espyna architecture supports **five storage providers**:

| Provider | Directory | Use Case | Build Tag |
|----------|-----------|----------|-----------|
| **Local Storage** | `local/` | Development, testing, single-server deployments | `local_storage` |
| **Google Cloud Storage** | `gcs/` | Production with GCP infrastructure | `google,gcp_storage` |
| **AWS S3** | `s3/` | Production with AWS infrastructure | `aws,s3` |
| **Azure Blob Storage** | `azure/` | Production with Azure infrastructure | `azure,azure_blob` |
| **Mock Storage** | `mock/` | Unit testing without filesystem | `mock_storage` |

Each provider implements the same `StorageProvider` interface, allowing you to switch between them without changing application code.

---

## Common Interface

All storage providers implement the same interface:

```go
type StorageProvider interface {
    // Name returns the provider name
    Name() string

    // Initialize sets up the provider with configuration
    Initialize(config map[string]any) error

    // Upload stores data and returns the storage path
    Upload(ctx context.Context, path string, data []byte) (string, error)

    // Download retrieves data from storage
    Download(ctx context.Context, path string) ([]byte, error)

    // Delete removes data from storage
    Delete(ctx context.Context, path string) error

    // IsHealthy checks if storage is accessible
    IsHealthy(ctx context.Context) error

    // Close cleans up resources
    Close() error

    // IsEnabled returns whether provider is active
    IsEnabled() bool
}
```

---

## Provider Manager Integration

All storage providers are initialized through the Provider Manager:

```go
package main

import (
    "context"
    "log"

    "leapfor.xyz/espyna/internal/composition"
    "leapfor.xyz/espyna/internal/infrastructure/config"
)

func main() {
    // 1. Load configuration
    cfg := config.NewConfig()

    // 2. Create Provider Manager
    pm := composition.NewProviderManager(cfg)

    // 3. Initialize all providers (includes storage)
    if err := pm.Initialize(); err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }
    defer pm.Close()

    // 4. Get active storage provider
    storageProvider := pm.GetActiveStorageProvider()
    if storageProvider == nil {
        log.Println("⚠️ No storage provider configured")
        return
    }

    log.Printf("✅ Active storage provider: %s", storageProvider.Name())

    // 5. Use storage operations
    ctx := context.Background()

    // Upload file
    data := []byte("Hello, storage!")
    path, err := storageProvider.Upload(ctx, "uploads/test.txt", data)
    if err != nil {
        log.Fatalf("Upload failed: %v", err)
    }
    log.Printf("✅ Uploaded to: %s", path)

    // Download file
    retrieved, err := storageProvider.Download(ctx, path)
    if err != nil {
        log.Fatalf("Download failed: %v", err)
    }
    log.Printf("✅ Downloaded: %s", string(retrieved))

    // Delete file
    if err := storageProvider.Delete(ctx, path); err != nil {
        log.Fatalf("Delete failed: %v", err)
    }
    log.Println("✅ File deleted")
}
```

---

# Local Storage Provider

## Overview

The Local Storage provider stores files on the local filesystem. Best for development, testing, and single-server deployments.

### When to Use

✅ **Use Local Storage for:**
- Local development and testing
- Single-server deployments
- Low-traffic applications
- Applications without cloud infrastructure
- Quick prototyping

❌ **Don't Use Local Storage for:**
- Multi-server/load-balanced deployments
- High-availability requirements
- Applications requiring CDN integration
- Scalable production systems

---

## Step 1: Environment Variables

```bash
# Provider Configuration
STORAGE_PROVIDER=local                        # Use local storage
STORAGE_LOCAL_BASE_PATH=./storage            # Local directory path
```

### Complete .env Example

```bash
# Storage Configuration
STORAGE_PROVIDER=local
STORAGE_LOCAL_BASE_PATH=./storage
```

---

## Step 2: Initialize Provider

The Provider Manager automatically initializes local storage:

```go
// Configuration handled by Provider Manager
cfg := config.NewConfig()
pm := composition.NewProviderManager(cfg)

if err := pm.Initialize(); err != nil {
    log.Fatalf("Failed to initialize: %v", err)
}

// Get storage provider
storageProvider := pm.GetActiveStorageProvider()
```

---

## Step 3: Storage Operations

### Upload Files

```go
ctx := context.Background()

// Upload text file
textData := []byte("Hello, World!")
path, err := storageProvider.Upload(ctx, "documents/hello.txt", textData)
if err != nil {
    log.Fatalf("Upload failed: %v", err)
}
log.Printf("✅ Uploaded to: %s", path)

// Upload image
imageData, _ := os.ReadFile("photo.jpg")
imagePath, err := storageProvider.Upload(ctx, "images/photo.jpg", imageData)
if err != nil {
    log.Fatalf("Upload failed: %v", err)
}
```

### Download Files

```go
// Download file
data, err := storageProvider.Download(ctx, "documents/hello.txt")
if err != nil {
    log.Fatalf("Download failed: %v", err)
}

log.Printf("File content: %s", string(data))

// Save to disk
os.WriteFile("downloaded.txt", data, 0644)
```

### Delete Files

```go
// Delete single file
err := storageProvider.Delete(ctx, "documents/hello.txt")
if err != nil {
    log.Fatalf("Delete failed: %v", err)
}
```

### Check Health

```go
// Health check
if err := storageProvider.IsHealthy(ctx); err != nil {
    log.Printf("⚠️ Storage unhealthy: %v", err)
} else {
    log.Println("✅ Storage healthy")
}
```

---

## Step 4: Build Tags

```bash
# Build with local storage
go build -tags="local_storage" -o app ./

# Run with local storage
go run -tags="local_storage" ./

# Test
go test -tags="local_storage" ./...
```

---

## Security Features

Local storage includes built-in security protections:

### Path Traversal Protection

```go
// ❌ These will be rejected
storageProvider.Upload(ctx, "../../../etc/passwd", data)  // Blocked
storageProvider.Upload(ctx, "/etc/passwd", data)          // Blocked
storageProvider.Upload(ctx, "../../secrets", data)        // Blocked

// ✅ These are allowed
storageProvider.Upload(ctx, "uploads/file.txt", data)     // OK
storageProvider.Upload(ctx, "images/photo.jpg", data)     // OK
```

The provider automatically:
- Blocks absolute paths
- Blocks `..` traversal patterns
- Validates paths stay within base directory
- Cleans and sanitizes all paths

---

## File Structure

```
./storage/                      # Base path
├── documents/
│   ├── report.pdf
│   └── invoice.pdf
├── images/
│   ├── photo1.jpg
│   └── photo2.jpg
└── uploads/
    └── user_file.txt
```

---

## Troubleshooting

### Error: "storage directory is not writable"

**Cause:** Insufficient permissions on storage directory

**Solution:**
```bash
# Fix permissions
chmod 755 ./storage
chown $USER:$USER ./storage
```

---

### Error: "path traversal attempt detected"

**Cause:** Path contains `..` or absolute path

**Solution:** Use only relative paths:
```go
// ❌ Wrong
storageProvider.Upload(ctx, "../file.txt", data)

// ✅ Correct
storageProvider.Upload(ctx, "uploads/file.txt", data)
```

---

# Google Cloud Storage (GCS) Provider

## Overview

The GCS provider stores files in Google Cloud Storage buckets. Best for production GCP deployments with global distribution.

### When to Use

✅ **Use GCS for:**
- Production deployments on GCP
- Multi-region applications
- High-availability requirements
- CDN integration needs
- Large file storage
- Integration with other GCP services

---

## Step 1: Environment Variables

```bash
# Provider Configuration
STORAGE_PROVIDER=gcs                          # Use Google Cloud Storage

# GCS Configuration
STORAGE_GCS_BUCKET_NAME=your-bucket-name      # GCS bucket name
STORAGE_GCS_PROJECT_ID=your-project-id        # Google Cloud project ID

# Authentication (choose one method)

# Method 1: Service Account File (Recommended)
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Method 2: Service Account JSON in Environment
STORAGE_GCS_USE_SERVICE_ACCOUNT_JSON=true
GOOGLE_CREDENTIALS_JSON='{"type":"service_account",...}'

# Optional Configuration
STORAGE_GCS_TIMEOUT=30s                       # Upload/download timeout
```

### Complete .env Example

```bash
# Storage Configuration
STORAGE_PROVIDER=gcs
STORAGE_GCS_BUCKET_NAME=my-app-storage
STORAGE_GCS_PROJECT_ID=my-gcp-project
GOOGLE_APPLICATION_CREDENTIALS=/home/user/gcp-key.json
STORAGE_GCS_TIMEOUT=60s
```

---

## Step 2: GCS Bucket Setup

### Create Bucket

```bash
# Using gcloud CLI
gcloud storage buckets create gs://my-app-storage \
  --location=us-central1 \
  --uniform-bucket-level-access

# Verify bucket
gcloud storage buckets describe gs://my-app-storage
```

### Set Permissions

```bash
# Grant service account storage admin role
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:SERVICE_ACCOUNT_EMAIL" \
  --role="roles/storage.objectAdmin"
```

### Bucket Lifecycle (Optional)

```bash
# Create lifecycle.json
cat > lifecycle.json <<EOF
{
  "lifecycle": {
    "rule": [
      {
        "action": {
          "type": "Delete"
        },
        "condition": {
          "age": 30,
          "matchesPrefix": ["temp/"]
        }
      }
    ]
  }
}
EOF

# Apply lifecycle policy
gsutil lifecycle set lifecycle.json gs://my-app-storage
```

---

## Step 3: Storage Operations

```go
ctx := context.Background()
storageProvider := pm.GetActiveStorageProvider()

// Upload to GCS
data := []byte("Hello from GCS!")
path, err := storageProvider.Upload(ctx, "documents/hello.txt", data)
if err != nil {
    log.Fatalf("GCS upload failed: %v", err)
}
log.Printf("✅ Uploaded to GCS: gs://%s/%s", bucketName, path)

// Download from GCS
retrieved, err := storageProvider.Download(ctx, path)
if err != nil {
    log.Fatalf("GCS download failed: %v", err)
}
log.Printf("✅ Retrieved: %s", string(retrieved))

// Delete from GCS
if err := storageProvider.Delete(ctx, path); err != nil {
    log.Fatalf("GCS delete failed: %v", err)
}
```

---

## Step 4: Build Tags

```bash
# Build with GCS
go build -tags="google,gcp_storage" -o app ./

# Run with GCS
go run -tags="google,gcp_storage" ./

# Test
go test -tags="google,gcp_storage" ./...
```

---

## Features

### Automatic Content-Type Detection

GCS provider automatically sets content types:

```go
// Image files
Upload(ctx, "photos/image.jpg", data)   // → image/jpeg
Upload(ctx, "photos/image.png", data)   // → image/png

// Documents
Upload(ctx, "docs/file.pdf", data)      // → application/pdf
Upload(ctx, "docs/file.txt", data)      // → text/plain

// Data files
Upload(ctx, "data/file.json", data)     // → application/json
Upload(ctx, "data/file.xml", data)      // → application/xml
```

### Path Sanitization

```go
// All these become: documents/file.txt
Upload(ctx, "/documents/file.txt", data)
Upload(ctx, "documents/file.txt", data)
Upload(ctx, "//documents//file.txt", data)
```

---

## Troubleshooting

### Error: "failed to initialize Google storage client"

**Cause:** Invalid credentials or missing permissions

**Solution:**
1. Verify service account key:
   ```bash
   cat $GOOGLE_APPLICATION_CREDENTIALS
   ```

2. Test authentication:
   ```bash
   gcloud auth activate-service-account --key-file=$GOOGLE_APPLICATION_CREDENTIALS
   gcloud storage buckets list
   ```

---

### Error: "GCS bucket access test failed"

**Cause:** Bucket doesn't exist or no access

**Solution:**
1. Check bucket exists:
   ```bash
   gcloud storage buckets describe gs://your-bucket-name
   ```

2. Grant permissions:
   ```bash
   gsutil iam ch serviceAccount:EMAIL:roles/storage.objectAdmin gs://your-bucket-name
   ```

---

### Error: "context deadline exceeded"

**Cause:** Network timeout or large file

**Solution:** Increase timeout:
```bash
export STORAGE_GCS_TIMEOUT=120s
```

---

# AWS S3 Provider

## Overview

The S3 provider stores files in AWS S3 buckets. Best for production AWS deployments with global distribution.

### When to Use

✅ **Use S3 for:**
- Production deployments on AWS
- Multi-region applications
- High-availability requirements
- CloudFront CDN integration
- Large file storage
- Integration with other AWS services

---

## Step 1: Environment Variables

```bash
# Provider Configuration
STORAGE_PROVIDER=s3                           # Use AWS S3

# S3 Configuration
STORAGE_S3_BUCKET_NAME=your-bucket-name       # S3 bucket name
STORAGE_S3_REGION=us-east-1                   # AWS region

# Authentication (choose one method)

# Method 1: IAM Role (Recommended for EC2/ECS)
# No credentials needed - uses instance IAM role

# Method 2: Access Keys
STORAGE_S3_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
STORAGE_S3_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# Optional Configuration
STORAGE_S3_TIMEOUT=30s                        # Upload/download timeout
```

### Complete .env Example

**For EC2/ECS with IAM Role:**
```bash
# Storage Configuration
STORAGE_PROVIDER=s3
STORAGE_S3_BUCKET_NAME=my-app-storage
STORAGE_S3_REGION=us-west-2
# Credentials automatically from IAM role
```

**For Local Development:**
```bash
# Storage Configuration
STORAGE_PROVIDER=s3
STORAGE_S3_BUCKET_NAME=my-app-storage
STORAGE_S3_REGION=us-west-2
STORAGE_S3_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
STORAGE_S3_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

---

## Step 2: S3 Bucket Setup

### Create Bucket

```bash
# Using AWS CLI
aws s3 mb s3://my-app-storage --region us-west-2

# Verify bucket
aws s3 ls s3://my-app-storage
```

### Set Permissions

**For IAM Role (EC2/ECS):**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-app-storage",
        "arn:aws:s3:::my-app-storage/*"
      ]
    }
  ]
}
```

**For Access Keys:**
```bash
# Create IAM user
aws iam create-user --user-name espyna-storage

# Attach policy
aws iam put-user-policy --user-name espyna-storage \
  --policy-name S3Access --policy-document file://policy.json

# Create access key
aws iam create-access-key --user-name espyna-storage
```

### Bucket Policy for Public Read (Optional)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicRead",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::my-app-storage/public/*"
    }
  ]
}
```

---

## Step 3: Storage Operations

```go
ctx := context.Background()
storageProvider := pm.GetActiveStorageProvider()

// Upload to S3
data := []byte("Hello from S3!")
path, err := storageProvider.Upload(ctx, "documents/hello.txt", data)
if err != nil {
    log.Fatalf("S3 upload failed: %v", err)
}
log.Printf("✅ Uploaded to S3: s3://%s/%s", bucketName, path)

// Download from S3
retrieved, err := storageProvider.Download(ctx, path)
if err != nil {
    log.Fatalf("S3 download failed: %v", err)
}
log.Printf("✅ Retrieved: %s", string(retrieved))

// Delete from S3
if err := storageProvider.Delete(ctx, path); err != nil {
    log.Fatalf("S3 delete failed: %v", err)
}
```

---

## Step 4: Build Tags

```bash
# Build with S3
go build -tags="aws,s3" -o app ./

# Run with S3
go run -tags="aws,s3" ./

# Test
go test -tags="aws,s3" ./...
```

---

## Features

### Automatic Content-Type Detection

S3 provider automatically sets content types (same as GCS).

### IAM Role Support

When running on AWS infrastructure, the provider automatically uses IAM roles:

```go
// No credentials needed in code
// AWS SDK automatically discovers:
// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
// 2. Shared credentials file (~/.aws/credentials)
// 3. IAM role (for EC2/ECS/Lambda)
```

### Region Support

The provider supports all AWS regions:
- `us-east-1`, `us-west-2`, etc. (US)
- `eu-west-1`, `eu-central-1`, etc. (Europe)
- `ap-southeast-1`, `ap-northeast-1`, etc. (Asia Pacific)
- All other AWS regions

---

## Troubleshooting

### Error: "failed to load AWS config"

**Cause:** Invalid credentials or configuration

**Solution:**
1. Verify credentials:
   ```bash
   aws sts get-caller-identity
   ```

2. Check AWS configuration:
   ```bash
   cat ~/.aws/credentials
   cat ~/.aws/config
   ```

---

### Error: "cannot access bucket"

**Cause:** Bucket doesn't exist or no permissions

**Solution:**
1. Check bucket exists:
   ```bash
   aws s3 ls s3://your-bucket-name
   ```

2. Verify permissions:
   ```bash
   aws s3api get-bucket-location --bucket your-bucket-name
   ```

---

### Error: "file not found"

**Cause:** Object doesn't exist in S3

**Solution:** List bucket contents:
```bash
aws s3 ls s3://your-bucket-name/ --recursive
```

---

## Provider Comparison

| Feature | Local Storage | Google Cloud Storage | AWS S3 |
|---------|--------------|---------------------|--------|
| **Cost** | Free | Pay per GB + requests | Pay per GB + requests |
| **Scalability** | Limited (single server) | Unlimited | Unlimited |
| **Availability** | Single server uptime | 99.95% SLA | 99.99% SLA |
| **CDN Integration** | Manual | Cloud CDN | CloudFront |
| **Setup Complexity** | Low | Medium | Medium |
| **Best For** | Development | GCP infrastructure | AWS infrastructure |
| **Global Distribution** | No | Yes (multi-region) | Yes (multi-region) |
| **Build Tag** | `local_storage` | `google,gcp_storage` | `aws,s3` |

---

## Summary

### Quick Start

1. **Choose your provider** based on infrastructure
2. **Set environment variables** (provider-specific)
3. **Initialize Provider Manager** (one line)
4. **Use storage operations** (Upload, Download, Delete)

### One Interface, Multiple Backends

All providers use the same interface - switch between them by changing:
- Environment variables (`STORAGE_PROVIDER=local|gcs|s3`)
- Build tags (`-tags="local_storage"` or `"google,gcp_storage"` or `"aws,s3"`)

### Key Benefits

- **Hexagonal architecture** - Business logic never touches storage code
- **Easy switching** - Change providers without code changes
- **Consistent interface** - Same operations across all providers
- **Security built-in** - Path validation, sanitization, timeout handling
- **Production-ready** - Health checks, error handling, resource cleanup

---

## Additional Resources

### Official Documentation
- [Local Storage (Go os package)](https://pkg.go.dev/os)
- [Google Cloud Storage Go Client](https://pkg.go.dev/cloud.google.com/go/storage)
- [AWS S3 Go SDK v2](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3)

### Internal Documentation
- **Espyna Architecture**: `packages/espyna/AGENTS.md`
- **Provider System**: `packages/espyna/internal/infrastructure/providers/`
- **Provider Manager**: `packages/espyna/internal/composition/provider_manager.go`