//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"math"
	"strings"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresBalanceRepository implements balance CRUD operations using PostgreSQL
type PostgresBalanceRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// SupportsBalanceServerPagination reports that this repository can execute
// balance list-page filtering, sorting, and pagination in PostgreSQL.
func (r *PostgresBalanceRepository) SupportsBalanceServerPagination() bool {
	return true
}

type balanceExecutorProvider interface {
	GetExecutor(context.Context) sqlexec.DBExecutor
}

var balanceListFilterFieldMap = map[string]string{
	"amount":          "b.amount",
	"client_id":       "b.client_id",
	"subscription_id": "b.subscription_id",
	"currency":        "b.currency",
	"balance_type":    "b.balance_type",
	"active":          "b.active",
	"date_created":    "b.date_created",
	"date_modified":   "b.date_modified",
}

var balanceListSortableColumns = []string{
	"amount",
	"client_id",
	"subscription_id",
	"currency",
	"balance_type",
	"date_created",
	"date_modified",
}

type balanceListQueries struct {
	countSQL  string
	dataSQL   string
	countArgs []any
	dataArgs  []any
	limit     int32
	offset    int32
	page      int32
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Balance, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres balance repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresBalanceRepository(dbOps, tableName), nil
	})
}

// NewPostgresBalanceRepository creates a new PostgreSQL balance repository
func NewPostgresBalanceRepository(dbOps interfaces.DatabaseOperation, tableName string) balancepb.BalanceDomainServiceServer {
	if tableName == "" {
		tableName = "balance" // default fallback
	}
	return &PostgresBalanceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateBalance creates a new balance using common PostgreSQL operations
func (r *PostgresBalanceRepository) CreateBalance(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("balance data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	balance := &balancepb.Balance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, balance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &balancepb.CreateBalanceResponse{
		Data: []*balancepb.Balance{balance},
	}, nil
}

// ReadBalance retrieves a balance using common PostgreSQL operations
func (r *PostgresBalanceRepository) ReadBalance(ctx context.Context, req *balancepb.ReadBalanceRequest) (*balancepb.ReadBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	balance := &balancepb.Balance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, balance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &balancepb.ReadBalanceResponse{
		Data: []*balancepb.Balance{balance},
	}, nil
}

// UpdateBalance updates a balance using common PostgreSQL operations
func (r *PostgresBalanceRepository) UpdateBalance(ctx context.Context, req *balancepb.UpdateBalanceRequest) (*balancepb.UpdateBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	balance := &balancepb.Balance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, balance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &balancepb.UpdateBalanceResponse{
		Data: []*balancepb.Balance{balance},
	}, nil
}

// DeleteBalance deletes a balance using common PostgreSQL operations
func (r *PostgresBalanceRepository) DeleteBalance(ctx context.Context, req *balancepb.DeleteBalanceRequest) (*balancepb.DeleteBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete balance: %w", err)
	}

	return &balancepb.DeleteBalanceResponse{
		Success: true,
	}, nil
}

// ListBalances lists balances using common PostgreSQL operations
func (r *PostgresBalanceRepository) ListBalances(ctx context.Context, req *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list balances: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var balances []*balancepb.Balance
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		balance := &balancepb.Balance{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, balance); err != nil {
			// Log error and continue with next item
			continue
		}
		balances = append(balances, balance)
	}

	return &balancepb.ListBalancesResponse{
		Data: balances,
	}, nil
}

// GetBalanceListPageData returns one real database page. Balance has no direct
// workspace_id column, so the mandatory tenant anchor is the joined subscription;
// client enrichment is independently constrained to the same workspace. Search,
// filters, and sort identifiers are adapter-owned allowlists, while every value is
// bound. Count and data reads share the active transaction executor when present.
func (r *PostgresBalanceRepository) GetBalanceListPageData(ctx context.Context, req *balancepb.GetBalanceListPageDataRequest) (*balancepb.GetBalanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	requestIdentity, err := identity.RequireWorkspace(ctx)
	if err != nil {
		return nil, fmt.Errorf("balance list workspace: %w", err)
	}
	queries, err := buildBalanceListQueries(req, requestIdentity.WorkspaceID)
	if err != nil {
		return nil, err
	}
	executorProvider, ok := r.dbOps.(balanceExecutorProvider)
	if !ok {
		return nil, fmt.Errorf("balance list requires a transaction-aware PostgreSQL executor")
	}
	exec := executorProvider.GetExecutor(ctx)

	var totalItems int64
	if err := exec.QueryRowContext(ctx, queries.countSQL, queries.countArgs...).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("failed to count balance list: %w", err)
	}
	if totalItems > math.MaxInt32 {
		return nil, fmt.Errorf("balance list total exceeds response capacity: %d", totalItems)
	}

	rows, err := exec.QueryContext(ctx, queries.dataSQL, queries.dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute balance list query: %w", err)
	}
	defer rows.Close()

	// Scan results
	var balances []*balancepb.Balance
	for rows.Next() {
		var (
			id               string
			amount           int64
			dateCreated      sql.NullInt64
			dateModified     sql.NullInt64
			active           bool
			clientID         string
			subscriptionID   string
			currency         string
			balanceType      string
			subscriptionData []byte
			clientData       []byte
		)

		err := rows.Scan(
			&id,
			&amount,
			&dateCreated,
			&dateModified,
			&active,
			&clientID,
			&subscriptionID,
			&currency,
			&balanceType,
			&subscriptionData,
			&clientData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance row: %w", err)
		}

		balance := &balancepb.Balance{
			Id:             id,
			Amount:         amount,
			Active:         active,
			ClientId:       clientID,
			SubscriptionId: subscriptionID,
			Currency:       currency,
			BalanceType:    balanceType,
		}

		// Handle nullable fields
		if dateCreated.Valid {
			balance.DateCreated = &dateCreated.Int64
		}
		if dateModified.Valid {
			balance.DateModified = &dateModified.Int64
		}

		// Unmarshal subscription data if present
		if len(subscriptionData) > 0 {
			var subscriptionMap map[string]any
			if err := json.Unmarshal(subscriptionData, &subscriptionMap); err == nil {
				subscriptionJSON, _ := json.Marshal(subscriptionMap)
				sub := &subscriptionpb.Subscription{}
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(subscriptionJSON, sub); err == nil {
					balance.Subscription = sub
				}
			}
		}

		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating balance rows: %w", err)
	}

	totalPages := int32(0)
	if totalItems > 0 {
		totalPages = int32((totalItems + int64(queries.limit) - 1) / int64(queries.limit))
	}
	currentPage := queries.page
	pagination := &commonpb.PaginationResponse{
		TotalItems:  int32(totalItems),
		CurrentPage: &currentPage,
		TotalPages:  &totalPages,
		HasNext:     currentPage < totalPages,
		HasPrev:     currentPage > 1,
	}

	return &balancepb.GetBalanceListPageDataResponse{
		BalanceList:   balances,
		Pagination:    pagination,
		SearchResults: []*commonpb.SearchResult{},
		Success:       true,
	}, nil
}

func buildBalanceListQueries(req *balancepb.GetBalanceListPageDataRequest, workspaceID string) (*balanceListQueries, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("balance list workspace is required")
	}

	limit, offset, page, err := postgresCore.BoundedOffsetPagination(req.GetPagination(), 50)
	if err != nil {
		return nil, fmt.Errorf("bounded balance pagination: %w", err)
	}
	clauses, filterArgs, nextArg, err := postgresCore.BuildFilterWhereMapped(
		req.GetFilters(),
		req.GetSearch(),
		balanceListFilterFieldMap,
		[]string{"b.currency", "b.balance_type"},
		2,
	)
	if err != nil {
		return nil, fmt.Errorf("bounded balance filters/search: %w", err)
	}
	orderBy, err := postgresCore.BuildOrderBy(balanceListSortableColumns, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("bounded balance sort: %w", err)
	}

	fromWhere := ` FROM ` + entityid.Balance + ` b
		JOIN ` + entityid.Subscription + ` s
		  ON s.id = b.subscription_id AND s.workspace_id = $1
		LEFT JOIN ` + entityid.Client + ` c
		  ON c.id = b.client_id AND c.workspace_id = $1
		WHERE b.active = true`
	if len(clauses) > 0 {
		fromWhere += " AND " + strings.Join(clauses, " AND ")
	}

	countArgs := make([]any, 0, 1+len(filterArgs))
	countArgs = append(countArgs, workspaceID)
	countArgs = append(countArgs, filterArgs...)
	dataArgs := append([]any(nil), countArgs...)
	dataArgs = append(dataArgs, limit, offset)

	return &balanceListQueries{
		countSQL: "SELECT COUNT(*)" + fromWhere,
		dataSQL: `WITH enriched AS (
		SELECT
			b.id,
			b.amount,
			b.date_created,
			b.date_modified,
			b.active,
			b.client_id,
			b.subscription_id,
			b.currency,
			b.balance_type,
			row_to_json(s.*) AS subscription_data,
			row_to_json(c.*) AS client_data` + fromWhere + `
		)
		SELECT * FROM enriched ` + orderBy +
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", nextArg, nextArg+1),
		countArgs: countArgs,
		dataArgs:  dataArgs,
		limit:     limit,
		offset:    offset,
		page:      page,
	}, nil
}

// GetBalanceItemPageData retrieves one balance through the same mandatory
// subscription workspace anchor as the list query.
func (r *PostgresBalanceRepository) GetBalanceItemPageData(ctx context.Context, req *balancepb.GetBalanceItemPageDataRequest) (*balancepb.GetBalanceItemPageDataResponse, error) {
	if req == nil || req.BalanceId == "" {
		return nil, fmt.Errorf("balance ID is required")
	}
	requestIdentity, err := identity.RequireWorkspace(ctx)
	if err != nil {
		return nil, fmt.Errorf("balance item workspace: %w", err)
	}
	executorProvider, ok := r.dbOps.(balanceExecutorProvider)
	if !ok {
		return nil, fmt.Errorf("balance item requires a transaction-aware PostgreSQL executor")
	}
	row := executorProvider.GetExecutor(ctx).QueryRowContext(
		ctx,
		balanceItemPageDataSQL(),
		req.BalanceId,
		requestIdentity.WorkspaceID,
	)

	var (
		id               string
		amount           int64
		dateCreated      sql.NullInt64
		dateModified     sql.NullInt64
		active           bool
		clientID         string
		subscriptionID   string
		currency         string
		balanceType      string
		subscriptionData []byte
		clientData       []byte
	)

	err = row.Scan(
		&id,
		&amount,
		&dateCreated,
		&dateModified,
		&active,
		&clientID,
		&subscriptionID,
		&currency,
		&balanceType,
		&subscriptionData,
		&clientData,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("balance not found with ID: %s", req.BalanceId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve balance: %w", err)
	}

	balance := &balancepb.Balance{
		Id:             id,
		Amount:         amount,
		Active:         active,
		ClientId:       clientID,
		SubscriptionId: subscriptionID,
		Currency:       currency,
		BalanceType:    balanceType,
	}
	if dateCreated.Valid {
		balance.DateCreated = &dateCreated.Int64
	}
	if dateModified.Valid {
		balance.DateModified = &dateModified.Int64
	}

	if len(subscriptionData) > 0 {
		var subscriptionMap map[string]any
		if err := json.Unmarshal(subscriptionData, &subscriptionMap); err == nil {
			subscriptionJSON, _ := json.Marshal(subscriptionMap)
			sub := &subscriptionpb.Subscription{}
			if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(subscriptionJSON, sub); err == nil {
				balance.Subscription = sub
			}
		}
	}

	return &balancepb.GetBalanceItemPageDataResponse{
		Balance: balance,
		Success: true,
	}, nil
}

func balanceItemPageDataSQL() string {
	return `
		WITH enriched AS (
			SELECT
				b.id,
				b.amount,
				b.date_created,
				b.date_modified,
				b.active,
				b.client_id,
				b.subscription_id,
				b.currency,
				b.balance_type,
				row_to_json(s.*) as subscription_data,
				row_to_json(c.*) as client_data
			FROM ` + entityid.Balance + ` b
			JOIN ` + entityid.Subscription + ` s
			  ON s.id = b.subscription_id AND s.workspace_id = $2
			LEFT JOIN ` + entityid.Client + ` c
			  ON c.id = b.client_id AND c.workspace_id = $2
			WHERE b.id = $1 AND b.active = true
		)
		SELECT * FROM enriched
		LIMIT 1
	`
}

// NewBalanceRepository creates a new PostgreSQL balance repository (old-style constructor)
func NewBalanceRepository(db *sql.DB, tableName string) balancepb.BalanceDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresBalanceRepository(dbOps, tableName)
}
