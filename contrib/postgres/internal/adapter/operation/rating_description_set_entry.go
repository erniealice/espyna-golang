//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	auditadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/audit"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	"github.com/erniealice/espyna-golang/shared/placeholder"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	"github.com/lib/pq"
	"google.golang.org/protobuf/encoding/protojson"
)

// noDescriptionBandRole is the score_scale_band.band_role value (schema-
// proposal.md §9.3, Q20) that can NEVER carry a rating_description_set_entry
// (typically level 0 — "changing a rating to 0 clears the note"). Data, not
// code: no "0" literal anywhere in this guard.
const noDescriptionBandRole = "no_description"

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RatingDescriptionSetEntry, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres rating_description_set_entry repository requires *sql.DB, got %T", conn)
		}
		// schema-proposal.md §9.4 / codex-review-impl2.out.md finding 9: join the
		// ambient transaction so Create/Update/Delete diff-audits roll back with
		// the write (see rating_description_set.go's init() for the full note).
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresRatingDescriptionSetEntryRepository(dbOps, tableName), nil
	})
}

// PostgresRatingDescriptionSetEntryRepository implements rating_description_set_entry
// CRUD via PostgreSQL. Unlike the plain CRUD adapters (score_scale, etc.), every
// write (Create/Update/Delete) enforces two invariants regardless of caller
// (schema-proposal.md §9.2, §9.3, "whatever the caller — drawer, import, generic
// CRUD route"):
//  1. the parent rating_description_set must be DRAFT (locked FOR SHARE so a
//     concurrent publish serializes against it — the loser gets SET_NOT_DRAFT);
//  2. the target score_scale_band must not carry band_role = 'no_description'
//     (BAND_NO_DESCRIPTION).
//
// Both checks run inside guardRatingDescriptionSetEntryWrite, which — like
// PostgresTaskOutcomeRepository.GuardCellWrite — requires an ambient
// transaction on ctx and fails closed otherwise. Callers (the entry use cases)
// must wrap every write in services.Transactor.ExecuteInTransaction.
type PostgresRatingDescriptionSetEntryRepository struct {
	pb.UnimplementedRatingDescriptionSetEntryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresRatingDescriptionSetEntryRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.RatingDescriptionSetEntryDomainServiceServer {
	if tableName == "" {
		tableName = entityid.RatingDescriptionSetEntry
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresRatingDescriptionSetEntryRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresRatingDescriptionSetEntryRepository) executor(ctx context.Context) sqlexec.DBExecutor {
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

// guardRatingDescriptionSetEntryWrite enforces the parent-DRAFT + band-role
// invariants (see type doc). bandID may be empty (Delete does not need to
// re-validate the band — it only needs the DRAFT guard); when non-empty it
// also checks the band belongs to the set's score_scale.
func (r *PostgresRatingDescriptionSetEntryRepository) guardRatingDescriptionSetEntryWrite(ctx context.Context, setID, bandID string) error {
	if setID == "" {
		return fmt.Errorf("rating description set entry write guard: rating_description_set_id is required (fail closed)")
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return fmt.Errorf("rating description set entry write guard: no trusted workspace in context (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return fmt.Errorf("rating description set entry write guard: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return fmt.Errorf("rating description set entry write guard: requires an ambient transaction (fail closed)")
	}

	var status, setWorkspaceID, scoreScaleID string
	if err := exec.QueryRowContext(ctx,
		`SELECT version_status, workspace_id, score_scale_id FROM `+entityid.RatingDescriptionSet+` WHERE id = $1 FOR SHARE`,
		setID).Scan(&status, &setWorkspaceID, &scoreScaleID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("rating description set entry write guard: parent set not found — fail closed")
		}
		return fmt.Errorf("rating description set entry write guard: lock parent set FOR SHARE: %w", err)
	}
	if setWorkspaceID != idn.WorkspaceID {
		return fmt.Errorf("rating description set entry write guard: parent set is in a different workspace — fail closed")
	}
	if status != ratingDescriptionSetVersionStatusDraft {
		return fmt.Errorf("SET_NOT_DRAFT: rating description set entries can only be written while the parent set is DRAFT")
	}

	if bandID == "" {
		return nil
	}
	var bandRole sql.NullString
	var bandScaleID string
	// fix2-backend (codex impl2 #4): band AND its scale must be owned by the
	// caller's workspace or global (workspace_id IS NULL — the score_scale /
	// score_scale_band convention of outcome_matrix_query.go).
	var ownedBand, ownedScale bool
	if err := exec.QueryRowContext(ctx,
		`SELECT COALESCE(b.band_role, ''), b.score_scale_id,
		        (b.workspace_id = $2 OR b.workspace_id IS NULL),
		        COALESCE(ss.workspace_id = $2 OR ss.workspace_id IS NULL, false)
		   FROM `+entityid.ScoreScaleBand+` b
		   LEFT JOIN `+entityid.ScoreScale+` ss ON ss.id = b.score_scale_id
		  WHERE b.id = $1`,
		bandID, idn.WorkspaceID).Scan(&bandRole, &bandScaleID, &ownedBand, &ownedScale); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("rating description set entry write guard: score_scale_band not found — fail closed")
		}
		return fmt.Errorf("rating description set entry write guard: read score_scale_band: %w", err)
	}
	if bandRole.Valid && bandRole.String == noDescriptionBandRole {
		return fmt.Errorf("BAND_NO_DESCRIPTION: this level (band_role=no_description) can never carry a rating description entry")
	}
	if !ownedBand || !ownedScale {
		return fmt.Errorf("INVALID_CONFIG: rating description set entry references a score scale level owned by another workspace — fail closed")
	}
	if bandScaleID != scoreScaleID {
		return fmt.Errorf("rating description set entry write guard: score_scale_band does not belong to the set's score_scale — fail closed")
	}
	return nil
}

// validateRatingDescriptionPlaceholders is the authoring guard of schema-
// proposal.md §10: entry text may only contain placeholder tags in the
// shared/placeholder allowlist, so a PUBLISHED set can only ever carry
// renderable tags. Braces that do not match the tag grammar are literal text
// and pass. The returned error starts with the bounded code
// UNKNOWN_PLACEHOLDER (same "CODE: message" convention as SET_NOT_DRAFT /
// BAND_NO_DESCRIPTION) and wraps the typed *placeholder.Error, so
// errors.Is(err, placeholder.ErrUnknownPlaceholder) holds. It runs before any
// DB access (create AND update — every entry write path in this adapter).
func validateRatingDescriptionPlaceholders(text string) error {
	if err := placeholder.ValidateAllowed(text); err != nil {
		return fmt.Errorf("%s: rating description text contains a placeholder tag that is not allowed: %w", placeholder.CodeUnknownPlaceholder, err)
	}
	return nil
}

func (r *PostgresRatingDescriptionSetEntryRepository) CreateRatingDescriptionSetEntry(ctx context.Context, req *pb.CreateRatingDescriptionSetEntryRequest) (*pb.CreateRatingDescriptionSetEntryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("rating description set entry data is required")
	}
	if err := validateRatingDescriptionPlaceholders(req.Data.GetDescription()); err != nil {
		return nil, err
	}
	if err := r.guardRatingDescriptionSetEntryWrite(ctx, req.Data.RatingDescriptionSetId, req.Data.ScoreScaleBandId); err != nil {
		return nil, err
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create rating description set entry: %w", err)
	}
	item, err := ratingDescriptionSetEntryFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateRatingDescriptionSetEntryResponse{Data: []*pb.RatingDescriptionSetEntry{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetEntryRepository) ReadRatingDescriptionSetEntry(ctx context.Context, req *pb.ReadRatingDescriptionSetEntryRequest) (*pb.ReadRatingDescriptionSetEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set entry ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rating description set entry: %w", err)
	}
	item, err := ratingDescriptionSetEntryFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadRatingDescriptionSetEntryResponse{Data: []*pb.RatingDescriptionSetEntry{item}, Success: true}, nil
}

// UpdateRatingDescriptionSetEntry locks and validates the entry's ACTUAL
// stored parent (not a request-supplied one) and rejects any attempt to
// reparent it (W3 follow-up finding, 2026-09-25 — codex-review-impl1.out.md:
// "validates a request-supplied DRAFT parent, then strips that parent from
// the update. Supplying a published entry ID plus a same-scale DRAFT parent
// therefore changes the published entry."). The real
// rating_description_set_id is read directly (not from the caller-supplied
// req.Data), so a caller can never point the guard at a different (e.g.
// DRAFT, caller-controlled) set while writing to an entry whose real parent
// is PUBLISHED/DEPRECATED. score_scale_band_id may still be changed by an
// update (only rating_description_set_id — the parent anchor — is immutable).
func (r *PostgresRatingDescriptionSetEntryRepository) UpdateRatingDescriptionSetEntry(ctx context.Context, req *pb.UpdateRatingDescriptionSetEntryRequest) (*pb.UpdateRatingDescriptionSetEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set entry ID is required")
	}
	if err := validateRatingDescriptionPlaceholders(req.Data.GetDescription()); err != nil {
		return nil, err
	}
	existingResult, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rating description set entry for update guard: %w", err)
	}
	existing, err := ratingDescriptionSetEntryFromResult(existingResult)
	if err != nil {
		return nil, err
	}
	actualSetID := existing.GetRatingDescriptionSetId()
	if req.Data.RatingDescriptionSetId != "" && req.Data.RatingDescriptionSetId != actualSetID {
		return nil, fmt.Errorf("REPARENT_FORBIDDEN: rating_description_set_id cannot be changed on update (entry's actual parent is authoritative, not a request-supplied value)")
	}
	effectiveBandID := existing.GetScoreScaleBandId()
	if req.Data.ScoreScaleBandId != "" {
		effectiveBandID = req.Data.ScoreScaleBandId
	}
	// guardRatingDescriptionSetEntryWrite locks the ACTUAL parent set row FOR
	// SHARE (never a request-supplied one) and checks it is DRAFT.
	if err := r.guardRatingDescriptionSetEntryWrite(ctx, actualSetID, effectiveBandID); err != nil {
		return nil, err
	}
	// The parent set an entry belongs to is immutable once written (a re-home
	// would let an entry escape the DRAFT lock it was created under); strip it
	// from the write payload even though the check above already rejects a
	// differing value — belt and suspenders, same as task_outcome stripping
	// its membership anchors.
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	delete(data, "rating_description_set_id")
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update rating description set entry: %w", err)
	}
	item, err := ratingDescriptionSetEntryFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateRatingDescriptionSetEntryResponse{Data: []*pb.RatingDescriptionSetEntry{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetEntryRepository) DeleteRatingDescriptionSetEntry(ctx context.Context, req *pb.DeleteRatingDescriptionSetEntryRequest) (*pb.DeleteRatingDescriptionSetEntryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set entry ID is required")
	}
	existingResult, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rating description set entry for delete guard: %w", err)
	}
	existing, err := ratingDescriptionSetEntryFromResult(existingResult)
	if err != nil {
		return nil, err
	}
	if err := r.guardRatingDescriptionSetEntryWrite(ctx, existing.GetRatingDescriptionSetId(), ""); err != nil {
		return nil, err
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete rating description set entry: %w", err)
	}
	return &pb.DeleteRatingDescriptionSetEntryResponse{Success: true}, nil
}

func (r *PostgresRatingDescriptionSetEntryRepository) ListRatingDescriptionSetEntries(ctx context.Context, req *pb.ListRatingDescriptionSetEntriesRequest) (*pb.ListRatingDescriptionSetEntriesResponse, error) {
	// fix3-backend (codex-review-impl3 #2): forward search/filters/sort/
	// pagination (see ratingDescriptionListParams). The frozen List response
	// has no pagination block; ListRatingDescriptionSetEntriesPage exposes the
	// same query with accurate total/has_next for callers that need it.
	items, _, err := r.ListRatingDescriptionSetEntriesPage(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.ListRatingDescriptionSetEntriesResponse{Data: items, Success: true}, nil
}

var _ portsdomain.RatingDescriptionSetEntryPager = (*PostgresRatingDescriptionSetEntryRepository)(nil)

// ListRatingDescriptionSetEntriesPage is ListRatingDescriptionSetEntries plus
// the adapter's pagination metadata (total_items from COUNT(*) over the same
// predicate, has_next/has_prev, current/total pages). Implements
// domainports.RatingDescriptionSetEntryPager.
func (r *PostgresRatingDescriptionSetEntryRepository) ListRatingDescriptionSetEntriesPage(ctx context.Context, req *pb.ListRatingDescriptionSetEntriesRequest) ([]*pb.RatingDescriptionSetEntry, *commonpb.PaginationResponse, error) {
	params, err := ratingDescriptionListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, nil, err
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list rating description set entries: %w", err)
	}
	var items []*pb.RatingDescriptionSetEntry
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal rating description set entry row: %w", err)
		}
		item := &pb.RatingDescriptionSetEntry{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			return nil, nil, fmt.Errorf("failed to decode rating description set entry row: %w", err)
		}
		items = append(items, item)
	}
	return items, listResult.Pagination, nil
}

var _ portsdomain.RatingDescriptionSetEntryCounter = (*PostgresRatingDescriptionSetEntryRepository)(nil)

// CountActiveRatingDescriptionSetEntriesBySet implements
// portsdomain.RatingDescriptionSetEntryCounter (codex-review-impl4.out.md
// round-3/round-4 disposition #2). Replaces the former workspace-wide,
// 5,000-row-capped page loop with one scoped GROUP BY aggregation, exactly
// as accurate for a workspace with 50 entries as for one with 50,000.
func (r *PostgresRatingDescriptionSetEntryRepository) CountActiveRatingDescriptionSetEntriesBySet(ctx context.Context, ratingDescriptionSetIds []string) (map[string]int32, error) {
	counts := map[string]int32{}
	if len(ratingDescriptionSetIds) == 0 {
		return counts, nil
	}
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("count active rating description set entries: no SQL executor available")
	}
	rows, err := exec.QueryContext(ctx,
		`SELECT rating_description_set_id, COUNT(*) FROM `+entityid.RatingDescriptionSetEntry+`
		 WHERE active = true AND rating_description_set_id = ANY($1)
		 GROUP BY rating_description_set_id`,
		pq.Array(ratingDescriptionSetIds))
	if err != nil {
		return nil, fmt.Errorf("count active rating description set entries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var setID string
		var count int32
		if err := rows.Scan(&setID, &count); err != nil {
			return nil, fmt.Errorf("count active rating description set entries: scan: %w", err)
		}
		counts[setID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("count active rating description set entries: %w", err)
	}
	return counts, nil
}

func ratingDescriptionSetEntryFromResult(result any) (*pb.RatingDescriptionSetEntry, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.RatingDescriptionSetEntry{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}
