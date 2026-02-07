//go:build firestore

package workflow

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "activity", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore activity repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreActivityRepository(dbOps, collectionName), nil
	})
}

// FirestoreActivityRepository implements activity CRUD operations using Firestore
type FirestoreActivityRepository struct {
	activitypb.UnimplementedActivityDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreActivityRepository creates a new Firestore activity repository
func NewFirestoreActivityRepository(dbOps interfaces.DatabaseOperation, collectionName string) activitypb.ActivityDomainServiceServer {
	if collectionName == "" {
		collectionName = "activity" // default fallback (singular to match database.go)
	}
	return &FirestoreActivityRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateActivity creates a new activity using common Firestore operations
func (r *FirestoreActivityRepository) CreateActivity(ctx context.Context, req *activitypb.CreateActivityRequest) (*activitypb.CreateActivityResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	activity := &activitypb.Activity{}
	convertedActivity, err := operations.ConvertMapToProtobuf(result, activity)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &activitypb.CreateActivityResponse{
		Data:    []*activitypb.Activity{convertedActivity},
		Success: true,
	}, nil
}

// ReadActivity retrieves an activity using common Firestore operations
func (r *FirestoreActivityRepository) ReadActivity(ctx context.Context, req *activitypb.ReadActivityRequest) (*activitypb.ReadActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	activity := &activitypb.Activity{}
	convertedActivity, err := operations.ConvertMapToProtobuf(result, activity)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &activitypb.ReadActivityResponse{
		Data:    []*activitypb.Activity{convertedActivity},
		Success: true,
	}, nil
}

// UpdateActivity updates an activity using common Firestore operations
func (r *FirestoreActivityRepository) UpdateActivity(ctx context.Context, req *activitypb.UpdateActivityRequest) (*activitypb.UpdateActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update activity: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	activity := &activitypb.Activity{}
	convertedActivity, err := operations.ConvertMapToProtobuf(result, activity)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &activitypb.UpdateActivityResponse{
		Data:    []*activitypb.Activity{convertedActivity},
		Success: true,
	}, nil
}

// DeleteActivity deletes an activity using common Firestore operations
func (r *FirestoreActivityRepository) DeleteActivity(ctx context.Context, req *activitypb.DeleteActivityRequest) (*activitypb.DeleteActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// Delete document using common operations
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete activity: %w", err)
	}

	return &activitypb.DeleteActivityResponse{
		Success: true,
	}, nil
}

// ListActivities retrieves activities using common Firestore operations
func (r *FirestoreActivityRepository) ListActivities(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
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
		return nil, fmt.Errorf("failed to list activities: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	activities, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *activitypb.Activity {
		return &activitypb.Activity{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if activities == nil {
		activities = make([]*activitypb.Activity, 0)
	}

	return &activitypb.ListActivitiesResponse{
		Data:    activities,
		Success: true,
	}, nil
}

// GetActivityListPageData retrieves activities with pagination using common Firestore operations
func (r *FirestoreActivityRepository) GetActivityListPageData(ctx context.Context, req *activitypb.GetActivityListPageDataRequest) (*activitypb.GetActivityListPageDataResponse, error) {
	// For now, implement basic list functionality
	// TODO: Implement full pagination, filtering, sorting, and search using common operations
	listReq := &activitypb.ListActivitiesRequest{}
	listResp, err := r.ListActivities(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity list page data: %w", err)
	}

	return &activitypb.GetActivityListPageDataResponse{
		ActivityList: listResp.Data,
		Success:      true,
	}, nil
}

// GetActivityItemPageData retrieves a single activity with enhanced data
func (r *FirestoreActivityRepository) GetActivityItemPageData(ctx context.Context, req *activitypb.GetActivityItemPageDataRequest) (*activitypb.GetActivityItemPageDataResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	readReq := &activitypb.ReadActivityRequest{
		Data: &activitypb.Activity{Id: req.ActivityId},
	}
	readResp, err := r.ReadActivity(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity not found")
	}

	return &activitypb.GetActivityItemPageDataResponse{
		Activity: readResp.Data[0],
		Success:  true,
	}, nil
}
