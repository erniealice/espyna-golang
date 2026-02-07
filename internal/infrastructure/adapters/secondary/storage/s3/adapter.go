//go:build aws && s3

package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"google.golang.org/protobuf/types/known/timestamppb"

	"leapfor.xyz/espyna/internal/application/ports"
	storagecommon "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/common"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	pb "leapfor.xyz/esqyma/golang/v1/infrastructure/storage"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterStorageProvider(
		"s3",
		func() ports.StorageProvider {
			return NewS3StorageProvider()
		},
		transformConfig,
	)
	registry.RegisterStorageBuildFromEnv("s3", buildFromEnv)
}

// buildFromEnv creates and initializes an S3 storage provider from environment variables.
func buildFromEnv() (ports.StorageProvider, error) {
	bucketName := os.Getenv("STORAGE_BUCKET_NAME")
	region := os.Getenv("AWS_REGION")

	protoConfig := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_AWS,
		Config: &pb.StorageProviderConfig_S3Config{
			S3Config: &pb.S3StorageConfig{
				DefaultBucket: bucketName,
				Region:        region,
			},
		},
	}
	p := NewS3StorageProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("s3: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to S3 storage proto config.
func transformConfig(rawConfig map[string]any) (*pb.StorageProviderConfig, error) {
	return nil, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// S3StorageProvider implements AWS S3 storage provider
// This adapter translates proto contracts to AWS S3 SDK operations
type S3StorageProvider struct {
	config     *pb.StorageProviderConfig
	client     *s3.Client
	bucketName string
	region     string
	enabled    bool
	timeout    time.Duration
}

// NewS3StorageProvider creates a new AWS S3 storage provider
func NewS3StorageProvider() ports.StorageProvider {
	return &S3StorageProvider{
		enabled: false,
		timeout: 30 * time.Second,
	}
}

// Name returns the name of this storage provider
func (p *S3StorageProvider) Name() string {
	return "s3"
}

// Initialize sets up the S3 storage provider with proto configuration
func (p *S3StorageProvider) Initialize(config *pb.StorageProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Verify provider type
	if config.Provider != pb.StorageProvider_STORAGE_PROVIDER_AWS {
		return fmt.Errorf("invalid provider type: expected AWS, got %v", config.Provider)
	}

	// Extract S3-specific configuration
	s3Config := config.GetS3Config()
	if s3Config == nil {
		return fmt.Errorf("S3 storage configuration is missing")
	}

	p.region = s3Config.Region
	p.bucketName = s3Config.DefaultBucket

	if p.bucketName == "" {
		return fmt.Errorf("default_bucket cannot be empty")
	}
	if p.region == "" {
		return fmt.Errorf("region cannot be empty")
	}

	// Initialize AWS config
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	var awsConfig aws.Config
	var err error

	// Check authentication method
	if s3Config.UseIamRole {
		// Use IAM role (EC2/ECS/Lambda)
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(p.region),
			awsconfig.WithEC2IMDSClientEnableState(imds.ClientEnabled),
		)
	} else if s3Config.AccessKeyId != "" && s3Config.SecretAccessKey != "" {
		// Use explicit credentials
		creds := credentials.NewStaticCredentialsProvider(
			s3Config.AccessKeyId,
			s3Config.SecretAccessKey,
			s3Config.SessionToken,
		)

		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(p.region),
			awsconfig.WithCredentialsProvider(creds),
		)
	} else {
		// Use default credential chain
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(p.region),
		)
	}

	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Configure endpoint URL if provided (for S3-compatible services)
	if s3Config.EndpointUrl != "" {
		awsConfig.BaseEndpoint = aws.String(s3Config.EndpointUrl)
	}

	// Create S3 client options
	s3Options := func(o *s3.Options) {
		if s3Config.UsePathStyle {
			o.UsePathStyle = true
		}
		if s3Config.UseDualStack {
			o.UsePathStyle = false // Dual-stack requires virtual-hosted style
		}
	}

	p.client = s3.NewFromConfig(awsConfig, s3Options)
	p.config = config
	p.enabled = true

	// Test bucket accessibility
	if err := p.testBucketAccess(ctx); err != nil {
		p.enabled = false
		return fmt.Errorf("S3 bucket access test failed: %w", err)
	}

	return nil
}

// testBucketAccess tests if we can access the configured bucket
func (p *S3StorageProvider) testBucketAccess(ctx context.Context) error {
	_, err := p.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(p.bucketName),
	})

	if err != nil {
		return fmt.Errorf("cannot access bucket %s: %w", p.bucketName, err)
	}

	return nil
}

// UploadObject stores an object in S3
func (p *S3StorageProvider) UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "S3 storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	// Validate request
	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	// Use container name as bucket (or default bucket)
	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}

	// Sanitize object key
	objectKey := strings.Trim(req.ObjectKey, "/")

	// Create context with timeout
	uploadCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Prepare upload input
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(req.Content),
	}

	// Set content type
	contentType := req.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(req.ObjectKey))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	input.ContentType = aws.String(contentType)

	// Set metadata
	if len(req.Metadata) > 0 {
		input.Metadata = req.Metadata
	}

	// Set cache control
	if req.CacheControl != "" {
		input.CacheControl = aws.String(req.CacheControl)
	}

	// Set content disposition
	if req.ContentDisposition != "" {
		input.ContentDisposition = aws.String(req.ContentDisposition)
	}

	// Apply S3-specific options if provided
	if s3Opts := req.GetS3Options(); s3Opts != nil {
		if s3Opts.StorageClass != "" {
			input.StorageClass = types.StorageClass(s3Opts.StorageClass)
		}
		if s3Opts.SseAlgorithm != "" {
			input.ServerSideEncryption = types.ServerSideEncryption(s3Opts.SseAlgorithm)
		}
		if s3Opts.KmsKeyId != "" {
			input.SSEKMSKeyId = aws.String(s3Opts.KmsKeyId)
		}
		if s3Opts.Acl != "" {
			input.ACL = types.ObjectCannedACL(s3Opts.Acl)
		}
	}

	// Check if exists and handle overwrite
	if !req.Overwrite {
		_, err := p.client.HeadObject(uploadCtx, &s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		})
		if err == nil {
			return &pb.UploadObjectResponse{
				Success: false,
				Message: "object already exists and overwrite is false",
			}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "object exists", nil)
		}
	}

	// Upload object
	result, err := p.client.PutObject(uploadCtx, input)
	if err != nil {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to upload: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeUploadFailed, "upload failed", err)
	}

	// Build storage object
	now := time.Now()
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(bucketName, objectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_AWS,
		ContainerName: bucketName,
		ObjectKey:     objectKey,
		Size:          int64(len(req.Content)),
		ContentType:   contentType,
		Etag:          aws.ToString(result.ETag),
		LastModified:  timestamppb.New(now),
		CreatedAt:     timestamppb.New(now),
		StorageClass:  string(input.StorageClass),
		IsEncrypted:   req.EnableEncryption,
		Metadata:      req.Metadata,
	}

	duration := time.Since(startTime)

	return &pb.UploadObjectResponse{
		Success:          true,
		Object:           storageObject,
		UploadDurationMs: duration.Milliseconds(),
		Message:          "upload successful",
	}, nil
}

// DownloadObject retrieves an object from S3
func (p *S3StorageProvider) DownloadObject(ctx context.Context, req *pb.DownloadObjectRequest) (*pb.DownloadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "S3 storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}

	objectKey := strings.Trim(req.ObjectKey, "/")

	// Create context with timeout
	downloadCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Prepare input
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	if req.VersionId != "" {
		input.VersionId = aws.String(req.VersionId)
	}

	if req.Range != "" {
		input.Range = aws.String(req.Range)
	}

	// Get object
	result, err := p.client.GetObject(downloadCtx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return &pb.DownloadObjectResponse{
				Success: false,
				Message: fmt.Sprintf("file not found: %s/%s", bucketName, objectKey),
			}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", err)
		}
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to get object: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "download failed", err)
	}
	defer result.Body.Close()

	// Read data
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to read data: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "read failed", err)
	}

	// Build storage object
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(bucketName, objectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_AWS,
		ContainerName: bucketName,
		ObjectKey:     objectKey,
		Size:          int64(len(data)), // Use actual data length
		ContentType:   aws.ToString(result.ContentType),
		Etag:          aws.ToString(result.ETag),
		StorageClass:  string(result.StorageClass),
	}

	// Set size from response if available
	if result.ContentLength != nil {
		storageObject.Size = *result.ContentLength
	}

	if result.LastModified != nil {
		storageObject.LastModified = timestamppb.New(*result.LastModified)
	}

	if result.VersionId != nil {
		storageObject.VersionId = aws.ToString(result.VersionId)
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

// GetPresignedUrl generates a presigned URL for S3 object
func (p *S3StorageProvider) GetPresignedUrl(ctx context.Context, req *pb.GetPresignedUrlRequest) (*pb.GetPresignedUrlResponse, error) {
	if !p.enabled {
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: "S3 storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}

	objectKey := strings.Trim(req.ObjectKey, "/")
	expiresIn := time.Duration(req.ExpiresInSeconds) * time.Second
	expiresAt := time.Now().Add(expiresIn)

	// Create presign client
	presignClient := s3.NewPresignClient(p.client)

	var url string
	var err error
	httpMethod := "GET"

	// Generate presigned URL based on operation
	switch req.Operation {
	case pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_DOWNLOAD:
		getReq := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		}
		presignResult, presignErr := presignClient.PresignGetObject(ctx, getReq, func(opts *s3.PresignOptions) {
			opts.Expires = expiresIn
		})
		if presignErr != nil {
			err = presignErr
		} else {
			url = presignResult.URL
			httpMethod = presignResult.Method
		}

	case pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_UPLOAD:
		putReq := &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		}
		if req.ContentType != "" {
			putReq.ContentType = aws.String(req.ContentType)
		}
		presignResult, presignErr := presignClient.PresignPutObject(ctx, putReq, func(opts *s3.PresignOptions) {
			opts.Expires = expiresIn
		})
		if presignErr != nil {
			err = presignErr
		} else {
			url = presignResult.URL
			httpMethod = presignResult.Method
		}

	case pb.PresignedUrlOperation_PRESIGNED_URL_OPERATION_DELETE:
		deleteReq := &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		}
		presignResult, presignErr := presignClient.PresignDeleteObject(ctx, deleteReq, func(opts *s3.PresignOptions) {
			opts.Expires = expiresIn
		})
		if presignErr != nil {
			err = presignErr
		} else {
			url = presignResult.URL
			httpMethod = presignResult.Method
		}

	default:
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: "unsupported operation",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "unsupported operation", nil)
	}

	if err != nil {
		return &pb.GetPresignedUrlResponse{
			Success: false,
			Message: fmt.Sprintf("failed to generate presigned URL: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "presign failed", err)
	}

	return &pb.GetPresignedUrlResponse{
		Success:    true,
		Url:        url,
		ExpiresAt:  timestamppb.New(expiresAt),
		HttpMethod: httpMethod,
		Message:    "presigned URL generated successfully",
	}, nil
}

// CreateContainer creates a new S3 bucket
func (p *S3StorageProvider) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*pb.CreateContainerResponse, error) {
	if !p.enabled {
		return &pb.CreateContainerResponse{
			Message: "S3 storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.CreateContainerResponse{
			Message: "container name is required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	createCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Prepare create bucket input
	input := &s3.CreateBucketInput{
		Bucket: aws.String(req.Name),
	}

	// Set location constraint (required for regions other than us-east-1)
	if p.region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(p.region),
		}
	}

	// Create bucket
	_, err := p.client.CreateBucket(createCtx, input)
	if err != nil {
		var bae *types.BucketAlreadyExists
		var bao *types.BucketAlreadyOwnedByYou
		if errors.As(err, &bae) || errors.As(err, &bao) {
			return &pb.CreateContainerResponse{
				Message: "bucket already exists",
			}, ports.NewStorageError(ports.StorageErrorCodeAlreadyExists, "already exists", err)
		}
		return &pb.CreateContainerResponse{
			Message: fmt.Sprintf("failed to create bucket: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "creation failed", err)
	}

	// Build container response
	now := time.Now()
	container := &pb.StorageContainer{
		Id:                req.Name,
		Provider:          pb.StorageProvider_STORAGE_PROVIDER_AWS,
		Name:              req.Name,
		Description:       req.Description,
		Location:          p.region,
		CreatedAt:         timestamppb.New(now),
		IsPublic:          req.IsPublic,
		VersioningEnabled: req.VersioningEnabled,
		EncryptionEnabled: false,
	}

	return &pb.CreateContainerResponse{
		Container: container,
		Message:   "bucket created successfully",
	}, nil
}

// GetContainer retrieves S3 bucket information
func (p *S3StorageProvider) GetContainer(ctx context.Context, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error) {
	if !p.enabled {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.Name == "" {
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "name required", nil)
	}

	getCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Check if bucket exists
	_, err := p.client.HeadBucket(getCtx, &s3.HeadBucketInput{
		Bucket: aws.String(req.Name),
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "NotFound" {
				return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "bucket not found", err)
			}
		}
		return &pb.GetContainerResponse{}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "failed to get bucket", err)
	}

	// Get bucket location
	location, _ := p.client.GetBucketLocation(getCtx, &s3.GetBucketLocationInput{
		Bucket: aws.String(req.Name),
	})

	locationStr := p.region
	if location != nil && location.LocationConstraint != "" {
		locationStr = string(location.LocationConstraint)
	}

	container := &pb.StorageContainer{
		Id:       req.Name,
		Provider: pb.StorageProvider_STORAGE_PROVIDER_AWS,
		Name:     req.Name,
		Location: locationStr,
	}

	return &pb.GetContainerResponse{
		Container: container,
	}, nil
}

// DeleteContainer deletes an S3 bucket
func (p *S3StorageProvider) DeleteContainer(ctx context.Context, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	if !p.enabled {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: "S3 storage provider is not initialized",
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

	// If force delete, empty bucket first
	if req.Force {
		// List and delete all objects
		paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
			Bucket: aws.String(req.Name),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(deleteCtx)
			if err != nil {
				return &pb.DeleteContainerResponse{
					Success: false,
					Message: fmt.Sprintf("failed to list objects: %v", err),
				}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "list failed", err)
			}

			// Delete objects in batch
			if len(page.Contents) > 0 {
				var objectIds []types.ObjectIdentifier
				for _, obj := range page.Contents {
					objectIds = append(objectIds, types.ObjectIdentifier{
						Key: obj.Key,
					})
				}

				_, err = p.client.DeleteObjects(deleteCtx, &s3.DeleteObjectsInput{
					Bucket: aws.String(req.Name),
					Delete: &types.Delete{
						Objects: objectIds,
						Quiet:   aws.Bool(true),
					},
				})

				if err != nil {
					return &pb.DeleteContainerResponse{
						Success: false,
						Message: fmt.Sprintf("failed to delete objects: %v", err),
					}, ports.NewStorageError(ports.StorageErrorCodeDeleteFailed, "deletion failed", err)
				}
			}
		}
	}

	// Delete bucket
	_, err := p.client.DeleteBucket(deleteCtx, &s3.DeleteBucketInput{
		Bucket: aws.String(req.Name),
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "NoSuchBucket" {
				return &pb.DeleteContainerResponse{
					Success: false,
					Message: "bucket not found",
				}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", err)
			}
		}
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: fmt.Sprintf("failed to delete bucket: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDeleteFailed, "deletion failed", err)
	}

	return &pb.DeleteContainerResponse{
		Success: true,
		Message: "bucket deleted successfully",
	}, nil
}

// IsHealthy checks if the S3 storage service is available
func (p *S3StorageProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("S3 storage provider is not initialized")
	}

	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return p.testBucketAccess(healthCtx)
}

// Close cleans up S3 client resources
func (p *S3StorageProvider) Close() error {
	p.enabled = false
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *S3StorageProvider) IsEnabled() bool {
	return p.enabled
}
