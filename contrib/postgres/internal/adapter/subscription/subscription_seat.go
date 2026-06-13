//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
	"google.golang.org/protobuf/encoding/protojson"
)

// subscriptionSeatSortableSQLCols is the sort-column whitelist that
// core.BuildOrderBy validates GetSubscriptionSeatListPageData requests against
// (A2 fail-closed guard). These match the columns projected by the list SELECT.
var subscriptionSeatSortableSQLCols = []string{
	"position",
	"status",
	"date_start",
	"date_end",
	"contracted_amount",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionSeat, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_seat repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionSeatRepository(dbOps, tableName), nil
	})
}

// PostgresSubscriptionSeatRepository implements subscription_seat CRUD operations using PostgreSQL.
//
// The seat models one staff member assigned to a subscription. The partial
// UNIQUE (subscription_id, position) WHERE status='active' constraint and the
// immutable-contracted_amount-while-active trigger are enforced at the DB; the
// SR-2 lifecycle arcs and atomic replace are enforced in the use case layer.
type PostgresSubscriptionSeatRepository struct {
	subscriptionseatpb.UnimplementedSubscriptionSeatDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSubscriptionSeatRepository creates a new PostgreSQL subscription seat repository
func NewPostgresSubscriptionSeatRepository(dbOps interfaces.DatabaseOperation, tableName string) subscriptionseatpb.SubscriptionSeatDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_seat"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresSubscriptionSeatRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSubscriptionSeat creates a new subscription seat using common PostgreSQL operations
func (r *PostgresSubscriptionSeatRepository) CreateSubscriptionSeat(ctx context.Context, req *subscriptionseatpb.CreateSubscriptionSeatRequest) (*subscriptionseatpb.CreateSubscriptionSeatResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription seat data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// The DB status column is a lowercase CHECK-pinned token; protojson serializes
	// the proto enum to its SCREAMING name. Translate so the write satisfies the
	// subscription_seat_status_chk constraint.
	subscriptionSeatStatusEnumToToken(req.Data.Status, data)

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription seat: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionSeat := subscriptionSeatFromResultJSON(resultJSON)

	return &subscriptionseatpb.CreateSubscriptionSeatResponse{
		Data:    []*subscriptionseatpb.SubscriptionSeat{subscriptionSeat},
		Success: true,
	}, nil
}

// ReadSubscriptionSeat retrieves a subscription seat using common PostgreSQL operations
func (r *PostgresSubscriptionSeatRepository) ReadSubscriptionSeat(ctx context.Context, req *subscriptionseatpb.ReadSubscriptionSeatRequest) (*subscriptionseatpb.ReadSubscriptionSeatResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription seat ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription seat: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionSeat := subscriptionSeatFromResultJSON(resultJSON)

	return &subscriptionseatpb.ReadSubscriptionSeatResponse{
		Data:    []*subscriptionseatpb.SubscriptionSeat{subscriptionSeat},
		Success: true,
	}, nil
}

// UpdateSubscriptionSeat updates a subscription seat using common PostgreSQL operations
func (r *PostgresSubscriptionSeatRepository) UpdateSubscriptionSeat(ctx context.Context, req *subscriptionseatpb.UpdateSubscriptionSeatRequest) (*subscriptionseatpb.UpdateSubscriptionSeatResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription seat ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// Translate the proto enum to the DB's lowercase CHECK-pinned token.
	subscriptionSeatStatusEnumToToken(req.Data.Status, data)

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription seat: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionSeat := subscriptionSeatFromResultJSON(resultJSON)

	return &subscriptionseatpb.UpdateSubscriptionSeatResponse{
		Data:    []*subscriptionseatpb.SubscriptionSeat{subscriptionSeat},
		Success: true,
	}, nil
}

// DeleteSubscriptionSeat soft-deletes a subscription seat (sets active=false).
// The seat carries a multi-stage status lifecycle, so a soft delete keeps the
// row for history while removing it from active surfaces.
func (r *PostgresSubscriptionSeatRepository) DeleteSubscriptionSeat(ctx context.Context, req *subscriptionseatpb.DeleteSubscriptionSeatRequest) (*subscriptionseatpb.DeleteSubscriptionSeatResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription seat ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription seat: %w", err)
	}

	return &subscriptionseatpb.DeleteSubscriptionSeatResponse{
		Success: true,
	}, nil
}

// ListSubscriptionSeats lists subscription seats using common PostgreSQL operations.
// Supports filters by subscription_id and client_id (the IDOR-scoping denorm) via
// the request's common.FilterRequest.
func (r *PostgresSubscriptionSeatRepository) ListSubscriptionSeats(ctx context.Context, req *subscriptionseatpb.ListSubscriptionSeatsRequest) (*subscriptionseatpb.ListSubscriptionSeatsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription seats: %w", err)
	}

	var subscriptionSeats []*subscriptionseatpb.SubscriptionSeat
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		subscriptionSeats = append(subscriptionSeats, subscriptionSeatFromResultJSON(resultJSON))
	}

	return &subscriptionseatpb.ListSubscriptionSeatsResponse{
		Data:    subscriptionSeats,
		Success: true,
	}, nil
}

// GetSubscriptionSeatListPageData retrieves paginated subscription seat list data.
// Tenancy scopes on the seat's own workspace_id column; the search matches on the
// human-readable text fields (role_title / position / status).
func (r *PostgresSubscriptionSeatRepository) GetSubscriptionSeatListPageData(ctx context.Context, req *subscriptionseatpb.GetSubscriptionSeatListPageDataRequest) (*subscriptionseatpb.GetSubscriptionSeatListPageDataResponse, error) {
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
	orderBy, err := postgresCore.BuildOrderBy(subscriptionSeatSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for subscription seat list: %w", err)
	}

	// subscription_seat carries its own workspace_id column; scope directly on it.
	// Empty wsID = service-to-service call → no scoping.
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	query := `SELECT id, subscription_id, staff_id, client_id, workspace_id, product_plan_id, product_variant_id, contracted_amount, contracted_currency, role_title, seniority, date_start, date_end, status, review_cadence_value, review_cadence_unit, position, replaces_id, work_request_id, active, date_created, date_modified
		FROM subscription_seat
		WHERE active = true
			AND ($4::text = '' OR workspace_id = $4::text)
			AND ($1::text IS NULL OR $1::text = '' OR COALESCE(role_title,'') ILIKE $1 OR COALESCE(position,'') ILIKE $1 OR status ILIKE $1) ` + orderBy + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var subscriptionSeats []*subscriptionseatpb.SubscriptionSeat
	for rows.Next() {
		seat, scanErr := scanSubscriptionSeatRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		subscriptionSeats = append(subscriptionSeats, seat)
	}
	return &subscriptionseatpb.GetSubscriptionSeatListPageDataResponse{SubscriptionSeatList: subscriptionSeats, Success: true}, nil
}

// GetSubscriptionSeatItemPageData retrieves subscription seat item page data
func (r *PostgresSubscriptionSeatRepository) GetSubscriptionSeatItemPageData(ctx context.Context, req *subscriptionseatpb.GetSubscriptionSeatItemPageDataRequest) (*subscriptionseatpb.GetSubscriptionSeatItemPageDataResponse, error) {
	if req == nil || req.SubscriptionSeatId == "" {
		return nil, fmt.Errorf("subscription seat ID required")
	}
	// Tenancy: scope on the seat's own workspace_id (IDOR defense — mirror the
	// list query at GetSubscriptionSeatListPageData and GetSubscriptionItemPageData).
	// Empty wsID = service-to-service call → no scoping.
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	query := `SELECT id, subscription_id, staff_id, client_id, workspace_id, product_plan_id, product_variant_id, contracted_amount, contracted_currency, role_title, seniority, date_start, date_end, status, review_cadence_value, review_cadence_unit, position, replaces_id, work_request_id, active, date_created, date_modified
		FROM subscription_seat WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.SubscriptionSeatId, wsID)
	seat, err := scanSubscriptionSeatRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription seat not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &subscriptionseatpb.GetSubscriptionSeatItemPageDataResponse{SubscriptionSeat: seat, Success: true}, nil
}

// seatExecutorProvider is the narrow type-assertion interface used to obtain a
// transaction-aware executor (mirrors the entity package's executorProvider).
type seatExecutorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// LockSubscriptionSeatForUpdate reads the seat by id WITH a row lock
// (SELECT ... FOR UPDATE), inside the caller's transaction, and returns it as a
// proto message. This is the SR-2 serialization primitive: the replace flow reads
// the OLD seat through this locked path so two concurrent replacers serialize —
// the second waits on the lock, then observes status=REPLACED and is rejected.
// It deliberately mirrors session_switch_principal.go's FOR UPDATE pattern.
//
// It MUST run inside an active transaction; it uses GetExecutor(ctx) so the lock
// participates in the transaction stored in ctx (a *sql.DB FOR UPDATE outside a
// tx releases the lock immediately and provides no serialization).
//
// Tenancy: scoped on workspace_id (empty wsID = service-to-service → no scope).
// Unlike the page-data reads it does NOT filter active=true: the replace flow must
// be able to lock a seat regardless of active flag to observe a concurrent
// REPLACED transition.
func (r *PostgresSubscriptionSeatRepository) LockSubscriptionSeatForUpdate(ctx context.Context, id string) (*subscriptionseatpb.SubscriptionSeat, error) {
	if id == "" {
		return nil, fmt.Errorf("subscription seat ID required")
	}
	exec, ok := r.dbOps.(seatExecutorProvider)
	if !ok {
		return nil, fmt.Errorf("subscription_seat adapter: dbOps does not provide a transaction-aware executor")
	}
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	query := `SELECT id, subscription_id, staff_id, client_id, workspace_id, product_plan_id, product_variant_id, contracted_amount, contracted_currency, role_title, seniority, date_start, date_end, status, review_cadence_value, review_cadence_unit, position, replaces_id, work_request_id, active, date_created, date_modified
		FROM subscription_seat WHERE id = $1 AND ($2::text = '' OR workspace_id = $2::text) FOR UPDATE`
	row := exec.GetExecutor(ctx).QueryRowContext(ctx, query, id, wsID)
	seat, err := scanSubscriptionSeatRow(row.Scan)
	if err == sql.ErrNoRows {
		// Not-found is signalled as (nil, nil) so the application-layer caller
		// stays free of a database/sql import (hexagonal rule).
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("lock subscription seat failed: %w", err)
	}
	return seat, nil
}

// scanSubscriptionSeatRow scans one subscription_seat row (column order must match
// the SELECT lists in GetSubscriptionSeatListPageData / ItemPageData) into a
// proto message, mapping nullable columns onto the optional proto fields and the
// status text column onto the SubscriptionSeatStatus enum.
func scanSubscriptionSeatRow(scan func(dest ...any) error) (*subscriptionseatpb.SubscriptionSeat, error) {
	var id, subscriptionId, staffId, clientId, workspaceId, productPlanId, statusStr string
	var productVariantId, reviewCadenceUnit sql.NullString
	var reviewCadenceValue sql.NullInt32
	var contractedAmount sql.NullInt64
	var contractedCurrency, roleTitle, seniority, position, replacesId, workRequestId sql.NullString
	var dateStart, dateEnd sql.NullInt64
	var active bool
	var dateCreated, dateModified time.Time
	if err := scan(&id, &subscriptionId, &staffId, &clientId, &workspaceId, &productPlanId, &productVariantId, &contractedAmount, &contractedCurrency, &roleTitle, &seniority, &dateStart, &dateEnd, &statusStr, &reviewCadenceValue, &reviewCadenceUnit, &position, &replacesId, &workRequestId, &active, &dateCreated, &dateModified); err != nil {
		return nil, err
	}
	seat := &subscriptionseatpb.SubscriptionSeat{
		Id:             id,
		SubscriptionId: subscriptionId,
		StaffId:        staffId,
		ClientId:       clientId,
		WorkspaceId:    workspaceId,
		ProductPlanId:  productPlanId,
		Status:         subscriptionSeatStatusFromString(statusStr),
		Active:         active,
	}
	if productVariantId.Valid && productVariantId.String != "" {
		v := productVariantId.String
		seat.ProductVariantId = &v
	}
	if reviewCadenceValue.Valid {
		v := reviewCadenceValue.Int32
		seat.ReviewCadenceValue = &v
	}
	if reviewCadenceUnit.Valid && reviewCadenceUnit.String != "" {
		v := reviewCadenceUnit.String
		seat.ReviewCadenceUnit = &v
	}
	if contractedAmount.Valid {
		v := contractedAmount.Int64
		seat.ContractedAmount = &v
	}
	if contractedCurrency.Valid && contractedCurrency.String != "" {
		v := contractedCurrency.String
		seat.ContractedCurrency = &v
	}
	if roleTitle.Valid && roleTitle.String != "" {
		v := roleTitle.String
		seat.RoleTitle = &v
	}
	if seniority.Valid && seniority.String != "" {
		v := seniority.String
		seat.Seniority = &v
	}
	if position.Valid && position.String != "" {
		v := position.String
		seat.Position = &v
	}
	if replacesId.Valid && replacesId.String != "" {
		v := replacesId.String
		seat.ReplacesId = &v
	}
	if workRequestId.Valid && workRequestId.String != "" {
		v := workRequestId.String
		seat.WorkRequestId = &v
	}
	if dateStart.Valid {
		v := dateStart.Int64
		seat.DateStart = &v
	}
	if dateEnd.Valid {
		v := dateEnd.Int64
		seat.DateEnd = &v
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		seat.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		seat.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		seat.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		seat.DateModifiedString = &dmStr
	}
	return seat, nil
}

// subscriptionSeatStatusFromString maps the DB CHECK-pinned status text column
// onto the proto SubscriptionSeatStatus enum. The DB stores lowercase tokens
// (proposed/active/replaced/ended); the proto enum value names are the SCREAMING
// form. Unknown / empty → UNSPECIFIED (fail-safe display).
func subscriptionSeatStatusFromString(s string) subscriptionseatpb.SubscriptionSeatStatus {
	switch s {
	case "proposed", "PROPOSED", "SUBSCRIPTION_SEAT_STATUS_PROPOSED":
		return subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_PROPOSED
	case "active", "ACTIVE", "SUBSCRIPTION_SEAT_STATUS_ACTIVE":
		return subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE
	case "replaced", "REPLACED", "SUBSCRIPTION_SEAT_STATUS_REPLACED":
		return subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_REPLACED
	case "ended", "ENDED", "SUBSCRIPTION_SEAT_STATUS_ENDED":
		return subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ENDED
	default:
		return subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_UNSPECIFIED
	}
}

// subscriptionSeatStatusTokenFromEnum maps the proto SubscriptionSeatStatus enum
// onto the DB's lowercase CHECK-pinned token. UNSPECIFIED has no valid token (the
// CHECK rejects it); callers must set a concrete status before persisting.
func subscriptionSeatStatusTokenFromEnum(s subscriptionseatpb.SubscriptionSeatStatus) string {
	switch s {
	case subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_PROPOSED:
		return "proposed"
	case subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE:
		return "active"
	case subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_REPLACED:
		return "replaced"
	case subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ENDED:
		return "ended"
	default:
		return ""
	}
}

// subscriptionSeatStatusEnumToToken overwrites the protojson-serialized status
// value in the write map with the DB lowercase token (or deletes it when the
// enum is UNSPECIFIED so a partial UPDATE does not write an invalid value).
func subscriptionSeatStatusEnumToToken(s subscriptionseatpb.SubscriptionSeatStatus, data map[string]any) {
	token := subscriptionSeatStatusTokenFromEnum(s)
	if token == "" {
		delete(data, "status")
		return
	}
	data["status"] = token
}

// subscriptionSeatFromResultJSON unmarshals an adapter result row (DB token form)
// into a proto SubscriptionSeat. The status column is the DB lowercase token; we
// strip it before protojson.Unmarshal (which would reject the unknown enum value)
// and set the enum field directly from the token.
func subscriptionSeatFromResultJSON(resultJSON []byte) *subscriptionseatpb.SubscriptionSeat {
	var raw map[string]any
	_ = json.Unmarshal(resultJSON, &raw)
	statusToken, _ := raw["status"].(string)
	delete(raw, "status")
	cleaned, _ := json.Marshal(raw)

	seat := &subscriptionseatpb.SubscriptionSeat{}
	_ = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(cleaned, seat)
	seat.Status = subscriptionSeatStatusFromString(statusToken)
	return seat
}

// NewSubscriptionSeatRepository creates a new PostgreSQL subscription_seat repository (old-style constructor)
func NewSubscriptionSeatRepository(db *sql.DB, tableName string) subscriptionseatpb.SubscriptionSeatDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresSubscriptionSeatRepository(dbOps, tableName)
}
