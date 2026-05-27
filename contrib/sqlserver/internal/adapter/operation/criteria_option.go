//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.CriteriaOption, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver criteria_option repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerCriteriaOptionRepository(dbOps, tableName), nil
	})
}

// SQLServerCriteriaOptionRepository implements criteria_option CRUD operations using SQL Server.
type SQLServerCriteriaOptionRepository struct {
	pb.UnimplementedCriteriaOptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerCriteriaOptionRepository creates a new SQL Server criteria_option repository.
func NewSQLServerCriteriaOptionRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.CriteriaOptionDomainServiceServer {
	if tableName == "" {
		tableName = "criteria_option"
	}
	return &SQLServerCriteriaOptionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerCriteriaOptionRepository) CreateCriteriaOption(ctx context.Context, req *pb.CreateCriteriaOptionRequest) (*pb.CreateCriteriaOptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("criteria option data is required")
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
		return nil, fmt.Errorf("failed to create criteria option: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	opt := &pb.CriteriaOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, opt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.CreateCriteriaOptionResponse{Success: true, Data: []*pb.CriteriaOption{opt}}, nil
}

func (r *SQLServerCriteriaOptionRepository) ReadCriteriaOption(ctx context.Context, req *pb.ReadCriteriaOptionRequest) (*pb.ReadCriteriaOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria option ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read criteria option: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	opt := &pb.CriteriaOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, opt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.ReadCriteriaOptionResponse{Success: true, Data: []*pb.CriteriaOption{opt}}, nil
}

func (r *SQLServerCriteriaOptionRepository) UpdateCriteriaOption(ctx context.Context, req *pb.UpdateCriteriaOptionRequest) (*pb.UpdateCriteriaOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria option ID is required")
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
		return nil, fmt.Errorf("failed to update criteria option: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	opt := &pb.CriteriaOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, opt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.UpdateCriteriaOptionResponse{Success: true, Data: []*pb.CriteriaOption{opt}}, nil
}

func (r *SQLServerCriteriaOptionRepository) DeleteCriteriaOption(ctx context.Context, req *pb.DeleteCriteriaOptionRequest) (*pb.DeleteCriteriaOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria option ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete criteria option: %w", err)
	}
	return &pb.DeleteCriteriaOptionResponse{Success: true}, nil
}

func (r *SQLServerCriteriaOptionRepository) ListCriteriaOptions(ctx context.Context, req *pb.ListCriteriaOptionsRequest) (*pb.ListCriteriaOptionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria options: %w", err)
	}
	var opts []*pb.CriteriaOption
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal criteria_option row: %v", err)
			continue
		}
		opt := &pb.CriteriaOption{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, opt); err != nil {
			log.Printf("WARN: protojson unmarshal criteria_option: %v", err)
			continue
		}
		opts = append(opts, opt)
	}
	return &pb.ListCriteriaOptionsResponse{Success: true, Data: opts}, nil
}

// GetCriteriaOptionListPageData retrieves criteria options with pagination.
//
// SQL Server differences vs postgres:
//   - ILIKE → LIKE; $N → @pN; active = true → active = 1.
//   - Pagination: OFFSET/FETCH; ORDER BY required.
//   - workspace_id filter.
func (r *SQLServerCriteriaOptionRepository) GetCriteriaOptionListPageData(
	ctx context.Context,
	req *pb.GetCriteriaOptionListPageDataRequest,
) (*pb.GetCriteriaOptionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	sortableCols := []string{"co.date_created", "co.date_modified", "co.label", "co.sort_order"}
	orderByClause, err := sqlserverCore.BuildOrderBy(sortableCols, req.GetSort(), "co.sort_order ASC")
	if err != nil {
		return nil, err
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if op := req.Pagination.GetOffset(); op != nil && op.Page > 0 {
			page = op.Page
			offset = (page - 1) * limit
		}
	}

	queryArgs := []any{offset, limit}
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				co.id,
				co.date_created,
				co.date_modified,
				co.active,
				co.outcome_criteria_id,
				co.label,
				co.sort_order
			FROM criteria_option co
			WHERE co.active = 1
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT e.*, c.total
		FROM enriched e, counted c
		%s OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY;
	`, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query criteria option list: %w", err)
	}
	defer rows.Close()

	var opts []*pb.CriteriaOption
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			outcomeCriteriaID string
			label             string
			sortOrder         int32
			total             int64
		)
		if err := rows.Scan(&id, &dateCreated, &dateModified, &active, &outcomeCriteriaID, &label, &sortOrder, &total); err != nil {
			return nil, fmt.Errorf("failed to scan criteria option row: %w", err)
		}
		totalCount = total
		opt := &pb.CriteriaOption{
			Id:                id,
			Active:            active,
			OutcomeCriteriaId: outcomeCriteriaID,
			OptionLabel:       label,
			DisplayOrder:      sortOrder,
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			opt.DateCreated = &ts
		}
		opts = append(opts, opt)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating criteria option rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetCriteriaOptionListPageDataResponse{
		CriteriaOptionList: opts,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

func (r *SQLServerCriteriaOptionRepository) GetCriteriaOptionItemPageData(ctx context.Context, req *pb.GetCriteriaOptionItemPageDataRequest) (*pb.GetCriteriaOptionItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetCriteriaOptionItemPageData not yet implemented")
}
