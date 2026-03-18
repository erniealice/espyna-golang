
package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JournalEntry, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres journal_entry repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresJournalEntryRepository(dbOps, tableName), nil
	})
}

// PostgresJournalEntryRepository implements journal_entry CRUD and lifecycle operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_journal_entry_status ON journal_entry(status)
//   - CREATE INDEX idx_journal_entry_fiscal_period_id ON journal_entry(fiscal_period_id)
//   - CREATE INDEX idx_journal_entry_entry_date ON journal_entry(entry_date DESC)
//   - CREATE INDEX idx_journal_entry_reference_number ON journal_entry(reference_number)
//   - CREATE INDEX idx_journal_entry_source_type ON journal_entry(source_type)
//
// TODO Phase 2: Implement GetJournalEntryListPageData with CTE + search/pagination/fiscal period filter
// TODO Phase 2: Implement GetJournalEntryItemPageData with enriched journal lines
// TODO Phase 2: Implement PostJournalEntry — validates debit=credit balance, sets status=POSTED, locks fiscal period
// TODO Phase 2: Implement ReverseJournalEntry — creates mirror entry, sets status=REVERSED on original
type PostgresJournalEntryRepository struct {
	journalentrypb.UnimplementedJournalEntryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJournalEntryRepository creates a new PostgreSQL journal_entry repository.
func NewPostgresJournalEntryRepository(dbOps interfaces.DatabaseOperation, tableName string) journalentrypb.JournalEntryDomainServiceServer {
	if tableName == "" {
		tableName = "journal_entry"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJournalEntryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJournalEntry creates a new journal_entry using common PostgreSQL operations.
func (r *PostgresJournalEntryRepository) CreateJournalEntry(ctx context.Context, req *journalentrypb.CreateJournalEntryRequest) (*journalentrypb.CreateJournalEntryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("journal_entry data is required")
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
		return nil, fmt.Errorf("failed to create journal_entry: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	journalEntry := &journalentrypb.JournalEntry{}
	if err := protojson.Unmarshal(resultJSON, journalEntry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &journalentrypb.CreateJournalEntryResponse{
		Success: true,
		Data:    []*journalentrypb.JournalEntry{journalEntry},
	}, nil
}

// ReadJournalEntry retrieves a journal_entry by ID using common PostgreSQL operations.
func (r *PostgresJournalEntryRepository) ReadJournalEntry(ctx context.Context, req *journalentrypb.ReadJournalEntryRequest) (*journalentrypb.ReadJournalEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_entry ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read journal_entry: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	journalEntry := &journalentrypb.JournalEntry{}
	if err := protojson.Unmarshal(resultJSON, journalEntry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &journalentrypb.ReadJournalEntryResponse{
		Success: true,
		Data:    []*journalentrypb.JournalEntry{journalEntry},
	}, nil
}

// UpdateJournalEntry updates a journal_entry using common PostgreSQL operations.
func (r *PostgresJournalEntryRepository) UpdateJournalEntry(ctx context.Context, req *journalentrypb.UpdateJournalEntryRequest) (*journalentrypb.UpdateJournalEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_entry ID is required")
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
		return nil, fmt.Errorf("failed to update journal_entry: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	journalEntry := &journalentrypb.JournalEntry{}
	if err := protojson.Unmarshal(resultJSON, journalEntry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &journalentrypb.UpdateJournalEntryResponse{
		Success: true,
		Data:    []*journalentrypb.JournalEntry{journalEntry},
	}, nil
}

// DeleteJournalEntry soft-deletes a journal_entry using common PostgreSQL operations.
func (r *PostgresJournalEntryRepository) DeleteJournalEntry(ctx context.Context, req *journalentrypb.DeleteJournalEntryRequest) (*journalentrypb.DeleteJournalEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_entry ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete journal_entry: %w", err)
	}

	return &journalentrypb.DeleteJournalEntryResponse{
		Success: true,
	}, nil
}

// ListJournalEntries lists journal_entries using common PostgreSQL operations.
func (r *PostgresJournalEntryRepository) ListJournalEntries(ctx context.Context, req *journalentrypb.ListJournalEntriesRequest) (*journalentrypb.ListJournalEntriesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list journal_entries: %w", err)
	}

	var journalEntries []*journalentrypb.JournalEntry
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		journalEntry := &journalentrypb.JournalEntry{}
		if err := protojson.Unmarshal(resultJSON, journalEntry); err != nil {
			continue
		}
		journalEntries = append(journalEntries, journalEntry)
	}

	return &journalentrypb.ListJournalEntriesResponse{
		Success: true,
		Data:    journalEntries,
	}, nil
}

// GetJournalEntryListPageData - TODO Phase 2: CTE with fiscal_period filter, status filter, search by reference_number.
func (r *PostgresJournalEntryRepository) GetJournalEntryListPageData(ctx context.Context, req *journalentrypb.GetJournalEntryListPageDataRequest) (*journalentrypb.GetJournalEntryListPageDataResponse, error) {
	// TODO Phase 2: CTE with total_debit/total_credit aggregation, fiscal_period join, pagination
	return nil, fmt.Errorf("GetJournalEntryListPageData not yet implemented — Phase 2")
}

// GetJournalEntryItemPageData - TODO Phase 2: implement with enriched journal lines and account names.
func (r *PostgresJournalEntryRepository) GetJournalEntryItemPageData(ctx context.Context, req *journalentrypb.GetJournalEntryItemPageDataRequest) (*journalentrypb.GetJournalEntryItemPageDataResponse, error) {
	// TODO Phase 2: fetch entry + all journal lines + account names via JOIN
	return nil, fmt.Errorf("GetJournalEntryItemPageData not yet implemented — Phase 2")
}

// PostJournalEntry - TODO Phase 2: validate debit=credit balance, set status=POSTED, record posting timestamp.
func (r *PostgresJournalEntryRepository) PostJournalEntry(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) (*journalentrypb.PostJournalEntryResponse, error) {
	// TODO Phase 2: validate entry is DRAFT, sum debits == sum credits, update status to POSTED
	return nil, fmt.Errorf("PostJournalEntry not yet implemented — Phase 2")
}

// ReverseJournalEntry - TODO Phase 2: create mirror entry with swapped debits/credits, mark original REVERSED.
func (r *PostgresJournalEntryRepository) ReverseJournalEntry(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) (*journalentrypb.ReverseJournalEntryResponse, error) {
	// TODO Phase 2: create reversing entry, link back to original, set original status=REVERSED
	return nil, fmt.Errorf("ReverseJournalEntry not yet implemented — Phase 2")
}
