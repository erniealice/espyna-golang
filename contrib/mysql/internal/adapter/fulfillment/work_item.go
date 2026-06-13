//go:build mysql

package fulfillment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Fulfillment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql fulfillment repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLFulfillmentRepository(dbOps, tableName), nil
	})
}

// MySQLFulfillmentRepository implements fulfillment CRUD operations using MySQL 8.0+.
type MySQLFulfillmentRepository struct {
	pb.UnimplementedFulfillmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLFulfillmentRepository creates a new MySQL fulfillment repository.
func NewMySQLFulfillmentRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.FulfillmentDomainServiceServer {
	if tableName == "" {
		tableName = "fulfillment"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLFulfillmentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateFulfillment creates a new fulfillment record.
func (r *MySQLFulfillmentRepository) CreateFulfillment(ctx context.Context, req *pb.CreateFulfillmentRequest) (*pb.CreateFulfillmentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("fulfillment data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateFulfillmentResponse{
		Data: fulfillment,
	}, nil
}

// GetFulfillment retrieves a fulfillment record by ID.
//
// Dialect: $1 → ?, read via dbOps.Read.
func (r *MySQLFulfillmentRepository) GetFulfillment(ctx context.Context, req *pb.GetFulfillmentRequest) (*pb.GetFulfillmentResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.GetFulfillmentResponse{
		Data: fulfillment,
	}, nil
}

// UpdateFulfillment updates a fulfillment record.
func (r *MySQLFulfillmentRepository) UpdateFulfillment(ctx context.Context, req *pb.UpdateFulfillmentRequest) (*pb.UpdateFulfillmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateFulfillmentResponse{
		Data: fulfillment,
	}, nil
}

// DeleteFulfillment soft-deletes a fulfillment record (SET active=false).
//
// Dialect: RETURNING → UPDATE + RowsAffected; active = true → active = 1;
// $1 → ?.
func (r *MySQLFulfillmentRepository) DeleteFulfillment(ctx context.Context, req *pb.DeleteFulfillmentRequest) (*pb.DeleteFulfillmentResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	// Dialect: active = true → active = 0 (soft delete sets to 0), $1 → ?
	query := `UPDATE fulfillment SET active = 0, date_modified = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete fulfillment: %w", err)
	}

	return &pb.DeleteFulfillmentResponse{
		Success: true,
	}, nil
}

var fulfillmentSortableSQLCols = []string{
	"id", "active", "workspace_id", "revenue_id", "supplier_id",
	"delivery_mode", "status", "provider_status", "provider_reference",
	"delivery_cost", "currency", "expenditure_id", "scheduled_at", "delivered_at",
	"date_created", "date_modified",
}

var fulfillmentSortSpec = espynahttp.SortSpec{AllowedCols: fulfillmentSortableSQLCols}

// ListFulfillments lists fulfillment records.
func (r *MySQLFulfillmentRepository) ListFulfillments(ctx context.Context, req *pb.ListFulfillmentsRequest) (*pb.ListFulfillmentsResponse, error) {
	if err := espynahttp.ValidateSortColumns(fulfillmentSortSpec, req.GetSort(), "fulfillment"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filter
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillments: %w", err)
	}

	var fulfillments []*pb.Fulfillment
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal fulfillment row: %v", err)
			continue
		}

		fulfillment := &pb.Fulfillment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
			log.Printf("WARN: protojson unmarshal fulfillment: %v", err)
			continue
		}
		fulfillments = append(fulfillments, fulfillment)
	}

	return &pb.ListFulfillmentsResponse{
		Data: fulfillments,
	}, nil
}

// GetFulfillmentListPageData retrieves fulfillments with pagination, filtering,
// sorting, and search. Joins supplier name, counts line items and status events.
//
// Dialect translation from postgres gold standard:
//   - $1,$2,$3,$4 → ? (positional, same left-to-right order)
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1 (and s.active = true → s.active = 1)
//   - COUNT(*) stays; no window function here (uses counted CTE instead)
//   - ORDER BY + LIMIT/OFFSET with safe interpolated sort column
//
// CRITICAL: workspace_id = ? predicate — required for multi-tenancy.
func (r *MySQLFulfillmentRepository) GetFulfillmentListPageData(
	ctx context.Context,
	req *pb.GetFulfillmentListPageDataRequest,
) (*pb.GetFulfillmentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get fulfillment list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	sortField := "f.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		} else {
			sortOrder = "ASC"
		}
	}

	// Dialect: active = true → active = 1; ILIKE → LIKE; $N → ?
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				f.id,
				f.date_created,
				f.date_modified,
				f.active,
				f.workspace_id,
				f.revenue_id,
				f.supplier_id,
				f.delivery_mode,
				f.status,
				f.provider_status,
				f.provider_reference,
				f.delivery_cost,
				f.currency,
				f.expenditure_id,
				f.scheduled_at,
				f.delivered_at,
				COALESCE(s.name, '') AS supplier_name,
				COUNT(DISTINCT fi.id) AS item_count,
				COUNT(DISTINCT fse.id) AS status_event_count
			FROM fulfillment f
			LEFT JOIN supplier s ON s.id = f.supplier_id AND s.active = 1
			LEFT JOIN fulfillment_item fi ON fi.fulfillment_id = f.id
			LEFT JOIN fulfillment_status_event fse ON fse.fulfillment_id = f.id
			WHERE f.active = 1
			  AND f.workspace_id = ?
			  AND (? = '' OR
			       f.status LIKE ? OR f.provider_reference LIKE ?)
			GROUP BY f.id, s.name
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.id,
			e.date_created,
			e.date_modified,
			e.active,
			e.workspace_id,
			e.revenue_id,
			e.supplier_id,
			e.delivery_mode,
			e.status,
			e.provider_status,
			e.provider_reference,
			e.delivery_cost,
			e.currency,
			e.expenditure_id,
			e.scheduled_at,
			e.delivered_at,
			e.supplier_name,
			e.item_count,
			e.status_event_count,
			c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT ? OFFSET ?;
	`, sortField, sortOrder)

	rows, err := r.db.QueryContext(ctx, query,
		workspaceID,
		searchPattern, searchPattern, searchPattern,
		limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillment list page data: %w", err)
	}
	defer rows.Close()

	var resultRows []*pb.FulfillmentListRow
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			workspaceIDVal    string
			revenueID         string
			supplierID        sql.NullString
			deliveryMode      string
			status            string
			providerStatus    string
			providerReference string
			deliveryCost      int64
			currency          string
			expenditureID     sql.NullString
			scheduledAt       sql.NullTime
			deliveredAt       sql.NullTime
			supplierName      string
			itemCount         int32
			statusEventCount  int32
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&workspaceIDVal,
			&revenueID,
			&supplierID,
			&deliveryMode,
			&status,
			&providerStatus,
			&providerReference,
			&deliveryCost,
			&currency,
			&expenditureID,
			&scheduledAt,
			&deliveredAt,
			&supplierName,
			&itemCount,
			&statusEventCount,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fulfillment list row: %w", err)
		}

		totalCount = total

		f := &pb.Fulfillment{
			Id:                id,
			Active:            active,
			WorkspaceId:       workspaceIDVal,
			RevenueId:         revenueID,
			DeliveryMode:      deliveryMode,
			Status:            status,
			ProviderStatus:    providerStatus,
			ProviderReference: providerReference,
			DeliveryCost:      deliveryCost,
			Currency:          currency,
		}

		if supplierID.Valid {
			f.SupplierId = &supplierID.String
		}
		if expenditureID.Valid {
			f.ExpenditureId = &expenditureID.String
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			f.DateCreated = &ts
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			f.DateModified = &ts
		}

		row := &pb.FulfillmentListRow{
			Fulfillment:      f,
			SupplierName:     supplierName,
			ItemCount:        itemCount,
			StatusEventCount: statusEventCount,
		}
		resultRows = append(resultRows, row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetFulfillmentListPageDataResponse{
		Rows: resultRows,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// convertMillisToTime converts epoch-millisecond fields in data maps to time.Time
// for MySQL-compatible datetime columns.
func convertMillisToTime(data map[string]any, key string) {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case float64:
			ms := int64(val)
			if ms > 0 {
				data[key] = time.UnixMilli(ms).UTC()
			}
		case int64:
			if val > 0 {
				data[key] = time.UnixMilli(val).UTC()
			}
		}
	}
}

// NewFulfillmentRepository creates a new MySQL fulfillment repository (old-style constructor).
func NewFulfillmentRepository(db *sql.DB, tableName string) pb.FulfillmentDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLFulfillmentRepository(dbOps, tableName)
}
