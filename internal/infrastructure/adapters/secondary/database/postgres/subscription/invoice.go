//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// PostgresInvoiceRepository implements invoice CRUD operations using PostgreSQL
type PostgresInvoiceRepository struct {
	invoicepb.UnimplementedInvoiceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "invoice", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres invoice repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInvoiceRepository(dbOps, tableName), nil
	})
}

// NewPostgresInvoiceRepository creates a new PostgreSQL invoice repository
func NewPostgresInvoiceRepository(dbOps interfaces.DatabaseOperation, tableName string) invoicepb.InvoiceDomainServiceServer {
	if tableName == "" {
		tableName = "invoice" // default fallback
	}
	return &PostgresInvoiceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateInvoice creates a new invoice using common PostgreSQL operations
func (r *PostgresInvoiceRepository) CreateInvoice(ctx context.Context, req *invoicepb.CreateInvoiceRequest) (*invoicepb.CreateInvoiceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("invoice data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoice := &invoicepb.Invoice{}
	if err := protojson.Unmarshal(resultJSON, invoice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoicepb.CreateInvoiceResponse{
		Data: []*invoicepb.Invoice{invoice},
	}, nil
}

// ReadInvoice retrieves an invoice using common PostgreSQL operations
func (r *PostgresInvoiceRepository) ReadInvoice(ctx context.Context, req *invoicepb.ReadInvoiceRequest) (*invoicepb.ReadInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read invoice: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoice := &invoicepb.Invoice{}
	if err := protojson.Unmarshal(resultJSON, invoice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoicepb.ReadInvoiceResponse{
		Data: []*invoicepb.Invoice{invoice},
	}, nil
}

// UpdateInvoice updates an invoice using common PostgreSQL operations
func (r *PostgresInvoiceRepository) UpdateInvoice(ctx context.Context, req *invoicepb.UpdateInvoiceRequest) (*invoicepb.UpdateInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoice := &invoicepb.Invoice{}
	if err := protojson.Unmarshal(resultJSON, invoice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoicepb.UpdateInvoiceResponse{
		Data: []*invoicepb.Invoice{invoice},
	}, nil
}

// DeleteInvoice deletes an invoice using common PostgreSQL operations
func (r *PostgresInvoiceRepository) DeleteInvoice(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) (*invoicepb.DeleteInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete invoice: %w", err)
	}

	return &invoicepb.DeleteInvoiceResponse{
		Success: true,
	}, nil
}

// ListInvoices lists invoices using common PostgreSQL operations
func (r *PostgresInvoiceRepository) ListInvoices(ctx context.Context, req *invoicepb.ListInvoicesRequest) (*invoicepb.ListInvoicesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var invoices []*invoicepb.Invoice
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		invoice := &invoicepb.Invoice{}
		if err := protojson.Unmarshal(resultJSON, invoice); err != nil {
			// Log error and continue with next item
			continue
		}
		invoices = append(invoices, invoice)
	}

	return &invoicepb.ListInvoicesResponse{
		Data: invoices,
	}, nil
}

// GetInvoiceListPageData retrieves paginated, filtered, and sorted invoice list with related data
// TODO: Add tests for GetInvoiceListPageData with various filter combinations
// TODO: Add tests for search functionality on invoice_number field
// TODO: Add tests for pagination with different page sizes
// TODO: Add tests for sorting by different columns
func (r *PostgresInvoiceRepository) GetInvoiceListPageData(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) (*invoicepb.GetInvoiceListPageDataResponse, error) {
	// Get the underlying *sql.DB from dbOps
	// This is needed for raw SQL queries with JOINs
	db, ok := r.dbOps.(*postgresCore.PostgresOperations)
	if !ok {
		return nil, fmt.Errorf("invalid database operations type")
	}

	// Build the query with CTE pattern for better performance
	// Performance indexes needed:
	// - invoice.subscription_id (foreign key)
	// - invoice.invoice_number (search field)
	// - invoice.date_created (filter range)
	// - invoice.active (filter flag)
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
				-- Subscription fields
				s.id as sub_id,
				s.name as sub_name,
				s.plan_id as sub_plan_id,
				s.client_id as sub_client_id,
				s.date_start as sub_date_start,
				s.date_end as sub_date_end,
				s.date_created as sub_date_created,
				s.date_modified as sub_date_modified,
				s.active as sub_active,
				-- Client fields (via subscription)
				c.id as client_id,
				c.user_id as client_user_id,
				c.internal_id as client_internal_id,
				c.date_created as client_date_created,
				c.date_modified as client_date_modified,
				c.active as client_active,
				-- User fields (nested via client)
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
			LEFT JOIN "user" u ON c.user_id = u.id
			WHERE i.active = true
	`

	// Build WHERE conditions for filters
	var args []interface{}
	argCounter := 1

	// Filter by invoice_number (exact match)
	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "invoice_number" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					query += fmt.Sprintf(" AND i.invoice_number = $%d", argCounter)
					args = append(args, strFilter.Value)
					argCounter++
				}
			}
			// Filter by subscription_id
			if filter.Field == "subscription_id" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					query += fmt.Sprintf(" AND i.subscription_id = $%d", argCounter)
					args = append(args, strFilter.Value)
					argCounter++
				}
			}
			// Filter by date_created range (start)
			if filter.Field == "date_created_start" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					query += fmt.Sprintf(" AND i.date_created >= $%d", argCounter)
					args = append(args, int64(numFilter.Value))
					argCounter++
				}
			}
			// Filter by date_created range (end)
			if filter.Field == "date_created_end" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					query += fmt.Sprintf(" AND i.date_created <= $%d", argCounter)
					args = append(args, int64(numFilter.Value))
					argCounter++
				}
			}
		}
	}

	// Search functionality on invoice_number (partial match)
	if req.Search != nil && req.Search.Query != "" {
		query += fmt.Sprintf(" AND i.invoice_number ILIKE $%d", argCounter)
		args = append(args, "%"+req.Search.Query+"%")
		argCounter++
	}

	query += `
		)
		SELECT * FROM filtered_data
	`

	// Add sorting
	orderBy := "date_created DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField := req.Sort.Fields[0].Field
		direction := "ASC"
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			direction = "DESC"
		}
		// Map protobuf field names to database column names
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
	query += fmt.Sprintf(" ORDER BY %s", orderBy)

	// Add pagination
	limit := int32(20) // default
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
			if limit > 100 {
				limit = 100 // Cap at 100 items per page
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
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCounter, argCounter+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := db.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	// Parse results
	var invoices []*invoicepb.Invoice
	for rows.Next() {
		var (
			// Invoice fields
			id             string
			invoiceNumber  string
			amount         float64
			dateCreated    sql.NullTime
			dateModified   sql.NullTime
			active         bool
			subscriptionID string
			// Subscription fields
			subID           sql.NullString
			subName         sql.NullString
			subPlanID       sql.NullString
			subClientID     sql.NullString
			subDateStart    sql.NullTime
			subDateEnd      sql.NullTime
			subDateCreated  sql.NullTime
			subDateModified sql.NullTime
			subActive       sql.NullBool
			// Client fields
			clientID           sql.NullString
			clientUserID       sql.NullString
			clientInternalID   sql.NullString
			clientDateCreated  sql.NullTime
			clientDateModified sql.NullTime
			clientActive       sql.NullBool
			// User fields
			userID           sql.NullString
			userFirstName    sql.NullString
			userLastName     sql.NullString
			userEmailAddress sql.NullString
			userDateCreated  sql.NullTime
			userDateModified sql.NullTime
			userActive       sql.NullBool
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

		// Build invoice protobuf
		invoice := &invoicepb.Invoice{
			Id:             id,
			InvoiceNumber:  invoiceNumber,
			Amount:         amount,
			Active:         active,
			SubscriptionId: subscriptionID,
		}

		// Handle nullable date fields
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

		// Build nested subscription if present
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
				ts := subDateStart.Time.Unix()
				subscription.DateStart = &ts
				dateStr := subDateStart.Time.Format("2006-01-02T15:04:05Z07:00")
				subscription.DateStartString = &dateStr
			}
			if subDateEnd.Valid {
				ts := subDateEnd.Time.Unix()
				subscription.DateEnd = &ts
				dateStr := subDateEnd.Time.Format("2006-01-02T15:04:05Z07:00")
				subscription.DateEndString = &dateStr
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

			// Build nested client if present
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

				// Build nested user if present
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

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*) FROM invoice i
		WHERE i.active = true
	`
	var countArgs []interface{}
	countArgCounter := 1

	// Apply same filters for count
	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "invoice_number" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					countQuery += fmt.Sprintf(" AND i.invoice_number = $%d", countArgCounter)
					countArgs = append(countArgs, strFilter.Value)
					countArgCounter++
				}
			}
			if filter.Field == "subscription_id" {
				if strFilter := filter.GetStringFilter(); strFilter != nil {
					countQuery += fmt.Sprintf(" AND i.subscription_id = $%d", countArgCounter)
					countArgs = append(countArgs, strFilter.Value)
					countArgCounter++
				}
			}
			if filter.Field == "date_created_start" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					countQuery += fmt.Sprintf(" AND i.date_created >= $%d", countArgCounter)
					countArgs = append(countArgs, int64(numFilter.Value))
					countArgCounter++
				}
			}
			if filter.Field == "date_created_end" {
				if numFilter := filter.GetNumberFilter(); numFilter != nil {
					countQuery += fmt.Sprintf(" AND i.date_created <= $%d", countArgCounter)
					countArgs = append(countArgs, int64(numFilter.Value))
					countArgCounter++
				}
			}
		}
	}
	if req.Search != nil && req.Search.Query != "" {
		countQuery += fmt.Sprintf(" AND i.invoice_number ILIKE $%d", countArgCounter)
		countArgs = append(countArgs, "%"+req.Search.Query+"%")
	}

	var totalCount int64
	err = db.GetDB().QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination metadata
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

// GetInvoiceItemPageData retrieves a single invoice with all related data
// TODO: Add tests for GetInvoiceItemPageData with valid invoice ID
// TODO: Add tests for GetInvoiceItemPageData with invalid/non-existent invoice ID
// TODO: Add tests for GetInvoiceItemPageData with soft-deleted invoice
func (r *PostgresInvoiceRepository) GetInvoiceItemPageData(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) (*invoicepb.GetInvoiceItemPageDataResponse, error) {
	if req.InvoiceId == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// Get the underlying *sql.DB from dbOps
	db, ok := r.dbOps.(*postgresCore.PostgresOperations)
	if !ok {
		return nil, fmt.Errorf("invalid database operations type")
	}

	// Build query with JOINs to get all related data
	// Performance indexes needed:
	// - invoice.id (primary key)
	// - invoice.subscription_id (foreign key)
	// - subscription.client_id (foreign key)
	// - client.user_id (foreign key)
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
				-- Subscription fields
				s.id as sub_id,
				s.name as sub_name,
				s.plan_id as sub_plan_id,
				s.client_id as sub_client_id,
				s.date_start as sub_date_start,
				s.date_end as sub_date_end,
				s.date_created as sub_date_created,
				s.date_modified as sub_date_modified,
				s.active as sub_active,
				-- Client fields (via subscription)
				c.id as client_id,
				c.user_id as client_user_id,
				c.internal_id as client_internal_id,
				c.date_created as client_date_created,
				c.date_modified as client_date_modified,
				c.active as client_active,
				-- User fields (nested via client)
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
			LEFT JOIN "user" u ON c.user_id = u.id
			WHERE i.id = $1 AND i.active = true
		)
		SELECT * FROM invoice_data
	`

	var (
		// Invoice fields
		id             string
		invoiceNumber  string
		amount         float64
		dateCreated    sql.NullTime
		dateModified   sql.NullTime
		active         bool
		subscriptionID string
		// Subscription fields
		subID           sql.NullString
		subName         sql.NullString
		subPlanID       sql.NullString
		subClientID     sql.NullString
		subDateStart    sql.NullTime
		subDateEnd      sql.NullTime
		subDateCreated  sql.NullTime
		subDateModified sql.NullTime
		subActive       sql.NullBool
		// Client fields
		clientID           sql.NullString
		clientUserID       sql.NullString
		clientInternalID   sql.NullString
		clientDateCreated  sql.NullTime
		clientDateModified sql.NullTime
		clientActive       sql.NullBool
		// User fields
		userID           sql.NullString
		userFirstName    sql.NullString
		userLastName     sql.NullString
		userEmailAddress sql.NullString
		userDateCreated  sql.NullTime
		userDateModified sql.NullTime
		userActive       sql.NullBool
	)

	err := db.GetDB().QueryRowContext(ctx, query, req.InvoiceId).Scan(
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

	// Build invoice protobuf
	invoice := &invoicepb.Invoice{
		Id:             id,
		InvoiceNumber:  invoiceNumber,
		Amount:         amount,
		Active:         active,
		SubscriptionId: subscriptionID,
	}

	// Handle nullable date fields
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

	// Build nested subscription if present
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
			ts := subDateStart.Time.Unix()
			subscription.DateStart = &ts
			dateStr := subDateStart.Time.Format("2006-01-02T15:04:05Z07:00")
			subscription.DateStartString = &dateStr
		}
		if subDateEnd.Valid {
			ts := subDateEnd.Time.Unix()
			subscription.DateEnd = &ts
			dateStr := subDateEnd.Time.Format("2006-01-02T15:04:05Z07:00")
			subscription.DateEndString = &dateStr
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

		// Build nested client if present
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

			// Build nested user if present
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

// NewInvoiceRepository creates a new PostgreSQL invoice repository (old-style constructor)
func NewInvoiceRepository(db *sql.DB, tableName string) invoicepb.InvoiceDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresInvoiceRepository(dbOps, tableName)
}
