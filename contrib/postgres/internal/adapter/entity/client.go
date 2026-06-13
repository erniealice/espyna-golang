//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Client, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres client repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresClientRepository(dbOps, tableName), nil
	})
}

// PostgresClientRepository implements client CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_client_user_id ON client(user_id) - Foreign key relationship to user table
//   - CREATE INDEX idx_client_active ON client(active) - Filter active records
//   - CREATE INDEX idx_client_date_created ON client(date_created DESC) - Default sorting
//   - CREATE INDEX idx_client_internal_id ON client(internal_id) - Search field
//   - CREATE INDEX idx_user_first_name ON "user"(first_name) - Search performance on joined table
//   - CREATE INDEX idx_user_last_name ON "user"(last_name) - Search performance on joined table
//   - CREATE INDEX idx_user_email_address ON "user"(email_address) - Search performance on joined table
//
// TODO: Add comprehensive tests for GetClientListPageData:
//   - Test with no search query (list all active clients)
//   - Test with search query matching user first_name
//   - Test with search query matching user last_name
//   - Test with search query matching user email_address
//   - Test with search query matching client internal_id
//   - Test pagination (page 1, page 2, page size variations)
//   - Test sorting (by different fields, ASC and DESC)
//   - Test with no matching results
//   - Test with inactive clients (should be filtered out)
//   - Test with null user_id (LEFT JOIN behavior)
//   - Test with inactive user (should be filtered out via JOIN condition)
//
// TODO: Add comprehensive tests for GetClientItemPageData:
//   - Test with valid client ID (with associated user)
//   - Test with valid client ID (without associated user - null user_id)
//   - Test with non-existent client ID
//   - Test with inactive client (should return not found)
//   - Test with client having inactive user (user fields should be null)
//   - Test timestamp parsing for date_created and date_modified
type PostgresClientRepository struct {
	clientpb.UnimplementedClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresClientRepository creates a new PostgreSQL client repository
func NewPostgresClientRepository(dbOps interfaces.DatabaseOperation, tableName string) clientpb.ClientDomainServiceServer {
	if tableName == "" {
		tableName = "client" // default fallback
	}

	return &PostgresClientRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateClient creates a new client using common PostgreSQL operations
func (r *PostgresClientRepository) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client data is required")
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
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientpb.CreateClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}

// ReadClient retrieves a client by ID using the canonical dbOps.Read +
// protojson DiscardUnknown round-trip, so new Client proto fields are picked
// up automatically without column-whitelist drift.
//
// Cross-table denorm: Client.user (nested User proto) is sourced from the
// "user" row pointed to by client.user_id, NOT from columns on the client
// row. The loadClientUser helper populates it after the canonical scan.
func (r *PostgresClientRepository) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Canonical Read — round-trip through protojson DiscardUnknown so every
	// proto-mapped column auto-resolves.
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("client with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	// User-table denorm — populate Client.User from the user row.
	if user, err := r.loadClientUser(ctx, client.GetUserId()); err == nil && user != nil {
		client.User = user
	}

	return &clientpb.ReadClientResponse{
		Data:    []*clientpb.Client{client},
		Success: true,
	}, nil
}

// loadClientPaymentTerm fetches the PaymentTerm row associated with a
// client.payment_term_id and returns a populated PaymentTerm proto. Returns
// (nil, nil) if paymentTermId is empty or the row is missing — keeps
// Client.PaymentTerm optional behavior intact.
//
// This helper mirrors loadClientUser/loadClientCategories — Client.payment_term
// is a cross-table denorm that dbOps.Read on the client table cannot resolve
// on its own. Used by GetClientListPageData so list views render the payment
// term name without a separate fetch per row in the view layer.
func (r *PostgresClientRepository) loadClientPaymentTerm(ctx context.Context, paymentTermId string) (*paymenttermpb.PaymentTerm, error) {
	if paymentTermId == "" {
		return nil, nil
	}
	result, err := r.dbOps.Read(ctx, "payment_term", paymentTermId)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment_term for client: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment_term result to JSON: %w", err)
	}

	pt := &paymenttermpb.PaymentTerm{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment_term JSON to protobuf: %w", err)
	}
	return pt, nil
}

// loadClientUser fetches the User row associated with a client.user_id and
// returns a populated User proto. Returns (nil, nil) if userId is empty or
// the user row is missing — keeps Client.User optional behavior intact.
//
// This helper exists because Client.user is a cross-table denorm that
// dbOps.Read on the client table cannot resolve on its own.
func (r *PostgresClientRepository) loadClientUser(ctx context.Context, userId string) (*userpb.User, error) {
	if userId == "" {
		return nil, nil
	}
	result, err := r.dbOps.Read(ctx, "user", userId)
	if err != nil {
		return nil, fmt.Errorf("failed to read user for client: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user result to JSON: %w", err)
	}

	user := &userpb.User{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user JSON to protobuf: %w", err)
	}
	return user, nil
}

// UpdateClient updates a client using common PostgreSQL operations
func (r *PostgresClientRepository) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
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
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientpb.UpdateClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}

// DeleteClient deletes a client using common PostgreSQL operations
func (r *PostgresClientRepository) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete client: %w", err)
	}

	return &clientpb.DeleteClientResponse{
		Success: true,
	}, nil
}

var clientSortableSQLCols = []string{
	"id", "user_id", "active", "internal_id", "name",
	"street_address", "city", "province", "postal_code", "notes",
	"payment_term_id", "billing_currency", "status", "country", "website",
	"date_created", "date_modified",
	// Derived column: computed by LATERAL JOIN in GetClientListPageData raw CTE.
	// Allows sorting the client list by active subscription count at DB level.
	"active_subscriptions",
}

var clientSortSpec = espynahttp.SortSpec{AllowedCols: clientSortableSQLCols}

// ListClients lists clients using common PostgreSQL operations.
func (r *PostgresClientRepository) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
	if err := espynahttp.ValidateSortColumns(clientSortSpec, req.GetSort(), "client"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var clients []*clientpb.Client
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		client := &clientpb.Client{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
			// Log error and continue with next item
			continue
		}
		clients = append(clients, client)
	}

	return &clientpb.ListClientsResponse{
		Data: clients,
	}, nil
}

// Example implementation for Create (commented out until database schema is defined):
/*
func (r *DBClientRepository) Create(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client data is required")
	}

	query := `
		INSERT INTO clients (user_id, active, internal_id, date_created, date_modified)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, date_created, date_modified
	`

	var id string
	var dateCreated, dateModified time.Time

	err := r.db.QueryRowContext(ctx, query,
		req.Data.UserId,
		req.Data.Active,
		req.Data.InternalId
	).Scan(&id, &dateCreated, &dateModified)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	client := &clientpb.Client{
		Id:                 id,
		UserId:             req.Data.UserId,
		DateCreated:        dateCreated.Unix(),
		DateCreatedString:  dateCreated.Format(time.RFC3339),
		DateModified:       dateModified.Unix(),
		DateModifiedString: dateModified.Format(time.RFC3339),
		Active:             req.Data.Active,
		InternalId:         req.Data.InternalId,
	}

	return &clientpb.CreateClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}
*/

// GetClientListPageData retrieves clients via a raw CTE query that includes a
// LATERAL JOIN computing active_subscriptions per client row. This enables
// server-side ORDER BY active_subscriptions without a separate count query per
// row and without client-side sorting.
//
// The query follows the same shape as GetSupplierListPageData (payment_term_name
// LATERAL JOIN exemplar) — single round-trip, CTE + windowed COUNT(*), user
// denorm via LEFT JOIN, payment_term name via LEFT JOIN, and the subscription
// count via LEFT JOIN LATERAL.
//
// Cross-table search (u.first_name, u.last_name, u.email_address) is supported
// via the enriched CTE joining the user table — callers pass Search through
// req.Search as before; searchFields wired to the user-joined aliases.
//
// The active_subscriptions column is not present on the Client proto, so it is
// used exclusively for DB-level ORDER BY. The view layer (entydad client list)
// continues to use GetActiveEngagementCounts to populate the cell value.
func (r *PostgresClientRepository) GetClientListPageData(
	ctx context.Context,
	req *clientpb.GetClientListPageDataRequest,
) (*clientpb.GetClientListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client list page data request is required")
	}

	// Validate sort columns against the extended allowlist (includes "active_subscriptions").
	if err := espynahttp.ValidateSortColumns(clientSortSpec, req.GetSort(), "client"); err != nil {
		return nil, err
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy).
	workspaceID := identity.Must(ctx).WorkspaceID

	// Default pagination values.
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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). Default
	// name ASC matches the view layer default. An unknown sort column now errors
	// instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(clientSortableSQLCols, req.GetSort(), "name ASC")
	if err != nil {
		return nil, err
	}

	// Build filter/search WHERE clauses ($1 reserved for workspace_id, start at $2).
	// Search spans client name + internal_id + representative user name/email.
	searchFields := []string{"c.name", "c.internal_id", "u.first_name", "u.last_name", "u.email_address"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE c.workspace_id = $1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

	// CTE query — single round-trip with:
	//   • User denorm via LEFT JOIN "user" u
	//   • PaymentTerm name via LEFT JOIN payment_term pt
	//   • Active subscription count via LEFT JOIN LATERAL subquery
	//   • Windowed total count via COUNT(*) OVER () — avoids double-materialization
	//     of the counted CTE pattern (A3 Q-PAGE-COUNT default tier).
	//
	// active_subscriptions is scoped to the same workspace_id so cross-workspace
	// counts are not leaked. The column is available to ORDER BY at the enriched
	// CTE level; it is not mapped to any Client proto field (DiscardUnknown).
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				c.id,
				c.user_id,
				c.active,
				c.internal_id,
				c.date_created,
				c.date_modified,
				c.name,
				c.street_address,
				c.city,
				c.province,
				c.postal_code,
				c.notes,
				c.payment_term_id,
				c.billing_currency,
				c.status,
				c.country,
				c.website,
				c.email,
				c.first_name,
				c.last_name,
				c.workspace_id,
				c.tax_id,
				c.registration_number,
				c.credit_limit,
				c.lead_time_days,
				COALESCE(pt.name, '') AS payment_term_name,
				-- Active subscription count — drives ORDER BY active_subscriptions.
				-- Scoped by workspace_id to prevent cross-workspace count leakage.
				sub.active_subscriptions,
				-- User fields (1:1 relationship via client.user_id)
				u.id AS user_id_value,
				u.first_name AS user_first_name,
				u.last_name AS user_last_name,
				u.email_address AS user_email_address,
				u.mobile_number AS user_phone_number,
				-- Windowed total — same filter as the page rows; no separate CTE needed.
				COUNT(*) OVER () AS total
			FROM client c
			LEFT JOIN "user" u ON c.user_id = u.id
			LEFT JOIN payment_term pt ON c.payment_term_id = pt.id
			LEFT JOIN LATERAL (
				SELECT COUNT(*) AS active_subscriptions
				FROM subscription s
				WHERE s.client_id = c.id
				  AND s.active = true
				  AND s.workspace_id = $1
			) sub ON true
			%s
		)
		SELECT * FROM enriched
		%s
		LIMIT $%d OFFSET $%d;
	`, whereSQL, orderByClause, limitIdx, offsetIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query client list page data: %w", err)
	}
	defer rows.Close()

	var clients []*clientpb.Client
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			userId             *string
			active             bool
			internalId         *string
			dateCreated        time.Time
			dateModified       time.Time
			name               *string
			streetAddress      *string
			city               *string
			province           *string
			postalCode         *string
			notes              *string
			paymentTermId      *string
			billingCurrency    *string
			status             *string
			country            *string
			website            *string
			email              *string
			firstName          *string
			lastName           *string
			workspaceId        *string
			taxId              *string
			registrationNumber *string
			creditLimit        *int64
			leadTimeDays       *int32
			paymentTermName    string
			activeSubCount     int64
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
			&name,
			&streetAddress,
			&city,
			&province,
			&postalCode,
			&notes,
			&paymentTermId,
			&billingCurrency,
			&status,
			&country,
			&website,
			&email,
			&firstName,
			&lastName,
			&workspaceId,
			&taxId,
			&registrationNumber,
			&creditLimit,
			&leadTimeDays,
			&paymentTermName,
			&activeSubCount,
			&userIdValue,
			&userFirstName,
			&userLastName,
			&userEmailAddress,
			&userPhoneNumber,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client row: %w", err)
		}

		totalCount = total

		c := &clientpb.Client{
			Id:     id,
			Active: active,
		}
		if userId != nil {
			c.UserId = *userId
		}
		if internalId != nil {
			c.InternalId = *internalId
		}
		dc := dateCreated.Unix()
		dm := dateModified.Unix()
		c.DateCreated = &dc
		c.DateModified = &dm
		if name != nil {
			c.Name = name
		}
		if streetAddress != nil {
			c.StreetAddress = streetAddress
		}
		if city != nil {
			c.City = city
		}
		if province != nil {
			c.Province = province
		}
		if postalCode != nil {
			c.PostalCode = postalCode
		}
		if notes != nil {
			c.Notes = notes
		}
		if paymentTermId != nil {
			c.PaymentTermId = paymentTermId
		}
		if billingCurrency != nil {
			c.BillingCurrency = billingCurrency
		}
		if status != nil {
			c.Status = status
		}
		if country != nil {
			c.Country = country
		}
		if website != nil {
			c.Website = website
		}
		if email != nil {
			c.Email = email
		}
		if firstName != nil {
			c.FirstName = firstName
		}
		if lastName != nil {
			c.LastName = lastName
		}
		if workspaceId != nil {
			c.WorkspaceId = workspaceId
		}
		if taxId != nil {
			c.TaxId = taxId
		}
		if registrationNumber != nil {
			c.RegistrationNumber = registrationNumber
		}
		if creditLimit != nil {
			c.CreditLimit = creditLimit
		}
		if leadTimeDays != nil {
			c.LeadTimeDays = leadTimeDays
		}

		// Denorm: PaymentTerm name inline from the JOIN (no extra round-trip).
		if paymentTermId != nil && paymentTermName != "" {
			c.PaymentTerm = &paymenttermpb.PaymentTerm{
				Id:   *paymentTermId,
				Name: paymentTermName,
			}
		}

		// Denorm: User fields from the LEFT JOIN "user" u.
		if userIdValue != nil {
			u := &userpb.User{Id: *userIdValue}
			if userFirstName != nil {
				u.FirstName = *userFirstName
			}
			if userLastName != nil {
				u.LastName = *userLastName
			}
			if userEmailAddress != nil {
				u.EmailAddress = *userEmailAddress
			}
			if userPhoneNumber != nil {
				u.MobileNumber = *userPhoneNumber
			}
			c.User = u
		}

		// activeSubCount is scanned but intentionally not stored on the Client
		// proto (no proto field). It serves only as the ORDER BY column at the
		// DB level. The view layer populates the cell via GetActiveEngagementCounts.
		_ = activeSubCount

		clients = append(clients, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client rows: %w", err)
	}

	// Load Categories per row — cannot be inlined via a simple LEFT JOIN without
	// multiplying rows; kept as a bounded N+1 read (≤ page size, default 50).
	for _, c := range clients {
		if cats, err := r.loadClientCategories(ctx, c.GetId()); err == nil && len(cats) > 0 {
			c.Categories = cats
		}
	}

	// Pagination metadata — total_items is the windowed count from the CTE.
	totalItems := int32(totalCount)
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &clientpb.GetClientListPageDataResponse{
		ClientList: clients,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetClientItemPageData retrieves a single client + categories via composition
// over the canonical ReadClient (which handles user denorm) and the adjacent
// loadClientCategories helper. Page-data layer adds the categories denorm
// that ReadClient does not need on its own.
func (r *PostgresClientRepository) GetClientItemPageData(
	ctx context.Context,
	req *clientpb.GetClientItemPageDataRequest,
) (*clientpb.GetClientItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client item page data request is required")
	}
	if req.ClientId == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	rr, err := r.ReadClient(ctx, &clientpb.ReadClientRequest{Data: &clientpb.Client{Id: req.ClientId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("client with ID '%s' not found", req.ClientId)
	}
	client := rr.GetData()[0]

	// Categories denorm — adjacent helper, kept separate from ReadClient.
	if categories, err := r.loadClientCategories(ctx, client.GetId()); err == nil && len(categories) > 0 {
		client.Categories = categories
	}

	return &clientpb.GetClientItemPageDataResponse{
		Client:  client,
		Success: true,
	}, nil
}

// loadClientCategories loads the category tags for a client via JOIN through client_category to category
func (r *PostgresClientRepository) loadClientCategories(ctx context.Context, clientId string) ([]*clientcategorypb.ClientCategory, error) {
	query := `
		SELECT
			cc.id,
			cc.client_id,
			cc.category_id,
			cat.name,
			cat.description
		FROM client_category cc
		INNER JOIN category cat ON cc.category_id = cat.id
		WHERE cc.client_id = $1 AND cc.active = true AND cat.active = true
		ORDER BY cat.name ASC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, clientId)
	if err != nil {
		return nil, fmt.Errorf("failed to load client categories: %w", err)
	}
	defer rows.Close()

	var categories []*clientcategorypb.ClientCategory
	for rows.Next() {
		var (
			ccId       string
			ccClientId string
			ccCatId    string
			catName    string
			catDesc    *string
		)
		if err := rows.Scan(&ccId, &ccClientId, &ccCatId, &catName, &catDesc); err != nil {
			return nil, fmt.Errorf("failed to scan client category row: %w", err)
		}

		cat := &commonpb.Category{
			Id:   ccCatId,
			Name: catName,
		}
		if catDesc != nil {
			cat.Description = *catDesc
		}

		categories = append(categories, &clientcategorypb.ClientCategory{
			Id:         ccId,
			ClientId:   ccClientId,
			CategoryId: ccCatId,
			Category:   cat,
			Active:     true,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client category rows: %w", err)
	}

	return categories, nil
}

// deref safely dereferences a *string, returning "" if nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SearchClientsByName searches clients by company name or user first/last name using ILIKE
func (r *PostgresClientRepository) SearchClientsByName(ctx context.Context, req *clientpb.SearchClientsByNameRequest) (*clientpb.SearchClientsByNameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search clients by name request is required")
	}

	limit := int32(20)
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	query := `
		SELECT
			c.id,
			COALESCE(
				NULLIF(c.name, ''),
				NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
				c.id
			) AS label
		FROM client c
		LEFT JOIN "user" u ON c.user_id = u.id
		WHERE c.active = true
			AND ($1::text = '' OR
				c.name ILIKE $1 OR
				u.first_name ILIKE $1 OR
				u.last_name ILIKE $1)
		ORDER BY label ASC
		LIMIT $2
	`

	pattern := ""
	if req.Query != "" {
		pattern = "%" + req.Query + "%"
	}

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search clients by name: %w", err)
	}
	defer rows.Close()

	var results []*clientpb.SearchClientResult
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, fmt.Errorf("failed to scan search client row: %w", err)
		}
		results = append(results, &clientpb.SearchClientResult{
			Id:    id,
			Label: label,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search client rows: %w", err)
	}

	return &clientpb.SearchClientsByNameResponse{
		Results: results,
		Success: true,
	}, nil
}

// NewClientRepository creates a new PostgreSQL client repository (old-style constructor)
func NewClientRepository(db *sql.DB, tableName string) clientpb.ClientDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresClientRepository(dbOps, tableName)
}
