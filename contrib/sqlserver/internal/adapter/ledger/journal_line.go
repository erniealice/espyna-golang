//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	journallinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_line"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JournalLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver journal_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJournalLineRepository(dbOps, tableName), nil
	})
}

// SQLServerJournalLineRepository implements journal_line CRUD using SQL Server.
type SQLServerJournalLineRepository struct {
	journallinepb.UnimplementedJournalLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerJournalLineRepository creates a new SQL Server journal_line repository.
func NewSQLServerJournalLineRepository(dbOps interfaces.DatabaseOperation, tableName string) journallinepb.JournalLineDomainServiceServer {
	if tableName == "" {
		tableName = "journal_line"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerJournalLineRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerJournalLineRepository) CreateJournalLine(ctx context.Context, req *journallinepb.CreateJournalLineRequest) (*journallinepb.CreateJournalLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("journal_line data is required")
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
		return nil, fmt.Errorf("failed to create journal_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	journalLine := &journallinepb.JournalLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, journalLine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &journallinepb.CreateJournalLineResponse{Data: []*journallinepb.JournalLine{journalLine}}, nil
}

func (r *SQLServerJournalLineRepository) ReadJournalLine(ctx context.Context, req *journallinepb.ReadJournalLineRequest) (*journallinepb.ReadJournalLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read journal_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	journalLine := &journallinepb.JournalLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, journalLine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &journallinepb.ReadJournalLineResponse{Data: []*journallinepb.JournalLine{journalLine}}, nil
}

func (r *SQLServerJournalLineRepository) UpdateJournalLine(ctx context.Context, req *journallinepb.UpdateJournalLineRequest) (*journallinepb.UpdateJournalLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_line ID is required")
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
		return nil, fmt.Errorf("failed to update journal_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	journalLine := &journallinepb.JournalLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, journalLine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &journallinepb.UpdateJournalLineResponse{Data: []*journallinepb.JournalLine{journalLine}}, nil
}

func (r *SQLServerJournalLineRepository) DeleteJournalLine(ctx context.Context, req *journallinepb.DeleteJournalLineRequest) (*journallinepb.DeleteJournalLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete journal_line: %w", err)
	}
	return &journallinepb.DeleteJournalLineResponse{Success: true}, nil
}

func (r *SQLServerJournalLineRepository) ListJournalLines(ctx context.Context, req *journallinepb.ListJournalLinesRequest) (*journallinepb.ListJournalLinesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list journal_lines: %w", err)
	}
	var journalLines []*journallinepb.JournalLine
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		journalLine := &journallinepb.JournalLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, journalLine); err != nil {
			continue
		}
		journalLines = append(journalLines, journalLine)
	}
	return &journallinepb.ListJournalLinesResponse{Data: journalLines}, nil
}

func (r *SQLServerJournalLineRepository) GetJournalLineListPageData(ctx context.Context, req *journallinepb.GetJournalLineListPageDataRequest) (*journallinepb.GetJournalLineListPageDataResponse, error) {
	return nil, fmt.Errorf("GetJournalLineListPageData not yet implemented — Phase 2")
}

func (r *SQLServerJournalLineRepository) GetJournalLineItemPageData(ctx context.Context, req *journallinepb.GetJournalLineItemPageDataRequest) (*journallinepb.GetJournalLineItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJournalLineItemPageData not yet implemented — Phase 2")
}
