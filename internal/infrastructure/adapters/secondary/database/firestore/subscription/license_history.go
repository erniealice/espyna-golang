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
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "license_history", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore license_history repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreLicenseHistoryRepository(dbOps, collectionName), nil
	})
}

// FirestoreLicenseHistoryRepository implements license_history CRUD operations using Firestore
type FirestoreLicenseHistoryRepository struct {
	licensehistorypb.UnimplementedLicenseHistoryDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreLicenseHistoryRepository creates a new Firestore license_history repository
func NewFirestoreLicenseHistoryRepository(dbOps interfaces.DatabaseOperation, collectionName string) licensehistorypb.LicenseHistoryDomainServiceServer {
	if collectionName == "" {
		collectionName = "license_history" // default fallback
	}
	return &FirestoreLicenseHistoryRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateLicenseHistory creates a new license history record using common Firestore operations
func (r *FirestoreLicenseHistoryRepository) CreateLicenseHistory(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) (*licensehistorypb.CreateLicenseHistoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("license history data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create license history: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	licenseHistory := &licensehistorypb.LicenseHistory{}
	convertedHistory, err := operations.ConvertMapToProtobuf(result, licenseHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensehistorypb.CreateLicenseHistoryResponse{
		Data:    []*licensehistorypb.LicenseHistory{convertedHistory},
		Success: true,
	}, nil
}

// ReadLicenseHistory retrieves a license history record using common Firestore operations
func (r *FirestoreLicenseHistoryRepository) ReadLicenseHistory(ctx context.Context, req *licensehistorypb.ReadLicenseHistoryRequest) (*licensehistorypb.ReadLicenseHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license history ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read license history: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	licenseHistory := &licensehistorypb.LicenseHistory{}
	convertedHistory, err := operations.ConvertMapToProtobuf(result, licenseHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &licensehistorypb.ReadLicenseHistoryResponse{
		Data:    []*licensehistorypb.LicenseHistory{convertedHistory},
		Success: true,
	}, nil
}

// ListLicenseHistory lists license history records using common Firestore operations
func (r *FirestoreLicenseHistoryRepository) ListLicenseHistory(ctx context.Context, req *licensehistorypb.ListLicenseHistoryRequest) (*licensehistorypb.ListLicenseHistoryResponse, error) {
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
		return nil, fmt.Errorf("failed to list license history: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	histories, _ := operations.ConvertSliceToProtobuf(listResult.Data, func() *licensehistorypb.LicenseHistory {
		return &licensehistorypb.LicenseHistory{}
	})

	// Filter by license_id if provided
	if req.LicenseId != nil && *req.LicenseId != "" {
		filteredHistories := make([]*licensehistorypb.LicenseHistory, 0)
		for _, history := range histories {
			if history.LicenseId == *req.LicenseId {
				filteredHistories = append(filteredHistories, history)
			}
		}
		histories = filteredHistories
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if histories == nil {
		histories = make([]*licensehistorypb.LicenseHistory, 0)
	}

	return &licensehistorypb.ListLicenseHistoryResponse{
		Data:    histories,
		Success: true,
	}, nil
}

// GetLicenseHistoryListPageData retrieves license history records with pagination support
func (r *FirestoreLicenseHistoryRepository) GetLicenseHistoryListPageData(ctx context.Context, req *licensehistorypb.GetLicenseHistoryListPageDataRequest) (*licensehistorypb.GetLicenseHistoryListPageDataResponse, error) {
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
		return nil, fmt.Errorf("failed to list license history for page data: %w", err)
	}

	// Convert results to protobuf slice
	histories, _ := operations.ConvertSliceToProtobuf(listResult.Data, func() *licensehistorypb.LicenseHistory {
		return &licensehistorypb.LicenseHistory{}
	})

	// Filter by license_id if provided
	if req.LicenseId != nil && *req.LicenseId != "" {
		filteredHistories := make([]*licensehistorypb.LicenseHistory, 0)
		for _, history := range histories {
			if history.LicenseId == *req.LicenseId {
				filteredHistories = append(filteredHistories, history)
			}
		}
		histories = filteredHistories
	}

	if histories == nil {
		histories = make([]*licensehistorypb.LicenseHistory, 0)
	}

	return &licensehistorypb.GetLicenseHistoryListPageDataResponse{
		LicenseHistoryList: histories,
		Pagination:         listResult.Pagination,
		Success:            true,
	}, nil
}
