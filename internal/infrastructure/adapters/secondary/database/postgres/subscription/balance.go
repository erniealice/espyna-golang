//go:build postgres

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// PostgresBalanceRepository implements balance CRUD operations using PostgreSQL
type PostgresBalanceRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "balance", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres balance repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
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
	if err := protojson.Unmarshal(resultJSON, balance); err != nil {
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
	if err := protojson.Unmarshal(resultJSON, balance); err != nil {
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
	if err := protojson.Unmarshal(resultJSON, balance); err != nil {
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
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
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
		if err := protojson.Unmarshal(resultJSON, balance); err != nil {
			// Log error and continue with next item
			continue
		}
		balances = append(balances, balance)
	}

	return &balancepb.ListBalancesResponse{
		Data: balances,
	}, nil
}

// GetBalanceListPageData retrieves balance list with enhanced data and filtering
// Uses CTE pattern with JOINs to subscription and client tables
// Supports filtering by: subscription_id, client_id, active status
// TODO: Add unit tests for GetBalanceListPageData
// TODO: Add integration tests with various filter combinations
// TODO: Test pagination edge cases
func (r *PostgresBalanceRepository) GetBalanceListPageData(ctx context.Context, req *balancepb.GetBalanceListPageDataRequest) (*balancepb.GetBalanceListPageDataResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	// Build query with CTE pattern for joining related entities
	// Performance notes:
	// - Add index on balance.subscription_id (foreign key)
	// - Add index on balance.active for filtering
	// - Add index on balance.date_created for sorting
	// - Subscription and client tables should have indexes on their id columns
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
				row_to_json(s.*) as subscription_data,
				row_to_json(c.*) as client_data
			FROM balance b
			LEFT JOIN subscription s ON b.subscription_id = s.id
			LEFT JOIN client c ON b.client_id = c.id
			WHERE b.active = true
	`

	var args []any
	argCount := 0

	// Apply optional filters
	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			switch filter.Field {
			case "subscription_id":
				if filter.GetStringFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.subscription_id = $%d", argCount)
					args = append(args, filter.GetStringFilter().Value)
				}
			case "client_id":
				if filter.GetStringFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.client_id = $%d", argCount)
					args = append(args, filter.GetStringFilter().Value)
				}
			case "active":
				if filter.GetBooleanFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.active = $%d", argCount)
					args = append(args, filter.GetBooleanFilter().Value)
				}
			case "balance_type":
				if filter.GetStringFilter() != nil {
					argCount++
					query += fmt.Sprintf(" AND b.balance_type = $%d", argCount)
					args = append(args, filter.GetStringFilter().Value)
				}
			}
		}
	}

	// Close the CTE and select from it
	query += `
		)
		SELECT * FROM enriched
		ORDER BY date_created DESC
	`

	// Execute query using raw database access
	// Note: This bypasses dbOps.List() to use custom CTE query
	db := r.dbOps.(*postgresCore.PostgresOperations).GetDB()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute balance list query: %w", err)
	}
	defer rows.Close()

	// Scan results
	var balances []*balancepb.Balance
	for rows.Next() {
		var (
			id                 string
			amount             float64
			dateCreated        sql.NullInt64
			dateCreatedString  sql.NullString
			dateModified       sql.NullInt64
			dateModifiedString sql.NullString
			active             bool
			clientID           string
			subscriptionID     string
			currency           string
			balanceType        string
			subscriptionData   []byte
			clientData         []byte
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
		if dateCreatedString.Valid {
			balance.DateCreatedString = &dateCreatedString.String
		}
		if dateModified.Valid {
			balance.DateModified = &dateModified.Int64
		}
		if dateModifiedString.Valid {
			balance.DateModifiedString = &dateModifiedString.String
		}

		// Unmarshal subscription data if present
		if len(subscriptionData) > 0 {
			var subscriptionMap map[string]any
			if err := json.Unmarshal(subscriptionData, &subscriptionMap); err == nil {
				subscriptionJSON, _ := json.Marshal(subscriptionMap)
				sub := &subscriptionpb.Subscription{}
				if err := protojson.Unmarshal(subscriptionJSON, sub); err == nil {
					balance.Subscription = sub
				}
			}
		}

		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating balance rows: %w", err)
	}

	// For now, return simple pagination metadata
	// TODO: Implement proper pagination with limit/offset support
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

// GetBalanceItemPageData retrieves a single balance with enhanced related data
// Uses CTE pattern with JOINs to subscription and client tables
// TODO: Add unit tests for GetBalanceItemPageData
// TODO: Test with missing/null related entities
// TODO: Verify proper error handling for not found cases
func (r *PostgresBalanceRepository) GetBalanceItemPageData(ctx context.Context, req *balancepb.GetBalanceItemPageDataRequest) (*balancepb.GetBalanceItemPageDataResponse, error) {
	// Input validation
	if req == nil || req.BalanceId == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	// Build query with CTE pattern for joining related entities
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
				row_to_json(s.*) as subscription_data,
				row_to_json(c.*) as client_data
			FROM balance b
			LEFT JOIN subscription s ON b.subscription_id = s.id
			LEFT JOIN client c ON b.client_id = c.id
			WHERE b.id = $1 AND b.active = true
		)
		SELECT * FROM enriched
		LIMIT 1
	`

	// Execute query using raw database access
	db := r.dbOps.(*postgresCore.PostgresOperations).GetDB()
	row := db.QueryRowContext(ctx, query, req.BalanceId)

	var (
		id                 string
		amount             float64
		dateCreated        sql.NullInt64
		dateCreatedString  sql.NullString
		dateModified       sql.NullInt64
		dateModifiedString sql.NullString
		active             bool
		clientID           string
		subscriptionID     string
		currency           string
		balanceType        string
		subscriptionData   []byte
		clientData         []byte
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

	// Handle nullable fields
	if dateCreated.Valid {
		balance.DateCreated = &dateCreated.Int64
	}
	if dateCreatedString.Valid {
		balance.DateCreatedString = &dateCreatedString.String
	}
	if dateModified.Valid {
		balance.DateModified = &dateModified.Int64
	}
	if dateModifiedString.Valid {
		balance.DateModifiedString = &dateModifiedString.String
	}

	// Unmarshal subscription data if present
	if len(subscriptionData) > 0 {
		var subscriptionMap map[string]any
		if err := json.Unmarshal(subscriptionData, &subscriptionMap); err == nil {
			subscriptionJSON, _ := json.Marshal(subscriptionMap)
			sub := &subscriptionpb.Subscription{}
			if err := protojson.Unmarshal(subscriptionJSON, sub); err == nil {
				balance.Subscription = sub
			}
		}
	}

	return &balancepb.GetBalanceItemPageDataResponse{
		Balance: balance,
		Success: true,
	}, nil
}

// NewBalanceRepository creates a new PostgreSQL balance repository (old-style constructor)
func NewBalanceRepository(db *sql.DB, tableName string) balancepb.BalanceDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresBalanceRepository(dbOps, tableName)
}
