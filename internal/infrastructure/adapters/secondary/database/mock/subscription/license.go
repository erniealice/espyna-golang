//go:build mock_db

package subscription

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "license", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockLicenseRepository(businessType), nil
	})
}

// MockLicenseRepository implements license.LicenseRepository using stateful mock data
type MockLicenseRepository struct {
	licensepb.UnimplementedLicenseDomainServiceServer
	businessType string
	licenses     map[string]*licensepb.License // Persistent in-memory store
	mutex        sync.RWMutex                  // Thread-safe concurrent access
	initialized  bool                          // Prevent double initialization
	processor    *listdata.ListDataProcessor   // List data processing utilities
}

// LicenseRepositoryOption allows configuration of repository behavior
type LicenseRepositoryOption func(*MockLicenseRepository)

// WithLicenseTestOptimizations enables test-specific optimizations
func WithLicenseTestOptimizations(enabled bool) LicenseRepositoryOption {
	return func(r *MockLicenseRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockLicenseRepository creates a new mock license repository
func NewMockLicenseRepository(businessType string, options ...LicenseRepositoryOption) licensepb.LicenseDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockLicenseRepository{
		businessType: businessType,
		licenses:     make(map[string]*licensepb.License),
		processor:    listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if err := repo.loadInitialData(); err != nil {
		// Log error but don't fail - allows graceful degradation
		fmt.Printf("Warning: Failed to load initial license mock data: %v\n", err)
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockLicenseRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawLicenses, err := datamock.LoadBusinessTypeModule(r.businessType, "license")
	if err != nil {
		// License data may not exist yet, this is not a critical error
		r.initialized = true
		return nil
	}

	// Convert and store each license
	for _, rawLicense := range rawLicenses {
		if license, err := r.mapToProtobufLicense(rawLicense); err == nil {
			r.licenses[license.Id] = license
		}
	}

	r.initialized = true
	return nil
}

// CreateLicense creates a new license with stateful storage
func (r *MockLicenseRepository) CreateLicense(ctx context.Context, req *licensepb.CreateLicenseRequest) (*licensepb.CreateLicenseResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create license request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("license data is required")
	}
	if req.Data.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription_id is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	licenseID := req.Data.Id
	if licenseID == "" {
		licenseID = fmt.Sprintf("license-%d-%d", now.UnixNano(), len(r.licenses))
	}

	// Generate license key if not provided
	licenseKey := req.Data.LicenseKey
	if licenseKey == "" {
		licenseKey = fmt.Sprintf("LIC-%d", now.UnixNano()%1000000)
	}

	// Create new license with proper timestamps and defaults
	newLicense := &licensepb.License{
		Id:                  licenseID,
		SubscriptionId:      req.Data.SubscriptionId,
		PlanId:              req.Data.PlanId,
		LicenseKey:          licenseKey,
		ExternalKey:         req.Data.ExternalKey,
		LicenseType:         req.Data.LicenseType,
		Status:              req.Data.Status,
		DateValidFrom:       req.Data.DateValidFrom,
		DateValidFromString: req.Data.DateValidFromString,
		DateValidUntil:      req.Data.DateValidUntil,
		DateValidUntilString: req.Data.DateValidUntilString,
		AssigneeId:          req.Data.AssigneeId,
		AssigneeType:        req.Data.AssigneeType,
		AssigneeName:        req.Data.AssigneeName,
		AssignedBy:          req.Data.AssignedBy,
		DateAssigned:        req.Data.DateAssigned,
		DateAssignedString:  req.Data.DateAssignedString,
		SequenceNumber:      req.Data.SequenceNumber,
		Metadata:            req.Data.Metadata,
		Entitlements:        req.Data.Entitlements,
		DateCreated:         &[]int64{now.UnixMilli()}[0],
		DateCreatedString:   &[]string{now.Format(time.RFC3339)}[0],
		DateModified:        &[]int64{now.UnixMilli()}[0],
		DateModifiedString:  &[]string{now.Format(time.RFC3339)}[0],
		Active:              true, // Default to active
	}

	// Set default status if not provided
	if newLicense.Status == licensepb.LicenseStatus_LICENSE_STATUS_UNSPECIFIED {
		newLicense.Status = licensepb.LicenseStatus_LICENSE_STATUS_PENDING
	}

	// Store in persistent map
	r.licenses[licenseID] = newLicense

	return &licensepb.CreateLicenseResponse{
		Data:    []*licensepb.License{newLicense},
		Success: true,
	}, nil
}

// ReadLicense retrieves a license by ID from stateful storage
func (r *MockLicenseRepository) ReadLicense(ctx context.Context, req *licensepb.ReadLicenseRequest) (*licensepb.ReadLicenseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read license request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated licenses)
	if license, exists := r.licenses[req.Data.Id]; exists {
		return &licensepb.ReadLicenseResponse{
			Data:    []*licensepb.License{license},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("license with ID '%s' not found", req.Data.Id)
}

// UpdateLicense updates an existing license in stateful storage
func (r *MockLicenseRepository) UpdateLicense(ctx context.Context, req *licensepb.UpdateLicenseRequest) (*licensepb.UpdateLicenseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update license request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify license exists
	existingLicense, exists := r.licenses[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedLicense := &licensepb.License{
		Id:                   req.Data.Id,
		SubscriptionId:       req.Data.SubscriptionId,
		PlanId:               req.Data.PlanId,
		LicenseKey:           req.Data.LicenseKey,
		ExternalKey:          req.Data.ExternalKey,
		LicenseType:          req.Data.LicenseType,
		Status:               req.Data.Status,
		DateValidFrom:        req.Data.DateValidFrom,
		DateValidFromString:  req.Data.DateValidFromString,
		DateValidUntil:       req.Data.DateValidUntil,
		DateValidUntilString: req.Data.DateValidUntilString,
		AssigneeId:           req.Data.AssigneeId,
		AssigneeType:         req.Data.AssigneeType,
		AssigneeName:         req.Data.AssigneeName,
		AssignedBy:           req.Data.AssignedBy,
		DateAssigned:         req.Data.DateAssigned,
		DateAssignedString:   req.Data.DateAssignedString,
		SequenceNumber:       req.Data.SequenceNumber,
		Metadata:             req.Data.Metadata,
		Entitlements:         req.Data.Entitlements,
		DateCreated:          existingLicense.DateCreated,       // Preserve original
		DateCreatedString:    existingLicense.DateCreatedString, // Preserve original
		DateModified:         &[]int64{now.UnixMilli()}[0],
		DateModifiedString:   &[]string{now.Format(time.RFC3339)}[0],
		Active:               req.Data.Active,
	}

	// Update in persistent store
	r.licenses[req.Data.Id] = updatedLicense

	return &licensepb.UpdateLicenseResponse{
		Data:    []*licensepb.License{updatedLicense},
		Success: true,
	}, nil
}

// DeleteLicense deletes a license from stateful storage
func (r *MockLicenseRepository) DeleteLicense(ctx context.Context, req *licensepb.DeleteLicenseRequest) (*licensepb.DeleteLicenseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete license request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify license exists before deletion
	if _, exists := r.licenses[req.Data.Id]; !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.licenses, req.Data.Id)

	return &licensepb.DeleteLicenseResponse{
		Success: true,
	}, nil
}

// ListLicenses retrieves all licenses from stateful storage
func (r *MockLicenseRepository) ListLicenses(ctx context.Context, req *licensepb.ListLicensesRequest) (*licensepb.ListLicensesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of licenses
	licenses := make([]*licensepb.License, 0, len(r.licenses))
	for _, license := range r.licenses {
		licenses = append(licenses, license)
	}

	return &licensepb.ListLicensesResponse{
		Data:    licenses,
		Success: true,
	}, nil
}

// GetLicenseListPageData retrieves licenses with advanced filtering, sorting, searching, and pagination
func (r *MockLicenseRepository) GetLicenseListPageData(
	ctx context.Context,
	req *licensepb.GetLicenseListPageDataRequest,
) (*licensepb.GetLicenseListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get license list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of licenses
	licenses := make([]*licensepb.License, 0, len(r.licenses))
	for _, license := range r.licenses {
		licenses = append(licenses, license)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		licenses,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process license list data: %w", err)
	}

	// Convert processed items back to license protobuf format
	processedLicenses := make([]*licensepb.License, len(result.Items))
	for i, item := range result.Items {
		if license, ok := item.(*licensepb.License); ok {
			processedLicenses[i] = license
		} else {
			return nil, fmt.Errorf("failed to convert item to license type")
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &licensepb.GetLicenseListPageDataResponse{
		LicenseList:   processedLicenses,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetLicenseItemPageData retrieves a single license with enhanced item page data
func (r *MockLicenseRepository) GetLicenseItemPageData(
	ctx context.Context,
	req *licensepb.GetLicenseItemPageDataRequest,
) (*licensepb.GetLicenseItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get license item page data request is required")
	}
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	license, exists := r.licenses[req.LicenseId]
	if !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.LicenseId)
	}

	return &licensepb.GetLicenseItemPageDataResponse{
		License: license,
		Success: true,
	}, nil
}

// AssignLicense assigns a license to an assignee
func (r *MockLicenseRepository) AssignLicense(ctx context.Context, req *licensepb.AssignLicenseRequest) (*licensepb.AssignLicenseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("assign license request is required")
	}
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.AssigneeId == "" {
		return nil, fmt.Errorf("assignee ID is required")
	}
	if req.AssigneeType == "" {
		return nil, fmt.Errorf("assignee type is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	license, exists := r.licenses[req.LicenseId]
	if !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.LicenseId)
	}

	now := time.Now()
	assigneeId := req.AssigneeId
	assigneeType := req.AssigneeType
	assigneeName := req.AssigneeName
	assignedBy := req.AssignedBy
	dateAssigned := now.UnixMilli()
	dateAssignedStr := now.Format(time.RFC3339)

	license.AssigneeId = &assigneeId
	license.AssigneeType = &assigneeType
	license.AssigneeName = assigneeName
	license.AssignedBy = &assignedBy
	license.DateAssigned = &dateAssigned
	license.DateAssignedString = &dateAssignedStr
	license.Status = licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return &licensepb.AssignLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// RevokeLicenseAssignment revokes the assignment of a license
func (r *MockLicenseRepository) RevokeLicenseAssignment(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) (*licensepb.RevokeLicenseAssignmentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("revoke license assignment request is required")
	}
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	license, exists := r.licenses[req.LicenseId]
	if !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.LicenseId)
	}

	now := time.Now()
	license.AssigneeId = nil
	license.AssigneeType = nil
	license.AssigneeName = nil
	license.AssignedBy = nil
	license.DateAssigned = nil
	license.DateAssignedString = nil
	license.Status = licensepb.LicenseStatus_LICENSE_STATUS_REVOKED
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return &licensepb.RevokeLicenseAssignmentResponse{
		License: license,
		Success: true,
	}, nil
}

// ReassignLicense reassigns a license to a new assignee
func (r *MockLicenseRepository) ReassignLicense(ctx context.Context, req *licensepb.ReassignLicenseRequest) (*licensepb.ReassignLicenseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("reassign license request is required")
	}
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.NewAssigneeId == "" {
		return nil, fmt.Errorf("new assignee ID is required")
	}
	if req.NewAssigneeType == "" {
		return nil, fmt.Errorf("new assignee type is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	license, exists := r.licenses[req.LicenseId]
	if !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.LicenseId)
	}

	now := time.Now()
	newAssigneeId := req.NewAssigneeId
	newAssigneeType := req.NewAssigneeType
	newAssigneeName := req.NewAssigneeName
	performedBy := req.PerformedBy
	dateAssigned := now.UnixMilli()
	dateAssignedStr := now.Format(time.RFC3339)

	license.AssigneeId = &newAssigneeId
	license.AssigneeType = &newAssigneeType
	license.AssigneeName = newAssigneeName
	license.AssignedBy = &performedBy
	license.DateAssigned = &dateAssigned
	license.DateAssignedString = &dateAssignedStr
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return &licensepb.ReassignLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// SuspendLicense suspends a license
func (r *MockLicenseRepository) SuspendLicense(ctx context.Context, req *licensepb.SuspendLicenseRequest) (*licensepb.SuspendLicenseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("suspend license request is required")
	}
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	license, exists := r.licenses[req.LicenseId]
	if !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.LicenseId)
	}

	now := time.Now()
	license.Status = licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return &licensepb.SuspendLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// ReactivateLicense reactivates a suspended license
func (r *MockLicenseRepository) ReactivateLicense(ctx context.Context, req *licensepb.ReactivateLicenseRequest) (*licensepb.ReactivateLicenseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("reactivate license request is required")
	}
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	license, exists := r.licenses[req.LicenseId]
	if !exists {
		return nil, fmt.Errorf("license with ID '%s' not found", req.LicenseId)
	}

	now := time.Now()
	license.Status = licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return &licensepb.ReactivateLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// ValidateLicenseAccess validates if a license grants access
func (r *MockLicenseRepository) ValidateLicenseAccess(ctx context.Context, req *licensepb.ValidateLicenseAccessRequest) (*licensepb.ValidateLicenseAccessResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("validate license access request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var license *licensepb.License
	var found bool

	// Find by ID or license key
	if req.LicenseId != "" {
		license, found = r.licenses[req.LicenseId]
	} else if req.LicenseKey != nil && *req.LicenseKey != "" {
		for _, l := range r.licenses {
			if l.LicenseKey == *req.LicenseKey {
				license = l
				found = true
				break
			}
		}
	}

	if !found {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: strPtr("License not found"),
			Success:           true,
		}, nil
	}

	// Check if license is active
	if license.Status != licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           license,
			ValidationMessage: strPtr(fmt.Sprintf("License is not active, current status: %s", license.Status.String())),
			Success:           true,
		}, nil
	}

	// Check assignee if specified
	if req.AssigneeId != nil && *req.AssigneeId != "" {
		if license.AssigneeId == nil || *license.AssigneeId != *req.AssigneeId {
			return &licensepb.ValidateLicenseAccessResponse{
				IsValid:           false,
				License:           license,
				ValidationMessage: strPtr("License is not assigned to the specified assignee"),
				Success:           true,
			}, nil
		}
	}

	// Check validity dates
	now := time.Now().UnixMilli()
	if license.DateValidFrom != nil && *license.DateValidFrom > now {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           license,
			ValidationMessage: strPtr("License is not yet valid"),
			Success:           true,
		}, nil
	}
	if license.DateValidUntil != nil && *license.DateValidUntil < now {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           license,
			ValidationMessage: strPtr("License has expired"),
			Success:           true,
		}, nil
	}

	return &licensepb.ValidateLicenseAccessResponse{
		IsValid:           true,
		License:           license,
		ValidationMessage: strPtr("License is valid"),
		Success:           true,
	}, nil
}

// CreateLicensesFromPlan creates multiple licenses from a plan
func (r *MockLicenseRepository) CreateLicensesFromPlan(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) (*licensepb.CreateLicensesFromPlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create licenses from plan request is required")
	}
	if req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	createdLicenses := make([]*licensepb.License, 0, req.Quantity)

	// Determine license type
	licenseType := licensepb.LicenseType_LICENSE_TYPE_USER
	if req.DefaultLicenseType != nil && *req.DefaultLicenseType != "" {
		switch *req.DefaultLicenseType {
		case "device":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_DEVICE
		case "tenant":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_TENANT
		case "floating":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_FLOATING
		}
	}

	for i := int32(0); i < req.Quantity; i++ {
		licenseID := fmt.Sprintf("license-%d-%d", now.UnixNano(), len(r.licenses)+int(i))
		licenseKey := fmt.Sprintf("LIC-%s-%04d", req.SubscriptionId[:8], i+1)
		seqNum := i + 1

		newLicense := &licensepb.License{
			Id:                 licenseID,
			SubscriptionId:     req.SubscriptionId,
			PlanId:             req.PlanId,
			LicenseKey:         licenseKey,
			LicenseType:        licenseType,
			Status:             licensepb.LicenseStatus_LICENSE_STATUS_PENDING,
			SequenceNumber:     &seqNum,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
			DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
			DateModified:       &[]int64{now.UnixMilli()}[0],
			DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
			Active:             true,
		}

		r.licenses[licenseID] = newLicense
		createdLicenses = append(createdLicenses, newLicense)
	}

	return &licensepb.CreateLicensesFromPlanResponse{
		Licenses:     createdLicenses,
		CreatedCount: int32(len(createdLicenses)),
		Success:      true,
	}, nil
}

// mapToProtobufLicense converts raw mock data to protobuf License
func (r *MockLicenseRepository) mapToProtobufLicense(rawLicense map[string]any) (*licensepb.License, error) {
	license := &licensepb.License{}

	// Map required fields
	if id, ok := rawLicense["id"].(string); ok {
		license.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if subscriptionId, ok := rawLicense["subscriptionId"].(string); ok {
		license.SubscriptionId = subscriptionId
	}

	if planId, ok := rawLicense["planId"].(string); ok {
		license.PlanId = planId
	}

	if licenseKey, ok := rawLicense["licenseKey"].(string); ok {
		license.LicenseKey = licenseKey
	}

	// Handle optional string fields
	if externalKey, ok := rawLicense["externalKey"].(string); ok {
		license.ExternalKey = &externalKey
	}

	if assigneeId, ok := rawLicense["assigneeId"].(string); ok {
		license.AssigneeId = &assigneeId
	}

	if assigneeType, ok := rawLicense["assigneeType"].(string); ok {
		license.AssigneeType = &assigneeType
	}

	if assigneeName, ok := rawLicense["assigneeName"].(string); ok {
		license.AssigneeName = &assigneeName
	}

	// Handle date fields
	if dateCreated, ok := rawLicense["dateCreated"].(string); ok {
		license.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			license.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawLicense["dateModified"].(string); ok {
		license.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			license.DateModified = &timestamp
		}
	}

	if active, ok := rawLicense["active"].(bool); ok {
		license.Active = active
	}

	return license, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockLicenseRepository) parseTimestamp(timestampStr string) (int64, error) {
	// Try parsing as Unix timestamp first
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	// Try parsing as RFC3339 format
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try parsing as other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}

// NewLicenseRepository creates a new mock license repository (registry constructor)
func NewLicenseRepository(data map[string]*licensepb.License) licensepb.LicenseDomainServiceServer {
	repo := &MockLicenseRepository{
		businessType: "education", // Default business type
		licenses:     data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.licenses = make(map[string]*licensepb.License)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
