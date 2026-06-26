//go:build sqlserver

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SQLServerInvoiceRepository implements invoice CRUD operations using SQL Server.
type SQLServerInvoiceRepository struct {
	invoicepb.UnimplementedInvoiceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Invoice, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver invoice repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInvoiceRepository(dbOps, tableName), nil
	})
}

// NewSQLServerInvoiceRepository creates a new SQL Server invoice repository.
func NewSQLServerInvoiceRepository(dbOps interfaces.DatabaseOperation, tableName string) invoicepb.InvoiceDomainServiceServer {
	if tableName == "" {
		tableName = "invoice"
	}
	return &SQLServerInvoiceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateInvoice creates a new invoice using common SQL Server operations.
func (r *SQLServerInvoiceRepository) CreateInvoice(ctx context.Context, req *invoicepb.CreateInvoiceRequest) (*invoicepb.CreateInvoiceResponse, error) {
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

// ReadInvoice retrieves an invoice using common SQL Server operations.
func (r *SQLServerInvoiceRepository) ReadInvoice(ctx context.Context, req *invoicepb.ReadInvoiceRequest) (*invoicepb.ReadInvoiceResponse, error) {
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

// UpdateInvoice updates an invoice using common SQL Server operations.
func (r *SQLServerInvoiceRepository) UpdateInvoice(ctx context.Context, req *invoicepb.UpdateInvoiceRequest) (*invoicepb.UpdateInvoiceResponse, error) {
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

// DeleteInvoice deletes an invoice using common SQL Server operations (soft delete).
func (r *SQLServerInvoiceRepository) DeleteInvoice(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) (*invoicepb.DeleteInvoiceResponse, error) {
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

// ListInvoices lists invoices using common SQL Server operations.
func (r *SQLServerInvoiceRepository) ListInvoices(ctx context.Context, req *invoicepb.ListInvoicesRequest) (*invoicepb.ListInvoicesResponse, error) {
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

// GetInvoiceListPageData retrieves paginated, filtered, and sorted invoice list
// with related data.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN (dynamically built with argCounter).
//   - "user" → [user] (T-SQL reserved word).
//   - ILIKE → LIKE (SQL Server CI collation).
//   - LIMIT/OFFSET → ORDER BY … OFFSET @pN ROWS FETCH NEXT @pN ROWS ONLY.
//   - active = true → active = 1.
//   - WHERE workspace_id = @pN added (MISSING in postgres version — added here per brief).
func (r *SQLServerInvoiceRepository) GetInvoiceListPageData(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) (*invoicepb.GetInvoiceListPageDataResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	// workspace_id guard — ADDED (was missing in postgres version per task brief).
	wsID := identity.Must(ctx).WorkspaceID

	// Build the base CTE query.
	// SQL Server differences:
	//   - [user] instead of "user".
	//   - active = 1 instead of active = true.
	//   - @pN placeholders (argCounter tracks the next index).
	query := `
		WITH filtered_data AS (
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
			LEFT JOIN [user] u ON c.user_id = u.id
			WHERE i.active = 1
			  AND (@p1 = '' OR i.workspace_id = @p1)
	`

	var args []any
	args = append(args, wsID) // @p1
	argCounter := 2

	// Filter by invoice_number (exact match).
	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "invoice_number" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					query += fmt.Sprintf(" AND i.invoice_number = @p%d", argCounter)
					args = append(args, strFilter.Value)
					argCounter++
				}
			}
			if filter.Field == "subscription_id" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					query += fmt.Sprintf(" AND i.subscription_id = @p%d", argCounter)
					args = append(args, strFilter.Value)
					argCounter++
				}
			}
			if filter.Field == "date_created_start" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					query += fmt.Sprintf(" AND i.date_created >= @p%d", argCounter)
					args = append(args, int64(numFilter.Value))
					argCounter++
				}
			}
			if filter.Field == "date_created_end" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					query += fmt.Sprintf(" AND i.date_created <= @p%d", argCounter)
					args = append(args, int64(numFilter.Value))
					argCounter++
				}
			}
		}
	}

	// Search on invoice_number (ILIKE → LIKE).
	if req.Search != nil && req.Search.Query != "" {
		query += fmt.Sprintf(" AND i.invoice_number LIKE @p%d", argCounter)
		args = append(args, "%"+req.Search.Query+"%")
		argCounter++
	}

	query += `
		)
		SELECT * FROM filtered_data
	`

	// Add sorting (column names are from a closed switch — safe to interpolate).
	orderBy := "date_created DESC"
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
			orderBy = fmt.Sprintf("date_created %s", direction)
		default:
			orderBy = fmt.Sprintf("date_created %s", direction)
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

	// SQL Server OFFSET/FETCH requires an ORDER BY on the outer query.
	query += fmt.Sprintf(" ORDER BY %s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY", orderBy, argCounter, argCounter+1)
	args = append(args, offset, limit)

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*invoicepb.Invoice
	for rows.Next() {
		var (
			id                 string
			invoiceNumber      string
			amount             int64
			dateCreated        sql.NullTime
			dateModified       sql.NullTime
			active             bool
			subscriptionID     string
			subID              sql.NullString
			subName            sql.NullString
			subPlanID          sql.NullString
			subClientID        sql.NullString
			subDateStart       sql.NullTime
			subDateEnd         sql.NullTime
			subDateCreated     sql.NullTime
			subDateModified    sql.NullTime
			subActive          sql.NullBool
			clientID           sql.NullString
			clientUserID       sql.NullString
			clientInternalID   sql.NullString
			clientDateCreated  sql.NullTime
			clientDateModified sql.NullTime
			clientActive       sql.NullBool
			userID             sql.NullString
			userFirstName      sql.NullString
			userLastName       sql.NullString
			userEmailAddress   sql.NullString
			userDateCreated    sql.NullTime
			userDateModified   sql.NullTime
			userActive         sql.NullBool
		)

		err := rows.Scan(
			&id, &invoiceNumber, &amount, &dateCreated, &dateModified, &active, &subscriptionID,
			&subID, &subName, &subPlanID, &subClientID, &subDateStart, &subDateEnd, &subDateCreated, &subDateModified, &subActive,
			&clientID, &clientUserID, &clientInternalID, &clientDateCreated, &clientDateModified, &clientActive,
			&userID, &userFirstName, &userLastName, &userEmailAddress, &userDateCreated, &userDateModified, &userActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice row: %w", err)
		}

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

		invoices = append(invoices, invoice)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invoice rows: %w", err)
	}

	// Count query — mirrors the main query filters but without pagination.
	// workspace_id guard also applied here.
	countQuery := `
		SELECT COUNT(*) FROM invoice i
		WHERE i.active = 1
		  AND (@p1 = '' OR i.workspace_id = @p1)
	`
	var countArgs []any
	countArgs = append(countArgs, wsID) // @p1
	countArgCounter := 2

	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "invoice_number" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					countQuery += fmt.Sprintf(" AND i.invoice_number = @p%d", countArgCounter)
					countArgs = append(countArgs, strFilter.Value)
					countArgCounter++
				}
			}
			if filter.Field == "subscription_id" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					countQuery += fmt.Sprintf(" AND i.subscription_id = @p%d", countArgCounter)
					countArgs = append(countArgs, strFilter.Value)
					countArgCounter++
				}
			}
			if filter.Field == "date_created_start" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					countQuery += fmt.Sprintf(" AND i.date_created >= @p%d", countArgCounter)
					countArgs = append(countArgs, int64(numFilter.Value))
					countArgCounter++
				}
			}
			if filter.Field == "date_created_end" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					countQuery += fmt.Sprintf(" AND i.date_created <= @p%d", countArgCounter)
					countArgs = append(countArgs, int64(numFilter.Value))
					countArgCounter++
				}
			}
		}
	}
	if req.Search != nil && req.Search.Query != "" {
		countQuery += fmt.Sprintf(" AND i.invoice_number LIKE @p%d", countArgCounter)
		countArgs = append(countArgs, "%"+req.Search.Query+"%")
	}

	var totalCount int64
	countRow := exec.QueryRowContext(ctx, countQuery, countArgs...)
	if err = countRow.Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &invoicepb.GetInvoiceListPageDataResponse{
		InvoiceList: invoices,
		Success:     true,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetInvoiceItemPageData retrieves a single invoice with all related data.
//
// SQL Server differences:
//   - $1 → @p1.
//   - "user" → [user].
//   - active = true → active = 1.
//   - workspace_id predicate ADDED (was missing in postgres version).
func (r *SQLServerInvoiceRepository) GetInvoiceItemPageData(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) (*invoicepb.GetInvoiceItemPageDataResponse, error) {
	if req.InvoiceId == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// workspace_id guard — ADDED (was missing in postgres version per task brief).
	wsID := identity.Must(ctx).WorkspaceID

	query := `
		WITH invoice_data AS (
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
			LEFT JOIN [user] u ON c.user_id = u.id
			WHERE i.id = @p1
			  AND i.active = 1
			  AND (@p2 = '' OR i.workspace_id = @p2)
		)
		SELECT * FROM invoice_data
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.InvoiceId, wsID)

	var (
		id                 string
		invoiceNumber      string
		amount             int64
		dateCreated        sql.NullTime
		dateModified       sql.NullTime
		active             bool
		subscriptionID     string
		subID              sql.NullString
		subName            sql.NullString
		subPlanID          sql.NullString
		subClientID        sql.NullString
		subDateStart       sql.NullTime
		subDateEnd         sql.NullTime
		subDateCreated     sql.NullTime
		subDateModified    sql.NullTime
		subActive          sql.NullBool
		clientID           sql.NullString
		clientUserID       sql.NullString
		clientInternalID   sql.NullString
		clientDateCreated  sql.NullTime
		clientDateModified sql.NullTime
		clientActive       sql.NullBool
		userID             sql.NullString
		userFirstName      sql.NullString
		userLastName       sql.NullString
		userEmailAddress   sql.NullString
		userDateCreated    sql.NullTime
		userDateModified   sql.NullTime
		userActive         sql.NullBool
	)

	err := row.Scan(
		&id, &invoiceNumber, &amount, &dateCreated, &dateModified, &active, &subscriptionID,
		&subID, &subName, &subPlanID, &subClientID, &subDateStart, &subDateEnd, &subDateCreated, &subDateModified, &subActive,
		&clientID, &clientUserID, &clientInternalID, &clientDateCreated, &clientDateModified, &clientActive,
		&userID, &userFirstName, &userLastName, &userEmailAddress, &userDateCreated, &userDateModified, &userActive,
	)
	if err == sql.ErrNoRows {
		return &invoicepb.GetInvoiceItemPageDataResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_FOUND",
				Message: "invoice not found",
			},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice: %w", err)
	}

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

	return &invoicepb.GetInvoiceItemPageDataResponse{
		Invoice: invoice,
		Success: true,
	}, nil
}

// NewInvoiceRepository creates a new SQL Server invoice repository (old-style constructor).
func NewInvoiceRepository(db *sql.DB, tableName string) invoicepb.InvoiceDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerInvoiceRepository(dbOps, tableName)
}
