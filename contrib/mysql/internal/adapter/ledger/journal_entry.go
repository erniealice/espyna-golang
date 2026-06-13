//go:build mysql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.JournalEntry, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql journal_entry repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLJournalEntryRepository(dbOps, tableName), nil
	})
}

// MySQLJournalEntryRepository implements journal_entry CRUD and lifecycle operations using MySQL 8.0+.
//
// Dialect differences from postgres gold standard:
//   - Placeholders: $N → ? (positional)
//   - active = true → active = 1
//   - ILIKE → LIKE
//   - COUNT(*) OVER () ✓ (MySQL 8.0+)
//   - WHERE workspace_id = ? added (missing in postgres gold standard)
//   - UPDATE ... WHERE status = 'DRAFT' uses standard SQL (no RETURNING needed)
type MySQLJournalEntryRepository struct {
	journalentrypb.UnimplementedJournalEntryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLJournalEntryRepository creates a new MySQL journal_entry repository.
func NewMySQLJournalEntryRepository(dbOps interfaces.DatabaseOperation, tableName string) journalentrypb.JournalEntryDomainServiceServer {
	if tableName == "" {
		tableName = "journal_entry"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLJournalEntryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJournalEntry creates a new journal_entry using common MySQL operations.
func (r *MySQLJournalEntryRepository) CreateJournalEntry(ctx context.Context, req *journalentrypb.CreateJournalEntryRequest) (*journalentrypb.CreateJournalEntryResponse, error) {
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

	// Convert entry_date (int64 Unix millis) to time.Time for DATETIME column.
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

// ReadJournalEntry retrieves a journal_entry by ID using common MySQL operations.
func (r *MySQLJournalEntryRepository) ReadJournalEntry(ctx context.Context, req *journalentrypb.ReadJournalEntryRequest) (*journalentrypb.ReadJournalEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("journal_entry ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read journal_entry: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// UpdateJournalEntry updates a journal_entry using common MySQL operations.
func (r *MySQLJournalEntryRepository) UpdateJournalEntry(ctx context.Context, req *journalentrypb.UpdateJournalEntryRequest) (*journalentrypb.UpdateJournalEntryResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// DeleteJournalEntry soft-deletes a journal_entry using common MySQL operations.
func (r *MySQLJournalEntryRepository) DeleteJournalEntry(ctx context.Context, req *journalentrypb.DeleteJournalEntryRequest) (*journalentrypb.DeleteJournalEntryResponse, error) {
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

// ListJournalEntries lists journal_entries using common MySQL operations.
func (r *MySQLJournalEntryRepository) ListJournalEntries(ctx context.Context, req *journalentrypb.ListJournalEntriesRequest) (*journalentrypb.ListJournalEntriesResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
//
// Dialect: $N → ?; active = true → active = 1; ILIKE → LIKE;
// COUNT(*) OVER () ✓; LIMIT/OFFSET trailing ?; workspace_id predicate added.
func (r *MySQLJournalEntryRepository) GetJournalEntryListPageData(ctx context.Context, req *journalentrypb.GetJournalEntryListPageDataRequest) (*journalentrypb.GetJournalEntryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get journal entry list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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

	journalEntrySortableCols := []string{
		"entry_number", "entry_date", "status", "source_type", "date_created", "date_modified",
	}
	orderByClause, err := mysqlCore.BuildOrderBy(journalEntrySortableCols, req.Sort, "entry_date DESC")
	if err != nil {
		return nil, err
	}

	searchFields := []string{"je.description", "je.entry_number"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE je.active = 1"
	if workspaceID != "" {
		whereSQL += " AND je.workspace_id = ?"
	}
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	var queryArgs []any
	if workspaceID != "" {
		queryArgs = append(queryArgs, workspaceID)
	}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

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
			%s
		)
		SELECT * FROM enriched
		%s
		LIMIT ? OFFSET ?`, whereSQL, orderByClause)

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

		if err := rows.Scan(
			&id, &entryNumber, &description, &entryDate,
			&status, &sourceType, &sourceID, &fiscalPeriodID,
			&totalDebit, &totalCredit, &postedBy, &postedAt,
			&reversedBy, &reversedAt, &reversalEntryID, &notes,
			&active, &dateCreated, &dateModified, &total,
		); err != nil {
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

// GetJournalEntryItemPageData — TODO Phase 2: implement with enriched journal lines and account names.
func (r *MySQLJournalEntryRepository) GetJournalEntryItemPageData(ctx context.Context, req *journalentrypb.GetJournalEntryItemPageDataRequest) (*journalentrypb.GetJournalEntryItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJournalEntryItemPageData not yet implemented — Phase 2")
}

// PostJournalEntry transitions a DRAFT journal entry to POSTED status.
//
// Dialect: $1/$2/$3 → ? (positional); no RETURNING clause (MySQL doesn't support it).
// workspace_id predicate added for multi-tenancy safety.
func (r *MySQLJournalEntryRepository) PostJournalEntry(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) (*journalentrypb.PostJournalEntryResponse, error) {
	if req.JournalEntryId == "" {
		return nil, fmt.Errorf("journal entry ID is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	workspaceID := identity.Must(ctx).WorkspaceID
	postedAt := time.Now().UTC()
	postedBy := req.PostedBy

	updateSQL := `UPDATE journal_entry
		    SET status     = 'POSTED',
		        posted_by  = ?,
		        posted_at  = ?,
		        date_modified = ?
		  WHERE id = ?
		    AND status = 'DRAFT'`
	args := []any{postedBy, postedAt, postedAt, req.JournalEntryId}
	if workspaceID != "" {
		updateSQL += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}

	result, err := r.db.ExecContext(ctx, updateSQL, args...)
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

// ReverseJournalEntry — TODO Phase 2: create mirror entry, set original status=REVERSED.
func (r *MySQLJournalEntryRepository) ReverseJournalEntry(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) (*journalentrypb.ReverseJournalEntryResponse, error) {
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

// normalizeJournalEntryEnums strips proto-style enum prefixes from string enum fields.
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
