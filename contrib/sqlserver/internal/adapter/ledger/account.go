//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Account, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver account repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerAccountRepository(dbOps, tableName), nil
	})
}

// accountSortableSQLCols is the fail-closed sort whitelist for account list page data.
var accountSortableSQLCols = []string{
	"code", "name", "element", "status", "date_created", "date_modified",
}

// SQLServerAccountRepository implements account CRUD operations using SQL Server.
//
// SQL Server differences from the postgres gold standard:
//   - Placeholders: @p1, @p2, … (not $1, $2, …)
//   - Identifier quoting: [ident] (not "ident")
//   - ILIKE → LIKE
//   - LIMIT n OFFSET m → ORDER BY … OFFSET m ROWS FETCH NEXT n ROWS ONLY
//   - active = true → active = 1 (BIT column)
//   - ($1::text IS NULL OR …) → (@p1 IS NULL OR …)
type SQLServerAccountRepository struct {
	accountpb.UnimplementedAccountDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerAccountRepository creates a new SQL Server account repository.
func NewSQLServerAccountRepository(dbOps interfaces.DatabaseOperation, tableName string) accountpb.AccountDomainServiceServer {
	if tableName == "" {
		tableName = "account"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerAccountRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccount creates a new account using common SQL Server operations.
func (r *SQLServerAccountRepository) CreateAccount(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
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

	normalizeAccountEnums(data)

	_, err = r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &accountpb.CreateAccountResponse{
		Data:    []*accountpb.Account{req.Data},
		Success: true,
	}, nil
}

// ReadAccount retrieves an account by ID.
func (r *SQLServerAccountRepository) ReadAccount(ctx context.Context, req *accountpb.ReadAccountRequest) (*accountpb.ReadAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// SQL Server: @p1/@p2 placeholders; no ::text cast; TOP 1 instead of LIMIT 1.
	query := `
		SELECT TOP 1
			id, code, name, description, element, classification,
			group_id, parent_id, cash_flow_activity, normal_balance,
			is_system_account, is_contra, status, notes,
			active, date_created, date_modified
		FROM [account]
		WHERE id = @p1
		AND (@p2 IS NULL OR workspace_id = @p2)`

	row := r.db.QueryRowContext(ctx, query, req.Data.Id, nilIfEmpty(workspaceID))

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
		&id, &code, &name, &description, &element, &classification,
		&groupID, &parentID, &cashFlowActivity, &normalBalance,
		&isSystemAccount, &isContra, &status, &notes,
		&active, &dateCreated, &dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account with ID '%s' not found", req.Data.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read account: %w", err)
	}

	account := buildAccount(id, code, name, description, element, classification,
		groupID, parentID, cashFlowActivity, normalBalance,
		isSystemAccount, isContra, status, notes, active, dateCreated, dateModified)

	return &accountpb.ReadAccountResponse{
		Data:    []*accountpb.Account{account},
		Success: true,
	}, nil
}

// UpdateAccount updates an account using common SQL Server operations.
func (r *SQLServerAccountRepository) UpdateAccount(ctx context.Context, req *accountpb.UpdateAccountRequest) (*accountpb.UpdateAccountResponse, error) {
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

	normalizeAccountEnums(data)

	_, err = r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	return &accountpb.UpdateAccountResponse{
		Data: []*accountpb.Account{req.Data},
	}, nil
}

// DeleteAccount soft-deletes an account.
func (r *SQLServerAccountRepository) DeleteAccount(ctx context.Context, req *accountpb.DeleteAccountRequest) (*accountpb.DeleteAccountResponse, error) {
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

// ListAccounts lists accounts using common SQL Server operations.
func (r *SQLServerAccountRepository) ListAccounts(ctx context.Context, req *accountpb.ListAccountsRequest) (*accountpb.ListAccountsResponse, error) {
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
//
// SQL Server differences:
//   - @pN placeholders; workspace_id = @p1 (no ::text IS NULL cast)
//   - ILIKE → LIKE
//   - LIMIT $n OFFSET $n → OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY
//   - active = 1 (BIT)
//   - COUNT(*) OVER () is supported in SQL Server 2017+
func (r *SQLServerAccountRepository) GetAccountListPageData(ctx context.Context, req *accountpb.GetAccountListPageDataRequest) (*accountpb.GetAccountListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get account list page data request is required")
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

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	orderByClause, err := sqlserverCore.BuildOrderBy(accountSortableSQLCols, req.GetSort(), "code ASC")
	if err != nil {
		return nil, err
	}

	searchFields := []string{"a.name", "a.code"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereStr := " AND a.workspace_id = @p1"
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
			FROM [account] a
			WHERE a.active = 1%s
		)
		SELECT * FROM enriched
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY`,
		whereStr, orderByClause, offsetIdx, limitIdx)

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
			&id, &code, &name, &description, &element, &classification,
			&groupID, &parentID, &cashFlowActivity, &normalBalance,
			&isSystemAccount, &isContra, &status, &notes,
			&active, &dateCreated, &dateModified, &total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row: %w", err)
		}

		totalCount = total
		account := buildAccount(id, code, name, description, element, classification,
			groupID, parentID, cashFlowActivity, normalBalance,
			isSystemAccount, isContra, status, notes, active, dateCreated, dateModified)
		accounts = append(accounts, account)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows: %w", err)
	}

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
func (r *SQLServerAccountRepository) GetAccountItemPageData(ctx context.Context, req *accountpb.GetAccountItemPageDataRequest) (*accountpb.GetAccountItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get account item page data request is required")
	}
	if req.AccountId == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	query := `
		SELECT TOP 1
			id, code, name, description, element, classification,
			group_id, parent_id, cash_flow_activity, normal_balance,
			is_system_account, is_contra, status, notes,
			active, date_created, date_modified
		FROM [account]
		WHERE id = @p1 AND active = 1
		AND (@p2 IS NULL OR workspace_id = @p2)`

	row := r.db.QueryRowContext(ctx, query, req.AccountId, nilIfEmpty(workspaceID))

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
		&id, &code, &name, &description, &element, &classification,
		&groupID, &parentID, &cashFlowActivity, &normalBalance,
		&isSystemAccount, &isContra, &status, &notes,
		&active, &dateCreated, &dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account with ID '%s' not found", req.AccountId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query account item page data: %w", err)
	}

	account := buildAccount(id, code, name, description, element, classification,
		groupID, parentID, cashFlowActivity, normalBalance,
		isSystemAccount, isContra, status, notes, active, dateCreated, dateModified)

	return &accountpb.GetAccountItemPageDataResponse{
		Account: account,
		Success: true,
	}, nil
}

// GetAccountTreePageData — TODO Phase 2.
func (r *SQLServerAccountRepository) GetAccountTreePageData(ctx context.Context, req *accountpb.GetAccountTreePageDataRequest) (*accountpb.GetAccountTreePageDataResponse, error) {
	return nil, fmt.Errorf("GetAccountTreePageData not yet implemented — Phase 2")
}

// buildAccount constructs an Account proto from raw scanned fields.
func buildAccount(
	id, code, name string, description *string, element, classification string,
	groupID, parentID *string, cashFlowActivity, normalBalance string,
	isSystemAccount, isContra bool, status string, notes *string,
	active bool, dateCreated, dateModified time.Time,
) *accountpb.Account {
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
		s := dateCreated.Format(time.RFC3339)
		account.DateCreatedString = &s
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		account.DateModified = &ts
		s := dateModified.Format(time.RFC3339)
		account.DateModifiedString = &s
	}
	return account
}

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

// normalizeAccountEnums strips proto-style enum prefixes so the DB stores short names.
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
