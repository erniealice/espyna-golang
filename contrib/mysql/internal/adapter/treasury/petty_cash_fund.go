//go:build mysql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.PettyCashFund, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql petty_cash_fund repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLPettyCashFundRepository(dbOps, tableName), nil
	})
}

// MySQLPettyCashFundRepository implements petty_cash_fund CRUD using MySQL 8.0+.
type MySQLPettyCashFundRepository struct {
	pettycashfundpb.UnimplementedPettyCashFundDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLPettyCashFundRepository creates a new MySQL petty_cash_fund repository.
func NewMySQLPettyCashFundRepository(dbOps interfaces.DatabaseOperation, tableName string) pettycashfundpb.PettyCashFundDomainServiceServer {
	if tableName == "" {
		tableName = "petty_cash_fund"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLPettyCashFundRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *MySQLPettyCashFundRepository) CreatePettyCashFund(ctx context.Context, req *pettycashfundpb.CreatePettyCashFundRequest) (*pettycashfundpb.CreatePettyCashFundResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("petty_cash_fund data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create petty_cash_fund: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pcf := &pettycashfundpb.PettyCashFund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pcf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashfundpb.CreatePettyCashFundResponse{Success: true, Data: []*pettycashfundpb.PettyCashFund{pcf}}, nil
}

func (r *MySQLPettyCashFundRepository) ReadPettyCashFund(ctx context.Context, req *pettycashfundpb.ReadPettyCashFundRequest) (*pettycashfundpb.ReadPettyCashFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read petty_cash_fund: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pcf := &pettycashfundpb.PettyCashFund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pcf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashfundpb.ReadPettyCashFundResponse{Success: true, Data: []*pettycashfundpb.PettyCashFund{pcf}}, nil
}

func (r *MySQLPettyCashFundRepository) UpdatePettyCashFund(ctx context.Context, req *pettycashfundpb.UpdatePettyCashFundRequest) (*pettycashfundpb.UpdatePettyCashFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update petty_cash_fund: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pcf := &pettycashfundpb.PettyCashFund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pcf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashfundpb.UpdatePettyCashFundResponse{Success: true, Data: []*pettycashfundpb.PettyCashFund{pcf}}, nil
}

func (r *MySQLPettyCashFundRepository) DeletePettyCashFund(ctx context.Context, req *pettycashfundpb.DeletePettyCashFundRequest) (*pettycashfundpb.DeletePettyCashFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete petty_cash_fund: %w", err)
	}
	return &pettycashfundpb.DeletePettyCashFundResponse{Success: true}, nil
}

func (r *MySQLPettyCashFundRepository) ListPettyCashFunds(ctx context.Context, req *pettycashfundpb.ListPettyCashFundsRequest) (*pettycashfundpb.ListPettyCashFundsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list petty_cash_funds: %w", err)
	}
	var pcfs []*pettycashfundpb.PettyCashFund
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal petty_cash_fund row: %v", err)
			continue
		}
		pcf := &pettycashfundpb.PettyCashFund{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pcf); err != nil {
			log.Printf("WARN: protojson unmarshal petty_cash_fund: %v", err)
			continue
		}
		pcfs = append(pcfs, pcf)
	}
	return &pettycashfundpb.ListPettyCashFundsResponse{Success: true, Data: pcfs}, nil
}

// GetPettyCashFundListPageData retrieves petty_cash_funds with pagination.
// Dialect: $N → ?; ILIKE → LIKE; active = true → active = 1.
func (r *MySQLPettyCashFundRepository) GetPettyCashFundListPageData(
	ctx context.Context,
	req *pettycashfundpb.GetPettyCashFundListPageDataRequest,
) (*pettycashfundpb.GetPettyCashFundListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_fund list page data request is required")
	}

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

	sortField := "pcf.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				pcf.id,
				pcf.date_created,
				pcf.date_modified,
				pcf.active,
				pcf.name,
				pcf.authorized_amount,
				pcf.current_balance,
				pcf.custodian_id,
				pcf.location_id
			FROM petty_cash_fund pcf
			WHERE pcf.active = 1
			  AND (? IS NULL OR ? = '' OR
			       pcf.name LIKE ?)
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

	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_fund list page data: %w", err)
	}
	defer rows.Close()

	var pcfs []*pettycashfundpb.PettyCashFund
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			dateModified     time.Time
			active           bool
			name             string
			authorizedAmount int64
			currentBalance   int64
			custodianID      *string
			locationID       *string
			total            int64
		)
		if err := rows.Scan(&id, &dateCreated, &dateModified, &active, &name, &authorizedAmount, &currentBalance, &custodianID, &locationID, &total); err != nil {
			return nil, fmt.Errorf("failed to scan petty_cash_fund row: %w", err)
		}
		totalCount = total
		pcf := &pettycashfundpb.PettyCashFund{Id: id, Active: active, Name: name, AuthorizedAmount: authorizedAmount, CurrentBalance: currentBalance}
		if custodianID != nil {
			pcf.CustodianId = custodianID
		}
		if locationID != nil {
			pcf.LocationId = locationID
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			pcf.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			pcf.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			pcf.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			pcf.DateModifiedString = &dmStr
		}
		pcfs = append(pcfs, pcf)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating petty_cash_fund rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	return &pettycashfundpb.GetPettyCashFundListPageDataResponse{
		PettyCashFundList: pcfs,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
		Success: true,
	}, nil
}

// GetPettyCashFundItemPageData retrieves a single petty_cash_fund.
func (r *MySQLPettyCashFundRepository) GetPettyCashFundItemPageData(
	ctx context.Context,
	req *pettycashfundpb.GetPettyCashFundItemPageDataRequest,
) (*pettycashfundpb.GetPettyCashFundItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_fund item page data request is required")
	}
	if req.PettyCashFundId == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}

	const query = `
		WITH enriched AS (
			SELECT pcf.id, pcf.date_created, pcf.date_modified, pcf.active,
			       pcf.name, pcf.authorized_amount, pcf.current_balance,
			       pcf.custodian_id, pcf.location_id
			FROM petty_cash_fund pcf
			WHERE pcf.id = ? AND pcf.active = 1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.PettyCashFundId)
	var (
		id               string
		dateCreated      time.Time
		dateModified     time.Time
		active           bool
		name             string
		authorizedAmount int64
		currentBalance   int64
		custodianID      *string
		locationID       *string
	)
	err := row.Scan(&id, &dateCreated, &dateModified, &active, &name, &authorizedAmount, &currentBalance, &custodianID, &locationID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("petty_cash_fund with ID '%s' not found", req.PettyCashFundId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_fund item page data: %w", err)
	}
	pcf := &pettycashfundpb.PettyCashFund{Id: id, Active: active, Name: name, AuthorizedAmount: authorizedAmount, CurrentBalance: currentBalance}
	if custodianID != nil {
		pcf.CustodianId = custodianID
	}
	if locationID != nil {
		pcf.LocationId = locationID
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		pcf.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		pcf.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		pcf.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		pcf.DateModifiedString = &dmStr
	}
	return &pettycashfundpb.GetPettyCashFundItemPageDataResponse{PettyCashFund: pcf, Success: true}, nil
}

func NewPettyCashFundRepository(db *sql.DB, tableName string) pettycashfundpb.PettyCashFundDomainServiceServer {
	return NewMySQLPettyCashFundRepository(mysqlCore.NewWorkspaceAwareOperations(db), tableName)
}
