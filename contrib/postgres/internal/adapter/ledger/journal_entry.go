//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JournalEntry, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres journal_entry repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
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

	// Convert entry_date (int64 Unix millis) to time.Time for TIMESTAMPTZ column.
	// protojson serialises int64 proto fields as JSON strings (e.g. "1774716817031")
	// for JavaScript compatibility. The generic postgres adapter cannot cast these
	// strings to TIMESTAMPTZ, so we convert them here to time.Time before calling Create.
	if rawDate, ok := data["entryDate"]; ok && rawDate != nil {
		switch v := rawDate.(type) {
		case float64:
			if v > 0 {
				data["entryDate"] = time.UnixMilli(int64(v)).UTC()
			}
		case string:
			if v != "" {
				if millis, err := strconv.ParseInt(v, 10, 64); err == nil && millis > 0 {
					data["entryDate"] = time.UnixMilli(millis).UTC()
				}
			}
		}
	}

	// Normalize enum values: strip proto-style prefixes so DB stores short names
	// consistent with seeded data (e.g. "DRAFT" not "JOURNAL_ENTRY_STATUS_DRAFT").
	normalizeJournalEntryEnums(data)

	_, err = r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create journal_entry: %w", err)
	}

	// Return the request proto directly — it already has correct Go enum values
	// and is the authoritative source of what was saved.
	// (The DB result uses short enum strings that protojson cannot unmarshal back.)
	return &journalentrypb.CreateJournalEntryResponse{
		Success: true,
		Data:    []*journalentrypb.JournalEntry{req.Data},
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, journalEntry); err != nil {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, journalEntry); err != nil {
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
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, journalEntry); err != nil {
			continue
		}
		journalEntries = append(journalEntries, journalEntry)
	}

	return &journalentrypb.ListJournalEntriesResponse{
		Success: true,
		Data:    journalEntries,
	}, nil
}

// GetJournalEntryListPageData retrieves journal entries with pagination, filtering, sorting, and search.
func (r *PostgresJournalEntryRepository) GetJournalEntryListPageData(ctx context.Context, req *journalentrypb.GetJournalEntryListPageDataRequest) (*journalentrypb.GetJournalEntryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get journal entry list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Default pagination values
	limit := int32(100)
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

	// Sort allowlist
	sortAllowlist := map[string]string{
		"entry_number":  "entry_number",
		"entry_date":    "entry_date",
		"status":        "status",
		"source_type":   "source_type",
		"date_created":  "date_created",
		"date_modified": "date_modified",
	}
	sortCol := "entry_date"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		if col, ok := sortAllowlist[req.Sort.Fields[0].Field]; ok {
			sortCol = col
		}
		if req.Sort.Fields[0].Direction == 1 { // SortDirection_DESC = 1
			sortOrder = "DESC"
		} else {
			sortOrder = "ASC"
		}
	}

	// Build WHERE clauses
	searchFields := []string{"je.description", "je.entry_number"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)

	var whereStr string
	if len(filterClauses) > 0 {
		whereStr = " AND " + strings.Join(filterClauses, " AND ")
	}

	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	queryArgs := append(filterArgs, limit, offset) //nolint:gocritic

	query := `
		WITH enriched AS (
			SELECT
				je.id,
				je.entry_number,
				je.description,
				je.entry_date,
				je.status,
				je.source_type,
				je.source_id,
				je.fiscal_period_id,
				je.total_debit,
				je.total_credit,
				je.posted_by,
				je.posted_at,
				je.reversed_by,
				je.reversed_at,
				je.reversal_entry_id,
				je.notes,
				je.active,
				je.date_created,
				je.date_modified,
				COUNT(*) OVER() AS total_count
			FROM journal_entry je
			WHERE je.active = true` + whereStr + `
		)
		SELECT * FROM enriched
		ORDER BY ` + sortCol + ` ` + sortOrder + fmt.Sprintf(`
		LIMIT $%d OFFSET $%d`, limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query journal entry list page data: %w", err)
	}
	defer rows.Close()

	var entries []*journalentrypb.JournalEntry
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			entryNumber     string
			description     string
			entryDate       *time.Time
			status          string
			sourceType      string
			sourceID        *string
			fiscalPeriodID  *string
			totalDebit      int64
			totalCredit     int64
			postedBy        *string
			postedAt        *time.Time
			reversedBy      *string
			reversedAt      *time.Time
			reversalEntryID *string
			notes           *string
			active          bool
			dateCreated     time.Time
			dateModified    time.Time
			total           int64
		)

		err := rows.Scan(
			&id,
			&entryNumber,
			&description,
			&entryDate,
			&status,
			&sourceType,
			&sourceID,
			&fiscalPeriodID,
			&totalDebit,
			&totalCredit,
			&postedBy,
			&postedAt,
			&reversedBy,
			&reversedAt,
			&reversalEntryID,
			&notes,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan journal entry row: %w", err)
		}

		totalCount = total

		entry := &journalentrypb.JournalEntry{
			Id:          id,
			EntryNumber: entryNumber,
			Description: description,
			TotalDebit:  totalDebit,
			TotalCredit: totalCredit,
			Active:      active,
			Status:      parseJournalEntryStatus(status),
			SourceType:  parseJournalSourceType(sourceType),
		}

		if entryDate != nil && !entryDate.IsZero() {
			ms := entryDate.UnixMilli()
			entry.EntryDate = ms
			s := entryDate.Format("2006-01-02")
			entry.EntryDateString = &s
		}
		if sourceID != nil {
			entry.SourceId = sourceID
		}
		if fiscalPeriodID != nil {
			entry.FiscalPeriodId = fiscalPeriodID
		}
		if postedBy != nil {
			entry.PostedBy = postedBy
		}
		if postedAt != nil && !postedAt.IsZero() {
			ms := postedAt.UnixMilli()
			entry.PostedAt = &ms
			s := postedAt.Format(time.RFC3339)
			entry.PostedAtString = &s
		}
		if reversedBy != nil {
			entry.ReversedBy = reversedBy
		}
		if reversedAt != nil && !reversedAt.IsZero() {
			ms := reversedAt.UnixMilli()
			entry.ReversedAt = &ms
			s := reversedAt.Format(time.RFC3339)
			entry.ReversedAtString = &s
		}
		if reversalEntryID != nil {
			entry.ReversalEntryId = reversalEntryID
		}
		if notes != nil {
			entry.Notes = notes
		}
		if !dateCreated.IsZero() {
			ms := dateCreated.UnixMilli()
			entry.DateCreated = &ms
			s := dateCreated.Format(time.RFC3339)
			entry.DateCreatedString = &s
		}
		if !dateModified.IsZero() {
			ms := dateModified.UnixMilli()
			entry.DateModified = &ms
			s := dateModified.Format(time.RFC3339)
			entry.DateModifiedString = &s
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating journal entry rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &journalentrypb.GetJournalEntryListPageDataResponse{
		JournalEntryList: entries,
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

func parseJournalEntryStatus(s string) journalentrypb.JournalEntryStatus {
	switch strings.ToUpper(s) {
	case "DRAFT":
		return journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_DRAFT
	case "POSTED":
		return journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_POSTED
	case "REVERSED":
		return journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_REVERSED
	default:
		return journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_UNSPECIFIED
	}
}

func parseJournalSourceType(s string) journalentrypb.JournalSourceType {
	switch strings.ToUpper(s) {
	case "MANUAL":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_MANUAL
	case "REVENUE":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_REVENUE
	case "EXPENDITURE":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_EXPENDITURE
	case "COLLECTION":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_COLLECTION
	case "DISBURSEMENT":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_DISBURSEMENT
	case "DEPRECIATION":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_DEPRECIATION
	case "ASSET_ACQUISITION":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_ASSET_ACQUISITION
	case "ASSET_DISPOSAL":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_ASSET_DISPOSAL
	case "PREPAYMENT":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_PREPAYMENT
	case "PREPAYMENT_AMORTIZATION":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_PREPAYMENT_AMORTIZATION
	case "LOAN_RECEIPT":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_LOAN_RECEIPT
	case "LOAN_PAYMENT":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_LOAN_PAYMENT
	case "PETTY_CASH_REPLENISHMENT":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_PETTY_CASH_REPLENISHMENT
	case "BAD_DEBT_PROVISION":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_BAD_DEBT_PROVISION
	case "DEFERRED_REVENUE":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_DEFERRED_REVENUE
	case "EQUITY_CONTRIBUTION":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_EQUITY_CONTRIBUTION
	case "EQUITY_WITHDRAWAL":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_EQUITY_WITHDRAWAL
	case "EQUITY_DISTRIBUTION":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_EQUITY_DISTRIBUTION
	case "YEAR_END_CLOSE":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_YEAR_END_CLOSE
	case "RECURRING":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_RECURRING
	case "PAYROLL":
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_PAYROLL
	default:
		return journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_MANUAL
	}
}

// GetJournalEntryItemPageData - TODO Phase 2: implement with enriched journal lines and account names.
func (r *PostgresJournalEntryRepository) GetJournalEntryItemPageData(ctx context.Context, req *journalentrypb.GetJournalEntryItemPageDataRequest) (*journalentrypb.GetJournalEntryItemPageDataResponse, error) {
	// TODO Phase 2: fetch entry + all journal lines + account names via JOIN
	return nil, fmt.Errorf("GetJournalEntryItemPageData not yet implemented — Phase 2")
}

// PostJournalEntry transitions a DRAFT journal entry to POSTED status.
// Validates that the entry exists and is in DRAFT status, then sets
// status=POSTED and records the posted_by user and posted_at timestamp.
func (r *PostgresJournalEntryRepository) PostJournalEntry(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) (*journalentrypb.PostJournalEntryResponse, error) {
	if req.JournalEntryId == "" {
		return nil, fmt.Errorf("journal entry ID is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	postedAt := time.Now().UTC()
	postedBy := req.PostedBy

	result, err := r.db.ExecContext(ctx,
		`UPDATE journal_entry
		    SET status     = 'POSTED',
		        posted_by  = $1,
		        posted_at  = $2,
		        date_modified = $2
		  WHERE id = $3
		    AND status = 'DRAFT'`,
		postedBy, postedAt, req.JournalEntryId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to post journal entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("journal entry %s not found or not in DRAFT status", req.JournalEntryId)
	}

	return &journalentrypb.PostJournalEntryResponse{
		Success: true,
	}, nil
}

// ReverseJournalEntry - TODO Phase 2: create mirror entry with swapped debits/credits, mark original REVERSED.
func (r *PostgresJournalEntryRepository) ReverseJournalEntry(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) (*journalentrypb.ReverseJournalEntryResponse, error) {
	// TODO Phase 2: create reversing entry, link back to original, set original status=REVERSED
	return nil, fmt.Errorf("ReverseJournalEntry not yet implemented — Phase 2")
}

// normalizeJournalEntryEnums strips the proto-style enum prefixes from string enum fields
// so the DB stores short names consistent with seeded data.
//
// protojson serializes enum values with their full proto names, e.g.:
//
//	"JOURNAL_ENTRY_STATUS_DRAFT"  → stored as "DRAFT"
//	"JOURNAL_SOURCE_TYPE_MANUAL"  → stored as "MANUAL"
//
// The data map uses camelCase keys (before normalizeKeys is called inside dbOps.Create).
func normalizeJournalEntryEnums(data map[string]any) {
	stripPrefix := func(key, prefix string) {
		if v, ok := data[key]; ok {
			if s, ok := v.(string); ok {
				data[key] = strings.TrimPrefix(s, prefix)
			}
		}
	}
	stripPrefix("status", "JOURNAL_ENTRY_STATUS_")
	stripPrefix("sourceType", "JOURNAL_SOURCE_TYPE_")
}
