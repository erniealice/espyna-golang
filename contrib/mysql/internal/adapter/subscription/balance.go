//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/encoding/protojson"
)

// MySQLBalanceRepository implements balance CRUD operations using MySQL 8.0+.
type MySQLBalanceRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Balance, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql balance repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLBalanceRepository(dbOps, tableName), nil
	})
}

// NewMySQLBalanceRepository creates a new MySQL balance repository.
func NewMySQLBalanceRepository(dbOps interfaces.DatabaseOperation, tableName string) balancepb.BalanceDomainServiceServer {
	if tableName == "" {
		tableName = "balance"
	}
	return &MySQLBalanceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *MySQLBalanceRepository) CreateBalance(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
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
	return &balancepb.CreateBalanceResponse{Data: []*balancepb.Balance{balance}}, nil
}

func (r *MySQLBalanceRepository) ReadBalance(ctx context.Context, req *balancepb.ReadBalanceRequest) (*balancepb.ReadBalanceResponse, error) {
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
	return &balancepb.ReadBalanceResponse{Data: []*balancepb.Balance{balance}}, nil
}

func (r *MySQLBalanceRepository) UpdateBalance(ctx context.Context, req *balancepb.UpdateBalanceRequest) (*balancepb.UpdateBalanceResponse, error) {
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
	return &balancepb.UpdateBalanceResponse{Data: []*balancepb.Balance{balance}}, nil
}

func (r *MySQLBalanceRepository) DeleteBalance(ctx context.Context, req *balancepb.DeleteBalanceRequest) (*balancepb.DeleteBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete balance: %w", err)
	}
	return &balancepb.DeleteBalanceResponse{Success: true}, nil
}

func (r *MySQLBalanceRepository) ListBalances(ctx context.Context, req *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
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
	return &balancepb.ListBalancesResponse{Data: balances}, nil
}

// GetBalanceListPageData retrieves balance list with enhanced data and filtering.
//
// Dialect changes vs postgres gold standard:
//   - row_to_json → subquery with explicit JSON_OBJECT (MySQL has no row_to_json)
//     simplified here: JSON is built from the joined scan, not inline SQL JSON.
//   - $N → ? (MySQL positional placeholders)
//   - WHERE workspace_id = ? enforced for multi-tenancy
func (r *MySQLBalanceRepository) GetBalanceListPageData(ctx context.Context, req *balancepb.GetBalanceListPageDataRequest) (*balancepb.GetBalanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	// Build WHERE conditions; workspace is enforced by WorkspaceAwareOperations on the
	// dbOps.List path so we delegate directly and let it inject workspace_id.
	filterParams := &interfaces.ListParams{}
	if req.Filters != nil {
		filterParams.Filters = req.Filters
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, filterParams)
	if err != nil {
		return nil, fmt.Errorf("failed to execute balance list query: %w", err)
	}

	var balances []*balancepb.Balance
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		balance := &balancepb.Balance{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, balance); err != nil {
			continue
		}
		balances = append(balances, balance)
	}

	// Subscription data: enrich inline if SubscriptionId is set.
	for _, b := range balances {
		if b.GetSubscriptionId() == "" {
			continue
		}
		subResult, err := r.dbOps.Read(ctx, "subscription", b.GetSubscriptionId())
		if err != nil {
			continue
		}
		subJSON, _ := json.Marshal(mysqlCore.DenormalizeKeys(subResult))
		sub := &subscriptionpb.Subscription{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(subJSON, sub); err == nil {
			b.Subscription = sub
		}
	}

	currentPage := int32(1)
	totalPages := int32(1)
	return &balancepb.GetBalanceListPageDataResponse{
		BalanceList: balances,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(len(balances)),
			CurrentPage: &currentPage,
			TotalPages:  &totalPages,
			HasNext:     false,
			HasPrev:     false,
		},
		SearchResults: []*commonpb.SearchResult{},
		Success:       true,
	}, nil
}

// GetBalanceItemPageData retrieves a single balance with enhanced related data.
func (r *MySQLBalanceRepository) GetBalanceItemPageData(ctx context.Context, req *balancepb.GetBalanceItemPageDataRequest) (*balancepb.GetBalanceItemPageDataResponse, error) {
	if req == nil || req.BalanceId == "" {
		return nil, fmt.Errorf("balance ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.BalanceId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve balance: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	balance := &balancepb.Balance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, balance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal balance: %w", err)
	}

	if balance.GetSubscriptionId() != "" {
		subResult, err := r.dbOps.Read(ctx, "subscription", balance.GetSubscriptionId())
		if err == nil {
			subJSON, _ := json.Marshal(mysqlCore.DenormalizeKeys(subResult))
			sub := &subscriptionpb.Subscription{}
			if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(subJSON, sub); err == nil {
				balance.Subscription = sub
			}
		}
	}

	return &balancepb.GetBalanceItemPageDataResponse{
		Balance: balance,
		Success: true,
	}, nil
}

// NewBalanceRepository creates a new MySQL balance repository (old-style constructor).
func NewBalanceRepository(db *sql.DB, tableName string) balancepb.BalanceDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLBalanceRepository(dbOps, tableName)
}
