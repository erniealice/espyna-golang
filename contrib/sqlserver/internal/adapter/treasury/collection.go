//go:build sqlserver

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// collectionSortableSQLCols lists the SQL column names that are safe to sort
// by in GetCollectionListPageData.
var collectionSortableSQLCols = []string{
	"tc.date_created",
	"tc.date_modified",
	"tc.name",
	"tc.amount",
	"tc.status",
	"tc.payment_date",
	"tc.reference_number",
}

// collectionViewToSQLColMap translates view-facing sort column keys to the SQL
// column names used in the query.
var collectionViewToSQLColMap = map[string]string{
	"date_created":     "tc.date_created",
	"date_modified":    "tc.date_modified",
	"name":             "tc.name",
	"amount":           "tc.amount",
	"status":           "tc.status",
	"payment_date":     "tc.payment_date",
	"reference_number": "tc.reference_number",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TreasuryCollection, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerCollectionRepository(dbOps, tableName), nil
	})
}

// SQLServerCollectionRepository implements collection CRUD operations using SQL Server.
type SQLServerCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerCollectionRepository creates a new SQL Server collection repository.
func NewSQLServerCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionpb.CollectionDomainServiceServer {
	if tableName == "" {
		tableName = "treasury_collection"
	}

	var db *sql.DB
	if ep, ok := dbOps.(executorProvider); ok {
		if rawDB, ok2 := ep.GetExecutor(context.Background()).(*sql.DB); ok2 {
			db = rawDB
		}
	}

	return &SQLServerCollectionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollection creates a new collection record.
func (r *SQLServerCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection data is required")
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
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.CreateCollectionResponse{
		Success: true,
		Data:    []*collectionpb.Collection{collection},
	}, nil
}

// ReadCollection retrieves a collection record by ID.
func (r *SQLServerCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.ReadCollectionResponse{
		Success: true,
		Data:    []*collectionpb.Collection{collection},
	}, nil
}

// UpdateCollection updates a collection record.
func (r *SQLServerCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
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
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.UpdateCollectionResponse{
		Success: true,
		Data:    []*collectionpb.Collection{collection},
	}, nil
}

// DeleteCollection soft-deletes a collection record.
func (r *SQLServerCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &collectionpb.DeleteCollectionResponse{Success: true}, nil
}

// ListCollections lists collection records with optional filters.
func (r *SQLServerCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var collections []*collectionpb.Collection
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal collection row: %v", err)
			continue
		}

		collection := &collectionpb.Collection{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
			log.Printf("WARN: protojson unmarshal collection: %v", err)
			continue
		}
		collections = append(collections, collection)
	}

	return &collectionpb.ListCollectionsResponse{
		Success: true,
		Data:    collections,
	}, nil
}

// GetCollectionListPageData retrieves collections with pagination, filtering, sorting, and search.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - @p1,@p2,… placeholders (not $1,$2,…).
//   - LIKE instead of ILIKE (SQL Server default CI collation is case-insensitive).
//   - active = 1 (BIT) instead of active = true.
//   - Pagination: ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - No FILTER (WHERE) — not needed in list queries (only dashboards).
//   - NULL-safe search: (@p2 = ” OR col LIKE @p2) replaces ($2::text IS NULL ...).
func (r *SQLServerCollectionRepository) GetCollectionListPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionListPageDataRequest,
) (*collectionpb.GetCollectionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection list page data request is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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

	// Translate view-facing column key to SQL column name.
	sortColKey := "tc.date_created"
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
	}
	if mapped, ok := collectionViewToSQLColMap[sortColKey]; ok {
		sortColKey = mapped
	}

	// A2 sort guard via BuildOrderBy — returns "ORDER BY [col] DIR".
	sortDir := commonpb.SortDirection_DESC
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortDir = req.Sort.Fields[0].Direction
	}
	orderByClause, err := sqlserverCore.BuildOrderBy(
		collectionSortableSQLCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: sortColKey, Direction: sortDir}}},
		"tc.date_created DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for collection: %w", err)
	}

	// 20260517 advance-cash-events: extend the CTE with all advance_* schedule
	// columns + client_id. SQL Server translation: LIKE, @pN, active = 1,
	// OFFSET/FETCH pagination.
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				tc.id,
				tc.date_created,
				tc.date_modified,
				tc.active,
				tc.name,
				tc.subscription_id,
				tc.amount,
				tc.status,
				tc.revenue_id,
				tc.collection_method_id,
				tc.currency,
				tc.reference_number,
				tc.payment_date,
				tc.received_by,
				tc.received_role,
				tc.collection_type,
				tc.advance_kind,
				tc.advance_status,
				tc.advance_start_date,
				tc.advance_end_date,
				tc.advance_period_count,
				tc.advance_period_unit,
				tc.advance_total_amount,
				tc.advance_remaining_amount,
				tc.advance_recognized_amount,
				tc.advance_balance_account_id,
				tc.advance_target_account_id,
				tc.advance_expiry_date,
				tc.advance_proration_policy,
				tc.client_id
			FROM treasury_collection tc
			WHERE tc.active = 1
			  AND tc.workspace_id = @p1
			  AND (@p2 = '' OR
			       tc.name LIKE @p2 OR
			       tc.reference_number LIKE @p2 OR
			       tc.status LIKE @p2 OR
			       tc.collection_type LIKE @p2)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY;
	`, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection list page data: %w", err)
	}
	defer rows.Close()

	var collections []*collectionpb.Collection
	var totalCount int64

	for rows.Next() {
		var (
			id                      string
			dateCreated             time.Time
			dateModified            time.Time
			active                  bool
			name                    string
			subscriptionID          *string
			amount                  int64
			status                  *string
			revenueID               *string
			collectionMethodID      *string
			currency                *string
			referenceNumber         *string
			paymentDate             *time.Time
			receivedBy              *string
			receivedRole            *string
			collectionType          *string
			advanceKind             sql.NullInt32
			advanceStatus           sql.NullInt32
			advanceStartDate        *string
			advanceEndDate          *string
			advancePeriodCount      sql.NullInt32
			advancePeriodUnit       *string
			advanceTotalAmount      sql.NullInt64
			advanceRemainingAmount  sql.NullInt64
			advanceRecognizedAmount sql.NullInt64
			advanceBalanceAccountID *string
			advanceTargetAccountID  *string
			advanceExpiryDate       *string
			advanceProrationPolicy  sql.NullInt32
			clientID                *string
			total                   int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&subscriptionID,
			&amount,
			&status,
			&revenueID,
			&collectionMethodID,
			&currency,
			&referenceNumber,
			&paymentDate,
			&receivedBy,
			&receivedRole,
			&collectionType,
			&advanceKind,
			&advanceStatus,
			&advanceStartDate,
			&advanceEndDate,
			&advancePeriodCount,
			&advancePeriodUnit,
			&advanceTotalAmount,
			&advanceRemainingAmount,
			&advanceRecognizedAmount,
			&advanceBalanceAccountID,
			&advanceTargetAccountID,
			&advanceExpiryDate,
			&advanceProrationPolicy,
			&clientID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan collection row: %w", err)
		}

		totalCount = total

		collection := &collectionpb.Collection{
			Id:     id,
			Active: active,
			Name:   name,
			Amount: amount,
		}

		if subscriptionID != nil {
			collection.SubscriptionId = *subscriptionID
		}
		if status != nil {
			collection.Status = *status
		}
		if revenueID != nil {
			collection.RevenueId = *revenueID
		}
		if collectionMethodID != nil {
			collection.CollectionMethodId = *collectionMethodID
		}
		if currency != nil {
			collection.Currency = *currency
		}
		if referenceNumber != nil {
			collection.ReferenceNumber = *referenceNumber
		}
		if receivedBy != nil {
			collection.ReceivedBy = *receivedBy
		}
		if receivedRole != nil {
			collection.ReceivedRole = *receivedRole
		}
		if collectionType != nil {
			collection.CollectionType = *collectionType
		}
		if paymentDate != nil && !paymentDate.IsZero() {
			collection.PaymentDate = paymentDate.Format("2006-01-02")
		}
		assignAdvanceFieldsCollection(collection,
			advanceKind, advanceStatus, advanceStartDate, advanceEndDate,
			advancePeriodCount, advancePeriodUnit,
			advanceTotalAmount, advanceRemainingAmount, advanceRecognizedAmount,
			advanceBalanceAccountID, advanceTargetAccountID, advanceExpiryDate,
			advanceProrationPolicy, clientID,
		)

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			collection.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			collection.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			collection.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			collection.DateModifiedString = &dmStr
		}

		collections = append(collections, collection)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &collectionpb.GetCollectionListPageDataResponse{
		CollectionList: collections,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetCollectionItemPageData retrieves a single collection with enriched data.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *SQLServerCollectionRepository) GetCollectionItemPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionItemPageDataRequest,
) (*collectionpb.GetCollectionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection item page data request is required")
	}
	if req.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// SQL Server: TOP 1 instead of LIMIT 1; @p1/@p2 instead of $1/$2.
	query := `
		WITH enriched AS (
			SELECT
				tc.id,
				tc.date_created,
				tc.date_modified,
				tc.active,
				tc.name,
				tc.subscription_id,
				tc.amount,
				tc.status,
				tc.revenue_id,
				tc.collection_method_id,
				tc.currency,
				tc.reference_number,
				tc.payment_date,
				tc.received_by,
				tc.received_role,
				tc.collection_type,
				tc.advance_kind,
				tc.advance_status,
				tc.advance_start_date,
				tc.advance_end_date,
				tc.advance_period_count,
				tc.advance_period_unit,
				tc.advance_total_amount,
				tc.advance_remaining_amount,
				tc.advance_recognized_amount,
				tc.advance_balance_account_id,
				tc.advance_target_account_id,
				tc.advance_expiry_date,
				tc.advance_proration_policy,
				tc.client_id
			FROM treasury_collection tc
			WHERE tc.id = @p1 AND tc.workspace_id = @p2 AND tc.active = 1
		)
		SELECT TOP 1 * FROM enriched;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.CollectionId, workspaceID)

	var (
		id                      string
		dateCreated             time.Time
		dateModified            time.Time
		active                  bool
		name                    string
		subscriptionID          *string
		amount                  int64
		status                  *string
		revenueID               *string
		collectionMethodID      *string
		currency                *string
		referenceNumber         *string
		paymentDate             *time.Time
		receivedBy              *string
		receivedRole            *string
		collectionType          *string
		advanceKind             sql.NullInt32
		advanceStatus           sql.NullInt32
		advanceStartDate        *string
		advanceEndDate          *string
		advancePeriodCount      sql.NullInt32
		advancePeriodUnit       *string
		advanceTotalAmount      sql.NullInt64
		advanceRemainingAmount  sql.NullInt64
		advanceRecognizedAmount sql.NullInt64
		advanceBalanceAccountID *string
		advanceTargetAccountID  *string
		advanceExpiryDate       *string
		advanceProrationPolicy  sql.NullInt32
		clientID                *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&subscriptionID,
		&amount,
		&status,
		&revenueID,
		&collectionMethodID,
		&currency,
		&referenceNumber,
		&paymentDate,
		&receivedBy,
		&receivedRole,
		&collectionType,
		&advanceKind,
		&advanceStatus,
		&advanceStartDate,
		&advanceEndDate,
		&advancePeriodCount,
		&advancePeriodUnit,
		&advanceTotalAmount,
		&advanceRemainingAmount,
		&advanceRecognizedAmount,
		&advanceBalanceAccountID,
		&advanceTargetAccountID,
		&advanceExpiryDate,
		&advanceProrationPolicy,
		&clientID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection with ID '%s' not found", req.CollectionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query collection item page data: %w", err)
	}

	collection := &collectionpb.Collection{
		Id:     id,
		Active: active,
		Name:   name,
		Amount: amount,
	}

	if subscriptionID != nil {
		collection.SubscriptionId = *subscriptionID
	}
	if status != nil {
		collection.Status = *status
	}
	if revenueID != nil {
		collection.RevenueId = *revenueID
	}
	if collectionMethodID != nil {
		collection.CollectionMethodId = *collectionMethodID
	}
	if currency != nil {
		collection.Currency = *currency
	}
	if referenceNumber != nil {
		collection.ReferenceNumber = *referenceNumber
	}
	if receivedBy != nil {
		collection.ReceivedBy = *receivedBy
	}
	if receivedRole != nil {
		collection.ReceivedRole = *receivedRole
	}
	if collectionType != nil {
		collection.CollectionType = *collectionType
	}
	if paymentDate != nil && !paymentDate.IsZero() {
		collection.PaymentDate = paymentDate.Format("2006-01-02")
	}
	assignAdvanceFieldsCollection(collection,
		advanceKind, advanceStatus, advanceStartDate, advanceEndDate,
		advancePeriodCount, advancePeriodUnit,
		advanceTotalAmount, advanceRemainingAmount, advanceRecognizedAmount,
		advanceBalanceAccountID, advanceTargetAccountID, advanceExpiryDate,
		advanceProrationPolicy, clientID,
	)

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		collection.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		collection.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		collection.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		collection.DateModifiedString = &dmStr
	}

	return &collectionpb.GetCollectionItemPageDataResponse{
		Collection: collection,
		Success:    true,
	}, nil
}

// assignAdvanceFieldsCollection folds the optional advance_* schedule columns
// into the Collection proto. Centralised so List and Item pages stay in lock-step.
func assignAdvanceFieldsCollection(
	out *collectionpb.Collection,
	advanceKind sql.NullInt32,
	advanceStatus sql.NullInt32,
	advanceStartDate *string,
	advanceEndDate *string,
	advancePeriodCount sql.NullInt32,
	advancePeriodUnit *string,
	advanceTotalAmount sql.NullInt64,
	advanceRemainingAmount sql.NullInt64,
	advanceRecognizedAmount sql.NullInt64,
	advanceBalanceAccountID *string,
	advanceTargetAccountID *string,
	advanceExpiryDate *string,
	advanceProrationPolicy sql.NullInt32,
	clientID *string,
) {
	if advanceKind.Valid {
		k := advancekindpb.AdvanceKind(advanceKind.Int32)
		out.AdvanceKind = &k
	}
	if advanceStatus.Valid {
		s := advancekindpb.AdvanceStatus(advanceStatus.Int32)
		out.AdvanceStatus = &s
	}
	if advanceStartDate != nil {
		out.AdvanceStartDate = advanceStartDate
	}
	if advanceEndDate != nil {
		out.AdvanceEndDate = advanceEndDate
	}
	if advancePeriodCount.Valid {
		pc := advancePeriodCount.Int32
		out.AdvancePeriodCount = &pc
	}
	if advancePeriodUnit != nil {
		out.AdvancePeriodUnit = advancePeriodUnit
	}
	if advanceTotalAmount.Valid {
		v := advanceTotalAmount.Int64
		out.AdvanceTotalAmount = &v
	}
	if advanceRemainingAmount.Valid {
		v := advanceRemainingAmount.Int64
		out.AdvanceRemainingAmount = &v
	}
	if advanceRecognizedAmount.Valid {
		v := advanceRecognizedAmount.Int64
		out.AdvanceRecognizedAmount = &v
	}
	if advanceBalanceAccountID != nil {
		out.AdvanceBalanceAccountId = advanceBalanceAccountID
	}
	if advanceTargetAccountID != nil {
		out.AdvanceTargetAccountId = advanceTargetAccountID
	}
	if advanceExpiryDate != nil {
		out.AdvanceExpiryDate = advanceExpiryDate
	}
	if advanceProrationPolicy.Valid {
		p := advancekindpb.AdvanceProrationPolicy(advanceProrationPolicy.Int32)
		out.AdvanceProrationPolicy = &p
	}
	if clientID != nil {
		out.ClientId = clientID
	}
}

// ListByClient lists collections for a given client by joining treasury_collection
// to revenue on revenue.client_id. Workspace isolation is applied.
func (r *SQLServerCollectionRepository) ListByClient(ctx context.Context, req *collectionpb.ListByClientRequest) (*collectionpb.ListByClientResponse, error) {
	if req.GetClientId() == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx,
		`SELECT c.id, c.active, c.revenue_id, c.amount, c.status, c.currency,
		        c.reference_number, c.payment_date, c.collection_type
		 FROM treasury_collection c
		 JOIN revenue r ON r.id = c.revenue_id
		 WHERE r.client_id = @p1
		   AND (@p2 = '' OR r.workspace_id = @p2)`,
		req.GetClientId(), wsID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListByClient query failed: %w", err)
	}
	defer rows.Close()

	var collections []*collectionpb.Collection
	for rows.Next() {
		c := &collectionpb.Collection{}
		if scanErr := rows.Scan(
			&c.Id, &c.Active, &c.RevenueId, &c.Amount, &c.Status, &c.Currency,
			&c.ReferenceNumber, &c.PaymentDate, &c.CollectionType,
		); scanErr != nil {
			return nil, fmt.Errorf("ListByClient scan failed: %w", scanErr)
		}
		collections = append(collections, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListByClient rows error: %w", err)
	}

	return &collectionpb.ListByClientResponse{Data: collections, Success: true}, nil
}

// NewCollectionRepository creates a new SQL Server collection repository (old-style constructor).
func NewCollectionRepository(db *sql.DB, tableName string) collectionpb.CollectionDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerCollectionRepository(dbOps, tableName)
}
