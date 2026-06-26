//go:build sqlserver

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
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
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
	registry.RegisterRepositoryFactory("sqlserver", entityid.Client, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver client repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerClientRepository(dbOps, tableName), nil
	})
}

// SQLServerClientRepository implements client CRUD operations using SQL Server.
type SQLServerClientRepository struct {
	clientpb.UnimplementedClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerClientRepository creates a new SQL Server client repository.
func NewSQLServerClientRepository(dbOps interfaces.DatabaseOperation, tableName string) clientpb.ClientDomainServiceServer {
	if tableName == "" {
		tableName = "client"
	}
	return &SQLServerClientRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateClient creates a new client using common SQL Server operations.
func (r *SQLServerClientRepository) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client data is required")
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
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientpb.CreateClientResponse{Data: []*clientpb.Client{client}}, nil
}

// ReadClient retrieves a client by ID via canonical dbOps.Read, then loads the joined user via a
// secondary read (same pattern as the postgres gold standard).
func (r *SQLServerClientRepository) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

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

	if user, err := r.loadClientUser(ctx, client.GetUserId()); err == nil && user != nil {
		client.User = user
	}

	return &clientpb.ReadClientResponse{Data: []*clientpb.Client{client}, Success: true}, nil
}

// loadClientUser fetches the User row associated with a client.user_id.
func (r *SQLServerClientRepository) loadClientUser(ctx context.Context, userId string) (*userpb.User, error) {
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

// loadClientPaymentTerm fetches the PaymentTerm row for a client.payment_term_id.
func (r *SQLServerClientRepository) loadClientPaymentTerm(ctx context.Context, paymentTermId string) (*paymenttermpb.PaymentTerm, error) {
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

// UpdateClient updates a client using common SQL Server operations.
func (r *SQLServerClientRepository) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
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
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientpb.UpdateClientResponse{Data: []*clientpb.Client{client}}, nil
}

// DeleteClient deletes a client using common SQL Server operations (soft delete).
func (r *SQLServerClientRepository) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete client: %w", err)
	}

	return &clientpb.DeleteClientResponse{Success: true}, nil
}

var clientSortableSQLCols = []string{
	"id", "user_id", "active", "internal_id", "name",
	"street_address", "city", "province", "postal_code", "notes",
	"payment_term_id", "billing_currency", "status", "country", "website",
	"date_created", "date_modified",
	// Derived column: computed by OUTER APPLY in GetClientListPageData.
	// Allows ORDER BY active_subscriptions at DB level without post-fetch sort.
	"active_subscriptions",
}

var clientSortSpec = espynahttp.SortSpec{AllowedCols: clientSortableSQLCols}

// ListClients lists clients using common SQL Server operations.
func (r *SQLServerClientRepository) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
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
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	var clients []*clientpb.Client
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		client := &clientpb.Client{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
			continue
		}
		clients = append(clients, client)
	}

	return &clientpb.ListClientsResponse{Data: clients}, nil
}

// GetClientListPageData retrieves clients via a CTE query with an OUTER APPLY computing
// active_subscriptions per client row. This enables ORDER BY active_subscriptions at the DB
// level without a per-row subquery and without client-side sorting.
//
// SQL Server translation notes:
//   - "user" → [user] (reserved word).
//   - $N → @pN.
//   - ILIKE → LIKE (CI collation).
//   - LEFT JOIN LATERAL (…) ON true → OUTER APPLY (…). This is the canonical SQL Server
//     equivalent — OUTER APPLY evaluates the subquery once per left-hand row and returns
//     NULL for the subquery columns when the subquery returns no rows.
//   - Workspace-scoped subscription count: scoped to c.workspace_id so cross-workspace
//     counts are not leaked (same logic as the postgres gold standard).
//   - LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER () retained (SQL Server 2017+).
//   - Sort is validated against clientSortableSQLCols; active_subscriptions is safe
//     because it is a computed alias in the same SELECT list.
func (r *SQLServerClientRepository) GetClientListPageData(
	ctx context.Context,
	req *clientpb.GetClientListPageDataRequest,
) (*clientpb.GetClientListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client list page data request is required")
	}

	if err := espynahttp.ValidateSortColumns(clientSortSpec, req.GetSort(), "client"); err != nil {
		return nil, err
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

	sortField := "name"
	sortOrder := "ASC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		} else {
			sortOrder = "ASC"
		}
	}

	// @p1 = workspaceID; filter/search params start at @p2.
	searchFields := []string{"c.name", "c.internal_id", "u.first_name", "u.last_name", "u.email_address"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE c.workspace_id = @p1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, offset, limit)

	// OUTER APPLY replaces LEFT JOIN LATERAL.
	// The OUTER APPLY subquery is scoped to the same workspace_id (@p1) so cross-workspace
	// subscription counts cannot leak. It is evaluated once per client row; when no rows
	// match, sub.active_subscriptions is NULL — COALESCE maps this to 0.
	//
	// active_subscriptions is not mapped to any Client proto field (DiscardUnknown);
	// it exists solely to support ORDER BY active_subscriptions at the DB level.
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
				COALESCE(sub.active_subscriptions, 0) AS active_subscriptions,
				-- User fields (1:1 relationship via client.user_id)
				u.id AS user_id_value,
				u.first_name AS user_first_name,
				u.last_name AS user_last_name,
				u.email_address AS user_email_address,
				u.mobile_number AS user_phone_number,
				-- Windowed total — same filter as the page rows; no separate CTE needed.
				COUNT(*) OVER () AS total
			FROM client c
			LEFT JOIN [user] u ON c.user_id = u.id
			LEFT JOIN payment_term pt ON c.payment_term_id = pt.id
			OUTER APPLY (
				SELECT COUNT(*) AS active_subscriptions
				FROM subscription s
				WHERE s.client_id = c.id
				  AND s.active = 1
				  AND s.workspace_id = @p1
			) sub
			%s
		)
		SELECT * FROM enriched
		ORDER BY [%s] %s
		OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, sortField, sortOrder, offsetIdx, limitIdx)

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

		if paymentTermId != nil && paymentTermName != "" {
			c.PaymentTerm = &paymenttermpb.PaymentTerm{
				Id:   *paymentTermId,
				Name: paymentTermName,
			}
		}

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

		// activeSubCount is for ORDER BY only — not stored on the Client proto.
		_ = activeSubCount

		clients = append(clients, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client rows: %w", err)
	}

	// Load categories per row — bounded N+1 read (≤ page size, default 50).
	for _, c := range clients {
		if cats, err := r.loadClientCategories(ctx, c.GetId()); err == nil && len(cats) > 0 {
			c.Categories = cats
		}
	}

	totalItems := int32(totalCount)
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &clientpb.GetClientListPageDataResponse{
		ClientList: clients,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
		Success: true,
	}, nil
}

// GetClientItemPageData retrieves a single client + categories via composition.
func (r *SQLServerClientRepository) GetClientItemPageData(
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

	if categories, err := r.loadClientCategories(ctx, client.GetId()); err == nil && len(categories) > 0 {
		client.Categories = categories
	}

	return &clientpb.GetClientItemPageDataResponse{Client: client, Success: true}, nil
}

// loadClientCategories loads the category tags for a client via JOIN through client_category.
//
// SQL Server translation: $1 → @p1; active = true → active = 1.
func (r *SQLServerClientRepository) loadClientCategories(ctx context.Context, clientId string) ([]*clientcategorypb.ClientCategory, error) {
	query := `
		SELECT
			cc.id,
			cc.client_id,
			cc.category_id,
			cat.name,
			cat.description
		FROM client_category cc
		INNER JOIN category cat ON cc.category_id = cat.id
		WHERE cc.client_id = @p1 AND cc.active = 1 AND cat.active = 1
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

		cat := &commonpb.Category{Id: ccCatId, Name: catName}
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

// SearchClientsByName searches clients by company name or user first/last name.
//
// SQL Server translation notes:
//   - "user" → [user].
//   - $1 → @p1, $2 → @p2.
//   - ILIKE → LIKE.
//   - active = true → active = 1.
//   - LIMIT n → TOP n (no ORDER BY needed for TOP in the outer query here;
//     ORDER BY is on the derived alias, safe for deterministic paging).
//   - ($1::text = ” OR ...) → (@p1 = ” OR ...) — SQL Server needs no ::text cast.
func (r *SQLServerClientRepository) SearchClientsByName(ctx context.Context, req *clientpb.SearchClientsByNameRequest) (*clientpb.SearchClientsByNameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search clients by name request is required")
	}

	limit := int32(20)
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	// SQL Server does not support LIMIT; use a CTE + TOP or subquery.
	// We use a subquery with ORDER BY + OFFSET/FETCH for correctness, but since
	// the caller only wants the top N matches (no offset), TOP is idiomatic.
	// The label alias drives ORDER BY — it is a computed COALESCE, safe to reference.
	query := fmt.Sprintf(`
		SELECT TOP %d
			c.id,
			COALESCE(
				NULLIF(c.name, ''),
				NULLIF(LTRIM(RTRIM(ISNULL(u.first_name, '') + ' ' + ISNULL(u.last_name, ''))), ''),
				c.id
			) AS label
		FROM client c
		LEFT JOIN [user] u ON c.user_id = u.id
		WHERE c.active = 1
			AND (@p1 = '' OR
				c.name LIKE @p1 OR
				u.first_name LIKE @p1 OR
				u.last_name LIKE @p1)
		ORDER BY label ASC
	`, limit)

	pattern := ""
	if req.Query != "" {
		pattern = "%" + req.Query + "%"
	}

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, pattern)
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
		results = append(results, &clientpb.SearchClientResult{Id: id, Label: label})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search client rows: %w", err)
	}

	return &clientpb.SearchClientsByNameResponse{Results: results, Success: true}, nil
}

// NewClientRepository creates a new SQL Server client repository (old-style constructor).
func NewClientRepository(db *sql.DB, tableName string) clientpb.ClientDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerClientRepository(dbOps, tableName)
}
