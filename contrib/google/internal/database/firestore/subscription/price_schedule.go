package subscription

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firestoreCore "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", entityid.PriceSchedule, func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore price_schedule repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePriceScheduleRepository(dbOps, collectionName), nil
	})
}

// FirestorePriceScheduleRepository implements price schedule CRUD operations using Firestore
type FirestorePriceScheduleRepository struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePriceScheduleRepository creates a new Firestore price schedule repository
func NewFirestorePriceScheduleRepository(dbOps interfaces.DatabaseOperation, collectionName string) priceschedulepb.PriceScheduleDomainServiceServer {
	if collectionName == "" {
		collectionName = "price_schedule" // default fallback
	}
	return &FirestorePriceScheduleRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePriceSchedule creates a new price schedule using common Firestore operations
func (r *FirestorePriceScheduleRepository) CreatePriceSchedule(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price schedule data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price schedule: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedPriceSchedule, err := operations.ConvertMapToProtobuf(result, &priceschedulepb.PriceSchedule{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceschedulepb.CreatePriceScheduleResponse{
		Data: []*priceschedulepb.PriceSchedule{convertedPriceSchedule},
	}, nil
}

// ReadPriceSchedule retrieves a price schedule using common Firestore operations
func (r *FirestorePriceScheduleRepository) ReadPriceSchedule(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price schedule: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedPriceSchedule, err := operations.ConvertMapToProtobuf(result, &priceschedulepb.PriceSchedule{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceschedulepb.ReadPriceScheduleResponse{
		Data: []*priceschedulepb.PriceSchedule{convertedPriceSchedule},
	}, nil
}

// UpdatePriceSchedule updates a price schedule using common Firestore operations
func (r *FirestorePriceScheduleRepository) UpdatePriceSchedule(ctx context.Context, req *priceschedulepb.UpdatePriceScheduleRequest) (*priceschedulepb.UpdatePriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price schedule: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedPriceSchedule, err := operations.ConvertMapToProtobuf(result, &priceschedulepb.PriceSchedule{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceschedulepb.UpdatePriceScheduleResponse{
		Data: []*priceschedulepb.PriceSchedule{convertedPriceSchedule},
	}, nil
}

// DeletePriceSchedule deletes a price schedule using common Firestore operations
func (r *FirestorePriceScheduleRepository) DeletePriceSchedule(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) (*priceschedulepb.DeletePriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete price schedule: %w", err)
	}

	return &priceschedulepb.DeletePriceScheduleResponse{
		Success: true,
	}, nil
}

// ListPriceSchedules lists price schedules using common Firestore operations
func (r *FirestorePriceScheduleRepository) ListPriceSchedules(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
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
		return nil, fmt.Errorf("failed to list price schedules: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	priceSchedules, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *priceschedulepb.PriceSchedule {
		return &priceschedulepb.PriceSchedule{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if priceSchedules == nil {
		priceSchedules = make([]*priceschedulepb.PriceSchedule, 0)
	}

	return &priceschedulepb.ListPriceSchedulesResponse{
		Data: priceSchedules,
	}, nil
}
