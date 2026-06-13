//go:build mysql

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
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
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
	registry.RegisterRepositoryFactory("mysql", entityid.Client, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql client repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLClientRepository(dbOps, tableName), nil
	})
}

// MySQLClientRepository implements client CRUD operations using MySQL 8.0+.
type MySQLClientRepository struct {
	clientpb.UnimplementedClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLClientRepository creates a new MySQL client repository.
func NewMySQLClientRepository(dbOps interfaces.DatabaseOperation, tableName string) clientpb.ClientDomainServiceServer {
	if tableName == "" {
		tableName = "client"
	}
	return &MySQLClientRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateClient creates a new client using common MySQL operations.
func (r *MySQLClientRepository) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// ReadClient retrieves a client by ID.
func (r *MySQLClientRepository) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
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

	return &clientpb.ReadClientResponse{
		Data:    []*clientpb.Client{client},
		Success: true,
	}, nil
}

// loadClientPaymentTerm fetches the PaymentTerm row associated with a client.payment_term_id.
func (r *MySQLClientRepository) loadClientPaymentTerm(ctx context.Context, paymentTermId string) (*paymenttermpb.PaymentTerm, error) {
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

// loadClientUser fetches the User row associated with a client.user_id.
func (r *MySQLClientRepository) loadClientUser(ctx context.Context, userId string) (*userpb.User, error) {
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

// UpdateClient updates a client using common MySQL operations.
func (r *MySQLClientRepository) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// DeleteClient deletes a client using common MySQL operations (soft delete).
func (r *MySQLClientRepository) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
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
	// Derived column: available for ORDER BY via LATERAL-equivalent subquery in GetClientListPageData.
	"active_subscriptions",
}

var clientSortSpec = espynahttp.SortSpec{AllowedCols: clientSortableSQLCols}

// ListClients lists clients using common MySQL operations.
func (r *MySQLClientRepository) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		client := &clientpb.Client{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
			continue
		}
		clients = append(clients, client)
	}

	return &clientpb.ListClientsResponse{
		Data: clients,
	}, nil
}

// GetClientListPageData retrieves clients with user/payment_term/subscription-count enrichment.
//
// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders, args in same left-to-right order)
//   - "user" → `user` (backtick-quoted reserved word)
//   - ILIKE → LIKE (MySQL ci collation handles case-insensitivity)
//   - LEFT JOIN LATERAL → inline correlated subquery (LEFT JOIN LATERAL is supported
//     in MySQL 8.0.14+ but the correlated subquery form is more portable for this use)
//   - COUNT(*) OVER () stays — MySQL 8.0+ supports window functions
//   - mysqlCore.BuildOrderBy uses backtick quoting
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLClientRepository) GetClientListPageData(
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

	// Default sort: name ASC matches the view layer default.
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

	// Build filter/search WHERE clauses.
	// First arg (?) is workspace_id; filter builder starts at index 2 (for parity with postgres).
	searchFields := []string{"c.name", "c.internal_id", "u.first_name", "u.last_name", "u.email_address"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE c.workspace_id = ?"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	// Args: [workspaceID, ...filterArgs, workspaceID(for subquery), limit, offset]
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, workspaceID, limit, offset)

	// CTE query — MySQL 8.0+ supports CTEs and COUNT(*) OVER ().
	// active_subscriptions: correlated subquery scoped by workspace_id.
	// Dialect: "user" → `user`; LIMIT/OFFSET use positional ?
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
				(SELECT COUNT(*) FROM subscription s WHERE s.client_id = c.id AND s.active = 1 AND s.workspace_id = ?) AS active_subscriptions,
				u.id AS user_id_value,
				u.first_name AS user_first_name,
				u.last_name AS user_last_name,
				u.email_address AS user_email_address,
				u.mobile_number AS user_phone_number,
				COUNT(*) OVER () AS total
			FROM client c
			LEFT JOIN `+"`user`"+` u ON c.user_id = u.id
			LEFT JOIN payment_term pt ON c.payment_term_id = pt.id
			%s
		)
		SELECT * FROM enriched
		ORDER BY %s %s
		LIMIT ? OFFSET ?;
	`, whereSQL, sortField, sortOrder)

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

		// activeSubCount is used only for ORDER BY at DB level.
		_ = activeSubCount

		clients = append(clients, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client rows: %w", err)
	}

	// Load Categories per row — bounded N+1 (≤ page size).
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

// GetClientItemPageData retrieves a single client + categories.
func (r *MySQLClientRepository) GetClientItemPageData(
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

	return &clientpb.GetClientItemPageDataResponse{
		Client:  client,
		Success: true,
	}, nil
}

// loadClientCategories loads category tags for a client via JOIN through client_category to category.
//
// Dialect change from postgres gold standard: $1 → ? (positional), active = true → active = 1.
func (r *MySQLClientRepository) loadClientCategories(ctx context.Context, clientId string) ([]*clientcategorypb.ClientCategory, error) {
	query := `
		SELECT
			cc.id,
			cc.client_id,
			cc.category_id,
			cat.name,
			cat.description
		FROM client_category cc
		INNER JOIN category cat ON cc.category_id = cat.id
		WHERE cc.client_id = ? AND cc.active = 1 AND cat.active = 1
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

// SearchClientsByName searches clients by company name or user first/last name using LIKE.
//
// Dialect translation: "user" → `user`; ILIKE → LIKE; $N → ?; $1::text cast removed.
func (r *MySQLClientRepository) SearchClientsByName(ctx context.Context, req *clientpb.SearchClientsByNameRequest) (*clientpb.SearchClientsByNameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search clients by name request is required")
	}

	limit := int32(20)
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	// Dialect: "user" → `user`; ILIKE → LIKE; $N → ?; boolean cast removed (MySQL accepts '' as empty string natively)
	query := `
		SELECT
			c.id,
			COALESCE(
				NULLIF(c.name, ''),
				NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
				c.id
			) AS label
		FROM client c
		LEFT JOIN ` + "`user`" + ` u ON c.user_id = u.id
		WHERE c.active = 1
			AND (? = '' OR
				c.name LIKE ? OR
				u.first_name LIKE ? OR
				u.last_name LIKE ?)
		ORDER BY label ASC
		LIMIT ?
	`

	pattern := ""
	if req.Query != "" {
		pattern = "%" + req.Query + "%"
	}

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, pattern, pattern, pattern, pattern, limit)
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

// NewClientRepository creates a new MySQL client repository (old-style constructor).
func NewClientRepository(db *sql.DB, tableName string) clientpb.ClientDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLClientRepository(dbOps, tableName)
}
