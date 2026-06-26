//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
	"google.golang.org/protobuf/encoding/protojson"
)

var licenseHistorySortableSQLCols = []string{"date_created", "action"}
var licenseHistoryViewToSQLColMap = map[string]string{}

// MySQLLicenseHistoryRepository implements license_history CRUD using MySQL 8.0+.
type MySQLLicenseHistoryRepository struct {
	licensehistorypb.UnimplementedLicenseHistoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.LicenseHistory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql license_history repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLLicenseHistoryRepository(dbOps, tableName), nil
	})
}

// NewMySQLLicenseHistoryRepository creates a new MySQL license_history repository.
func NewMySQLLicenseHistoryRepository(dbOps interfaces.DatabaseOperation, tableName string) licensehistorypb.LicenseHistoryDomainServiceServer {
	if tableName == "" {
		tableName = "license_history"
	}
	return &MySQLLicenseHistoryRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLLicenseHistoryRepository) CreateLicenseHistory(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) (*licensehistorypb.CreateLicenseHistoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("license_history data is required")
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
		return nil, fmt.Errorf("failed to create license_history: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lh := &licensehistorypb.LicenseHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lh); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &licensehistorypb.CreateLicenseHistoryResponse{Data: []*licensehistorypb.LicenseHistory{lh}, Success: true}, nil
}

func (r *MySQLLicenseHistoryRepository) ReadLicenseHistory(ctx context.Context, req *licensehistorypb.ReadLicenseHistoryRequest) (*licensehistorypb.ReadLicenseHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license_history ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read license_history: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lh := &licensehistorypb.LicenseHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lh); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &licensehistorypb.ReadLicenseHistoryResponse{Data: []*licensehistorypb.LicenseHistory{lh}, Success: true}, nil
}

func (r *MySQLLicenseHistoryRepository) ListLicenseHistory(ctx context.Context, req *licensehistorypb.ListLicenseHistoryRequest) (*licensehistorypb.ListLicenseHistoryResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list license_history: %w", err)
	}
	var histories []*licensehistorypb.LicenseHistory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		lh := &licensehistorypb.LicenseHistory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lh); err != nil {
			continue
		}
		if req != nil && req.LicenseId != nil && *req.LicenseId != "" {
			if lh.LicenseId != *req.LicenseId {
				continue
			}
		}
		histories = append(histories, lh)
	}
	return &licensehistorypb.ListLicenseHistoryResponse{Data: histories, Success: true}, nil
}

// GetLicenseHistoryListPageData retrieves a paginated list of license_history records.
//
// Dialect changes vs postgres gold standard:
//   - $N → ? (MySQL positional placeholders)
//   - active = true → active = 1
//   - CROSS JOIN total_count → COUNT(*) OVER ()
//   - WHERE workspace_id = ? added for multi-tenancy
func (r *MySQLLicenseHistoryRepository) GetLicenseHistoryListPageData(ctx context.Context, req *licensehistorypb.GetLicenseHistoryListPageDataRequest) (*licensehistorypb.GetLicenseHistoryListPageDataResponse, error) {
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

	licenseIdFilter := ""
	if req.LicenseId != nil && *req.LicenseId != "" {
		licenseIdFilter = *req.LicenseId
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

	if mapped, ok := licenseHistoryViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	if sortField != "" && !slices.Contains(licenseHistorySortableSQLCols, sortField) {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortField, "license_history", licenseHistorySortableSQLCols)
	}

	// Dialect: $N → ?, active = true → active = 1, CROSS JOIN → COUNT(*) OVER ().
	query := fmt.Sprintf(`
		WITH filtered AS (
			SELECT lh.*
			FROM license_history lh
			WHERE lh.active = 1
				AND (? = '' OR lh.license_id = ?)
		),
		sorted AS (
			SELECT *,
				COUNT(*) OVER () AS _total_count
			FROM filtered
			ORDER BY
				CASE WHEN ('%s' = 'date_created' OR '%s' = '') AND '%s' = 'DESC' THEN date_created END DESC,
				CASE WHEN '%s' = 'date_created' AND '%s' = 'ASC' THEN date_created END ASC,
				CASE WHEN '%s' = 'action' AND '%s' = 'ASC' THEN action END ASC,
				CASE WHEN '%s' = 'action' AND '%s' = 'DESC' THEN action END DESC
		)
		SELECT * FROM sorted
		LIMIT ? OFFSET ?
	`,
		sortField, sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
	)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, licenseIdFilter, licenseIdFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetLicenseHistoryListPageData query: %w", err)
	}
	defer rows.Close()

	// Dynamic scan — column set may vary; _total_count is appended last.
	var histories []*licensehistorypb.LicenseHistory
	var totalCount int32

	for rows.Next() {
		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("rows.Columns: %w", err)
		}
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("failed to scan license_history row: %w", err)
		}
		raw := map[string]any{}
		for i, c := range cols {
			raw[c] = normalizeScanValue(vals[i])
		}
		if t, ok := raw["_total_count"].(int64); ok {
			totalCount = int32(t)
		}
		delete(raw, "_total_count")
		dataJSON, _ := json.Marshal(mysqlCore.DenormalizeKeys(raw))
		lh := &licensehistorypb.LicenseHistory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dataJSON, lh); err == nil {
			histories = append(histories, lh)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating license_history rows: %w", err)
	}

	totalPages := (totalCount + limit - 1) / limit
	return &licensehistorypb.GetLicenseHistoryListPageDataResponse{
		Success:            true,
		LicenseHistoryList: histories,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalCount,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
	}, nil
}
