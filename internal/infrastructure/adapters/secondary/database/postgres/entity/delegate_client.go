//go:build postgres

package entity

import (
	"context"
	"database/sql"
	"time"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "delegate_client", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres delegate_client repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresDelegateClientRepository(dbOps, tableName), nil
	})
}

// PostgresDelegateClientRepository implements delegate client CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_delegate_client_active ON delegate_client(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_delegate_client_delegate_id ON delegate_client(delegate_id) - Filter by delegate
//   - CREATE INDEX idx_delegate_client_client_id ON delegate_client(client_id) - Filter by client
//   - CREATE INDEX idx_delegate_client_date_created ON delegate_client(date_created DESC) - Default sorting
type PostgresDelegateClientRepository struct {
	delegateclientpb.UnimplementedDelegateClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresDelegateClientRepository creates a new PostgreSQL delegate client repository
func NewPostgresDelegateClientRepository(dbOps interfaces.DatabaseOperation, tableName string) delegateclientpb.DelegateClientDomainServiceServer {
	if tableName == "" {
		tableName = "delegate_client" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresDelegateClientRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateDelegateClient creates a new delegate client using common PostgreSQL operations
func (r *PostgresDelegateClientRepository) CreateDelegateClient(ctx context.Context, req *delegateclientpb.CreateDelegateClientRequest) (*delegateclientpb.CreateDelegateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("delegate client data is required")
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
		return nil, fmt.Errorf("failed to create delegate client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegateClient := &delegateclientpb.DelegateClient{}
	if err := protojson.Unmarshal(resultJSON, delegateClient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegateclientpb.CreateDelegateClientResponse{
		Data: []*delegateclientpb.DelegateClient{delegateClient},
	}, nil
}

// ReadDelegateClient retrieves a delegate client using common PostgreSQL operations
func (r *PostgresDelegateClientRepository) ReadDelegateClient(ctx context.Context, req *delegateclientpb.ReadDelegateClientRequest) (*delegateclientpb.ReadDelegateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read delegate client: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegateClient := &delegateclientpb.DelegateClient{}
	if err := protojson.Unmarshal(resultJSON, delegateClient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegateclientpb.ReadDelegateClientResponse{
		Data: []*delegateclientpb.DelegateClient{delegateClient},
	}, nil
}

// UpdateDelegateClient updates a delegate client using common PostgreSQL operations
func (r *PostgresDelegateClientRepository) UpdateDelegateClient(ctx context.Context, req *delegateclientpb.UpdateDelegateClientRequest) (*delegateclientpb.UpdateDelegateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required")
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
		return nil, fmt.Errorf("failed to update delegate client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegateClient := &delegateclientpb.DelegateClient{}
	if err := protojson.Unmarshal(resultJSON, delegateClient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegateclientpb.UpdateDelegateClientResponse{
		Data: []*delegateclientpb.DelegateClient{delegateClient},
	}, nil
}

// DeleteDelegateClient deletes a delegate client using common PostgreSQL operations
func (r *PostgresDelegateClientRepository) DeleteDelegateClient(ctx context.Context, req *delegateclientpb.DeleteDelegateClientRequest) (*delegateclientpb.DeleteDelegateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete delegate client: %w", err)
	}

	return &delegateclientpb.DeleteDelegateClientResponse{
		Success: true,
	}, nil
}

// ListDelegateClients lists delegate clients using common PostgreSQL operations
func (r *PostgresDelegateClientRepository) ListDelegateClients(ctx context.Context, req *delegateclientpb.ListDelegateClientsRequest) (*delegateclientpb.ListDelegateClientsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list delegate clients: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var delegateClients []*delegateclientpb.DelegateClient
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		delegateClient := &delegateclientpb.DelegateClient{}
		if err := protojson.Unmarshal(resultJSON, delegateClient); err != nil {
			// Log error and continue with next item
			continue
		}
		delegateClients = append(delegateClients, delegateClient)
	}

	return &delegateclientpb.ListDelegateClientsResponse{
		Data: delegateClients,
	}, nil
}

// GetDelegateClientListPageData retrieves paginated delegate client list data with CTE
func (r *PostgresDelegateClientRepository) GetDelegateClientListPageData(ctx context.Context, req *delegateclientpb.GetDelegateClientListPageDataRequest) (*delegateclientpb.GetDelegateClientListPageDataResponse, error) {
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

	query := `WITH enriched AS (SELECT id, delegate_id, client_id, active, date_created, date_modified FROM delegate_client WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR delegate_id ILIKE $1 OR client_id ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var delegateClients []*delegateclientpb.DelegateClient
	var totalCount int64
	for rows.Next() {
		var id, delegateId, clientId string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &delegateId, &clientId, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		delegateClient := &delegateclientpb.DelegateClient{Id: id, DelegateId: delegateId, ClientId: clientId, Active: active}
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		delegateClient.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		delegateClient.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		delegateClient.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		delegateClient.DateModifiedString = &dmStr
	}
		delegateClients = append(delegateClients, delegateClient)
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &delegateclientpb.GetDelegateClientListPageDataResponse{DelegateClientList: delegateClients, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetDelegateClientItemPageData retrieves delegate client item page data
func (r *PostgresDelegateClientRepository) GetDelegateClientItemPageData(ctx context.Context, req *delegateclientpb.GetDelegateClientItemPageDataRequest) (*delegateclientpb.GetDelegateClientItemPageDataResponse, error) {
	if req == nil || req.DelegateClientId == "" {
		return nil, fmt.Errorf("delegate client ID required")
	}
	query := `SELECT id, delegate_id, client_id, active, date_created, date_modified FROM delegate_client WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.DelegateClientId)
	var id, delegateId, clientId string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &delegateId, &clientId, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("delegate client not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	delegateClient := &delegateclientpb.DelegateClient{Id: id, DelegateId: delegateId, ClientId: clientId, Active: active}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		delegateClient.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		delegateClient.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		delegateClient.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		delegateClient.DateModifiedString = &dmStr
	}
	return &delegateclientpb.GetDelegateClientItemPageDataResponse{DelegateClient: delegateClient, Success: true}, nil
}


// NewDelegateClientRepository creates a new PostgreSQL delegate_client repository (old-style constructor)
func NewDelegateClientRepository(db *sql.DB, tableName string) delegateclientpb.DelegateClientDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresDelegateClientRepository(dbOps, tableName)
}
