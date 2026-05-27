//go:build mysql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
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
	registry.RegisterRepositoryFactory("mysql", entityid.Delegate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql delegate repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLDelegateRepository(dbOps, tableName), nil
	})
}

// MySQLDelegateRepository implements delegate CRUD operations using MySQL 8.0+.
type MySQLDelegateRepository struct {
	delegatepb.UnimplementedDelegateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLDelegateRepository creates a new MySQL delegate repository.
func NewMySQLDelegateRepository(dbOps interfaces.DatabaseOperation, tableName string) delegatepb.DelegateDomainServiceServer {
	if tableName == "" {
		tableName = "delegate"
	}
	return &MySQLDelegateRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateDelegate creates a new delegate using common MySQL operations.
func (r *MySQLDelegateRepository) CreateDelegate(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
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

// ReadDelegate retrieves a delegate using common MySQL operations.
func (r *MySQLDelegateRepository) ReadDelegate(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
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

// UpdateDelegate updates a delegate using common MySQL operations.
func (r *MySQLDelegateRepository) UpdateDelegate(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
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

// DeleteDelegate deletes a delegate using common MySQL operations.
func (r *MySQLDelegateRepository) DeleteDelegate(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete delegate: %w", err)
	}

	return &delegatepb.DeleteDelegateResponse{
		Success: true,
	}, nil
}

// ListDelegates lists delegates using common MySQL operations.
func (r *MySQLDelegateRepository) ListDelegates(ctx context.Context, req *delegatepb.ListDelegatesRequest) (*delegatepb.ListDelegatesResponse, error) {
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

	return &delegatepb.ListDelegatesResponse{
		Data: delegates,
	}, nil
}

// GetDelegateListPageData retrieves a paginated list of delegates with user, client, and supplier relationships.
//
// Dialect translation from postgres gold standard:
//   - $1/$2/$3/$4/$5 → ? (MySQL positional placeholders in the same arg order)
//   - "user" → `user` (backtick-quoted reserved word)
//   - jsonb_build_object → JSON_OBJECT
//   - jsonb_agg(...) → JSON_ARRAYAGG(...)
//   - ILIKE → LIKE
//   - active = true → active = 1
//   - Sorted CTE (CTE 4) simplified: MySQL supports CASE-in-ORDER-BY natively
//
// CRITICAL: Workspace isolation is enforced by WorkspaceAwareOperations; delegate
// does not carry workspace_id directly — the active filter prevents cross-tenant data.
func (r *MySQLDelegateRepository) GetDelegateListPageData(ctx context.Context, req *delegatepb.GetDelegateListPageDataRequest) (*delegatepb.GetDelegateListPageDataResponse, error) {
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

	// Dialect translation:
	//   - "user" → `user`
	//   - jsonb_build_object → JSON_OBJECT
	//   - jsonb_agg(obj ORDER BY ...) → JSON_ARRAYAGG(obj) (MySQL 8.0+; ORDER inside not supported, stable by PK)
	//   - ILIKE → LIKE
	//   - active = true → active = 1
	//   - $N → ? (positional)
	// MySQL does not support ORDER BY inside JSON_ARRAYAGG; rows are pre-sorted
	// in the inner CTE via a derived table trick to preserve stable PK order.
	query := `
		WITH
		delegate_clients_rows AS (
			SELECT
				dc.delegate_id,
				dc.id AS dc_id,
				JSON_OBJECT(
					'id', dc.id,
					'delegate_id', dc.delegate_id,
					'client_id', dc.client_id,
					'date_created', dc.date_created,
					'date_modified', dc.date_modified,
					'active', dc.active,
					'client', JSON_OBJECT(
						'id', c.id,
						'user_id', c.user_id,
						'date_created', c.date_created,
						'date_modified', c.date_modified,
						'active', c.active,
						'user', CASE
							WHEN cu.id IS NOT NULL THEN JSON_OBJECT(
								'id', cu.id,
								'first_name', cu.first_name,
								'last_name', cu.last_name,
								'email_address', cu.email_address,
								'date_created', cu.date_created,
								'date_modified', cu.date_modified,
								'active', cu.active
							)
							ELSE NULL
						END
					)
				) AS obj
			FROM delegate_client dc
			INNER JOIN client c ON dc.client_id = c.id
			LEFT JOIN ` + "`user`" + ` cu ON c.user_id = cu.id
			WHERE dc.active = 1 AND c.active = 1
		),
		delegate_clients_agg AS (
			SELECT
				delegate_id,
				JSON_ARRAYAGG(obj) AS delegate_clients
			FROM (SELECT * FROM delegate_clients_rows ORDER BY dc_id ASC) ordered_dc
			GROUP BY delegate_id
		),

		delegate_suppliers_rows AS (
			SELECT
				ds.delegate_id,
				ds.id AS ds_id,
				JSON_OBJECT(
					'id', ds.id,
					'delegate_id', ds.delegate_id,
					'supplier_id', ds.supplier_id,
					'date_created', ds.date_created,
					'date_modified', ds.date_modified,
					'active', ds.active,
					'supplier', CASE
						WHEN s.id IS NOT NULL THEN JSON_OBJECT(
							'id', s.id,
							'name', s.name,
							'date_created', s.date_created,
							'date_modified', s.date_modified,
							'active', s.active
						)
						ELSE NULL
					END
				) AS obj
			FROM delegate_supplier ds
			LEFT JOIN supplier s ON ds.supplier_id = s.id
			WHERE ds.active = 1
		),
		delegate_suppliers_agg AS (
			SELECT
				delegate_id,
				JSON_ARRAYAGG(obj) AS delegate_suppliers
			FROM (SELECT * FROM delegate_suppliers_rows ORDER BY ds_id ASC) ordered_ds
			GROUP BY delegate_id
		),

		search_filtered AS (
			SELECT d.*
			FROM delegate d
			LEFT JOIN ` + "`user`" + ` u ON d.user_id = u.id
			WHERE d.active = 1
				AND (? = '' OR
					u.first_name LIKE ? OR
					u.last_name LIKE ? OR
					u.email_address LIKE ?)
		),

		enriched AS (
			SELECT
				sf.id,
				sf.user_id,
				sf.active,
				sf.date_created,
				sf.date_modified,
				CASE
					WHEN u.id IS NOT NULL THEN JSON_OBJECT(
						'id', u.id,
						'first_name', u.first_name,
						'last_name', u.last_name,
						'email_address', u.email_address,
						'date_created', u.date_created,
						'date_modified', u.date_modified,
						'active', u.active
					)
					ELSE NULL
				END AS user_json,
				COALESCE(dca.delegate_clients, JSON_ARRAY()) AS delegate_clients,
				COALESCE(dsa.delegate_suppliers, JSON_ARRAY()) AS delegate_suppliers
			FROM search_filtered sf
			LEFT JOIN ` + "`user`" + ` u ON sf.user_id = u.id
			LEFT JOIN delegate_clients_agg dca ON sf.id = dca.delegate_id
			LEFT JOIN delegate_suppliers_agg dsa ON sf.id = dsa.delegate_id
		),

		total_count AS (
			SELECT COUNT(*) AS total FROM enriched
		)

		SELECT
			e.id,
			e.user_id,
			e.active,
			e.date_created,
			e.date_modified,
			e.user_json,
			e.delegate_clients,
			e.delegate_suppliers,
			tc.total AS _total_count
		FROM enriched e
		CROSS JOIN total_count tc
		ORDER BY
			CASE WHEN ? = 'user_id' AND ? = 'ASC' THEN e.user_id END ASC,
			CASE WHEN ? = 'user_id' AND ? = 'DESC' THEN e.user_id END DESC,
			CASE WHEN (? = 'date_created' OR ? = '') AND ? = 'DESC' THEN e.date_created END DESC,
			CASE WHEN ? = 'date_created' AND ? = 'ASC' THEN e.date_created END ASC
		LIMIT ? OFFSET ?
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query,
		searchQuery, // search_filtered: ? = ''
		searchQuery, // u.first_name LIKE ?
		searchQuery, // u.last_name LIKE ?
		searchQuery, // u.email_address LIKE ?
		sortField,   // ORDER BY CASE user_id ASC
		sortDirection,
		sortField, // ORDER BY CASE user_id DESC
		sortDirection,
		sortField, // ORDER BY CASE date_created DESC
		sortField,
		sortDirection,
		sortField, // ORDER BY CASE date_created ASC
		sortDirection,
		limit,
		offset,
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

		if len(delegateClientsJSON) > 0 {
			var delegateClients []map[string]any
			if err := json.Unmarshal(delegateClientsJSON, &delegateClients); err == nil {
				for _, dcData := range delegateClients {
					dcJSON, _ := json.Marshal(dcData)
					var delegateClient delegateclientpb.DelegateClient
					if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dcJSON, &delegateClient); err == nil {
						delegate.DelegateClients = append(delegate.DelegateClients, &delegateClient)
					}
				}
			}
		}

		if len(delegateSuppliersJSON) > 0 {
			var delegateSuppliers []map[string]any
			if err := json.Unmarshal(delegateSuppliersJSON, &delegateSuppliers); err == nil {
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

// GetDelegateItemPageData retrieves a single delegate with all related user, client, and supplier data.
//
// Dialect translation: "user" → `user`; jsonb_build_object → JSON_OBJECT; jsonb_agg → JSON_ARRAYAGG;
// $1 → ?; active = true → active = 1.
func (r *MySQLDelegateRepository) GetDelegateItemPageData(ctx context.Context, req *delegatepb.GetDelegateItemPageDataRequest) (*delegatepb.GetDelegateItemPageDataResponse, error) {
	if req.DelegateId == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	query := `
		WITH
		delegate_clients_rows AS (
			SELECT
				dc.delegate_id,
				dc.id AS dc_id,
				JSON_OBJECT(
					'id', dc.id,
					'delegate_id', dc.delegate_id,
					'client_id', dc.client_id,
					'date_created', dc.date_created,
					'date_modified', dc.date_modified,
					'active', dc.active,
					'client', JSON_OBJECT(
						'id', c.id,
						'user_id', c.user_id,
						'date_created', c.date_created,
						'date_modified', c.date_modified,
						'active', c.active,
						'user', CASE
							WHEN cu.id IS NOT NULL THEN JSON_OBJECT(
								'id', cu.id,
								'first_name', cu.first_name,
								'last_name', cu.last_name,
								'email_address', cu.email_address,
								'date_created', cu.date_created,
								'date_modified', cu.date_modified,
								'active', cu.active
							)
							ELSE NULL
						END
					)
				) AS obj
			FROM delegate_client dc
			INNER JOIN client c ON dc.client_id = c.id
			LEFT JOIN ` + "`user`" + ` cu ON c.user_id = cu.id
			WHERE dc.delegate_id = ? AND dc.active = 1 AND c.active = 1
		),
		delegate_clients_agg AS (
			SELECT
				delegate_id,
				JSON_ARRAYAGG(obj) AS delegate_clients
			FROM (SELECT * FROM delegate_clients_rows ORDER BY dc_id ASC) ordered_dc
			GROUP BY delegate_id
		),

		delegate_suppliers_rows AS (
			SELECT
				ds.delegate_id,
				ds.id AS ds_id,
				JSON_OBJECT(
					'id', ds.id,
					'delegate_id', ds.delegate_id,
					'supplier_id', ds.supplier_id,
					'date_created', ds.date_created,
					'date_modified', ds.date_modified,
					'active', ds.active,
					'supplier', CASE
						WHEN s.id IS NOT NULL THEN JSON_OBJECT(
							'id', s.id,
							'name', s.name,
							'date_created', s.date_created,
							'date_modified', s.date_modified,
							'active', s.active
						)
						ELSE NULL
					END
				) AS obj
			FROM delegate_supplier ds
			LEFT JOIN supplier s ON ds.supplier_id = s.id
			WHERE ds.delegate_id = ? AND ds.active = 1
		),
		delegate_suppliers_agg AS (
			SELECT
				delegate_id,
				JSON_ARRAYAGG(obj) AS delegate_suppliers
			FROM (SELECT * FROM delegate_suppliers_rows ORDER BY ds_id ASC) ordered_ds
			GROUP BY delegate_id
		)

		SELECT
			d.id,
			d.user_id,
			d.active,
			d.date_created,
			d.date_modified,
			CASE
				WHEN u.id IS NOT NULL THEN JSON_OBJECT(
					'id', u.id,
					'first_name', u.first_name,
					'last_name', u.last_name,
					'email_address', u.email_address,
					'date_created', u.date_created,
					'date_modified', u.date_modified,
					'active', u.active
				)
				ELSE NULL
			END AS user_json,
			COALESCE(dca.delegate_clients, JSON_ARRAY()) AS delegate_clients,
			COALESCE(dsa.delegate_suppliers, JSON_ARRAY()) AS delegate_suppliers
		FROM delegate d
		LEFT JOIN ` + "`user`" + ` u ON d.user_id = u.id
		LEFT JOIN delegate_clients_agg dca ON d.id = dca.delegate_id
		LEFT JOIN delegate_suppliers_agg dsa ON d.id = dsa.delegate_id
		WHERE d.id = ? AND d.active = 1
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

	// Args: delegate_clients_rows.?, delegate_suppliers_rows.?, final WHERE ?
	err := exec.QueryRowContext(ctx, query, req.DelegateId, req.DelegateId, req.DelegateId).Scan(
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

	if len(delegateClientsJSON) > 0 {
		var delegateClients []map[string]any
		if err := json.Unmarshal(delegateClientsJSON, &delegateClients); err == nil {
			for _, dcData := range delegateClients {
				dcJSON, _ := json.Marshal(dcData)
				var delegateClient delegateclientpb.DelegateClient
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dcJSON, &delegateClient); err == nil {
					delegate.DelegateClients = append(delegate.DelegateClients, &delegateClient)
				}
			}
		}
	}

	if len(delegateSuppliersJSON) > 0 {
		var delegateSuppliers []map[string]any
		if err := json.Unmarshal(delegateSuppliersJSON, &delegateSuppliers); err == nil {
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

// NewDelegateRepository creates a new MySQL delegate repository (old-style constructor).
func NewDelegateRepository(db *sql.DB, tableName string) delegatepb.DelegateDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLDelegateRepository(dbOps, tableName)
}
