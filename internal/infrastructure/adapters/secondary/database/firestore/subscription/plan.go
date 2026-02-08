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
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "plan", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore plan repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePlanRepository(dbOps, collectionName), nil
	})
}

// FirestorePlanRepository implements plan CRUD operations using Firestore
type FirestorePlanRepository struct {
	planpb.UnimplementedPlanDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePlanRepository creates a new Firestore plan repository
func NewFirestorePlanRepository(dbOps interfaces.DatabaseOperation, collectionName string) planpb.PlanDomainServiceServer {
	if collectionName == "" {
		collectionName = "plan" // default fallback
	}
	return &FirestorePlanRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePlan creates a new plan using common Firestore operations
func (r *FirestorePlanRepository) CreatePlan(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Remove nested objects before storing to Firestore - only store references
	// Handle plan_locations if they exist
	if _, exists := data["plan_locations"]; exists {
		delete(data, "plan_locations")
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	plan := &planpb.Plan{}
	convertedPlan, err := operations.ConvertMapToProtobuf(result, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &planpb.CreatePlanResponse{
		Data: []*planpb.Plan{convertedPlan},
	}, nil
}

// ReadPlan retrieves a plan using common Firestore operations
func (r *FirestorePlanRepository) ReadPlan(ctx context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan data and ID are required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, *req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	plan := &planpb.Plan{}
	convertedPlan, err := operations.ConvertMapToProtobuf(result, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &planpb.ReadPlanResponse{
		Data: []*planpb.Plan{convertedPlan},
	}, nil
}

// UpdatePlan updates a plan using common Firestore operations
func (r *FirestorePlanRepository) UpdatePlan(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan data is required")
	}
	if req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Remove nested objects before storing to Firestore - only store references
	// Handle plan_locations if they exist
	if _, exists := data["plan_locations"]; exists {
		delete(data, "plan_locations")
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, *req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	plan := &planpb.Plan{}
	convertedPlan, err := operations.ConvertMapToProtobuf(result, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &planpb.UpdatePlanResponse{
		Data: []*planpb.Plan{convertedPlan},
	}, nil
}

// DeletePlan deletes a plan using common Firestore operations
func (r *FirestorePlanRepository) DeletePlan(ctx context.Context, req *planpb.DeletePlanRequest) (*planpb.DeletePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan data is required")
	}
	if req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, *req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete plan: %w", err)
	}

	return &planpb.DeletePlanResponse{
		Success: true,
	}, nil
}

// ListPlans lists plans using common Firestore operations
func (r *FirestorePlanRepository) ListPlans(ctx context.Context, req *planpb.ListPlansRequest) (*planpb.ListPlansResponse, error) {
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
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	plans, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *planpb.Plan {
		return &planpb.Plan{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if plans == nil {
		plans = make([]*planpb.Plan, 0)
	}

	return &planpb.ListPlansResponse{
		Data: plans,
	}, nil
}
