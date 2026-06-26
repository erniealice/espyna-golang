//go:build mysql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Account, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql account repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLAccountRepository(dbOps, tableName), nil
	})
}

// MySQLAccountRepository implements account CRUD operations using MySQL 8.0+.
//
// Dialect differences from postgres gold standard:
//   - Placeholders: $N → ? (positional)
//   - active = true → active = 1
//   - ILIKE → LIKE
//   - COUNT(*) OVER () ✓ (MySQL 8.0+)
//   - WHERE workspace_id = ? added (missing in postgres gold standard)
type MySQLAccountRepository struct {
	accountpb.UnimplementedAccountDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLAccountRepository creates a new MySQL account repository.
func NewMySQLAccountRepository(dbOps interfaces.DatabaseOperation, tableName string) accountpb.AccountDomainServiceServer {
	if tableName == "" {
		tableName = "account"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLAccountRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccount creates a new account using common MySQL operations.
func (r *MySQLAccountRepository) CreateAccount(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
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
//
// Dialect: $1 → ?; active = true → active = 1; workspace_id predicate added.
func (r *MySQLAccountRepository) ReadAccount(ctx context.Context, req *accountpb.ReadAccountRequest) (*accountpb.ReadAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		SELECT
			id, code, name, description, element, classification,
			group_id, parent_id, cash_flow_activity, normal_balance,
			is_system_account, is_contra, status, notes,
			active, date_created, date_modified
		FROM account
		WHERE id = ?`
	args := []any{req.Data.Id}
	if workspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}
	query += " LIMIT 1"

	row := r.db.QueryRowContext(ctx, query, args...)

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

	account := buildAccountFromScan(
		id, code, name, description, element, classification,
		groupID, parentID, cashFlowActivity, normalBalance,
		isSystemAccount, isContra, status, notes, active, dateCreated, dateModified,
	)

	return &accountpb.ReadAccountResponse{
		Data:    []*accountpb.Account{account},
		Success: true,
	}, nil
}

// UpdateAccount updates an account using common MySQL operations.
func (r *MySQLAccountRepository) UpdateAccount(ctx context.Context, req *accountpb.UpdateAccountRequest) (*accountpb.UpdateAccountResponse, error) {
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

// DeleteAccount soft-deletes an account using common MySQL operations.
func (r *MySQLAccountRepository) DeleteAccount(ctx context.Context, req *accountpb.DeleteAccountRequest) (*accountpb.DeleteAccountResponse, error) {
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

// ListAccounts lists accounts using common MySQL operations.
func (r *MySQLAccountRepository) ListAccounts(ctx context.Context, req *accountpb.ListAccountsRequest) (*accountpb.ListAccountsResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
// Dialect: $N → ?; ILIKE → LIKE; active = true → active = 1;
// COUNT(*) OVER() ✓; LIMIT/OFFSET use trailing ?; workspace_id predicate added.
func (r *MySQLAccountRepository) GetAccountListPageData(ctx context.Context, req *accountpb.GetAccountListPageDataRequest) (*accountpb.GetAccountListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get account list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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

	// Sort allowlist — bare column names (ORDER BY applied on outer enriched CTE).
	accountSortableCols := []string{
		"code", "name", "element", "status", "date_created", "date_modified",
	}
	orderByClause, err := mysqlCore.BuildOrderBy(accountSortableCols, req.Sort, "code ASC")
	if err != nil {
		return nil, err
	}

	// Build filter/search WHERE clauses.
	// workspace_id occupies first ?; filter builder starts at index 2 for parity.
	searchFields := []string{"a.name", "a.code"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE a.active = 1"
	if workspaceID != "" {
		whereSQL += " AND a.workspace_id = ?"
	}
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	// Args: [workspaceID (optional), ...filterArgs, limit, offset]
	var queryArgs []any
	if workspaceID != "" {
		queryArgs = append(queryArgs, workspaceID)
	}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

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
			FROM account a
			%s
		)
		SELECT * FROM enriched
		%s
		LIMIT ? OFFSET ?`, whereSQL, orderByClause)

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

		if err := rows.Scan(
			&id, &code, &name, &description, &element, &classification,
			&groupID, &parentID, &cashFlowActivity, &normalBalance,
			&isSystemAccount, &isContra, &status, &notes,
			&active, &dateCreated, &dateModified, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan account row: %w", err)
		}

		totalCount = total
		accounts = append(accounts, buildAccountFromScan(
			id, code, name, description, element, classification,
			groupID, parentID, cashFlowActivity, normalBalance,
			isSystemAccount, isContra, status, notes, active, dateCreated, dateModified,
		))
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
//
// Dialect: $1 → ?; active = true → active = 1; workspace_id predicate added.
func (r *MySQLAccountRepository) GetAccountItemPageData(ctx context.Context, req *accountpb.GetAccountItemPageDataRequest) (*accountpb.GetAccountItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get account item page data request is required")
	}
	if req.AccountId == "" {
		return nil, fmt.Errorf("account ID is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		SELECT
			id, code, name, description, element, classification,
			group_id, parent_id, cash_flow_activity, normal_balance,
			is_system_account, is_contra, status, notes,
			active, date_created, date_modified
		FROM account
		WHERE id = ? AND active = 1`
	args := []any{req.AccountId}
	if workspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}
	query += " LIMIT 1"

	row := r.db.QueryRowContext(ctx, query, args...)

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

	account := buildAccountFromScan(
		id, code, name, description, element, classification,
		groupID, parentID, cashFlowActivity, normalBalance,
		isSystemAccount, isContra, status, notes, active, dateCreated, dateModified,
	)

	return &accountpb.GetAccountItemPageDataResponse{
		Account: account,
		Success: true,
	}, nil
}

// GetAccountTreePageData — TODO Phase 2: recursive CTE for hierarchical CoA display.
func (r *MySQLAccountRepository) GetAccountTreePageData(ctx context.Context, req *accountpb.GetAccountTreePageDataRequest) (*accountpb.GetAccountTreePageDataResponse, error) {
	return nil, fmt.Errorf("GetAccountTreePageData not yet implemented — Phase 2")
}

// buildAccountFromScan constructs an Account proto from scanned SQL fields.
// Shared by ReadAccount, GetAccountListPageData, and GetAccountItemPageData
// to keep scan order consistent across all query paths.
func buildAccountFromScan(
	id, code, name string, description *string,
	element, classification string, groupID, parentID *string,
	cashFlowActivity, normalBalance string,
	isSystemAccount, isContra bool,
	status string, notes *string, active bool,
	dateCreated, dateModified time.Time,
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
		dcStr := dateCreated.Format(time.RFC3339)
		account.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		account.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		account.DateModifiedString = &dmStr
	}
	return account
}

// parseAccountElement converts a DB short name to the proto enum value.
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

// normalizeAccountEnums strips proto-style enum prefixes from string enum fields
// so the DB stores short names consistent with seeded data.
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
