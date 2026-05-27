//go:build sqlserver

package entity

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
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
	delegatesupplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_supplier"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Delegate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver delegate repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerDelegateRepository(dbOps, tableName), nil
	})
}

// SQLServerDelegateRepository implements delegate CRUD operations using SQL Server.
type SQLServerDelegateRepository struct {
	delegatepb.UnimplementedDelegateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerDelegateRepository creates a new SQL Server delegate repository.
func NewSQLServerDelegateRepository(dbOps interfaces.DatabaseOperation, tableName string) delegatepb.DelegateDomainServiceServer {
	if tableName == "" {
		tableName = "delegate"
	}
	return &SQLServerDelegateRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateDelegate creates a new delegate using common SQL Server operations.
func (r *SQLServerDelegateRepository) CreateDelegate(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("delegate data is required")
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
		return nil, fmt.Errorf("failed to create delegate: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegate := &delegatepb.Delegate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, delegate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegatepb.CreateDelegateResponse{
		Data: []*delegatepb.Delegate{delegate},
	}, nil
}

// ReadDelegate retrieves a delegate using common SQL Server operations.
func (r *SQLServerDelegateRepository) ReadDelegate(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read delegate: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegate := &delegatepb.Delegate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, delegate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegatepb.ReadDelegateResponse{
		Data: []*delegatepb.Delegate{delegate},
	}, nil
}

// UpdateDelegate updates a delegate using common SQL Server operations.
func (r *SQLServerDelegateRepository) UpdateDelegate(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
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
		return nil, fmt.Errorf("failed to update delegate: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegate := &delegatepb.Delegate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, delegate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegatepb.UpdateDelegateResponse{
		Data: []*delegatepb.Delegate{delegate},
	}, nil
}

// DeleteDelegate deletes a delegate using common SQL Server operations.
func (r *SQLServerDelegateRepository) DeleteDelegate(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete delegate: %w", err)
	}

	return &delegatepb.DeleteDelegateResponse{Success: true}, nil
}

// ListDelegates lists delegates using common SQL Server operations.
func (r *SQLServerDelegateRepository) ListDelegates(ctx context.Context, req *delegatepb.ListDelegatesRequest) (*delegatepb.ListDelegatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list delegates: %w", err)
	}

	var delegates []*delegatepb.Delegate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		delegate := &delegatepb.Delegate{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, delegate); err != nil {
			continue
		}
		delegates = append(delegates, delegate)
	}

	return &delegatepb.ListDelegatesResponse{Data: delegates}, nil
}

// GetDelegateListPageData retrieves a paginated list of delegates with user and client/supplier relationships.
//
// SQL Server translation notes:
//   - "user" → [user] (reserved word, must be bracket-quoted).
//   - $N → @pN throughout.
//   - ILIKE → LIKE (SQL Server CI collation matches postgres ILIKE).
//   - jsonb_build_object / jsonb_agg: Replaced with FOR JSON PATH subqueries scoped per delegate row,
//     materialised into NVARCHAR(MAX) columns and re-parsed on the Go side as JSON.
//   - LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER () window function retained (SQL Server 2017+).
//
// The postgres query used CASE-level per-row JSON aggregation across CTEs.
// SQL Server does not support ordered jsonb_agg; the equivalent is a correlated
// FOR JSON PATH subquery per parent row. The results are identical columns/scan order.
func (r *SQLServerDelegateRepository) GetDelegateListPageData(ctx context.Context, req *delegatepb.GetDelegateListPageDataRequest) (*delegatepb.GetDelegateListPageDataResponse, error) {
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	// SQL Server translation:
	//   - [user] for the reserved keyword.
	//   - @p1=searchQuery, @p2=limit, @p3=offset, @p4=sortField, @p5=sortDirection.
	//   - FOR JSON PATH subqueries replace jsonb_agg CTEs; they return NULL when no
	//     rows match, so COALESCE(..., '[]') is applied in Go after scanning.
	//   - OUTER APPLY not needed here (no LATERAL); correlated subquery suffices for
	//     the JSON aggregation pattern.
	//   - ORDER BY / OFFSET FETCH replaces LIMIT / OFFSET.
	//   - COUNT(*) OVER () window is supported in SQL Server 2017+.
	query := `
		WITH
		search_filtered AS (
			SELECT d.*
			FROM delegate d
			LEFT JOIN [user] u ON d.user_id = u.id
			WHERE d.active = 1
				AND (@p1 = '' OR
					u.first_name LIKE @p1 OR
					u.last_name LIKE @p1 OR
					u.email_address LIKE @p1)
		),
		enriched AS (
			SELECT
				sf.id,
				sf.user_id,
				sf.active,
				sf.date_created,
				sf.date_modified,
				(SELECT
					u.id,
					u.first_name,
					u.last_name,
					u.email_address,
					u.date_created,
					u.date_modified,
					u.active
				 FROM [user] u
				 WHERE u.id = sf.user_id
				 FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS [user],
				(SELECT
					dc.id,
					dc.delegate_id,
					dc.client_id,
					dc.date_created,
					dc.date_modified,
					dc.active
				 FROM delegate_client dc
				 INNER JOIN client c ON dc.client_id = c.id
				 WHERE dc.delegate_id = sf.id AND dc.active = 1 AND c.active = 1
				 ORDER BY dc.id ASC
				 FOR JSON PATH) AS delegate_clients,
				(SELECT
					ds.id,
					ds.delegate_id,
					ds.supplier_id,
					ds.date_created,
					ds.date_modified,
					ds.active
				 FROM delegate_supplier ds
				 LEFT JOIN supplier s ON ds.supplier_id = s.id
				 WHERE ds.delegate_id = sf.id AND ds.active = 1
				 ORDER BY ds.id ASC
				 FOR JSON PATH) AS delegate_suppliers,
				COUNT(*) OVER () AS _total_count
			FROM search_filtered sf
		)
		SELECT
			id,
			user_id,
			active,
			date_created,
			date_modified,
			[user],
			delegate_clients,
			delegate_suppliers,
			_total_count
		FROM enriched
		ORDER BY
			CASE WHEN @p4 = 'user_id' AND @p5 = 'ASC'  THEN user_id    END ASC,
			CASE WHEN @p4 = 'user_id' AND @p5 = 'DESC' THEN user_id    END DESC,
			CASE WHEN (@p4 = 'date_created' OR @p4 = '') AND @p5 = 'DESC' THEN date_created END DESC,
			CASE WHEN @p4 = 'date_created' AND @p5 = 'ASC'               THEN date_created END ASC
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query,
		searchQuery,   // @p1
		limit,         // @p2
		offset,        // @p3
		sortField,     // @p4
		sortDirection, // @p5
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetDelegateListPageData query: %w", err)
	}
	defer rows.Close()

	var delegates []*delegatepb.Delegate
	var totalCount int32

	for rows.Next() {
		var (
			id                    string
			userId                string
			active                bool
			dateCreated           sql.NullInt64
			dateModified          sql.NullInt64
			userJSON              []byte
			delegateClientsJSON   []byte
			delegateSuppliersJSON []byte
			rowTotalCount         int32
		)

		err := rows.Scan(
			&id,
			&userId,
			&active,
			&dateCreated,
			&dateModified,
			&userJSON,
			&delegateClientsJSON,
			&delegateSuppliersJSON,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delegate row: %w", err)
		}

		totalCount = rowTotalCount

		delegate := &delegatepb.Delegate{
			Id:     id,
			UserId: userId,
			Active: active,
		}

		if dateCreated.Valid {
			delegate.DateCreated = &dateCreated.Int64
		}
		if dateModified.Valid {
			delegate.DateModified = &dateModified.Int64
		}

		if len(userJSON) > 0 && string(userJSON) != "null" {
			var userData map[string]any
			if err := json.Unmarshal(userJSON, &userData); err == nil {
				userDataJSON, _ := json.Marshal(userData)
				var user userpb.User
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(userDataJSON, &user); err == nil {
					delegate.User = &user
				}
			}
		}

		// FOR JSON PATH returns NULL when no rows — treat as empty array.
		clientsJSON := delegateClientsJSON
		if len(clientsJSON) == 0 {
			clientsJSON = []byte("[]")
		}
		if len(clientsJSON) > 0 {
			var delegateClients []map[string]any
			if err := json.Unmarshal(clientsJSON, &delegateClients); err == nil {
				for _, dcData := range delegateClients {
					dcJSON, _ := json.Marshal(dcData)
					var delegateClient delegateclientpb.DelegateClient
					if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dcJSON, &delegateClient); err == nil {
						delegate.DelegateClients = append(delegate.DelegateClients, &delegateClient)
					}
				}
			}
		}

		suppliersJSON := delegateSuppliersJSON
		if len(suppliersJSON) == 0 {
			suppliersJSON = []byte("[]")
		}
		if len(suppliersJSON) > 0 {
			var delegateSuppliers []map[string]any
			if err := json.Unmarshal(suppliersJSON, &delegateSuppliers); err == nil {
				for _, dsData := range delegateSuppliers {
					dsJSON, _ := json.Marshal(dsData)
					var delegateSupplier delegatesupplierpb.DelegateSupplier
					if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dsJSON, &delegateSupplier); err == nil {
						delegate.DelegateSuppliers = append(delegate.DelegateSuppliers, &delegateSupplier)
					}
				}
			}
		}

		delegates = append(delegates, delegate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating delegate rows: %w", err)
	}

	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	return &delegatepb.GetDelegateListPageDataResponse{
		Success:      true,
		DelegateList: delegates,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalCount,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetDelegateItemPageData retrieves a single delegate with all related user and client/supplier data.
//
// SQL Server translation: same as GetDelegateListPageData but scoped to a single delegate ID.
// FOR JSON PATH subqueries replace jsonb_agg; [user] replaces "user"; @p1 replaces $1.
func (r *SQLServerDelegateRepository) GetDelegateItemPageData(ctx context.Context, req *delegatepb.GetDelegateItemPageDataRequest) (*delegatepb.GetDelegateItemPageDataResponse, error) {
	if req.DelegateId == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	query := `
		SELECT
			d.id,
			d.user_id,
			d.active,
			d.date_created,
			d.date_modified,
			(SELECT
				u.id,
				u.first_name,
				u.last_name,
				u.email_address,
				u.date_created,
				u.date_modified,
				u.active
			 FROM [user] u
			 WHERE u.id = d.user_id
			 FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS [user],
			(SELECT
				dc.id,
				dc.delegate_id,
				dc.client_id,
				dc.date_created,
				dc.date_modified,
				dc.active
			 FROM delegate_client dc
			 INNER JOIN client c ON dc.client_id = c.id
			 WHERE dc.delegate_id = d.id AND dc.active = 1 AND c.active = 1
			 ORDER BY dc.id ASC
			 FOR JSON PATH) AS delegate_clients,
			(SELECT
				ds.id,
				ds.delegate_id,
				ds.supplier_id,
				ds.date_created,
				ds.date_modified,
				ds.active
			 FROM delegate_supplier ds
			 LEFT JOIN supplier s ON ds.supplier_id = s.id
			 WHERE ds.delegate_id = d.id AND ds.active = 1
			 ORDER BY ds.id ASC
			 FOR JSON PATH) AS delegate_suppliers
		FROM delegate d
		LEFT JOIN [user] u ON d.user_id = u.id
		WHERE d.id = @p1 AND d.active = 1
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	var (
		id                    string
		userId                string
		active                bool
		dateCreated           sql.NullInt64
		dateModified          sql.NullInt64
		userJSON              []byte
		delegateClientsJSON   []byte
		delegateSuppliersJSON []byte
	)

	err := exec.QueryRowContext(ctx, query, req.DelegateId).Scan(
		&id,
		&userId,
		&active,
		&dateCreated,
		&dateModified,
		&userJSON,
		&delegateClientsJSON,
		&delegateSuppliersJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("delegate not found with ID: %s", req.DelegateId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetDelegateItemPageData query: %w", err)
	}

	delegate := &delegatepb.Delegate{
		Id:     id,
		UserId: userId,
		Active: active,
	}

	if dateCreated.Valid {
		delegate.DateCreated = &dateCreated.Int64
	}
	if dateModified.Valid {
		delegate.DateModified = &dateModified.Int64
	}

	if len(userJSON) > 0 && string(userJSON) != "null" {
		var userData map[string]any
		if err := json.Unmarshal(userJSON, &userData); err == nil {
			userDataJSON, _ := json.Marshal(userData)
			var user userpb.User
			if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(userDataJSON, &user); err == nil {
				delegate.User = &user
			}
		}
	}

	clientsJSON := delegateClientsJSON
	if len(clientsJSON) == 0 {
		clientsJSON = []byte("[]")
	}
	if len(clientsJSON) > 0 {
		var delegateClients []map[string]any
		if err := json.Unmarshal(clientsJSON, &delegateClients); err == nil {
			for _, dcData := range delegateClients {
				dcJSON, _ := json.Marshal(dcData)
				var delegateClient delegateclientpb.DelegateClient
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dcJSON, &delegateClient); err == nil {
					delegate.DelegateClients = append(delegate.DelegateClients, &delegateClient)
				}
			}
		}
	}

	suppliersJSON := delegateSuppliersJSON
	if len(suppliersJSON) == 0 {
		suppliersJSON = []byte("[]")
	}
	if len(suppliersJSON) > 0 {
		var delegateSuppliers []map[string]any
		if err := json.Unmarshal(suppliersJSON, &delegateSuppliers); err == nil {
			for _, dsData := range delegateSuppliers {
				dsJSON, _ := json.Marshal(dsData)
				var delegateSupplier delegatesupplierpb.DelegateSupplier
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dsJSON, &delegateSupplier); err == nil {
					delegate.DelegateSuppliers = append(delegate.DelegateSuppliers, &delegateSupplier)
				}
			}
		}
	}

	return &delegatepb.GetDelegateItemPageDataResponse{
		Success:  true,
		Delegate: delegate,
	}, nil
}

// NewDelegateRepository creates a new SQL Server delegate repository (old-style constructor).
func NewDelegateRepository(db *sql.DB, tableName string) delegatepb.DelegateDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerDelegateRepository(dbOps, tableName)
}
