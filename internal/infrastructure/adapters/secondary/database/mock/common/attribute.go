//go:build mock_db

package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "attribute", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockAttributeRepository(businessType), nil
	})
}

// MockAttributeRepository implements attributepb.AttributeServiceServer using stateful mock data
type MockAttributeRepository struct {
	attributepb.UnimplementedAttributeDomainServiceServer
	businessType string
	attributes   map[string]*attributepb.Attribute // Persistent in-memory store
	mutex        sync.RWMutex                      // Thread-safe concurrent access
	initialized  bool                              // Prevent double initialization
}

// AttributeRepositoryOption allows configuration of repository behavior
type AttributeRepositoryOption func(*MockAttributeRepository)

// WithAttributeTestOptimizations enables test-specific optimizations
func WithAttributeTestOptimizations(enabled bool) AttributeRepositoryOption {
	return func(r *MockAttributeRepository) {
		// Test optimizations placeholder
	}
}

// NewMockAttributeRepository creates a new mock attribute repository
func NewMockAttributeRepository(businessType string, options ...AttributeRepositoryOption) attributepb.AttributeDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockAttributeRepository{
		businessType: businessType,
		attributes:   make(map[string]*attributepb.Attribute),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if !repo.initialized {
				repo.initializeMockData()
		repo.initialized = true
	}

	return repo
}

// InitializeMockData loads mock data from copya package (public interface method)
func (r *MockAttributeRepository) InitializeMockData() {
	r.initializeMockData()
}

// initializeMockData loads mock data from copya package (internal implementation)
func (r *MockAttributeRepository) initializeMockData() {
	// Use datamock.LoadBusinessTypeModule instead of direct file reading
	rawAttributes, err := datamock.LoadBusinessTypeModule(r.businessType, "attribute")
	if err != nil {
		// Silently fail and use empty dataset if loading fails
		return
	}

	// Convert raw data to protobuf models
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, rawAttr := range rawAttributes {
		if attr, err := r.mapToProtobufAttribute(rawAttr); err == nil {
			if attr.Id != "" {
				r.attributes[attr.Id] = attr
			}
		}
	}
}

// mapToProtobufAttribute converts raw mock data to protobuf Attribute
func (r *MockAttributeRepository) mapToProtobufAttribute(rawAttr map[string]any) (*attributepb.Attribute, error) {
	attr := &attributepb.Attribute{}

	// Map required fields
	if id, ok := rawAttr["id"].(string); ok {
		attr.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawAttr["name"].(string); ok {
		attr.Name = name
	}

	// Map active status
	if active, ok := rawAttr["active"].(bool); ok {
		attr.Active = active
	} else {
		attr.Active = true // Default to active
	}

	return attr, nil
}

// CreateAttribute creates a new attribute
func (r *MockAttributeRepository) CreateAttribute(ctx context.Context, req *attributepb.CreateAttributeRequest) (*attributepb.CreateAttributeResponse, error) {
	if req == nil || req.Data == nil {
		return &attributepb.CreateAttributeResponse{
			Success: false,
			Error:   &attributepb.Error{Code: "INVALID_REQUEST", Message: "Request data is required"},
		}, nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate ID if not provided
	if req.Data.Id == "" {
		req.Data.Id = fmt.Sprintf("attr_%d", time.Now().UnixNano())
	}

	// Set timestamps
	now := time.Now()
	nowUnix := now.Unix()
	nowString := now.Format(time.RFC3339)
	req.Data.DateCreated = &nowUnix
	req.Data.DateCreatedString = &nowString
	req.Data.DateModified = &nowUnix
	req.Data.DateModifiedString = &nowString
	req.Data.Active = true

	// Store in memory
	r.attributes[req.Data.Id] = req.Data

	return &attributepb.CreateAttributeResponse{
		Data:    []*attributepb.Attribute{req.Data},
		Success: true,
	}, nil
}

// ReadAttribute retrieves an attribute by ID
func (r *MockAttributeRepository) ReadAttribute(ctx context.Context, req *attributepb.ReadAttributeRequest) (*attributepb.ReadAttributeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return &attributepb.ReadAttributeResponse{
			Success: false,
			Error:   &attributepb.Error{Code: "INVALID_REQUEST", Message: "Attribute ID is required"},
		}, nil
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	attribute, exists := r.attributes[req.Data.Id]
	if !exists {
		return &attributepb.ReadAttributeResponse{
			Success: false,
			Error:   &attributepb.Error{Code: "NOT_FOUND", Message: fmt.Sprintf("Attribute with ID '%s' not found", req.Data.Id)},
		}, nil
	}

	return &attributepb.ReadAttributeResponse{
		Data:    []*attributepb.Attribute{attribute},
		Success: true,
	}, nil
}

// UpdateAttribute updates an existing attribute
func (r *MockAttributeRepository) UpdateAttribute(ctx context.Context, req *attributepb.UpdateAttributeRequest) (*attributepb.UpdateAttributeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return &attributepb.UpdateAttributeResponse{
			Success: false,
			Error:   &attributepb.Error{Code: "INVALID_REQUEST", Message: "Attribute ID is required"},
		}, nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, exists := r.attributes[req.Data.Id]
	if !exists {
		return &attributepb.UpdateAttributeResponse{
			Success: false,
			Error:   &attributepb.Error{Code: "NOT_FOUND", Message: fmt.Sprintf("Attribute with ID '%s' not found", req.Data.Id)},
		}, nil
	}

	// Update timestamps
	now := time.Now()
	nowUnix := now.Unix()
	nowString := now.Format(time.RFC3339)
	req.Data.DateModified = &nowUnix
	req.Data.DateModifiedString = &nowString

	// Preserve creation timestamps
	req.Data.DateCreated = existing.DateCreated
	req.Data.DateCreatedString = existing.DateCreatedString

	// Update in memory
	r.attributes[req.Data.Id] = req.Data

	return &attributepb.UpdateAttributeResponse{
		Data:    []*attributepb.Attribute{req.Data},
		Success: true,
	}, nil
}

// DeleteAttribute deletes an attribute
func (r *MockAttributeRepository) DeleteAttribute(ctx context.Context, req *attributepb.DeleteAttributeRequest) (*attributepb.DeleteAttributeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return &attributepb.DeleteAttributeResponse{
			Success: false,
			Error:   &attributepb.Error{Code: "INVALID_REQUEST", Message: "Attribute ID is required"},
		}, nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	attribute, exists := r.attributes[req.Data.Id]
	if !exists {
		return &attributepb.DeleteAttributeResponse{
			Success: false,
			Error:   &attributepb.Error{Code: "NOT_FOUND", Message: fmt.Sprintf("Attribute with ID '%s' not found", req.Data.Id)},
		}, nil
	}

	// Perform soft delete
	attribute.Active = false
	now := time.Now()
	nowUnix := now.Unix()
	nowString := now.Format(time.RFC3339)
	attribute.DateModified = &nowUnix
	attribute.DateModifiedString = &nowString

	return &attributepb.DeleteAttributeResponse{
		Data:    []*attributepb.Attribute{attribute},
		Success: true,
	}, nil
}

// ListAttributes lists all attributes
func (r *MockAttributeRepository) ListAttributes(ctx context.Context, req *attributepb.ListAttributesRequest) (*attributepb.ListAttributesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var attributes []*attributepb.Attribute
	for _, attr := range r.attributes {
		if attr.Active { // Only return active attributes
			attributes = append(attributes, attr)
		}
	}

	// Apply limit if specified via pagination
	if req != nil && req.Pagination != nil && req.Pagination.Limit > 0 {
		limit := int(req.Pagination.Limit)
		if len(attributes) > limit {
			attributes = attributes[:limit]
		}
	}

	return &attributepb.ListAttributesResponse{
		Data:    attributes,
		Success: true,
	}, nil
}

// NewAttributeRepository creates a new mock attribute repository (legacy constructor)
func NewAttributeRepository(businessType string) attributepb.AttributeDomainServiceServer {
	return NewMockAttributeRepository(businessType)
}
