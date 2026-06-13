package adapter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strconv"
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

	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
	storagecommon "github.com/erniealice/espyna-golang/storage/helpers"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/storage"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterStorageProvider(
		"aws_storage",
		func() ports.StorageProvider {
			return NewS3StorageProvider()
		},
		transformConfig,
	)
	registry.RegisterStorageBuildFromEnv("aws_storage", buildFromEnv)
}

// buildFromEnv creates and initializes an S3 storage provider from environment variables.
//
// Q-ST-S3COMPAT (LOCKED, A): ONE S3 adapter serves AWS S3 AND every S3-compatible
// endpoint (DO Spaces / MinIO / Wasabi / Cloudflare R2). Initialize already honors
// EndpointUrl/UsePathStyle/credentials/UseIamRole (see adapter.go below); only this
// buildFromEnv was missing the plumbing. The fix is buildFromEnv-only.
//
// Env key scheme (ST-M2): the canonical keys converge on STORAGE_S3_<KEY>. The
// legacy STORAGE_BUCKET_NAME / AWS_REGION keys remain accepted fallbacks for one
// release so existing deploys do not break.
func buildFromEnv() (ports.StorageProvider, error) {
	// Bucket + region: prefer the standardized STORAGE_S3_* keys, fall back to the
	// legacy keys for one release.
	bucketName := firstNonEmpty(os.Getenv("STORAGE_S3_BUCKET_NAME"), os.Getenv("STORAGE_BUCKET_NAME"))
	region := firstNonEmpty(os.Getenv("STORAGE_S3_REGION"), os.Getenv("AWS_REGION"))

	// S3-compatible endpoint + path-style (MinIO/Spaces/R2/Wasabi). Initialize sets
	// awsConfig.BaseEndpoint from EndpointUrl and o.UsePathStyle from UsePathStyle.
	endpointURL := os.Getenv("STORAGE_S3_ENDPOINT")
	usePathStyle, _ := strconv.ParseBool(os.Getenv("STORAGE_S3_FORCE_PATH_STYLE"))

	// Credentials. UseIamRole is mutually exclusive with explicit creds; Initialize
	// branches on it first.
	useIamRole, _ := strconv.ParseBool(os.Getenv("STORAGE_S3_USE_IAM_ROLE"))
	accessKeyID := os.Getenv("STORAGE_S3_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("STORAGE_S3_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("STORAGE_S3_SESSION_TOKEN")

	protoConfig := &pb.StorageProviderConfig{
		Provider: pb.StorageProvider_STORAGE_PROVIDER_AWS,
		Config: &pb.StorageProviderConfig_S3Config{
			S3Config: &pb.S3StorageConfig{
				DefaultBucket:   bucketName,
				Region:          region,
				EndpointUrl:     endpointURL,
				UsePathStyle:    usePathStyle,
				UseIamRole:      useIamRole,
				AccessKeyId:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				SessionToken:    sessionToken,
			},
		},
	}
	p := NewS3StorageProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("s3: failed to initialize: %w", err)
	}
	return p, nil
}

// firstNonEmpty returns the first non-empty string among its arguments, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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
	return "aws_storage"
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
		// Disposition + safe-content-type pinning AT SIGNING TIME (Q-ST-STREAM B+C,
		// ST-H3 baked into the signed URL). These two response-header overrides are
		// part of the signature, so the browser is forced to honor them no matter
		// what was stored on the object:
		//
		//  (1) ResponseContentDisposition = attachment; filename="<sanitized>" — the
		//      object always DOWNLOADS as an attachment, never inline-renders (defeats
		//      stored-HTML/SVG XSS). nosniff is irrelevant across the S3 origin on a
		//      302 redirect; this disposition pin is the equivalent protection.
		//  (2) ResponseContentType = <server-derived safe content_type> — the browser
		//      receives the pinned, server-sniffed type rather than whatever MIME was
		//      persisted on the object (defeats MIME confusion).
		//
		// The caller threads the server-authoritative values via the existing proto
		// fields: req.Filename (field 7) for the attachment filename and
		// req.ContentType (field 6, today used only for upload at the PUT branch) for
		// the safe type. The mint happens against the already-authorized
		// att.StorageContainer/att.StorageKey (never a client-supplied key) AFTER the
		// metadata-row authz (ReadAttachmentByEntity) has passed.
		filename := sanitizeDownloadFilename(req.Filename, objectKey)
		getReq.ResponseContentDisposition = aws.String(
			fmt.Sprintf("attachment; filename=%q", filename),
		)
		if req.ContentType != "" {
			getReq.ResponseContentType = aws.String(req.ContentType)
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

// =============================================================================
// Streaming tier (StreamingStorageProvider) + capability discovery
// =============================================================================

// Compile-time assertions that S3 implements both optional sub-interfaces.
var (
	_ ports.StreamingStorageProvider  = (*S3StorageProvider)(nil)
	_ ports.StorageCapabilityProvider = (*S3StorageProvider)(nil)
)

// UploadStream streams body directly to S3 via PutObject. The AWS SDK accepts any
// io.Reader as the request Body and streams it natively to the service — no
// io.ReadAll, so a large object never lands fully in RAM. The proto req carries the
// container/key/content-type/metadata envelope; req.Content is ignored.
func (p *S3StorageProvider) UploadStream(ctx context.Context, req *pb.UploadObjectRequest, body io.Reader) (*pb.UploadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "S3 storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.ContainerName == "" || req.ObjectKey == "" {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}
	objectKey := strings.Trim(req.ObjectKey, "/")

	uploadCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   body, // streamed natively — io.Reader is consumed lazily by the SDK
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(req.ObjectKey))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	input.ContentType = aws.String(contentType)

	if len(req.Metadata) > 0 {
		input.Metadata = req.Metadata
	}
	if req.CacheControl != "" {
		input.CacheControl = aws.String(req.CacheControl)
	}
	if req.ContentDisposition != "" {
		input.ContentDisposition = aws.String(req.ContentDisposition)
	}
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

	result, err := p.client.PutObject(uploadCtx, input)
	if err != nil {
		return &pb.UploadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to upload: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeUploadFailed, "upload failed", err)
	}

	now := time.Now()
	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(bucketName, objectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_AWS,
		ContainerName: bucketName,
		ObjectKey:     objectKey,
		ContentType:   contentType,
		Etag:          aws.ToString(result.ETag),
		LastModified:  timestamppb.New(now),
		CreatedAt:     timestamppb.New(now),
		StorageClass:  string(input.StorageClass),
		IsEncrypted:   req.EnableEncryption,
		Metadata:      req.Metadata,
	}

	return &pb.UploadObjectResponse{
		Success:          true,
		Object:           storageObject,
		UploadDurationMs: time.Since(startTime).Milliseconds(),
		Message:          "upload successful",
	}, nil
}

// DownloadStream returns the S3 GetObjectOutput.Body (an io.ReadCloser) directly —
// the caller MUST Close it. No io.ReadAll, so the object streams through to the
// HTTP response without buffering. The metadata response carries object attributes
// but leaves Content nil (the bytes flow through the ReadCloser).
func (p *S3StorageProvider) DownloadStream(ctx context.Context, req *pb.DownloadObjectRequest) (io.ReadCloser, *pb.DownloadObjectResponse, error) {
	startTime := time.Now()

	if !p.enabled {
		return nil, &pb.DownloadObjectResponse{
			Success: false,
			Message: "S3 storage provider is not initialized",
		}, ports.NewStorageError(ports.StorageErrorCodeProviderError, "not initialized", nil)
	}

	if req.ContainerName == "" || req.ObjectKey == "" {
		return nil, &pb.DownloadObjectResponse{
			Success: false,
			Message: "container_name and object_key are required",
		}, ports.NewStorageError(ports.StorageErrorCodeInvalidPath, "missing required fields", nil)
	}

	bucketName := req.ContainerName
	if bucketName == "" {
		bucketName = p.bucketName
	}
	objectKey := strings.Trim(req.ObjectKey, "/")

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

	// NOTE: no per-call timeout here — the stream may outlive p.timeout while the
	// caller copies bytes. The parent ctx governs cancellation.
	result, err := p.client.GetObject(ctx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, &pb.DownloadObjectResponse{
				Success: false,
				Message: fmt.Sprintf("file not found: %s/%s", bucketName, objectKey),
			}, ports.NewStorageError(ports.StorageErrorCodeNotFound, "not found", err)
		}
		return nil, &pb.DownloadObjectResponse{
			Success: false,
			Message: fmt.Sprintf("failed to get object: %v", err),
		}, ports.NewStorageError(ports.StorageErrorCodeDownloadFailed, "download failed", err)
	}

	storageObject := &pb.StorageObject{
		Id:            storagecommon.GenerateObjectID(bucketName, objectKey),
		Provider:      pb.StorageProvider_STORAGE_PROVIDER_AWS,
		ContainerName: bucketName,
		ObjectKey:     objectKey,
		ContentType:   aws.ToString(result.ContentType),
		Etag:          aws.ToString(result.ETag),
		StorageClass:  string(result.StorageClass),
	}
	if result.ContentLength != nil {
		storageObject.Size = *result.ContentLength
	}
	if result.LastModified != nil {
		storageObject.LastModified = timestamppb.New(*result.LastModified)
	}
	if result.VersionId != nil {
		storageObject.VersionId = aws.ToString(result.VersionId)
	}

	resp := &pb.DownloadObjectResponse{
		Success:            true,
		Object:             storageObject,
		Content:            nil, // bytes flow through the ReadCloser, not the proto
		DownloadDurationMs: time.Since(startTime).Milliseconds(),
		Message:            "download stream opened",
	}
	return result.Body, resp, nil
}

// GetCapabilities returns the full S3 capability set. S3 is the primary cloud
// beneficiary of BOTH the stream tier and the presigned-direct tier.
func (p *S3StorageProvider) GetCapabilities() []ports.StorageCapability {
	return []ports.StorageCapability{
		ports.StorageCapabilityUpload,
		ports.StorageCapabilityDownload,
		ports.StorageCapabilityDelete,
		ports.StorageCapabilityStreaming,
		ports.StorageCapabilityPresignedUrls,
		ports.StorageCapabilityMetadata,
	}
}

// SupportsCapability reports whether S3 supports a given capability.
func (p *S3StorageProvider) SupportsCapability(capability ports.StorageCapability) bool {
	for _, c := range p.GetCapabilities() {
		if c == capability {
			return true
		}
	}
	return false
}

// sanitizeDownloadFilename derives a safe Content-Disposition filename. It prefers
// the server-supplied display name, falls back to the object key's basename, strips
// any path separators and control/quote characters, and defaults to "download".
func sanitizeDownloadFilename(name, objectKey string) string {
	if name == "" {
		name = filepath.Base(objectKey)
	}
	// Drop any directory components a caller may have left in.
	name = filepath.Base(name)
	// Remove characters that could break out of the quoted header value or inject
	// additional header directives.
	name = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20: // control chars (incl. CR/LF)
			return -1
		case r == '"' || r == '\\' || r == '/':
			return -1
		default:
			return r
		}
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "download"
	}
	return name
}
