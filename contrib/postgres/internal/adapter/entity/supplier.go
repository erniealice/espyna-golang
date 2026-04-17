//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Supplier, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierRepository implements supplier CRUD operations using PostgreSQL
type PostgresSupplierRepository struct {
	supplierpb.UnimplementedSupplierDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresSupplierRepository creates a new PostgreSQL supplier repository
func NewPostgresSupplierRepository(dbOps interfaces.DatabaseOperation, tableName string) supplierpb.SupplierDomainServiceServer {
	if tableName == "" {
		tableName = "supplier" // default fallback
	}

	return &PostgresSupplierRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateSupplier creates a new supplier using common PostgreSQL operations
func (r *PostgresSupplierRepository) CreateSupplier(ctx context.Context, req *supplierpb.CreateSupplierRequest) (*supplierpb.CreateSupplierResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier data is required")
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
		return nil, fmt.Errorf("failed to create supplier: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	supplier := &supplierpb.Supplier{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, supplier); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &supplierpb.CreateSupplierResponse{
		Data: []*supplierpb.Supplier{supplier},
	}, nil
}

// ReadSupplier retrieves a supplier with joined user data using a custom SQL query
func (r *PostgresSupplierRepository) ReadSupplier(ctx context.Context, req *supplierpb.ReadSupplierRequest) (*supplierpb.ReadSupplierResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier ID is required")
	}

	// Custom query that JOINs with user table to populate nested User field
	query := `
		SELECT
			s.id,
			s.user_id,
			s.active,
			s.internal_id,
			s.date_created,
			s.date_modified,
			s.supplier_type,
			s.company_name,
			s.tax_id,
			s.registration_number,
			s.street_address,
			s.city,
			s.province,
			s.postal_code,
			s.country,
			s.default_currency,
			s.payment_terms,
			s.lead_time_days,
			s.credit_limit,
			s.status,
			s.client_id,
			s.website,
			s.notes,
			s.category_id,
			s.payment_term_id,
			u.id as user_id_value,
			u.first_name as user_first_name,
			u.last_name as user_last_name,
			u.email_address as user_email_address,
			u.mobile_number as user_phone_number
		FROM supplier s
		LEFT JOIN "user" u ON s.user_id = u.id
		WHERE s.id = $1 AND s.active = true
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.Data.Id)

	var (
		id                 string
		userId             *string
		active             bool
		internalId         *string
		dateCreated        time.Time
		dateModified       time.Time
		supplierType       *string
		companyName        *string
		taxId              *string
		registrationNumber *string
		streetAddress      *string
		city               *string
		province           *string
		postalCode         *string
		country            *string
		defaultCurrency    *string
		paymentTerms       *string
		leadTimeDays       *int32
		creditLimit        *int64
		status             *string
		clientId           *string
		website            *string
		notes              *string
		categoryId         *string
		userIdValue        *string
		userFirstName      *string
		userLastName       *string
		userEmailAddress   *string
		userPhoneNumber    *string
		paymentTermID      *string
	)

	err := row.Scan(
		&id,
		&userId,
		&active,
		&internalId,
		&dateCreated,
		&dateModified,
		&supplierType,
		&companyName,
		&taxId,
		&registrationNumber,
		&streetAddress,
		&city,
		&province,
		&postalCode,
		&country,
		&defaultCurrency,
		&paymentTerms,
		&leadTimeDays,
		&creditLimit,
		&status,
		&clientId,
		&website,
		&notes,
		&categoryId,
		&paymentTermID,
		&userIdValue,
		&userFirstName,
		&userLastName,
		&userEmailAddress,
		&userPhoneNumber,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("supplier with ID '%s' not found", req.Data.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier: %w", err)
	}

	supplier := buildSupplierFromScan(
		id, userId, active, internalId, dateCreated, dateModified,
		supplierType, companyName, taxId, registrationNumber,
		streetAddress, city, province, postalCode, country,
		defaultCurrency, paymentTerms, leadTimeDays, creditLimit,
		status, clientId, website, notes, categoryId,
		paymentTermID,
		userIdValue, userFirstName, userLastName, userEmailAddress, userPhoneNumber,
	)

	return &supplierpb.ReadSupplierResponse{
		Data: []*supplierpb.Supplier{supplier},
	}, nil
}

// UpdateSupplier updates a supplier using common PostgreSQL operations
func (r *PostgresSupplierRepository) UpdateSupplier(ctx context.Context, req *supplierpb.UpdateSupplierRequest) (*supplierpb.UpdateSupplierResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier ID is required")
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
		return nil, fmt.Errorf("failed to update supplier: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	supplier := &supplierpb.Supplier{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, supplier); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &supplierpb.UpdateSupplierResponse{
		Data: []*supplierpb.Supplier{supplier},
	}, nil
}

// DeleteSupplier deletes a supplier using common PostgreSQL operations
func (r *PostgresSupplierRepository) DeleteSupplier(ctx context.Context, req *supplierpb.DeleteSupplierRequest) (*supplierpb.DeleteSupplierResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete supplier: %w", err)
	}

	return &supplierpb.DeleteSupplierResponse{
		Success: true,
	}, nil
}

// ListSuppliers lists suppliers using common PostgreSQL operations
func (r *PostgresSupplierRepository) ListSuppliers(ctx context.Context, req *supplierpb.ListSuppliersRequest) (*supplierpb.ListSuppliersResponse, error) {
	// Pass through filters from the request
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list suppliers: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var suppliers []*supplierpb.Supplier
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		supplier := &supplierpb.Supplier{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, supplier); err != nil {
			continue
		}
		suppliers = append(suppliers, supplier)
	}

	return &supplierpb.ListSuppliersResponse{
		Data: suppliers,
	}, nil
}

// GetSupplierListPageData retrieves suppliers with advanced filtering, sorting, searching, and pagination using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresSupplierRepository) GetSupplierListPageData(
	ctx context.Context,
	req *supplierpb.GetSupplierListPageDataRequest,
) (*supplierpb.GetSupplierListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier list page data request is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Build filter/search WHERE clauses ($1 is reserved for workspace_id, start at $2)
	searchFields := []string{"s.company_name", "s.internal_id", "u.first_name", "u.last_name", "u.email_address"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE s.workspace_id = $1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

	// CTE Query - Single round-trip with enriched user data
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				s.id,
				s.user_id,
				s.active,
				s.internal_id,
				s.date_created,
				s.date_modified,
				s.supplier_type,
				s.company_name,
				s.tax_id,
				s.registration_number,
				s.street_address,
				s.city,
				s.province,
				s.postal_code,
				s.country,
				s.default_currency,
				s.payment_terms,
				s.lead_time_days,
				s.credit_limit,
				s.status,
				s.client_id,
				s.website,
				s.notes,
				s.category_id,
				s.payment_term_id,
				COALESCE(pt.name, '') as payment_term_name,
				-- User fields (1:1 relationship)
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.mobile_number as user_phone_number
			FROM supplier s
			LEFT JOIN "user" u ON s.user_id = u.id
			LEFT JOIN payment_term pt ON s.payment_term_id = pt.id
			%s
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d;
	`, whereSQL, sortField, sortOrder, limitIdx, offsetIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query supplier list page data: %w", err)
	}
	defer rows.Close()

	var suppliers []*supplierpb.Supplier
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			userId             *string
			active             bool
			internalId         *string
			dateCreated        time.Time
			dateModified       time.Time
			supplierType       *string
			companyName        *string
			taxId              *string
			registrationNumber *string
			streetAddress      *string
			city               *string
			province           *string
			postalCode         *string
			country            *string
			defaultCurrency    *string
			paymentTerms       *string
			leadTimeDays       *int32
			creditLimit        *int64
			status             *string
			clientId           *string
			website            *string
			notes              *string
			categoryId         *string
			paymentTermID      *string
			paymentTermName    string
			userIdValue        *string
			userFirstName      *string
			userLastName       *string
			userEmailAddress   *string
			userPhoneNumber    *string
			total              int64
		)

		err := rows.Scan(
			&id,
			&userId,
			&active,
			&internalId,
			&dateCreated,
			&dateModified,
			&supplierType,
			&companyName,
			&taxId,
			&registrationNumber,
			&streetAddress,
			&city,
			&province,
			&postalCode,
			&country,
			&defaultCurrency,
			&paymentTerms,
			&leadTimeDays,
			&creditLimit,
			&status,
			&clientId,
			&website,
			&notes,
			&categoryId,
			&paymentTermID,
			&paymentTermName,
			&userIdValue,
			&userFirstName,
			&userLastName,
			&userEmailAddress,
			&userPhoneNumber,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan supplier row: %w", err)
		}

		totalCount = total

		supplier := buildSupplierFromScan(
			id, userId, active, internalId, dateCreated, dateModified,
			supplierType, companyName, taxId, registrationNumber,
			streetAddress, city, province, postalCode, country,
			defaultCurrency, paymentTerms, leadTimeDays, creditLimit,
			status, clientId, website, notes, categoryId,
			paymentTermID,
			userIdValue, userFirstName, userLastName, userEmailAddress, userPhoneNumber,
		)

		if paymentTermID != nil {
			supplier.PaymentTermId = paymentTermID
		}
		if paymentTermName != "" {
			supplier.PaymentTerms = &paymentTermName
		}

		suppliers = append(suppliers, supplier)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating supplier rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &supplierpb.GetSupplierListPageDataResponse{
		SupplierList: suppliers,
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

// GetSupplierItemPageData retrieves a single supplier with enhanced item page data using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresSupplierRepository) GetSupplierItemPageData(
	ctx context.Context,
	req *supplierpb.GetSupplierItemPageDataRequest,
) (*supplierpb.GetSupplierItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier item page data request is required")
	}
	if req.SupplierId == "" {
		return nil, fmt.Errorf("supplier ID is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// CTE Query - Single round-trip with enriched user data and supplier fields
	query := `
		WITH enriched AS (
			SELECT
				s.id,
				s.user_id,
				s.active,
				s.internal_id,
				s.date_created,
				s.date_modified,
				s.supplier_type,
				s.company_name,
				s.tax_id,
				s.registration_number,
				s.street_address,
				s.city,
				s.province,
				s.postal_code,
				s.country,
				s.default_currency,
				s.payment_terms,
				s.lead_time_days,
				s.credit_limit,
				s.status,
				s.client_id,
				s.website,
				s.notes,
				s.category_id,
				s.payment_term_id,
				-- User fields (1:1 relationship)
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.mobile_number as user_phone_number
			FROM supplier s
			LEFT JOIN "user" u ON s.user_id = u.id
			WHERE s.id = $1 AND s.workspace_id = $2
		)
		SELECT * FROM enriched LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.SupplierId, workspaceID)

	var (
		id                 string
		userId             *string
		active             bool
		internalId         *string
		dateCreated        time.Time
		dateModified       time.Time
		supplierType       *string
		companyName        *string
		taxId              *string
		registrationNumber *string
		streetAddress      *string
		city               *string
		province           *string
		postalCode         *string
		country            *string
		defaultCurrency    *string
		paymentTerms       *string
		leadTimeDays       *int32
		creditLimit        *int64
		status             *string
		clientId           *string
		website            *string
		notes              *string
		categoryId         *string
		paymentTermID      *string
		userIdValue        *string
		userFirstName      *string
		userLastName       *string
		userEmailAddress   *string
		userPhoneNumber    *string
	)

	err := row.Scan(
		&id,
		&userId,
		&active,
		&internalId,
		&dateCreated,
		&dateModified,
		&supplierType,
		&companyName,
		&taxId,
		&registrationNumber,
		&streetAddress,
		&city,
		&province,
		&postalCode,
		&country,
		&defaultCurrency,
		&paymentTerms,
		&leadTimeDays,
		&creditLimit,
		&status,
		&clientId,
		&website,
		&notes,
		&categoryId,
		&paymentTermID,
		&userIdValue,
		&userFirstName,
		&userLastName,
		&userEmailAddress,
		&userPhoneNumber,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("supplier with ID '%s' not found", req.SupplierId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query supplier item page data: %w", err)
	}

	supplier := buildSupplierFromScan(
		id, userId, active, internalId, dateCreated, dateModified,
		supplierType, companyName, taxId, registrationNumber,
		streetAddress, city, province, postalCode, country,
		defaultCurrency, paymentTerms, leadTimeDays, creditLimit,
		status, clientId, website, notes, categoryId,
		paymentTermID,
		userIdValue, userFirstName, userLastName, userEmailAddress, userPhoneNumber,
	)

	// Load categories (tags) for this supplier via separate query
	categories, err := r.loadSupplierCategories(ctx, id)
	if err == nil && len(categories) > 0 {
		supplier.Categories = categories
	}

	return &supplierpb.GetSupplierItemPageDataResponse{
		Supplier: supplier,
		Success:  true,
	}, nil
}

// loadSupplierCategories loads the category tags for a supplier via JOIN through supplier_category to category
func (r *PostgresSupplierRepository) loadSupplierCategories(ctx context.Context, supplierId string) ([]*suppliercategorypb.SupplierCategory, error) {
	query := `
		SELECT
			sc.id,
			sc.supplier_id,
			sc.category_id,
			cat.name,
			cat.description
		FROM supplier_category sc
		INNER JOIN category cat ON sc.category_id = cat.id
		WHERE sc.supplier_id = $1 AND sc.active = true AND cat.active = true
		ORDER BY cat.name ASC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, supplierId)
	if err != nil {
		return nil, fmt.Errorf("failed to load supplier categories: %w", err)
	}
	defer rows.Close()

	var categories []*suppliercategorypb.SupplierCategory
	for rows.Next() {
		var (
			scId         string
			scSupplierId string
			scCatId      string
			catName      string
			catDesc      *string
		)
		if err := rows.Scan(&scId, &scSupplierId, &scCatId, &catName, &catDesc); err != nil {
			return nil, fmt.Errorf("failed to scan supplier category row: %w", err)
		}

		cat := &commonpb.Category{
			Id:   scCatId,
			Name: catName,
		}
		if catDesc != nil {
			cat.Description = *catDesc
		}

		categories = append(categories, &suppliercategorypb.SupplierCategory{
			Id:         scId,
			SupplierId: scSupplierId,
			CategoryId: scCatId,
			Category:   cat,
			Active:     true,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating supplier category rows: %w", err)
	}

	return categories, nil
}

// buildSupplierFromScan constructs a Supplier protobuf from scanned SQL fields
func buildSupplierFromScan(
	id string, userId *string, active bool, internalId *string,
	dateCreated time.Time, dateModified time.Time,
	supplierType *string, companyName *string, taxId *string, registrationNumber *string,
	streetAddress *string, city *string, province *string, postalCode *string, country *string,
	defaultCurrency *string, paymentTerms *string, leadTimeDays *int32, creditLimit *int64,
	status *string, clientId *string, website *string, notes *string, categoryId *string,
	paymentTermID *string,
	userIdValue *string, userFirstName *string, userLastName *string,
	userEmailAddress *string, userPhoneNumber *string,
) *supplierpb.Supplier {
	supplier := &supplierpb.Supplier{
		Id:     id,
		Active: active,
	}
	if userId != nil {
		supplier.UserId = *userId
	}

	// Handle nullable fields
	if internalId != nil {
		supplier.InternalId = *internalId
	}

	// Supplier-specific fields
	if supplierType != nil {
		supplier.SupplierType = *supplierType
	}
	if companyName != nil {
		supplier.CompanyName = *companyName
	}
	supplier.TaxId = taxId
	supplier.RegistrationNumber = registrationNumber
	supplier.StreetAddress = streetAddress
	supplier.City = city
	supplier.Province = province
	supplier.PostalCode = postalCode
	supplier.Country = country
	supplier.DefaultCurrency = defaultCurrency
	supplier.PaymentTerms = paymentTerms
	supplier.LeadTimeDays = leadTimeDays
	supplier.CreditLimit = creditLimit
	supplier.Status = status
	supplier.ClientId = clientId
	supplier.Website = website
	supplier.Notes = notes
	supplier.CategoryId = categoryId
	if paymentTermID != nil {
		supplier.PaymentTermId = paymentTermID
	}

	// Populate joined user data
	if userIdValue != nil {
		supplier.User = &userpb.User{Id: deref(userIdValue)}
		supplier.User.FirstName = deref(userFirstName)
		supplier.User.LastName = deref(userLastName)
		supplier.User.EmailAddress = deref(userEmailAddress)
		supplier.User.MobileNumber = deref(userPhoneNumber)
	}

	// Parse timestamps
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		supplier.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		supplier.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		supplier.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		supplier.DateModifiedString = &dmStr
	}

	return supplier
}

// NewSupplierRepository creates a new PostgreSQL supplier repository (old-style constructor)
func NewSupplierRepository(db *sql.DB, tableName string) supplierpb.SupplierDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresSupplierRepository(dbOps, tableName)
}