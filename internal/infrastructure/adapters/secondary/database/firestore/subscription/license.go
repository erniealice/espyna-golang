//go:build firestore

package subscription

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "license", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore license repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreLicenseRepository(dbOps, collectionName), nil
	})
}

// FirestoreLicenseRepository implements license CRUD operations using Firestore
type FirestoreLicenseRepository struct {
	licensepb.UnimplementedLicenseDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreLicenseRepository creates a new Firestore license repository
func NewFirestoreLicenseRepository(dbOps interfaces.DatabaseOperation, collectionName string) licensepb.LicenseDomainServiceServer {
	if collectionName == "" {
		collectionName = "license" // default fallback
	}
	return &FirestoreLicenseRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateLicense creates a new license using common Firestore operations
func (r *FirestoreLicenseRepository) CreateLicense(ctx context.Context, req *licensepb.CreateLicenseRequest) (*licensepb.CreateLicenseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("license data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create license: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(result, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.CreateLicenseResponse{
		Data:    []*licensepb.License{convertedLicense},
		Success: true,
	}, nil
}

// ReadLicense retrieves a license using common Firestore operations
func (r *FirestoreLicenseRepository) ReadLicense(ctx context.Context, req *licensepb.ReadLicenseRequest) (*licensepb.ReadLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(result, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.ReadLicenseResponse{
		Data:    []*licensepb.License{convertedLicense},
		Success: true,
	}, nil
}

// UpdateLicense updates a license using common Firestore operations
func (r *FirestoreLicenseRepository) UpdateLicense(ctx context.Context, req *licensepb.UpdateLicenseRequest) (*licensepb.UpdateLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(result, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.UpdateLicenseResponse{
		Data:    []*licensepb.License{convertedLicense},
		Success: true,
	}, nil
}

// DeleteLicense deletes a license using common Firestore operations
func (r *FirestoreLicenseRepository) DeleteLicense(ctx context.Context, req *licensepb.DeleteLicenseRequest) (*licensepb.DeleteLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete license: %w", err)
	}

	return &licensepb.DeleteLicenseResponse{
		Success: true,
	}, nil
}

// ListLicenses lists licenses using common Firestore operations
func (r *FirestoreLicenseRepository) ListLicenses(ctx context.Context, req *licensepb.ListLicensesRequest) (*licensepb.ListLicensesResponse, error) {
	// Build ListParams from request
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list licenses: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	licenses, _ := operations.ConvertSliceToProtobuf(listResult.Data, func() *licensepb.License {
		return &licensepb.License{}
	})

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if licenses == nil {
		licenses = make([]*licensepb.License, 0)
	}

	return &licensepb.ListLicensesResponse{
		Data:    licenses,
		Success: true,
	}, nil
}

// GetLicenseListPageData retrieves licenses with pagination support
func (r *FirestoreLicenseRepository) GetLicenseListPageData(ctx context.Context, req *licensepb.GetLicenseListPageDataRequest) (*licensepb.GetLicenseListPageDataResponse, error) {
	// Build ListParams from request
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list licenses for page data: %w", err)
	}

	// Convert results to protobuf slice
	licenses, _ := operations.ConvertSliceToProtobuf(listResult.Data, func() *licensepb.License {
		return &licensepb.License{}
	})

	if licenses == nil {
		licenses = make([]*licensepb.License, 0)
	}

	return &licensepb.GetLicenseListPageDataResponse{
		LicenseList: licenses,
		Pagination:  listResult.Pagination,
		Success:     true,
	}, nil
}

// GetLicenseItemPageData retrieves a single license for item page display
func (r *FirestoreLicenseRepository) GetLicenseItemPageData(ctx context.Context, req *licensepb.GetLicenseItemPageDataRequest) (*licensepb.GetLicenseItemPageDataResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Convert result to protobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(result, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.GetLicenseItemPageDataResponse{
		License: convertedLicense,
		Success: true,
	}, nil
}

// AssignLicense assigns a license to an assignee
func (r *FirestoreLicenseRepository) AssignLicense(ctx context.Context, req *licensepb.AssignLicenseRequest) (*licensepb.AssignLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.AssigneeId == "" {
		return nil, fmt.Errorf("assignee ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.collectionName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update assignment fields
	now := time.Now()
	result["assignee_id"] = req.AssigneeId
	result["assignee_type"] = req.AssigneeType
	if req.AssigneeName != nil {
		result["assignee_name"] = *req.AssigneeName
	}
	result["assigned_by"] = req.AssignedBy
	result["date_assigned"] = now.UnixMilli()
	result["date_assigned_string"] = now.Format(time.RFC3339)
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE)
	result["date_modified"] = now.UnixMilli()
	result["date_modified_string"] = now.Format(time.RFC3339)

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.collectionName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license assignment: %w", err)
	}

	// Convert result to protobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(updatedResult, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.AssignLicenseResponse{
		License: convertedLicense,
		Success: true,
	}, nil
}

// RevokeLicenseAssignment revokes the assignment of a license
func (r *FirestoreLicenseRepository) RevokeLicenseAssignment(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) (*licensepb.RevokeLicenseAssignmentResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.collectionName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Clear assignment fields
	now := time.Now()
	delete(result, "assignee_id")
	delete(result, "assignee_type")
	delete(result, "assignee_name")
	delete(result, "assigned_by")
	delete(result, "date_assigned")
	delete(result, "date_assigned_string")
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_REVOKED)
	result["date_modified"] = now.UnixMilli()
	result["date_modified_string"] = now.Format(time.RFC3339)

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.collectionName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(updatedResult, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.RevokeLicenseAssignmentResponse{
		License: convertedLicense,
		Success: true,
	}, nil
}

// ReassignLicense reassigns a license to a new assignee
func (r *FirestoreLicenseRepository) ReassignLicense(ctx context.Context, req *licensepb.ReassignLicenseRequest) (*licensepb.ReassignLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.NewAssigneeId == "" {
		return nil, fmt.Errorf("new assignee ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.collectionName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update assignment fields
	now := time.Now()
	result["assignee_id"] = req.NewAssigneeId
	result["assignee_type"] = req.NewAssigneeType
	if req.NewAssigneeName != nil {
		result["assignee_name"] = *req.NewAssigneeName
	}
	result["assigned_by"] = req.PerformedBy
	result["date_assigned"] = now.UnixMilli()
	result["date_assigned_string"] = now.Format(time.RFC3339)
	result["date_modified"] = now.UnixMilli()
	result["date_modified_string"] = now.Format(time.RFC3339)

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.collectionName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(updatedResult, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.ReassignLicenseResponse{
		License: convertedLicense,
		Success: true,
	}, nil
}

// SuspendLicense suspends a license
func (r *FirestoreLicenseRepository) SuspendLicense(ctx context.Context, req *licensepb.SuspendLicenseRequest) (*licensepb.SuspendLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.collectionName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update status
	now := time.Now()
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED)
	result["date_modified"] = now.UnixMilli()
	result["date_modified_string"] = now.Format(time.RFC3339)

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.collectionName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(updatedResult, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.SuspendLicenseResponse{
		License: convertedLicense,
		Success: true,
	}, nil
}

// ReactivateLicense reactivates a suspended license
func (r *FirestoreLicenseRepository) ReactivateLicense(ctx context.Context, req *licensepb.ReactivateLicenseRequest) (*licensepb.ReactivateLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.collectionName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update status
	now := time.Now()
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE)
	result["date_modified"] = now.UnixMilli()
	result["date_modified_string"] = now.Format(time.RFC3339)

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.collectionName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(updatedResult, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensepb.ReactivateLicenseResponse{
		License: convertedLicense,
		Success: true,
	}, nil
}

// ValidateLicenseAccess validates if a license grants access
func (r *FirestoreLicenseRepository) ValidateLicenseAccess(ctx context.Context, req *licensepb.ValidateLicenseAccessRequest) (*licensepb.ValidateLicenseAccessResponse, error) {
	var result map[string]any
	var err error

	// Find by ID or license key
	if req.LicenseId != "" {
		result, err = r.dbOps.Read(ctx, r.collectionName, req.LicenseId)
	} else if req.LicenseKey != nil && *req.LicenseKey != "" {
		// Search by license key using filter
		listParams := &interfaces.ListParams{}
		listResult, listErr := r.dbOps.List(ctx, r.collectionName, listParams)
		if listErr != nil {
			return nil, fmt.Errorf("failed to search licenses: %w", listErr)
		}
		for _, item := range listResult.Data {
			if licenseKey, ok := item["license_key"].(string); ok && licenseKey == *req.LicenseKey {
				result = item
				break
			}
		}
		if result == nil {
			err = fmt.Errorf("license not found")
		}
	} else {
		return nil, fmt.Errorf("license ID or license key is required")
	}

	if err != nil {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: strPtr("License not found"),
			Success:           true,
		}, nil
	}

	// Convert to protobuf
	license := &licensepb.License{}
	convertedLicense, err := operations.ConvertMapToProtobuf(result, license)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	// Check if license is active
	if convertedLicense.Status != licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           convertedLicense,
			ValidationMessage: strPtr(fmt.Sprintf("License is not active, current status: %s", convertedLicense.Status.String())),
			Success:           true,
		}, nil
	}

	// Check assignee if specified
	if req.AssigneeId != nil && *req.AssigneeId != "" {
		if convertedLicense.AssigneeId == nil || *convertedLicense.AssigneeId != *req.AssigneeId {
			return &licensepb.ValidateLicenseAccessResponse{
				IsValid:           false,
				License:           convertedLicense,
				ValidationMessage: strPtr("License is not assigned to the specified assignee"),
				Success:           true,
			}, nil
		}
	}

	// Check validity dates
	now := time.Now().UnixMilli()
	if convertedLicense.DateValidFrom != nil && *convertedLicense.DateValidFrom > now {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           convertedLicense,
			ValidationMessage: strPtr("License is not yet valid"),
			Success:           true,
		}, nil
	}
	if convertedLicense.DateValidUntil != nil && *convertedLicense.DateValidUntil < now {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           convertedLicense,
			ValidationMessage: strPtr("License has expired"),
			Success:           true,
		}, nil
	}

	return &licensepb.ValidateLicenseAccessResponse{
		IsValid:           true,
		License:           convertedLicense,
		ValidationMessage: strPtr("License is valid"),
		Success:           true,
	}, nil
}

// CreateLicensesFromPlan creates multiple licenses from a plan
func (r *FirestoreLicenseRepository) CreateLicensesFromPlan(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) (*licensepb.CreateLicensesFromPlanResponse, error) {
	if req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

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
		seqNum := i + 1
		licenseKey := fmt.Sprintf("LIC-%s-%04d", req.SubscriptionId[:8], seqNum)

		data := map[string]any{
			"subscription_id":      req.SubscriptionId,
			"plan_id":              req.PlanId,
			"license_key":          licenseKey,
			"license_type":         int32(licenseType),
			"status":               int32(licensepb.LicenseStatus_LICENSE_STATUS_PENDING),
			"sequence_number":      seqNum,
			"date_created":         now.UnixMilli(),
			"date_created_string":  now.Format(time.RFC3339),
			"date_modified":        now.UnixMilli(),
			"date_modified_string": now.Format(time.RFC3339),
			"active":               true,
		}

		// Create document
		result, err := r.dbOps.Create(ctx, r.collectionName, data)
		if err != nil {
			return nil, fmt.Errorf("failed to create license %d: %w", i+1, err)
		}

		// Convert to protobuf
		license := &licensepb.License{}
		convertedLicense, err := operations.ConvertMapToProtobuf(result, license)
		if err != nil {
			return nil, fmt.Errorf("failed to convert license %d to protobuf: %w", i+1, err)
		}

		createdLicenses = append(createdLicenses, convertedLicense)
	}

	return &licensepb.CreateLicensesFromPlanResponse{
		Licenses:     createdLicenses,
		CreatedCount: int32(len(createdLicenses)),
		Success:      true,
	}, nil
}

// strPtr returns a pointer to a string (helper function)
func strPtr(s string) *string {
	return &s
}
