//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductCollection, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductCollectionRepository(dbOps, tableName), nil
	})
}

// SQLServerProductCollectionRepository implements product_collection CRUD using SQL Server.
type SQLServerProductCollectionRepository struct {
	productcollectionpb.UnimplementedProductCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductCollectionRepository creates a new SQL Server product_collection repository.
func NewSQLServerProductCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) productcollectionpb.ProductCollectionDomainServiceServer {
	if tableName == "" {
		tableName = "product_collection"
	}
	return &SQLServerProductCollectionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductCollectionRepository) CreateProductCollection(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) (*productcollectionpb.CreateProductCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_collection data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_collection: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pc := &productcollectionpb.ProductCollection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productcollectionpb.CreateProductCollectionResponse{Data: []*productcollectionpb.ProductCollection{pc}}, nil
}

func (r *SQLServerProductCollectionRepository) ReadProductCollection(ctx context.Context, req *productcollectionpb.ReadProductCollectionRequest) (*productcollectionpb.ReadProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_collection ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_collection: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pc := &productcollectionpb.ProductCollection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productcollectionpb.ReadProductCollectionResponse{Data: []*productcollectionpb.ProductCollection{pc}}, nil
}

func (r *SQLServerProductCollectionRepository) UpdateProductCollection(ctx context.Context, req *productcollectionpb.UpdateProductCollectionRequest) (*productcollectionpb.UpdateProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_collection ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product_collection: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pc := &productcollectionpb.ProductCollection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productcollectionpb.UpdateProductCollectionResponse{Data: []*productcollectionpb.ProductCollection{pc}}, nil
}

func (r *SQLServerProductCollectionRepository) DeleteProductCollection(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) (*productcollectionpb.DeleteProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_collection ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_collection: %w", err)
	}
	return &productcollectionpb.DeleteProductCollectionResponse{Success: true}, nil
}

func (r *SQLServerProductCollectionRepository) ListProductCollections(ctx context.Context, req *productcollectionpb.ListProductCollectionsRequest) (*productcollectionpb.ListProductCollectionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_collections: %w", err)
	}
	var pcs []*productcollectionpb.ProductCollection
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pc := &productcollectionpb.ProductCollection{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
			continue
		}
		pcs = append(pcs, pc)
	}
	return &productcollectionpb.ListProductCollectionsResponse{Data: pcs}, nil
}

func (r *SQLServerProductCollectionRepository) GetProductCollectionListPageData(ctx context.Context, req *productcollectionpb.GetProductCollectionListPageDataRequest) (*productcollectionpb.GetProductCollectionListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductCollectionListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductCollectionRepository) GetProductCollectionItemPageData(ctx context.Context, req *productcollectionpb.GetProductCollectionItemPageDataRequest) (*productcollectionpb.GetProductCollectionItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductCollectionItemPageData not yet implemented — Phase 2")
}
