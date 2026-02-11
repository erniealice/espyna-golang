//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "price_list", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres price_list repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPriceListRepository(dbOps, tableName), nil
	})
}

// PostgresPriceListRepository implements price_list CRUD operations using PostgreSQL
type PostgresPriceListRepository struct {
	pricelistpb.UnimplementedPriceListDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPriceListRepository creates a new PostgreSQL price list repository
func NewPostgresPriceListRepository(dbOps interfaces.DatabaseOperation, tableName string) pricelistpb.PriceListDomainServiceServer {
	if tableName == "" {
		tableName = "price_list"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPriceListRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePriceList creates a new price list using common PostgreSQL operations
func (r *PostgresPriceListRepository) CreatePriceList(ctx context.Context, req *pricelistpb.CreatePriceListRequest) (*pricelistpb.CreatePriceListResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price list data is required")
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
		return nil, fmt.Errorf("failed to create price list: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceList := &pricelistpb.PriceList{}
	if err := protojson.Unmarshal(resultJSON, priceList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pricelistpb.CreatePriceListResponse{
		Data: []*pricelistpb.PriceList{priceList},
	}, nil
}

// ReadPriceList retrieves a price list using common PostgreSQL operations
func (r *PostgresPriceListRepository) ReadPriceList(ctx context.Context, req *pricelistpb.ReadPriceListRequest) (*pricelistpb.ReadPriceListResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price list ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price list: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceList := &pricelistpb.PriceList{}
	if err := protojson.Unmarshal(resultJSON, priceList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pricelistpb.ReadPriceListResponse{
		Data: []*pricelistpb.PriceList{priceList},
	}, nil
}

// UpdatePriceList updates a price list using common PostgreSQL operations
func (r *PostgresPriceListRepository) UpdatePriceList(ctx context.Context, req *pricelistpb.UpdatePriceListRequest) (*pricelistpb.UpdatePriceListResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price list ID is required")
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
		return nil, fmt.Errorf("failed to update price list: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceList := &pricelistpb.PriceList{}
	if err := protojson.Unmarshal(resultJSON, priceList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pricelistpb.UpdatePriceListResponse{
		Data: []*pricelistpb.PriceList{priceList},
	}, nil
}

// DeletePriceList deletes a price list using common PostgreSQL operations
func (r *PostgresPriceListRepository) DeletePriceList(ctx context.Context, req *pricelistpb.DeletePriceListRequest) (*pricelistpb.DeletePriceListResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price list ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete price list: %w", err)
	}

	return &pricelistpb.DeletePriceListResponse{
		Success: true,
	}, nil
}

// ListPriceLists lists price lists using common PostgreSQL operations
func (r *PostgresPriceListRepository) ListPriceLists(ctx context.Context, req *pricelistpb.ListPriceListsRequest) (*pricelistpb.ListPriceListsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list price lists: %w", err)
	}

	var priceLists []*pricelistpb.PriceList
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		priceList := &pricelistpb.PriceList{}
		if err := protojson.Unmarshal(resultJSON, priceList); err != nil {
			continue
		}
		priceLists = append(priceLists, priceList)
	}

	return &pricelistpb.ListPriceListsResponse{
		Data: priceLists,
	}, nil
}

// GetPriceListListPageData retrieves price lists with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresPriceListRepository) GetPriceListListPageData(
	ctx context.Context,
	req *pricelistpb.GetPriceListListPageDataRequest,
) (*pricelistpb.GetPriceListListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}

	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `WITH enriched AS (SELECT id, name, description, active, date_start, date_end, location_id, date_created, date_modified FROM price_list WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR name ILIKE $1 OR description ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var priceLists []*pricelistpb.PriceList
	var totalCount int64
	for rows.Next() {
		var id, name string
		var description, locationId sql.NullString
		var active bool
		var dateStart int64
		var dateEnd sql.NullInt64
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &name, &description, &active, &dateStart, &dateEnd, &locationId, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		priceList := &pricelistpb.PriceList{Id: id, Name: name, Active: active, DateStart: dateStart}
		if description.Valid {
			priceList.Description = &description.String
		}
		if locationId.Valid {
			priceList.LocationId = &locationId.String
		}
		if dateEnd.Valid {
			priceList.DateEnd = &dateEnd.Int64
		}
		dsStr := time.Unix(dateStart/1000, 0).Format(time.RFC3339)
		priceList.DateStartString = dsStr
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			priceList.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			priceList.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			priceList.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			priceList.DateModifiedString = &dmStr
		}
		priceLists = append(priceLists, priceList)
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &pricelistpb.GetPriceListListPageDataResponse{PriceListList: priceLists, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetPriceListItemPageData retrieves price list item page data including associated price products
func (r *PostgresPriceListRepository) GetPriceListItemPageData(ctx context.Context, req *pricelistpb.GetPriceListItemPageDataRequest) (*pricelistpb.GetPriceListItemPageDataResponse, error) {
	if req == nil || req.PriceListId == "" {
		return nil, fmt.Errorf("price list ID required")
	}

	// Query price list
	query := `SELECT id, name, description, active, date_start, date_end, location_id, date_created, date_modified FROM price_list WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.PriceListId)
	var id, name string
	var description, locationId sql.NullString
	var active bool
	var dateStart int64
	var dateEnd sql.NullInt64
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &name, &description, &active, &dateStart, &dateEnd, &locationId, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("price list not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	priceList := &pricelistpb.PriceList{Id: id, Name: name, Active: active, DateStart: dateStart}
	if description.Valid {
		priceList.Description = &description.String
	}
	if locationId.Valid {
		priceList.LocationId = &locationId.String
	}
	if dateEnd.Valid {
		priceList.DateEnd = &dateEnd.Int64
	}
	dsStr := time.Unix(dateStart/1000, 0).Format(time.RFC3339)
	priceList.DateStartString = dsStr
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		priceList.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		priceList.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		priceList.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		priceList.DateModifiedString = &dmStr
	}

	// Query price products associated with this price list
	ppQuery := `SELECT id, product_id, amount, currency, active, date_created, date_modified FROM price_product WHERE price_list_id = $1 AND active = true`
	ppRows, err := r.db.QueryContext(ctx, ppQuery, req.PriceListId)
	if err != nil {
		return nil, fmt.Errorf("price products query failed: %w", err)
	}
	defer ppRows.Close()

	var priceProducts []*priceproductpb.PriceProduct
	for ppRows.Next() {
		var ppId, productId, currency string
		var amount int64
		var ppActive bool
		var ppDateCreated, ppDateModified time.Time
		if err := ppRows.Scan(&ppId, &productId, &amount, &currency, &ppActive, &ppDateCreated, &ppDateModified); err != nil {
			return nil, fmt.Errorf("price product scan failed: %w", err)
		}
		pp := &priceproductpb.PriceProduct{Id: ppId, ProductId: productId, Amount: amount, Currency: currency, Active: ppActive}
		if !ppDateCreated.IsZero() {
			ts := ppDateCreated.UnixMilli()
			pp.DateCreated = &ts
			dcStr := ppDateCreated.Format(time.RFC3339)
			pp.DateCreatedString = &dcStr
		}
		if !ppDateModified.IsZero() {
			ts := ppDateModified.UnixMilli()
			pp.DateModified = &ts
			dmStr := ppDateModified.Format(time.RFC3339)
			pp.DateModifiedString = &dmStr
		}
		priceProducts = append(priceProducts, pp)
	}

	return &pricelistpb.GetPriceListItemPageDataResponse{PriceList: priceList, PriceProducts: priceProducts, Success: true}, nil
}

// NewPriceListRepository creates a new PostgreSQL price_list repository (old-style constructor)
func NewPriceListRepository(db *sql.DB, tableName string) pricelistpb.PriceListDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPriceListRepository(dbOps, tableName)
}
