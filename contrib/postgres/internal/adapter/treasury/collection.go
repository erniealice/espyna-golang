//go:build postgresql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// collectionSortableSQLCols lists the SQL column names that are safe to sort by
// in GetCollectionListPageData. The query uses direct ORDER BY interpolation so
// this guard is critical — an unrecognised column is a potential SQL-injection
// vector and must be rejected loudly before query execution.
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
// column names used in the query. Columns absent from the map pass through unchanged.
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
	registry.RegisterRepositoryFactory("postgresql", entityid.TreasuryCollection, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCollectionRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionRepository implements collection CRUD operations using PostgreSQL
type PostgresCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCollectionRepository creates a new PostgreSQL collection repository
func NewPostgresCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionpb.CollectionDomainServiceServer {
	if tableName == "" {
		tableName = "treasury_collection"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollection creates a new collection record
func (r *PostgresCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection data is required")
	}

	// BURN_DOWN guard moved to the use case layer (Phase 1.C-iv of
	// 20260518-hexagonal-strict-adherence). See
	// internal/application/usecases/domain/treasury/collection/validate_advance_kind.go.

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
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

// ReadCollection retrieves a collection record by ID
func (r *PostgresCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
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

// UpdateCollection updates a collection record
func (r *PostgresCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// BURN_DOWN guard moved to the use case layer (Phase 1.C-iv of
	// 20260518-hexagonal-strict-adherence). See
	// internal/application/usecases/domain/treasury/collection/validate_advance_kind.go.

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
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

// DeleteCollection deletes a collection record (soft delete)
func (r *PostgresCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &collectionpb.DeleteCollectionResponse{
		Success: true,
	}, nil
}

// ListCollections lists collection records with optional filters
func (r *PostgresCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
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
		postgresCore.ConvertMillisToDateStr(result, "payment_date")
		resultJSON, err := json.Marshal(result)
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

// GetCollectionListPageData retrieves collections with pagination, filtering, sorting, and search using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresCollectionRepository) GetCollectionListPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionListPageDataRequest,
) (*collectionpb.GetCollectionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection list page data request is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
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

	sortField := "tc.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Translate view-facing column key to SQL column name via ColMap.
	if mapped, ok := collectionViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	// Loud-failure guard: reject any sort column not in the allowlist. This query
	// uses direct ORDER BY interpolation, so an unrecognised value is a potential
	// SQL-injection vector and must be rejected loudly before query execution.
	if sortField != "" && !slices.Contains(collectionSortableSQLCols, sortField) {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortField, "collection", collectionSortableSQLCols)
	}

	// 20260517 advance-cash-events: extend the CTE with all advance_* schedule
	// columns + client_id. The list view doesn't render every column today,
	// but downstream filter chips + Treasury dashboard need the data flowing
	// through the proto without a second round-trip.
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
			WHERE tc.active = true
			  AND tc.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
			       tc.name ILIKE $2 OR
			       tc.reference_number ILIKE $2 OR
			       tc.status ILIKE $2 OR
			       tc.collection_type ILIKE $2)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $3 OFFSET $4;
	`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, limit, offset)
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

// GetCollectionItemPageData retrieves a single collection with enriched data using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresCollectionRepository) GetCollectionItemPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionItemPageDataRequest,
) (*collectionpb.GetCollectionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection item page data request is required")
	}
	if req.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// 20260517 advance-cash-events: extend the CTE with all advance_* schedule
	// columns + client_id (mirrors GetCollectionListPageData; needed by the
	// Advance Schedule tab + Treasury dashboard tile).
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
			WHERE tc.id = $1 AND tc.workspace_id = $2 AND tc.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.CollectionId, workspaceID)

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
// scanned from a treasury_collection row into the Collection proto. Centralised
// so GetCollectionListPageData and GetCollectionItemPageData stay in lock-step.
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
// to revenue on revenue.client_id. Workspace isolation is applied automatically.
func (r *PostgresCollectionRepository) ListByClient(ctx context.Context, req *collectionpb.ListByClientRequest) (*collectionpb.ListByClientResponse, error) {
	if req.GetClientId() == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	rows, err := db.GetDB().QueryContext(ctx,
		`SELECT c.id, c.active, c.revenue_id, c.amount, c.status, c.currency,
		        c.reference_number, c.payment_date, c.collection_type
		 FROM treasury_collection c
		 JOIN revenue r ON r.id = c.revenue_id
		 WHERE r.client_id = $1
		   AND ($2::text = '' OR r.workspace_id = $2::text)`,
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

// NewCollectionRepository creates a new PostgreSQL collection repository (old-style constructor)
func NewCollectionRepository(db *sql.DB, tableName string) collectionpb.CollectionDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCollectionRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey, _ string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}
