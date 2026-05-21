//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Account, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres account repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresAccountRepository(dbOps, tableName), nil
	})
}

// PostgresAccountRepository implements account CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_account_active ON account(active)
//   - CREATE INDEX idx_account_account_group_id ON account(account_group_id)
//   - CREATE INDEX idx_account_element ON account(element)
//   - CREATE INDEX idx_account_date_created ON account(date_created DESC)
//   - CREATE INDEX idx_account_code ON account(code)
//
// TODO Phase 2: Implement GetAccountListPageData with CTE + search/pagination
// TODO Phase 2: Implement GetAccountItemPageData with enriched data
// TODO Phase 2: Implement GetAccountTreePageData for hierarchical CoA display
type PostgresAccountRepository struct {
	accountpb.UnimplementedAccountDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresAccountRepository creates a new PostgreSQL account repository.
func NewPostgresAccountRepository(dbOps interfaces.DatabaseOperation, tableName string) accountpb.AccountDomainServiceServer {
	if tableName == "" {
		tableName = "account"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresAccountRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccount creates a new account using common PostgreSQL operations.
func (r *PostgresAccountRepository) CreateAccount(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("account data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Normalize enum values: strip proto-style prefixes so DB stores short names
	// consistent with seeded data (e.g. "ASSET" not "ACCOUNT_ELEMENT_ASSET").
	normalizeAccountEnums(data)

	_, err = r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Return the request proto directly — it already has correct Go enum values
	// and is the authoritative source of what was saved.
	return &accountpb.CreateAccountResponse{
		Data:    []*accountpb.Account{req.Data},
		Success: true,
	}, nil
}

// ReadAccount retrieves an account by ID using a direct SQL query.
// Uses manual field scanning (same approach as GetAccountItemPageData) to correctly
// handle DB short-name enums (e.g. "ASSET") that protojson.Unmarshal cannot parse.
func (r *PostgresAccountRepository) ReadAccount(ctx context.Context, req *accountpb.ReadAccountRequest) (*accountpb.ReadAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	query := `
		SELECT
			id, code, name, description, element, classification,
			group_id, parent_id, cash_flow_activity, normal_balance,
			is_system_account, is_contra, status, notes,
			active, date_created, date_modified
		FROM account
		WHERE id = $1
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, req.Data.Id)

	var (
		id               string
		code             string
		name             string
		description      *string
		element          string
		classification   string
		groupID          *string
		parentID         *string
		cashFlowActivity string
		normalBalance    string
		isSystemAccount  bool
		isContra         bool
		status           string
		notes            *string
		active           bool
		dateCreated      time.Time
		dateModified     time.Time
	)

	err := row.Scan(
		&id,
		&code,
		&name,
		&description,
		&element,
		&classification,
		&groupID,
		&parentID,
		&cashFlowActivity,
		&normalBalance,
		&isSystemAccount,
		&isContra,
		&status,
		&notes,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account with ID '%s' not found", req.Data.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read account: %w", err)
	}

	account := &accountpb.Account{
		Id:               id,
		Code:             code,
		Name:             name,
		Element:          parseAccountElement(element),
		Classification:   parseAccountClassification(classification),
		CashFlowActivity: parseCashFlowActivity(cashFlowActivity),
		NormalBalance:    parseNormalBalance(normalBalance),
		IsSystemAccount:  isSystemAccount,
		IsContra:         isContra,
		Status:           parseAccountStatus(status),
		Active:           active,
	}

	if description != nil {
		account.Description = description
	}
	if groupID != nil {
		account.GroupId = groupID
	}
	if parentID != nil {
		account.ParentId = parentID
	}
	if notes != nil {
		account.Notes = notes
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		account.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		account.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		account.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		account.DateModifiedString = &dmStr
	}

	return &accountpb.ReadAccountResponse{
		Data:    []*accountpb.Account{account},
		Success: true,
	}, nil
}

// UpdateAccount updates an account using common PostgreSQL operations.
func (r *PostgresAccountRepository) UpdateAccount(ctx context.Context, req *accountpb.UpdateAccountRequest) (*accountpb.UpdateAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Normalize enum values: strip proto-style prefixes so DB stores short names.
	normalizeAccountEnums(data)

	_, err = r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	// Return the request proto directly — it already has correct Go enum values.
	return &accountpb.UpdateAccountResponse{
		Data: []*accountpb.Account{req.Data},
	}, nil
}

// DeleteAccount soft-deletes an account using common PostgreSQL operations.
func (r *PostgresAccountRepository) DeleteAccount(ctx context.Context, req *accountpb.DeleteAccountRequest) (*accountpb.DeleteAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete account: %w", err)
	}

	return &accountpb.DeleteAccountResponse{
		Success: true,
	}, nil
}

// ListAccounts lists accounts using common PostgreSQL operations.
func (r *PostgresAccountRepository) ListAccounts(ctx context.Context, req *accountpb.ListAccountsRequest) (*accountpb.ListAccountsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var accounts []*accountpb.Account
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		account := &accountpb.Account{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, account); err != nil {
			continue
		}
		accounts = append(accounts, account)
	}

	return &accountpb.ListAccountsResponse{
		Data: accounts,
	}, nil
}

// GetAccountListPageData retrieves accounts with pagination, filtering, sorting, and search.
func (r *PostgresAccountRepository) GetAccountListPageData(ctx context.Context, req *accountpb.GetAccountListPageDataRequest) (*accountpb.GetAccountListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get account list page data request is required")
	}

	// Default pagination values
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

	// Sort with allowlist validation.
	// Use bare column names (no table alias) because ORDER BY applies to the
	// outer "SELECT * FROM enriched" which has no "a" alias.
	sortAllowlist := map[string]string{
		"code":          "code",
		"name":          "name",
		"element":       "element",
		"status":        "status",
		"date_created":  "date_created",
		"date_modified": "date_modified",
	}
	sortCol := "code"
	sortOrder := "ASC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		if col, ok := sortAllowlist[req.Sort.Fields[0].Field]; ok {
			sortCol = col
		}
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	// Build parameterized WHERE clauses via shared helper (starts at $1)
	searchFields := []string{"a.name", "a.code"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)

	var whereStr string
	if len(filterClauses) > 0 {
		whereStr = " AND " + strings.Join(filterClauses, " AND ")
	}

	// Parameterized LIMIT/OFFSET come after filter args
	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	queryArgs := append(filterArgs, limit, offset) //nolint:gocritic

	query := `
		WITH enriched AS (
			SELECT
				a.id,
				a.code,
				a.name,
				a.description,
				a.element,
				a.classification,
				a.group_id,
				a.parent_id,
				a.cash_flow_activity,
				a.normal_balance,
				a.is_system_account,
				a.is_contra,
				a.status,
				a.notes,
				a.active,
				a.date_created,
				a.date_modified,
				COUNT(*) OVER() AS total_count
			FROM account a
			WHERE a.active = true` + whereStr + `
		)
		SELECT * FROM enriched
		ORDER BY ` + sortCol + ` ` + sortOrder + fmt.Sprintf(`
		LIMIT $%d OFFSET $%d`, limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query account list page data: %w", err)
	}
	defer rows.Close()

	var accounts []*accountpb.Account
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			code             string
			name             string
			description      *string
			element          string
			classification   string
			groupID          *string
			parentID         *string
			cashFlowActivity string
			normalBalance    string
			isSystemAccount  bool
			isContra         bool
			status           string
			notes            *string
			active           bool
			dateCreated      time.Time
			dateModified     time.Time
			total            int64
		)

		err := rows.Scan(
			&id,
			&code,
			&name,
			&description,
			&element,
			&classification,
			&groupID,
			&parentID,
			&cashFlowActivity,
			&normalBalance,
			&isSystemAccount,
			&isContra,
			&status,
			&notes,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row: %w", err)
		}

		totalCount = total

		account := &accountpb.Account{
			Id:               id,
			Code:             code,
			Name:             name,
			Element:          parseAccountElement(element),
			Classification:   parseAccountClassification(classification),
			CashFlowActivity: parseCashFlowActivity(cashFlowActivity),
			NormalBalance:    parseNormalBalance(normalBalance),
			IsSystemAccount:  isSystemAccount,
			IsContra:         isContra,
			Status:           parseAccountStatus(status),
			Active:           active,
		}

		if description != nil {
			account.Description = description
		}
		if groupID != nil {
			account.GroupId = groupID
		}
		if parentID != nil {
			account.ParentId = parentID
		}
		if notes != nil {
			account.Notes = notes
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			account.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			account.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			account.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			account.DateModifiedString = &dmStr
		}

		accounts = append(accounts, account)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &accountpb.GetAccountListPageDataResponse{
		AccountList: accounts,
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

// GetAccountItemPageData retrieves a single account by ID.
func (r *PostgresAccountRepository) GetAccountItemPageData(ctx context.Context, req *accountpb.GetAccountItemPageDataRequest) (*accountpb.GetAccountItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get account item page data request is required")
	}
	if req.AccountId == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	query := `
		SELECT
			id, code, name, description, element, classification,
			group_id, parent_id, cash_flow_activity, normal_balance,
			is_system_account, is_contra, status, notes,
			active, date_created, date_modified
		FROM account
		WHERE id = $1 AND active = true
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, req.AccountId)

	var (
		id               string
		code             string
		name             string
		description      *string
		element          string
		classification   string
		groupID          *string
		parentID         *string
		cashFlowActivity string
		normalBalance    string
		isSystemAccount  bool
		isContra         bool
		status           string
		notes            *string
		active           bool
		dateCreated      time.Time
		dateModified     time.Time
	)

	err := row.Scan(
		&id,
		&code,
		&name,
		&description,
		&element,
		&classification,
		&groupID,
		&parentID,
		&cashFlowActivity,
		&normalBalance,
		&isSystemAccount,
		&isContra,
		&status,
		&notes,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account with ID '%s' not found", req.AccountId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query account item page data: %w", err)
	}

	account := &accountpb.Account{
		Id:               id,
		Code:             code,
		Name:             name,
		Element:          parseAccountElement(element),
		Classification:   parseAccountClassification(classification),
		CashFlowActivity: parseCashFlowActivity(cashFlowActivity),
		NormalBalance:    parseNormalBalance(normalBalance),
		IsSystemAccount:  isSystemAccount,
		IsContra:         isContra,
		Status:           parseAccountStatus(status),
		Active:           active,
	}

	if description != nil {
		account.Description = description
	}
	if groupID != nil {
		account.GroupId = groupID
	}
	if parentID != nil {
		account.ParentId = parentID
	}
	if notes != nil {
		account.Notes = notes
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		account.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		account.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		account.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		account.DateModifiedString = &dmStr
	}

	return &accountpb.GetAccountItemPageDataResponse{
		Account: account,
		Success: true,
	}, nil
}

// parseAccountElement converts a DB short name (e.g. "ASSET") to the proto enum value.
func parseAccountElement(s string) accountpb.AccountElement {
	switch s {
	case "ASSET":
		return accountpb.AccountElement_ACCOUNT_ELEMENT_ASSET
	case "LIABILITY":
		return accountpb.AccountElement_ACCOUNT_ELEMENT_LIABILITY
	case "EQUITY":
		return accountpb.AccountElement_ACCOUNT_ELEMENT_EQUITY
	case "REVENUE":
		return accountpb.AccountElement_ACCOUNT_ELEMENT_REVENUE
	case "EXPENSE":
		return accountpb.AccountElement_ACCOUNT_ELEMENT_EXPENSE
	default:
		return accountpb.AccountElement_ACCOUNT_ELEMENT_UNSPECIFIED
	}
}

// parseAccountClassification converts a DB short name to the proto enum value.
func parseAccountClassification(s string) accountpb.AccountClassification {
	switch s {
	case "CURRENT_ASSET":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_CURRENT_ASSET
	case "NON_CURRENT_ASSET":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_NON_CURRENT_ASSET
	case "CURRENT_LIABILITY":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_CURRENT_LIABILITY
	case "NON_CURRENT_LIABILITY":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_NON_CURRENT_LIABILITY
	case "EQUITY":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_EQUITY
	case "OPERATING_REVENUE":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_OPERATING_REVENUE
	case "OTHER_INCOME":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_OTHER_INCOME
	case "COST_OF_SALES":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_COST_OF_SALES
	case "OPERATING_EXPENSE":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_OPERATING_EXPENSE
	case "FINANCE_COST":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_FINANCE_COST
	case "INCOME_TAX":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_INCOME_TAX
	case "OTHER_EXPENSE":
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_OTHER_EXPENSE
	default:
		return accountpb.AccountClassification_ACCOUNT_CLASSIFICATION_UNSPECIFIED
	}
}

// parseCashFlowActivity converts a DB short name to the proto enum value.
func parseCashFlowActivity(s string) accountpb.CashFlowActivity {
	switch s {
	case "OPERATING":
		return accountpb.CashFlowActivity_CASH_FLOW_ACTIVITY_OPERATING
	case "INVESTING":
		return accountpb.CashFlowActivity_CASH_FLOW_ACTIVITY_INVESTING
	case "FINANCING":
		return accountpb.CashFlowActivity_CASH_FLOW_ACTIVITY_FINANCING
	case "NONE":
		return accountpb.CashFlowActivity_CASH_FLOW_ACTIVITY_NONE
	default:
		return accountpb.CashFlowActivity_CASH_FLOW_ACTIVITY_UNSPECIFIED
	}
}

// parseNormalBalance converts a DB short name to the proto enum value.
func parseNormalBalance(s string) accountpb.NormalBalance {
	switch s {
	case "DEBIT":
		return accountpb.NormalBalance_NORMAL_BALANCE_DEBIT
	case "CREDIT":
		return accountpb.NormalBalance_NORMAL_BALANCE_CREDIT
	default:
		return accountpb.NormalBalance_NORMAL_BALANCE_UNSPECIFIED
	}
}

// parseAccountStatus converts a DB short name to the proto enum value.
func parseAccountStatus(s string) accountpb.AccountStatus {
	switch s {
	case "ACTIVE":
		return accountpb.AccountStatus_ACCOUNT_STATUS_ACTIVE
	case "INACTIVE":
		return accountpb.AccountStatus_ACCOUNT_STATUS_INACTIVE
	case "LOCKED":
		return accountpb.AccountStatus_ACCOUNT_STATUS_LOCKED
	default:
		return accountpb.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED
	}
}

// GetAccountTreePageData - TODO Phase 2: implement recursive CTE for hierarchical CoA display.
func (r *PostgresAccountRepository) GetAccountTreePageData(ctx context.Context, req *accountpb.GetAccountTreePageDataRequest) (*accountpb.GetAccountTreePageDataResponse, error) {
	// TODO Phase 2: recursive CTE (WITH RECURSIVE) to build account tree grouped by element/classification
	return nil, fmt.Errorf("GetAccountTreePageData not yet implemented — Phase 2")
}

// normalizeAccountEnums strips proto-style enum prefixes from string enum fields
// so the DB stores short names consistent with seeded data.
//
// protojson serializes enum values with their full proto names, e.g.:
//
//	"ACCOUNT_ELEMENT_ASSET"                   → stored as "ASSET"
//	"ACCOUNT_CLASSIFICATION_CURRENT_ASSET"    → stored as "CURRENT_ASSET"
//	"CASH_FLOW_ACTIVITY_OPERATING"            → stored as "OPERATING"
//	"NORMAL_BALANCE_DEBIT"                    → stored as "DEBIT"
//	"ACCOUNT_STATUS_ACTIVE"                   → stored as "ACTIVE"
//
// The data map uses camelCase keys (before normalizeKeys is called inside dbOps.Create/Update).
func normalizeAccountEnums(data map[string]any) {
	stripPrefix := func(key, prefix string) {
		if v, ok := data[key]; ok {
			if s, ok := v.(string); ok {
				data[key] = strings.TrimPrefix(s, prefix)
			}
		}
	}
	stripPrefix("element", "ACCOUNT_ELEMENT_")
	stripPrefix("classification", "ACCOUNT_CLASSIFICATION_")
	stripPrefix("cashFlowActivity", "CASH_FLOW_ACTIVITY_")
	stripPrefix("normalBalance", "NORMAL_BALANCE_")
	stripPrefix("status", "ACCOUNT_STATUS_")
}
