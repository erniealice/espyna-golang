//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	journallinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_line"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JournalLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres journal_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJournalLineRepository(dbOps, tableName), nil
	})
}

// PostgresJournalLineRepository implements journal_line CRUD operations using PostgreSQL.
// Journal lines are child entities of journal entries — typically queried via their parent entry.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_journal_line_journal_entry_id ON journal_line(journal_entry_id)
//   - CREATE INDEX idx_journal_line_account_id ON journal_line(account_id)
//   - CREATE INDEX idx_journal_line_active ON journal_line(active)
//
// TODO Phase 2: Implement GetJournalLineListPageData with CTE + account name join
// TODO Phase 2: Implement GetJournalLineItemPageData with account and entry enrichment
type PostgresJournalLineRepository struct {
	journallinepb.UnimplementedJournalLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJournalLineRepository creates a new PostgreSQL journal_line repository.
func NewPostgresJournalLineRepository(dbOps interfaces.DatabaseOperation, tableName string) journallinepb.JournalLineDomainServiceServer {
	if tableName == "" {
		tableName = "journal_line"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJournalLineRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJournalLine creates a new journal_line using common PostgreSQL operations.
func (r *PostgresJournalLineRepository) CreateJournalLine(ctx context.Context, req *journallinepb.CreateJournalLineRequest) (*journallinepb.CreateJournalLineResponse, error) {
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

	return &journallinepb.CreateJournalLineResponse{
		Data: []*journallinepb.JournalLine{journalLine},
	}, nil
}

// ReadJournalLine retrieves a journal_line by ID using common PostgreSQL operations.
func (r *PostgresJournalLineRepository) ReadJournalLine(ctx context.Context, req *journallinepb.ReadJournalLineRequest) (*journallinepb.ReadJournalLineResponse, error) {
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

	return &journallinepb.ReadJournalLineResponse{
		Data: []*journallinepb.JournalLine{journalLine},
	}, nil
}

// UpdateJournalLine updates a journal_line using common PostgreSQL operations.
func (r *PostgresJournalLineRepository) UpdateJournalLine(ctx context.Context, req *journallinepb.UpdateJournalLineRequest) (*journallinepb.UpdateJournalLineResponse, error) {
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

	return &journallinepb.UpdateJournalLineResponse{
		Data: []*journallinepb.JournalLine{journalLine},
	}, nil
}

// DeleteJournalLine soft-deletes a journal_line using common PostgreSQL operations.
func (r *PostgresJournalLineRepository) DeleteJournalLine(ctx context.Context, req *journallinepb.DeleteJournalLineRequest) (*journallinepb.DeleteJournalLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_line ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete journal_line: %w", err)
	}

	return &journallinepb.DeleteJournalLineResponse{
		Success: true,
	}, nil
}

// ListJournalLines lists journal_lines using common PostgreSQL operations.
func (r *PostgresJournalLineRepository) ListJournalLines(ctx context.Context, req *journallinepb.ListJournalLinesRequest) (*journallinepb.ListJournalLinesResponse, error) {
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

	return &journallinepb.ListJournalLinesResponse{
		Data: journalLines,
	}, nil
}

// GetJournalLineListPageData - TODO Phase 2: CTE with account name join, filter by journal_entry_id.
func (r *PostgresJournalLineRepository) GetJournalLineListPageData(ctx context.Context, req *journallinepb.GetJournalLineListPageDataRequest) (*journallinepb.GetJournalLineListPageDataResponse, error) {
	// TODO Phase 2: JOIN with account table to get account names and codes
	return nil, fmt.Errorf("GetJournalLineListPageData not yet implemented — Phase 2")
}

// GetJournalLineItemPageData - TODO Phase 2: implement with account and parent entry enrichment.
func (r *PostgresJournalLineRepository) GetJournalLineItemPageData(ctx context.Context, req *journallinepb.GetJournalLineItemPageDataRequest) (*journallinepb.GetJournalLineItemPageDataResponse, error) {
	// TODO Phase 2: fetch line + account + parent journal_entry context
	return nil, fmt.Errorf("GetJournalLineItemPageData not yet implemented — Phase 2")
}