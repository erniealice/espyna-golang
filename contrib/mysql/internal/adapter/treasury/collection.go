//go:build mysql

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
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// collectionSortableSQLCols lists the SQL column names safe to sort by in
// GetCollectionListPageData. Unrecognised column → loud error (A2 guard).
var collectionSortableSQLCols = []string{
	"tc.date_created",
	"tc.date_modified",
	"tc.name",
	"tc.amount",
	"tc.status",
	"tc.payment_date",
	"tc.reference_number",
}

// collectionViewToSQLColMap translates view-facing sort column keys to SQL column names.
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
	registry.RegisterRepositoryFactory("mysql", entityid.TreasuryCollection, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLCollectionRepository(dbOps, tableName), nil
	})
}

// MySQLCollectionRepository implements collection CRUD operations using MySQL 8.0+.
type MySQLCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLCollectionRepository creates a new MySQL collection repository.
func NewMySQLCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionpb.CollectionDomainServiceServer {
	if tableName == "" {
		tableName = "treasury_collection"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLCollectionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollection creates a new collection record.
func (r *MySQLCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
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

	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionpb.CreateCollectionResponse{Success: true, Data: []*collectionpb.Collection{collection}}, nil
}

// ReadCollection retrieves a collection record by ID.
func (r *MySQLCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionpb.ReadCollectionResponse{Success: true, Data: []*collectionpb.Collection{collection}}, nil
}

// UpdateCollection updates a collection record.
func (r *MySQLCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
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

	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionpb.UpdateCollectionResponse{Success: true, Data: []*collectionpb.Collection{collection}}, nil
}

// DeleteCollection deletes a collection record (soft delete).
func (r *MySQLCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}
	return &collectionpb.DeleteCollectionResponse{Success: true}, nil
}

// ListCollections lists collection records with optional filters.
func (r *MySQLCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
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
	return &collectionpb.ListCollectionsResponse{Success: true, Data: collections}, nil
}

// GetCollectionListPageData retrieves collections with pagination, filtering, sorting, and search.
//
// Dialect changes from postgres gold standard:
//   - $1/$2/$3/$4 → ? (positional, re-sequenced in arg order)
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1 (MySQL TINYINT boolean)
//   - $2::text IS NULL OR $2::text = ” → ? IS NULL OR ? = ” — MySQL does not need cast
//   - mysqlCore.BuildOrderBy for backtick-quoted ORDER BY
//   - LIMIT ? OFFSET ? at end (MySQL syntax, same as postgres)
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLCollectionRepository) GetCollectionListPageData(
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

	sortField := "tc.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}
	if mapped, ok := collectionViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}
	if sortField != "" && !slices.Contains(collectionSortableSQLCols, sortField) {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortField, "collection", collectionSortableSQLCols)
	}

	// Dialect: active = 1 (MySQL); LIKE not ILIKE; ? placeholders.
	// Args: workspaceID, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset
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
			WHERE tc.active = 1
			  AND tc.workspace_id = ?
			  AND (? IS NULL OR ? = '' OR
			       tc.name LIKE ? OR
			       tc.reference_number LIKE ? OR
			       tc.status LIKE ? OR
			       tc.collection_type LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT ? OFFSET ?;
	`

	// Args: workspaceID, searchPattern (null check x2), searchPattern x4, limit, offset
	queryArgs := []any{workspaceID, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset}

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
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

		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active, &name,
			&subscriptionID, &amount, &status, &revenueID, &collectionMethodID,
			&currency, &referenceNumber, &paymentDate, &receivedBy, &receivedRole,
			&collectionType, &advanceKind, &advanceStatus, &advanceStartDate, &advanceEndDate,
			&advancePeriodCount, &advancePeriodUnit, &advanceTotalAmount, &advanceRemainingAmount,
			&advanceRecognizedAmount, &advanceBalanceAccountID, &advanceTargetAccountID,
			&advanceExpiryDate, &advanceProrationPolicy, &clientID, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection row: %w", err)
		}

		totalCount = total
		collection := buildCollectionFromScan(
			id, dateCreated, dateModified, active, name,
			subscriptionID, amount, status, revenueID, collectionMethodID,
			currency, referenceNumber, paymentDate, receivedBy, receivedRole, collectionType,
			advanceKind, advanceStatus, advanceStartDate, advanceEndDate,
			advancePeriodCount, advancePeriodUnit, advanceTotalAmount, advanceRemainingAmount,
			advanceRecognizedAmount, advanceBalanceAccountID, advanceTargetAccountID,
			advanceExpiryDate, advanceProrationPolicy, clientID,
		)
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
//
// Dialect changes: $1/$2 → ? (positional); active = true → active = 1.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLCollectionRepository) GetCollectionItemPageData(
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

	// Args: collectionId, workspaceID — same order as postgres.
	const query = `
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
			WHERE tc.id = ? AND tc.workspace_id = ? AND tc.active = 1
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
		&id, &dateCreated, &dateModified, &active, &name,
		&subscriptionID, &amount, &status, &revenueID, &collectionMethodID,
		&currency, &referenceNumber, &paymentDate, &receivedBy, &receivedRole,
		&collectionType, &advanceKind, &advanceStatus, &advanceStartDate, &advanceEndDate,
		&advancePeriodCount, &advancePeriodUnit, &advanceTotalAmount, &advanceRemainingAmount,
		&advanceRecognizedAmount, &advanceBalanceAccountID, &advanceTargetAccountID,
		&advanceExpiryDate, &advanceProrationPolicy, &clientID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection with ID '%s' not found", req.CollectionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query collection item page data: %w", err)
	}

	collection := buildCollectionFromScan(
		id, dateCreated, dateModified, active, name,
		subscriptionID, amount, status, revenueID, collectionMethodID,
		currency, referenceNumber, paymentDate, receivedBy, receivedRole, collectionType,
		advanceKind, advanceStatus, advanceStartDate, advanceEndDate,
		advancePeriodCount, advancePeriodUnit, advanceTotalAmount, advanceRemainingAmount,
		advanceRecognizedAmount, advanceBalanceAccountID, advanceTargetAccountID,
		advanceExpiryDate, advanceProrationPolicy, clientID,
	)

	return &collectionpb.GetCollectionItemPageDataResponse{Collection: collection, Success: true}, nil
}

// buildCollectionFromScan constructs a Collection protobuf from scanned SQL fields.
// Column order and set are preserved exactly from the postgres gold standard.
func buildCollectionFromScan(
	id string, dateCreated, dateModified time.Time, active bool, name string,
	subscriptionID *string, amount int64, status, revenueID, collectionMethodID,
	currency, referenceNumber *string, paymentDate *time.Time, receivedBy, receivedRole, collectionType *string,
	advanceKind, advanceStatus sql.NullInt32,
	advanceStartDate, advanceEndDate *string,
	advancePeriodCount sql.NullInt32, advancePeriodUnit *string,
	advanceTotalAmount, advanceRemainingAmount, advanceRecognizedAmount sql.NullInt64,
	advanceBalanceAccountID, advanceTargetAccountID, advanceExpiryDate *string,
	advanceProrationPolicy sql.NullInt32, clientID *string,
) *collectionpb.Collection {
	c := &collectionpb.Collection{Id: id, Active: active, Name: name, Amount: amount}
	if subscriptionID != nil {
		c.SubscriptionId = *subscriptionID
	}
	if status != nil {
		c.Status = *status
	}
	if revenueID != nil {
		c.RevenueId = *revenueID
	}
	if collectionMethodID != nil {
		c.CollectionMethodId = *collectionMethodID
	}
	if currency != nil {
		c.Currency = *currency
	}
	if referenceNumber != nil {
		c.ReferenceNumber = *referenceNumber
	}
	if receivedBy != nil {
		c.ReceivedBy = *receivedBy
	}
	if receivedRole != nil {
		c.ReceivedRole = *receivedRole
	}
	if collectionType != nil {
		c.CollectionType = *collectionType
	}
	if paymentDate != nil && !paymentDate.IsZero() {
		c.PaymentDate = paymentDate.Format("2006-01-02")
	}
	assignAdvanceFieldsCollection(c,
		advanceKind, advanceStatus, advanceStartDate, advanceEndDate,
		advancePeriodCount, advancePeriodUnit,
		advanceTotalAmount, advanceRemainingAmount, advanceRecognizedAmount,
		advanceBalanceAccountID, advanceTargetAccountID, advanceExpiryDate,
		advanceProrationPolicy, clientID,
	)
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		c.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		c.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		c.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		c.DateModifiedString = &dmStr
	}
	return c
}

// assignAdvanceFieldsCollection folds the optional advance_* schedule columns into the Collection proto.
func assignAdvanceFieldsCollection(
	out *collectionpb.Collection,
	advanceKind sql.NullInt32, advanceStatus sql.NullInt32,
	advanceStartDate, advanceEndDate *string,
	advancePeriodCount sql.NullInt32, advancePeriodUnit *string,
	advanceTotalAmount, advanceRemainingAmount, advanceRecognizedAmount sql.NullInt64,
	advanceBalanceAccountID, advanceTargetAccountID, advanceExpiryDate *string,
	advanceProrationPolicy sql.NullInt32, clientID *string,
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

// ListByClient lists collections for a given client by joining treasury_collection to revenue.
// Workspace isolation is applied automatically.
//
// Dialect changes: $1/$2 → ?; $2::text = ” → ? = ” (MySQL no cast needed).
func (r *MySQLCollectionRepository) ListByClient(ctx context.Context, req *collectionpb.ListByClientRequest) (*collectionpb.ListByClientResponse, error) {
	if req.GetClientId() == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	// Dialect: ? placeholders; no ::text cast; LIKE not ILIKE.
	const query = `
		SELECT c.id, c.active, c.revenue_id, c.amount, c.status, c.currency,
		       c.reference_number, c.payment_date, c.collection_type
		FROM treasury_collection c
		JOIN revenue r ON r.id = c.revenue_id
		WHERE r.client_id = ?
		  AND (? = '' OR r.workspace_id = ?)`

	rows, err := r.db.QueryContext(ctx, query, req.GetClientId(), wsID, wsID)
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

// NewCollectionRepository creates a new MySQL collection repository (old-style constructor).
func NewCollectionRepository(db *sql.DB, tableName string) collectionpb.CollectionDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLCollectionRepository(dbOps, tableName)
}
