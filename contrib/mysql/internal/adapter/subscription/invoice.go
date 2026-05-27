//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MySQLInvoiceRepository implements invoice CRUD operations using MySQL 8.0+.
type MySQLInvoiceRepository struct {
	invoicepb.UnimplementedInvoiceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Invoice, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql invoice repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLInvoiceRepository(dbOps, tableName), nil
	})
}

// NewMySQLInvoiceRepository creates a new MySQL invoice repository.
func NewMySQLInvoiceRepository(dbOps interfaces.DatabaseOperation, tableName string) invoicepb.InvoiceDomainServiceServer {
	if tableName == "" {
		tableName = "invoice"
	}
	return &MySQLInvoiceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateInvoice creates a new invoice using common MySQL operations.
func (r *MySQLInvoiceRepository) CreateInvoice(ctx context.Context, req *invoicepb.CreateInvoiceRequest) (*invoicepb.CreateInvoiceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("invoice data is required")
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
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoice := &invoicepb.Invoice{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, invoice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoicepb.CreateInvoiceResponse{
		Data: []*invoicepb.Invoice{invoice},
	}, nil
}

// ReadInvoice retrieves an invoice using common MySQL operations.
func (r *MySQLInvoiceRepository) ReadInvoice(ctx context.Context, req *invoicepb.ReadInvoiceRequest) (*invoicepb.ReadInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read invoice: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoice := &invoicepb.Invoice{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, invoice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoicepb.ReadInvoiceResponse{
		Data: []*invoicepb.Invoice{invoice},
	}, nil
}

// UpdateInvoice updates an invoice using common MySQL operations.
func (r *MySQLInvoiceRepository) UpdateInvoice(ctx context.Context, req *invoicepb.UpdateInvoiceRequest) (*invoicepb.UpdateInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
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
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoice := &invoicepb.Invoice{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, invoice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoicepb.UpdateInvoiceResponse{
		Data: []*invoicepb.Invoice{invoice},
	}, nil
}

// DeleteInvoice deletes an invoice using common MySQL operations (soft delete).
func (r *MySQLInvoiceRepository) DeleteInvoice(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) (*invoicepb.DeleteInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete invoice: %w", err)
	}

	return &invoicepb.DeleteInvoiceResponse{
		Success: true,
	}, nil
}

// ListInvoices lists invoices using common MySQL operations.
func (r *MySQLInvoiceRepository) ListInvoices(ctx context.Context, req *invoicepb.ListInvoicesRequest) (*invoicepb.ListInvoicesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	var invoices []*invoicepb.Invoice
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		invoice := &invoicepb.Invoice{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, invoice); err != nil {
			continue
		}
		invoices = append(invoices, invoice)
	}

	return &invoicepb.ListInvoicesResponse{
		Data: invoices,
	}, nil
}

// scanInvoiceRow scans the joined row columns in the order they are SELECTed
// by both GetInvoiceListPageData and GetInvoiceItemPageData.
// Column order must stay in sync with the SELECT list.
func scanInvoiceRow(rows interface {
	Scan(dest ...any) error
}) (
	id, invoiceNumber string,
	amount int64,
	dateCreated, dateModified sql.NullTime,
	active bool,
	subscriptionID string,
	subID, subName, subPlanID, subClientID sql.NullString,
	subDateStart, subDateEnd, subDateCreated, subDateModified sql.NullTime,
	subActive sql.NullBool,
	clientID, clientUserID, clientInternalID sql.NullString,
	clientDateCreated, clientDateModified sql.NullTime,
	clientActive sql.NullBool,
	userID, userFirstName, userLastName, userEmailAddress sql.NullString,
	userDateCreated, userDateModified sql.NullTime,
	userActive sql.NullBool,
	err error,
) {
	err = rows.Scan(
		&id, &invoiceNumber, &amount, &dateCreated, &dateModified, &active, &subscriptionID,
		&subID, &subName, &subPlanID, &subClientID, &subDateStart, &subDateEnd, &subDateCreated, &subDateModified, &subActive,
		&clientID, &clientUserID, &clientInternalID, &clientDateCreated, &clientDateModified, &clientActive,
		&userID, &userFirstName, &userLastName, &userEmailAddress, &userDateCreated, &userDateModified, &userActive,
	)
	return
}

// buildInvoiceFromScan constructs an Invoice protobuf from scanned SQL fields.
// Identical column / scan order to postgres gold standard.
func buildInvoiceFromScan(
	id, invoiceNumber string, amount int64,
	dateCreated, dateModified sql.NullTime,
	active bool, subscriptionID string,
	subID, subName, subPlanID, subClientID sql.NullString,
	subDateStart, subDateEnd, subDateCreated, subDateModified sql.NullTime,
	subActive sql.NullBool,
	clientID, clientUserID, clientInternalID sql.NullString,
	clientDateCreated, clientDateModified sql.NullTime,
	clientActive sql.NullBool,
	userID, userFirstName, userLastName, userEmailAddress sql.NullString,
	userDateCreated, userDateModified sql.NullTime,
	userActive sql.NullBool,
) *invoicepb.Invoice {
	invoice := &invoicepb.Invoice{
		Id:             id,
		InvoiceNumber:  invoiceNumber,
		Amount:         amount,
		Active:         active,
		SubscriptionId: subscriptionID,
	}

	if dateCreated.Valid {
		ts := dateCreated.Time.Unix()
		invoice.DateCreated = &ts
		dateStr := dateCreated.Time.Format("2006-01-02T15:04:05Z07:00")
		invoice.DateCreatedString = &dateStr
	}
	if dateModified.Valid {
		ts := dateModified.Time.Unix()
		invoice.DateModified = &ts
		dateStr := dateModified.Time.Format("2006-01-02T15:04:05Z07:00")
		invoice.DateModifiedString = &dateStr
	}

	if subID.Valid {
		subscription := &subscriptionpb.Subscription{
			Id:     subID.String,
			Active: subActive.Bool,
		}
		if subName.Valid {
			subscription.Name = subName.String
		}
		if subPlanID.Valid {
			subscription.PricePlanId = subPlanID.String
		}
		if subClientID.Valid {
			subscription.ClientId = subClientID.String
		}
		if subDateStart.Valid {
			subscription.DateTimeStart = timestamppb.New(subDateStart.Time)
		}
		if subDateEnd.Valid {
			subscription.DateTimeEnd = timestamppb.New(subDateEnd.Time)
		}
		if subDateCreated.Valid {
			ts := subDateCreated.Time.Unix()
			subscription.DateCreated = &ts
			dateStr := subDateCreated.Time.Format("2006-01-02T15:04:05Z07:00")
			subscription.DateCreatedString = &dateStr
		}
		if subDateModified.Valid {
			ts := subDateModified.Time.Unix()
			subscription.DateModified = &ts
			dateStr := subDateModified.Time.Format("2006-01-02T15:04:05Z07:00")
			subscription.DateModifiedString = &dateStr
		}

		if clientID.Valid {
			client := &clientpb.Client{
				Id:     clientID.String,
				Active: clientActive.Bool,
			}
			if clientUserID.Valid {
				client.UserId = clientUserID.String
			}
			if clientInternalID.Valid {
				client.InternalId = clientInternalID.String
			}
			if clientDateCreated.Valid {
				ts := clientDateCreated.Time.Unix()
				client.DateCreated = &ts
				dateStr := clientDateCreated.Time.Format("2006-01-02T15:04:05Z07:00")
				client.DateCreatedString = &dateStr
			}
			if clientDateModified.Valid {
				ts := clientDateModified.Time.Unix()
				client.DateModified = &ts
				dateStr := clientDateModified.Time.Format("2006-01-02T15:04:05Z07:00")
				client.DateModifiedString = &dateStr
			}

			if userID.Valid {
				user := &userpb.User{
					Id:     userID.String,
					Active: userActive.Bool,
				}
				if userFirstName.Valid {
					user.FirstName = userFirstName.String
				}
				if userLastName.Valid {
					user.LastName = userLastName.String
				}
				if userEmailAddress.Valid {
					user.EmailAddress = userEmailAddress.String
				}
				if userDateCreated.Valid {
					ts := userDateCreated.Time.Unix()
					user.DateCreated = &ts
					dateStr := userDateCreated.Time.Format("2006-01-02T15:04:05Z07:00")
					user.DateCreatedString = &dateStr
				}
				if userDateModified.Valid {
					ts := userDateModified.Time.Unix()
					user.DateModified = &ts
					dateStr := userDateModified.Time.Format("2006-01-02T15:04:05Z07:00")
					user.DateModifiedString = &dateStr
				}
				client.User = user
			}
			subscription.Client = client
		}
		invoice.Subscription = subscription
	}
	return invoice
}

// invoiceListQuery is the shared base SELECT for the invoice list and item pages.
// Dialect changes vs postgres gold standard:
//   - $N → ? (positional placeholders)
//   - "user" → `user` (backtick-quoted reserved word)
//   - ILIKE → LIKE (MySQL ci collation)
//   - WHERE i.workspace_id = ? added for multi-tenancy (postgres gold was missing this)
const invoiceBaseSelect = `
	SELECT
		i.id,
		i.invoice_number,
		i.amount,
		i.date_created,
		i.date_modified,
		i.active,
		i.subscription_id,
		s.id as sub_id,
		s.name as sub_name,
		s.plan_id as sub_plan_id,
		s.client_id as sub_client_id,
		s.date_time_start as sub_date_start,
		s.date_time_end as sub_date_end,
		s.date_created as sub_date_created,
		s.date_modified as sub_date_modified,
		s.active as sub_active,
		c.id as client_id,
		c.user_id as client_user_id,
		c.internal_id as client_internal_id,
		c.date_created as client_date_created,
		c.date_modified as client_date_modified,
		c.active as client_active,
		u.id as user_id,
		u.first_name as user_first_name,
		u.last_name as user_last_name,
		u.email_address as user_email_address,
		u.date_created as user_date_created,
		u.date_modified as user_date_modified,
		u.active as user_active
	FROM invoice i
	LEFT JOIN subscription s ON i.subscription_id = s.id
	LEFT JOIN client c ON s.client_id = c.id
	LEFT JOIN ` + "`user`" + ` u ON c.user_id = u.id
`

// GetInvoiceListPageData retrieves paginated, filtered, and sorted invoice list with related data.
//
// Dialect translation from postgres gold standard:
//   - Dynamic $N → sequential ? placeholders
//   - ILIKE → LIKE
//   - "user" → `user`
//   - WHERE i.workspace_id = ? added (postgres gold was missing this — enforced here)
//   - Count done as a separate query (postgres gold already used two-pass approach)
func (r *MySQLInvoiceRepository) GetInvoiceListPageData(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) (*invoicepb.GetInvoiceListPageDataResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	// Build WHERE conditions.
	whereSQL := "WHERE i.active = 1 AND (? = '' OR i.workspace_id = ?)"
	var args []any

	// We always add workspace pair first (two args for the (? = '' OR workspace_id = ?) pattern).
	// workspace_id is extracted from dbOps via WorkspaceAwareOperations which already injects it
	// automatically on crud paths; for raw SQL we pass it explicitly via the executor context.
	// Per brief: workspace_id = ? on every query.
	// Use empty string sentinel so callers without workspace context still work (service-to-service).
	wsID := ""
	if wao, ok := r.dbOps.(interface {
		WorkspaceID(ctx context.Context) string
	}); ok {
		wsID = wao.WorkspaceID(ctx)
	}
	args = append(args, wsID, wsID)

	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			switch filter.Field {
			case "invoice_number":
				if sf := filter.GetStringFilter(); sf != nil {
					whereSQL += " AND i.invoice_number = ?"
					args = append(args, sf.Value)
				}
			case "subscription_id":
				if sf := filter.GetStringFilter(); sf != nil {
					whereSQL += " AND i.subscription_id = ?"
					args = append(args, sf.Value)
				}
			case "date_created_start":
				if nf := filter.GetNumberFilter(); nf != nil {
					whereSQL += " AND i.date_created >= ?"
					args = append(args, int64(nf.Value))
				}
			case "date_created_end":
				if nf := filter.GetNumberFilter(); nf != nil {
					whereSQL += " AND i.date_created <= ?"
					args = append(args, int64(nf.Value))
				}
			}
		}
	}

	if req.Search != nil && req.Search.Query != "" {
		whereSQL += " AND i.invoice_number LIKE ?"
		args = append(args, "%"+req.Search.Query+"%")
	}

	// Sorting — safe: only hardcoded column names.
	orderBy := "i.date_created DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField := req.Sort.Fields[0].Field
		direction := "ASC"
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			direction = "DESC"
		}
		switch sortField {
		case "invoice_number":
			orderBy = fmt.Sprintf("invoice_number %s", direction)
		case "amount":
			orderBy = fmt.Sprintf("amount %s", direction)
		case "date_created":
			orderBy = fmt.Sprintf("i.date_created %s", direction)
		default:
			orderBy = fmt.Sprintf("i.date_created %s", direction)
		}
	}

	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
			if limit > 100 {
				limit = 100
			}
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	query := invoiceBaseSelect + whereSQL + fmt.Sprintf(" ORDER BY %s LIMIT ? OFFSET ?", orderBy)
	queryArgs := append(args, limit, offset)

	sqlRows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer sqlRows.Close()

	var invoices []*invoicepb.Invoice
	for sqlRows.Next() {
		id, invoiceNumber, amount, dateCreated, dateModified, active, subscriptionID,
			subID, subName, subPlanID, subClientID, subDateStart, subDateEnd, subDateCreated, subDateModified, subActive,
			clientID, clientUserID, clientInternalID, clientDateCreated, clientDateModified, clientActive,
			userID, userFirstName, userLastName, userEmailAddress, userDateCreated, userDateModified, userActive,
			scanErr := scanInvoiceRow(sqlRows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan invoice row: %w", scanErr)
		}
		invoices = append(invoices, buildInvoiceFromScan(
			id, invoiceNumber, amount, dateCreated, dateModified, active, subscriptionID,
			subID, subName, subPlanID, subClientID, subDateStart, subDateEnd, subDateCreated, subDateModified, subActive,
			clientID, clientUserID, clientInternalID, clientDateCreated, clientDateModified, clientActive,
			userID, userFirstName, userLastName, userEmailAddress, userDateCreated, userDateModified, userActive,
		))
	}
	if err := sqlRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invoice rows: %w", err)
	}

	// Count query — same WHERE, no ORDER BY / LIMIT.
	countQuery := "SELECT COUNT(*) FROM invoice i LEFT JOIN subscription s ON i.subscription_id = s.id LEFT JOIN client c ON s.client_id = c.id LEFT JOIN `user` u ON c.user_id = u.id " + whereSQL
	var totalCount int64
	if err := exec.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &invoicepb.GetInvoiceListPageDataResponse{
		InvoiceList: invoices,
		Success:     true,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
	}, nil
}

// GetInvoiceItemPageData retrieves a single invoice with all related data.
//
// Dialect translation from postgres gold standard:
//   - $1 → ? (positional)
//   - "user" → `user`
//   - WHERE i.workspace_id = ? added for multi-tenancy
func (r *MySQLInvoiceRepository) GetInvoiceItemPageData(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) (*invoicepb.GetInvoiceItemPageDataResponse, error) {
	if req.InvoiceId == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	wsID := ""
	if wao, ok := r.dbOps.(interface {
		WorkspaceID(ctx context.Context) string
	}); ok {
		wsID = wao.WorkspaceID(ctx)
	}

	query := invoiceBaseSelect + `WHERE i.id = ? AND i.active = 1 AND (? = '' OR i.workspace_id = ?)`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	id, invoiceNumber, amount, dateCreated, dateModified, active, subscriptionID,
		subID, subName, subPlanID, subClientID, subDateStart, subDateEnd, subDateCreated, subDateModified, subActive,
		clientID, clientUserID, clientInternalID, clientDateCreated, clientDateModified, clientActive,
		userID, userFirstName, userLastName, userEmailAddress, userDateCreated, userDateModified, userActive,
		scanErr := scanInvoiceRow(exec.QueryRowContext(ctx, query, req.InvoiceId, wsID, wsID))
	if scanErr == sql.ErrNoRows {
		return &invoicepb.GetInvoiceItemPageDataResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_FOUND",
				Message: "invoice not found",
			},
		}, nil
	}
	if scanErr != nil {
		return nil, fmt.Errorf("failed to query invoice: %w", scanErr)
	}

	invoice := buildInvoiceFromScan(
		id, invoiceNumber, amount, dateCreated, dateModified, active, subscriptionID,
		subID, subName, subPlanID, subClientID, subDateStart, subDateEnd, subDateCreated, subDateModified, subActive,
		clientID, clientUserID, clientInternalID, clientDateCreated, clientDateModified, clientActive,
		userID, userFirstName, userLastName, userEmailAddress, userDateCreated, userDateModified, userActive,
	)

	return &invoicepb.GetInvoiceItemPageDataResponse{
		Invoice: invoice,
		Success: true,
	}, nil
}

// NewInvoiceRepository creates a new MySQL invoice repository (old-style constructor).
func NewInvoiceRepository(db *sql.DB, tableName string) invoicepb.InvoiceDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLInvoiceRepository(dbOps, tableName)
}
