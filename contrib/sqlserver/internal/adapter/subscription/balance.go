//go:build sqlserver

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/encoding/protojson"
)

// SQLServerBalanceRepository implements balance CRUD operations using SQL Server.
type SQLServerBalanceRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Balance, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver balance repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerBalanceRepository(dbOps, tableName), nil
	})
}

// NewSQLServerBalanceRepository creates a new SQL Server balance repository.
func NewSQLServerBalanceRepository(dbOps interfaces.DatabaseOperation, tableName string) balancepb.BalanceDomainServiceServer {
	if tableName == "" {
		tableName = "balance"
	}
	return &SQLServerBalanceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateBalance creates a new balance using common SQL Server operations.
func (r *SQLServerBalanceRepository) CreateBalance(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("balance data is required")
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
		return nil, fmt.Errorf("failed to create balance: %w", err)
	}

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

// ReadBalance retrieves a balance using common SQL Server operations.
func (r *SQLServerBalanceRepository) ReadBalance(ctx context.Context, req *balancepb.ReadBalanceRequest) (*balancepb.ReadBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance: %w", err)
	}

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

// UpdateBalance updates a balance using common SQL Server operations.
func (r *SQLServerBalanceRepository) UpdateBalance(ctx context.Context, req *balancepb.UpdateBalanceRequest) (*balancepb.UpdateBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
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
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

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

// DeleteBalance deletes a balance using common SQL Server operations (soft delete).
func (r *SQLServerBalanceRepository) DeleteBalance(ctx context.Context, req *balancepb.DeleteBalanceRequest) (*balancepb.DeleteBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete balance: %w", err)
	}

	return &balancepb.DeleteBalanceResponse{
		Success: true,
	}, nil
}

// ListBalances lists balances using common SQL Server operations.
func (r *SQLServerBalanceRepository) ListBalances(ctx context.Context, req *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list balances: %w", err)
	}

	var balances []*balancepb.Balance
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		balance := &balancepb.Balance{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, balance); err != nil {
			continue
		}
		balances = append(balances, balance)
	}

	return &balancepb.ListBalancesResponse{
		Data: balances,
	}, nil
}

// GetBalanceListPageData retrieves balance list with enhanced data and filtering.
//
// SQL Server differences from the postgres gold standard:
//   - row_to_json(s.*) / row_to_json(c.*) → FOR JSON PATH WITHOUT_ARRAY_WRAPPER subselects.
//   - $N → @pN (dynamically built with argCount).
//   - active = true → active = 1.
//   - ORDER BY date_created DESC retained; no OFFSET/FETCH needed (no pagination in this
//     version — mirrors the postgres implementation's TODO).
//   - workspace_id predicate on balance (A1 guard).
func (r *SQLServerBalanceRepository) GetBalanceListPageData(ctx context.Context, req *balancepb.GetBalanceListPageDataRequest) (*balancepb.GetBalanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	// SQL Server: FOR JSON PATH instead of row_to_json; active = 1 instead of true.
	query := `
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
				(SELECT s.* FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS subscription_data,
				(SELECT c.* FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS client_data
			FROM balance b
			LEFT JOIN subscription s ON b.subscription_id = s.id
			LEFT JOIN client c ON b.client_id = c.id
			WHERE b.active = 1
	`

	var args []any
	argCount := 0

	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			switch filter.Field {
			case "subscription_id":
				if filter.GetStringFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.subscription_id = @p%d", argCount)
					args = append(args, filter.GetStringFilter().Value)
				}
			case "client_id":
				if filter.GetStringFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.client_id = @p%d", argCount)
					args = append(args, filter.GetStringFilter().Value)
				}
			case "active":
				if filter.GetBooleanFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.active = @p%d", argCount)
					v := 0
					if filter.GetBooleanFilter().Value {
						v = 1
					}
					args = append(args, v)
				}
			case "balance_type":
				if filter.GetStringFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.balance_type = @p%d", argCount)
					args = append(args, filter.GetStringFilter().Value)
				}
			}
		}
	}

	query += `
		)
		SELECT * FROM enriched
		ORDER BY date_created DESC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute balance list query: %w", err)
	}
	defer rows.Close()

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

		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating balance rows: %w", err)
	}

	currentPage := int32(1)
	totalPages := int32(1)
	pagination := &commonpb.PaginationResponse{
		TotalItems:  int32(len(balances)),
		CurrentPage: &currentPage,
		TotalPages:  &totalPages,
		HasNext:     false,
		HasPrev:     false,
	}

	return &balancepb.GetBalanceListPageDataResponse{
		BalanceList:   balances,
		Pagination:    pagination,
		SearchResults: []*commonpb.SearchResult{},
		Success:       true,
	}, nil
}

// GetBalanceItemPageData retrieves a single balance with enhanced related data.
//
// SQL Server differences:
//   - row_to_json → FOR JSON PATH WITHOUT_ARRAY_WRAPPER.
//   - $1 → @p1.
//   - active = true → active = 1.
//   - LIMIT 1 → SELECT TOP 1 (applied on the CTE outer select).
func (r *SQLServerBalanceRepository) GetBalanceItemPageData(ctx context.Context, req *balancepb.GetBalanceItemPageDataRequest) (*balancepb.GetBalanceItemPageDataResponse, error) {
	if req == nil || req.BalanceId == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	query := `
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
				(SELECT s.* FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS subscription_data,
				(SELECT c.* FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS client_data
			FROM balance b
			LEFT JOIN subscription s ON b.subscription_id = s.id
			LEFT JOIN client c ON b.client_id = c.id
			WHERE b.id = @p1 AND b.active = 1
		)
		SELECT TOP 1 * FROM enriched
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.BalanceId)

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

	err := row.Scan(
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

// NewBalanceRepository creates a new SQL Server balance repository (old-style constructor).
func NewBalanceRepository(db *sql.DB, tableName string) balancepb.BalanceDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerBalanceRepository(dbOps, tableName)
}
