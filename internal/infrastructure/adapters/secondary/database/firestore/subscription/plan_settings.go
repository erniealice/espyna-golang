//go:build firestore

package subscription

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "plan_settings", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore plan_settings repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePlanSettingsRepository(dbOps, collectionName), nil
	})
}

// FirestorePlanSettingsRepository implements plan settings CRUD operations using Firestore
type FirestorePlanSettingsRepository struct {
	plansettingspb.UnimplementedPlanSettingsDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePlanSettingsRepository creates a new Firestore plan settings repository
func NewFirestorePlanSettingsRepository(dbOps interfaces.DatabaseOperation, collectionName string) plansettingspb.PlanSettingsDomainServiceServer {
	if collectionName == "" {
		collectionName = "plan_setting" // default fallback
	}
	return &FirestorePlanSettingsRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePlanSettings creates a new plan settings using common Firestore operations
func (r *FirestorePlanSettingsRepository) CreatePlanSettings(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) (*plansettingspb.CreatePlanSettingsResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan settings data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan settings: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	planSettings := &plansettingspb.PlanSettings{}
	convertedPlanSettings, err := operations.ConvertMapToProtobuf(result, planSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &plansettingspb.CreatePlanSettingsResponse{
		Data: []*plansettingspb.PlanSettings{convertedPlanSettings},
	}, nil
}

// ReadPlanSettings retrieves a plan settings using common Firestore operations
func (r *FirestorePlanSettingsRepository) ReadPlanSettings(ctx context.Context, req *plansettingspb.ReadPlanSettingsRequest) (*plansettingspb.ReadPlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan settings: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	planSettings := &plansettingspb.PlanSettings{}
	convertedPlanSettings, err := operations.ConvertMapToProtobuf(result, planSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &plansettingspb.ReadPlanSettingsResponse{
		Data: []*plansettingspb.PlanSettings{convertedPlanSettings},
	}, nil
}

// UpdatePlanSettings updates a plan settings using common Firestore operations
func (r *FirestorePlanSettingsRepository) UpdatePlanSettings(ctx context.Context, req *plansettingspb.UpdatePlanSettingsRequest) (*plansettingspb.UpdatePlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan settings: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	planSettings := &plansettingspb.PlanSettings{}
	convertedPlanSettings, err := operations.ConvertMapToProtobuf(result, planSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &plansettingspb.UpdatePlanSettingsResponse{
		Data: []*plansettingspb.PlanSettings{convertedPlanSettings},
	}, nil
}

// DeletePlanSettings deletes a plan settings using common Firestore operations
func (r *FirestorePlanSettingsRepository) DeletePlanSettings(ctx context.Context, req *plansettingspb.DeletePlanSettingsRequest) (*plansettingspb.DeletePlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete plan settings: %w", err)
	}

	return &plansettingspb.DeletePlanSettingsResponse{
		Success: true,
	}, nil
}

// ListPlanSettings lists plan settings using common Firestore operations
func (r *FirestorePlanSettingsRepository) ListPlanSettings(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) (*plansettingspb.ListPlanSettingsResponse, error) {
	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan settings: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	planSettings, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *plansettingspb.PlanSettings {
		return &plansettingspb.PlanSettings{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if planSettings == nil {
		planSettings = make([]*plansettingspb.PlanSettings, 0)
	}

	return &plansettingspb.ListPlanSettingsResponse{
		Data: planSettings,
	}, nil
}

// ListPlanSettingsByPlan lists plan settings by plan using common Firestore operations
func (r *FirestorePlanSettingsRepository) ListPlanSettingsByPlan(ctx context.Context, req *plansettingspb.ListPlanSettingsByPlanRequest) (*plansettingspb.ListPlanSettingsByPlanResponse, error) {
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// List documents using common operations with a filter
	listResult, err := r.dbOps.List(ctx, r.collectionName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan settings by plan: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	planSettings, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *plansettingspb.PlanSettings {
		return &plansettingspb.PlanSettings{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// CRITICAL FIX: Ensure we always return a non-nil slice for proper JSON marshaling
	// This guarantees the "data" field is always included in the JSON response
	if planSettings == nil {
		planSettings = make([]*plansettingspb.PlanSettings, 0)
	}

	return &plansettingspb.ListPlanSettingsByPlanResponse{
		Data: planSettings,
	}, nil
}
