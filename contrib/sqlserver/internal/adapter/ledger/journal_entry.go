//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JournalEntry, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver journal_entry repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJournalEntryRepository(dbOps, tableName), nil
	})
}

// journalEntrySortableSQLCols is the fail-closed sort whitelist for journal entry list page data.
var journalEntrySortableSQLCols = []string{
	"entry_number", "entry_date", "status", "source_type",
	"date_created", "date_modified",
}

// SQLServerJournalEntryRepository implements journal_entry CRUD and lifecycle operations using SQL Server.
//
// SQL Server differences from the postgres gold standard:
//   - @pN placeholders; [ident] quoting; LIKE instead of ILIKE
//   - LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY
//   - active = 1 (BIT); no ::text casts
//   - PostJournalEntry UPDATE: $N → @pN
//   - workspace_id added to GetJournalEntryListPageData WHERE (postgres had it; preserved here)
type SQLServerJournalEntryRepository struct {
	journalentrypb.UnimplementedJournalEntryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerJournalEntryRepository creates a new SQL Server journal_entry repository.
func NewSQLServerJournalEntryRepository(dbOps interfaces.DatabaseOperation, tableName string) journalentrypb.JournalEntryDomainServiceServer {
	if tableName == "" {
		tableName = "journal_entry"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerJournalEntryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerJournalEntryRepository) CreateJournalEntry(ctx context.Context, req *journalentrypb.CreateJournalEntryRequest) (*journalentrypb.CreateJournalEntryResponse, error) {
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
	// Convert entry_date int64 millis → time.Time for DATETIME2 column.
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
	normalizeJournalEntryEnums(data)
	_, err = r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create journal_entry: %w", err)
	}
	return &journalentrypb.CreateJournalEntryResponse{
		Success: true,
		Data:    []*journalentrypb.JournalEntry{req.Data},
	}, nil
}

func (r *SQLServerJournalEntryRepository) ReadJournalEntry(ctx context.Context, req *journalentrypb.ReadJournalEntryRequest) (*journalentrypb.ReadJournalEntryResponse, error) {
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
	return &journalentrypb.ReadJournalEntryResponse{Success: true, Data: []*journalentrypb.JournalEntry{journalEntry}}, nil
}

func (r *SQLServerJournalEntryRepository) UpdateJournalEntry(ctx context.Context, req *journalentrypb.UpdateJournalEntryRequest) (*journalentrypb.UpdateJournalEntryResponse, error) {
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
	return &journalentrypb.UpdateJournalEntryResponse{Success: true, Data: []*journalentrypb.JournalEntry{journalEntry}}, nil
}

func (r *SQLServerJournalEntryRepository) DeleteJournalEntry(ctx context.Context, req *journalentrypb.DeleteJournalEntryRequest) (*journalentrypb.DeleteJournalEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_entry ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete journal_entry: %w", err)
	}
	return &journalentrypb.DeleteJournalEntryResponse{Success: true}, nil
}

func (r *SQLServerJournalEntryRepository) ListJournalEntries(ctx context.Context, req *journalentrypb.ListJournalEntriesRequest) (*journalentrypb.ListJournalEntriesResponse, error) {
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
	return &journalentrypb.ListJournalEntriesResponse{Success: true, Data: journalEntries}, nil
}

// GetJournalEntryListPageData retrieves journal entries with pagination, filtering, sorting, and search.
//
// SQL Server differences: @pN; workspace_id = @p1 (no ::text IS NULL); LIKE; OFFSET/FETCH pagination;
// COUNT(*) OVER () supported in SQL Server 2017+.
func (r *SQLServerJournalEntryRepository) GetJournalEntryListPageData(ctx context.Context, req *journalentrypb.GetJournalEntryListPageDataRequest) (*journalentrypb.GetJournalEntryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get journal entry list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

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

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	orderByClause, err := sqlserverCore.BuildOrderBy(journalEntrySortableSQLCols, req.GetSort(), "entry_date DESC")
	if err != nil {
		return nil, err
	}

	searchFields := []string{"je.description", "je.entry_number"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereStr := " AND je.workspace_id = @p1"
	if len(filterClauses) > 0 {
		whereStr += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, offset, limit)

	query := fmt.Sprintf(`
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
			WHERE je.active = 1%s
		)
		SELECT * FROM enriched
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY`,
		whereStr, orderByClause, offsetIdx, limitIdx)

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
			&id, &entryNumber, &description, &entryDate, &status, &sourceType,
			&sourceID, &fiscalPeriodID, &totalDebit, &totalCredit,
			&postedBy, &postedAt, &reversedBy, &reversedAt, &reversalEntryID,
			&notes, &active, &dateCreated, &dateModified, &total,
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

func (r *SQLServerJournalEntryRepository) GetJournalEntryItemPageData(ctx context.Context, req *journalentrypb.GetJournalEntryItemPageDataRequest) (*journalentrypb.GetJournalEntryItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJournalEntryItemPageData not yet implemented — Phase 2")
}

// PostJournalEntry transitions a DRAFT journal entry to POSTED status.
//
// SQL Server differences: @p1/@p2/@p3 instead of $1/$2/$3.
// workspace_id predicate NOT added here (PostJournalEntry uses the entry ID only
// for the status transition — matches the postgres pattern).
func (r *SQLServerJournalEntryRepository) PostJournalEntry(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) (*journalentrypb.PostJournalEntryResponse, error) {
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
		    SET status        = 'POSTED',
		        posted_by     = @p1,
		        posted_at     = @p2,
		        date_modified = @p2
		  WHERE id     = @p3
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

	return &journalentrypb.PostJournalEntryResponse{Success: true}, nil
}

func (r *SQLServerJournalEntryRepository) ReverseJournalEntry(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) (*journalentrypb.ReverseJournalEntryResponse, error) {
	return nil, fmt.Errorf("ReverseJournalEntry not yet implemented — Phase 2")
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
