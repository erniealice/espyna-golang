//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.OutcomeCriteria, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres outcome_criteria repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresOutcomeCriteriaRepository(dbOps, tableName), nil
	})
}

// PostgresOutcomeCriteriaRepository implements outcome_criteria CRUD operations using PostgreSQL
type PostgresOutcomeCriteriaRepository struct {
	pb.UnimplementedOutcomeCriteriaDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresOutcomeCriteriaRepository creates a new PostgreSQL outcome_criteria repository
func NewPostgresOutcomeCriteriaRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.OutcomeCriteriaDomainServiceServer {
	if tableName == "" {
		tableName = "outcome_criteria"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresOutcomeCriteriaRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateOutcomeCriteria creates a new outcome_criteria record
func (r *PostgresOutcomeCriteriaRepository) CreateOutcomeCriteria(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest) (*pb.CreateOutcomeCriteriaResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("outcome criteria data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create outcome criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.OutcomeCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateOutcomeCriteriaResponse{
		Success: true,
		Data:    []*pb.OutcomeCriteria{item},
	}, nil
}

// ReadOutcomeCriteria retrieves an outcome_criteria record by ID
func (r *PostgresOutcomeCriteriaRepository) ReadOutcomeCriteria(ctx context.Context, req *pb.ReadOutcomeCriteriaRequest) (*pb.ReadOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read outcome criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.OutcomeCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadOutcomeCriteriaResponse{
		Success: true,
		Data:    []*pb.OutcomeCriteria{item},
	}, nil
}

// UpdateOutcomeCriteria updates an outcome_criteria record
func (r *PostgresOutcomeCriteriaRepository) UpdateOutcomeCriteria(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest) (*pb.UpdateOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update outcome criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.OutcomeCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateOutcomeCriteriaResponse{
		Success: true,
		Data:    []*pb.OutcomeCriteria{item},
	}, nil
}

// DeleteOutcomeCriteria deletes an outcome_criteria record (soft delete)
func (r *PostgresOutcomeCriteriaRepository) DeleteOutcomeCriteria(ctx context.Context, req *pb.DeleteOutcomeCriteriaRequest) (*pb.DeleteOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete outcome criteria: %w", err)
	}

	return &pb.DeleteOutcomeCriteriaResponse{
		Success: true,
	}, nil
}

// executor returns the transaction-aware SQL executor: the active *sql.Tx when
// one is present on ctx (so the purpose-built anchor reads participate in the
// use-case transaction and see its uncommitted writes), else the pooled *sql.DB.
// Mirrors task_outcome.go's helper: the workspace-aware dbOps exposes
// GetExecutor(ctx); we type-assert so this works regardless of the concrete
// dbOps wrapping, falling back to the stored *sql.DB handle.
func (r *PostgresOutcomeCriteriaRepository) executor(ctx context.Context) sqlexec.DBExecutor {
	if ep, ok := r.dbOps.(interface {
		GetExecutor(ctx context.Context) sqlexec.DBExecutor
	}); ok {
		if e := ep.GetExecutor(ctx); e != nil {
			return e
		}
	}
	if r.db != nil {
		return r.db
	}
	return nil
}

// LineageEstablishedCode is the purpose-built lineage read: a single indexed
// point lookup against the criteria_group anchor (PK id -> code). The anchor IS
// the authoritative "established code across ALL versions" — the populate
// trigger claims it on the first coded write and the composite FK pins every
// coded version to it — so no outcome_criteria scan (let alone a paginated one)
// is needed. Returns "" when the lineage has no coded version yet. Single
// statement -> inherently snapshot-consistent; the caller supplies the context
// deadline. Parameterized; not workspace-scoped because a lineage's established
// code is a global fact the composite FK enforces regardless of tenant (only
// the code string is disclosed).
func (r *PostgresOutcomeCriteriaRepository) LineageEstablishedCode(ctx context.Context, criteriaGroupID string) (string, error) {
	if criteriaGroupID == "" {
		return "", nil
	}
	var code string
	err := r.executor(ctx).QueryRowContext(ctx,
		`SELECT "code" FROM "criteria_group" WHERE "id" = $1`, criteriaGroupID,
	).Scan(&code)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read criteria_group anchor for lineage %q: %w", criteriaGroupID, err)
	}
	return code, nil
}

// CodeOwnerGroup is the purpose-built collision read: a single indexed point
// lookup against the criteria_group anchor's normalized domain claim
// (uq_criteria_group_domain_code). Returns the criteria_group_id that owns
// (scopeKey, workspaceKey, industryKey, code) across ALL version statuses and
// both active states, or "" when unclaimed. Keys are the normalized ”-for-NULL
// forms the anchor stores (the caller derives workspaceKey from the TRUSTED
// request context, never from client input). Single statement -> inherently
// snapshot-consistent; the unique index behind it remains the authoritative
// race backstop.
func (r *PostgresOutcomeCriteriaRepository) CodeOwnerGroup(ctx context.Context, scopeKey, workspaceKey, industryKey, code string) (string, error) {
	if code == "" {
		return "", nil
	}
	var id string
	err := r.executor(ctx).QueryRowContext(ctx,
		`SELECT "id" FROM "criteria_group"
		 WHERE "scope" = $1 AND "workspace_key" = $2 AND "industry_key" = $3 AND "code" = $4`,
		scopeKey, workspaceKey, industryKey, code,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read criteria_group domain claim for code %q: %w", code, err)
	}
	return id, nil
}

// LineageClaimedDomain is the purpose-built domain read backing the NEW-1
// domain-consistency guard: a single indexed point lookup against the
// criteria_group anchor (PK id -> scope/workspace_key/industry_key). Returns
// claimed=false when the lineage has no anchor yet (no coded version). The
// columns are NOT NULL, ”-normalized forms (the populate trigger COALESCEs
// NULL domain parts to ”), so plain string scans are exact. Single statement
// -> inherently snapshot-consistent; parameterized; the caller supplies the
// context deadline. Not workspace-scoped for the same reason as
// LineageEstablishedCode: the anchor's claim is the global fact the guard
// compares against (only normalized domain keys are disclosed).
func (r *PostgresOutcomeCriteriaRepository) LineageClaimedDomain(ctx context.Context, criteriaGroupID string) (string, string, string, bool, error) {
	if criteriaGroupID == "" {
		return "", "", "", false, nil
	}
	var scopeKey, workspaceKey, industryKey string
	err := r.executor(ctx).QueryRowContext(ctx,
		`SELECT "scope", "workspace_key", "industry_key" FROM "criteria_group" WHERE "id" = $1`, criteriaGroupID,
	).Scan(&scopeKey, &workspaceKey, &industryKey)
	if err == sql.ErrNoRows {
		return "", "", "", false, nil
	}
	if err != nil {
		return "", "", "", false, fmt.Errorf("failed to read criteria_group domain claim for lineage %q: %w", criteriaGroupID, err)
	}
	return scopeKey, workspaceKey, industryKey, true, nil
}

// ListOutcomeCriterias lists outcome_criteria records with optional filters
//
// Pagination pass-through (mirrors ListJobTasks, job_task.go): the caller's
// req.Pagination/Sort/Search MUST be forwarded into ListParams. Dropping them
// forced the generic core List onto its 100-row default at offset 0 on EVERY
// call, so a paging caller (the code-validation fallback loop) re-read the same
// first 100 rows for every page and never advanced. Callers that omit all four
// keep the prior nil-params behavior byte-for-byte.
func (r *PostgresOutcomeCriteriaRepository) ListOutcomeCriterias(ctx context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil || req.Sort != nil || req.Search != nil) {
		params = &interfaces.ListParams{Search: req.Search, Filters: req.Filters, Sort: req.Sort, Pagination: req.Pagination}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list outcome criterias: %w", err)
	}

	var items []*pb.OutcomeCriteria
	for _, result := range listResult.Data {
		// Decode failures propagate as errors (fail-closed) rather than being
		// logged-and-dropped. A silently-skipped row would corrupt the code
		// lineage/collision pre-checks (a dropped row could hide an established
		// code or a collision), so a genuinely undecodable row must fail the whole
		// read instead of returning a short, wrong answer.
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal outcome_criteria row: %w", err)
		}

		item := &pb.OutcomeCriteria{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
			return nil, fmt.Errorf("failed to decode outcome_criteria row: %w", err)
		}
		items = append(items, item)
	}

	return &pb.ListOutcomeCriteriasResponse{
		Success: true,
		Data:    items,
	}, nil
}

var outcomeCriteriaSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "criteria_group_id",
	"version", "version_status", "scope", "name", "criteria_type",
}

// GetOutcomeCriteriaListPageData retrieves outcome_criterias with pagination, filtering, sorting, and search
func (r *PostgresOutcomeCriteriaRepository) GetOutcomeCriteriaListPageData(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaListPageDataRequest,
) (*pb.GetOutcomeCriteriaListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get outcome criteria list page data request is required")
	}

	searchPattern, err := postgresCore.BoundedContainsSearchPattern(req.GetSearch())
	if err != nil {
		return nil, fmt.Errorf("invalid list search: %w", err)
	}

	limit, offset, page, err := postgresCore.BoundedOffsetPagination(req.GetPagination(), 50)
	if err != nil {
		return nil, fmt.Errorf("invalid list pagination: %w", err)
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The ORDER BY
	// runs against the outer `enriched e` projection (unprefixed cols), so the
	// whitelist + fallback are unprefixed. Default preserves name ASC.
	orderByClause, err := postgresCore.BuildOrderBy(outcomeCriteriaSortableSQLCols, req.GetSort(), "name ASC")
	if err != nil {
		return nil, err
	}

	// A1 (CRITICAL): scope to the caller's workspace. This raw-SQL path bypasses
	// the WorkspaceAwareOperations decorator. outcome_criteria carries its own
	// workspace_id column (verified against the baseline schema; the item method
	// and ListByScope in this same file already scope oc.workspace_id). Empty wsID
	// = service-to-service call → no scoping. $4 carries the workspace_id.
	wsID := identity.Must(ctx).WorkspaceID

	query := `
		WITH enriched AS (
			SELECT
				oc.id,
				oc.date_created,
				oc.date_modified,
				oc.active,
				oc.criteria_group_id,
				oc.version,
				oc.version_status,
				oc.scope,
				oc.name,
				oc.criteria_type
			FROM ` + entityid.OutcomeCriteria + ` oc
			WHERE oc.active = true
			  AND ($4::text = '' OR oc.workspace_id = $4::text)
			  AND ($1::text IS NULL OR $1::text = '' OR
			       oc.name ILIKE $1)
		)
		-- A3 (Q-PAGE-COUNT default tier): COUNT(*) OVER () computes the total in the
		-- same scan as the page rows (the prior counted CTE forced a second scan).
		SELECT
			e.*,
			COUNT(*) OVER () AS total
		FROM enriched e
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outcome criteria list page data: %w", err)
	}
	defer rows.Close()

	var items []*pb.OutcomeCriteria
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			criteriaGroupID string
			version         int32
			versionStatus   int32
			scope           int32
			name            string
			criteriaType    int32
			total           int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&criteriaGroupID,
			&version,
			&versionStatus,
			&scope,
			&name,
			&criteriaType,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outcome criteria row: %w", err)
		}

		totalCount = total

		item := &pb.OutcomeCriteria{
			Id:              id,
			Active:          active,
			CriteriaGroupId: criteriaGroupID,
			Version:         version,
			Name:            name,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			item.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			item.DateModifiedString = &dmStr
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outcome criteria rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetOutcomeCriteriaListPageDataResponse{
		OutcomeCriteriaList: items,
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

// GetOutcomeCriteriaItemPageData retrieves a single outcome_criteria with enriched data
func (r *PostgresOutcomeCriteriaRepository) GetOutcomeCriteriaItemPageData(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaItemPageDataRequest,
) (*pb.GetOutcomeCriteriaItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get outcome criteria item page data request is required")
	}
	if req.OutcomeCriteriaId == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	// A1 (CRITICAL): scope to the caller's workspace. This method bypasses the
	// WorkspaceAwareOperations decorator (raw SQL via db.GetDB()). outcome_criteria
	// carries its own workspace_id column (verified baseline; ListByScope in this
	// same file scopes oc.workspace_id). Empty wsID = service-to-service call → no
	// scoping. $2 carries the workspace_id.
	wsID := identity.Must(ctx).WorkspaceID

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM ` + entityid.OutcomeCriteria + ` oc
		WHERE oc.id = $1 AND oc.active = true
		  AND ($2::text = '' OR oc.workspace_id = $2::text)
	`

	row := r.db.QueryRowContext(ctx, query, req.OutcomeCriteriaId, wsID)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		criteriaGroupID string
		version         int32
		versionStatus   int32
		scope           int32
		name            string
		criteriaType    int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&criteriaGroupID,
		&version,
		&versionStatus,
		&scope,
		&name,
		&criteriaType,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("outcome criteria with ID '%s' not found", req.OutcomeCriteriaId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query outcome criteria item page data: %w", err)
	}

	item := &pb.OutcomeCriteria{
		Id:              id,
		Active:          active,
		CriteriaGroupId: criteriaGroupID,
		Version:         version,
		Name:            name,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		item.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		item.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		item.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		item.DateModifiedString = &dmStr
	}

	return &pb.GetOutcomeCriteriaItemPageDataResponse{
		OutcomeCriteria: item,
		Success:         true,
	}, nil
}

// ListByGroup retrieves all outcome_criteria for a given criteria group, ordered by version DESC
func (r *PostgresOutcomeCriteriaRepository) ListByGroup(
	ctx context.Context,
	req *pb.ListOutcomeCriteriasByGroupRequest,
) (*pb.ListOutcomeCriteriasByGroupResponse, error) {
	if req == nil || req.CriteriaGroupId == "" {
		return nil, fmt.Errorf("criteria group ID is required")
	}

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM ` + entityid.OutcomeCriteria + ` oc
		WHERE oc.criteria_group_id = $1 AND oc.active = true
		ORDER BY oc.version DESC
	`

	rows, err := r.db.QueryContext(ctx, query, req.CriteriaGroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to list outcome criterias by group: %w", err)
	}
	defer rows.Close()

	var items []*pb.OutcomeCriteria
	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			criteriaGroupID string
			version         int32
			versionStatus   int32
			scope           int32
			name            string
			criteriaType    int32
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&criteriaGroupID,
			&version,
			&versionStatus,
			&scope,
			&name,
			&criteriaType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outcome criteria row: %w", err)
		}

		item := &pb.OutcomeCriteria{
			Id:              id,
			Active:          active,
			CriteriaGroupId: criteriaGroupID,
			Version:         version,
			Name:            name,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			item.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			item.DateModifiedString = &dmStr
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outcome criteria rows: %w", err)
	}

	return &pb.ListOutcomeCriteriasByGroupResponse{
		OutcomeCriterias: items,
		Success:          true,
	}, nil
}

// GetCurrentPublished retrieves the current published outcome_criteria for a criteria group
func (r *PostgresOutcomeCriteriaRepository) GetCurrentPublished(
	ctx context.Context,
	req *pb.GetCurrentPublishedOutcomeCriteriaRequest,
) (*pb.GetCurrentPublishedOutcomeCriteriaResponse, error) {
	if req == nil || req.CriteriaGroupId == "" {
		return nil, fmt.Errorf("criteria group ID is required")
	}

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM ` + entityid.OutcomeCriteria + ` oc
		WHERE oc.criteria_group_id = $1 AND oc.version_status = 2 AND oc.active = true
		ORDER BY oc.version DESC
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, req.CriteriaGroupId)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		criteriaGroupID string
		version         int32
		versionStatus   int32
		scope           int32
		name            string
		criteriaType    int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&criteriaGroupID,
		&version,
		&versionStatus,
		&scope,
		&name,
		&criteriaType,
	)
	if err == sql.ErrNoRows {
		return &pb.GetCurrentPublishedOutcomeCriteriaResponse{
			Success: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query current published outcome criteria: %w", err)
	}

	item := &pb.OutcomeCriteria{
		Id:              id,
		Active:          active,
		CriteriaGroupId: criteriaGroupID,
		Version:         version,
		Name:            name,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		item.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		item.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		item.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		item.DateModifiedString = &dmStr
	}

	return &pb.GetCurrentPublishedOutcomeCriteriaResponse{
		OutcomeCriteria: item,
		Success:         true,
	}, nil
}

// ListByScope retrieves outcome_criterias filtered by scope, industry code, and workspace
func (r *PostgresOutcomeCriteriaRepository) ListByScope(
	ctx context.Context,
	req *pb.ListOutcomeCriteriasByScopeRequest,
) (*pb.ListOutcomeCriteriasByScopeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("list outcome criterias by scope request is required")
	}

	industryCode := ""
	if req.IndustryCode != nil {
		industryCode = *req.IndustryCode
	}
	workspaceID := ""
	if req.WorkspaceId != nil {
		workspaceID = *req.WorkspaceId
	}

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM ` + entityid.OutcomeCriteria + ` oc
		WHERE oc.scope = $1
		  AND ($2 = '' OR oc.industry_code = $2)
		  AND ($3 = '' OR oc.workspace_id = $3)
		  AND oc.active = true
		ORDER BY oc.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, int32(req.Scope), industryCode, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list outcome criterias by scope: %w", err)
	}
	defer rows.Close()

	var items []*pb.OutcomeCriteria
	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			criteriaGroupID string
			version         int32
			versionStatus   int32
			scope           int32
			name            string
			criteriaType    int32
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&criteriaGroupID,
			&version,
			&versionStatus,
			&scope,
			&name,
			&criteriaType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outcome criteria row: %w", err)
		}

		item := &pb.OutcomeCriteria{
			Id:              id,
			Active:          active,
			CriteriaGroupId: criteriaGroupID,
			Version:         version,
			Name:            name,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			item.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			item.DateModifiedString = &dmStr
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outcome criteria rows: %w", err)
	}

	return &pb.ListOutcomeCriteriasByScopeResponse{
		OutcomeCriterias: items,
		Success:          true,
	}, nil
}

// NewOutcomeCriteriaRepository creates a new PostgreSQL outcome_criteria repository (old-style constructor)
func NewOutcomeCriteriaRepository(db *sql.DB, tableName string) pb.OutcomeCriteriaDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresOutcomeCriteriaRepository(dbOps, tableName)
}
