//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	"google.golang.org/protobuf/encoding/protojson"
)

// lockedScoreScaleFields are the score_scale columns whose meaning must not
// change once any of the scale's own bands is referenced by a
// rating_description_set_entry of a PUBLISHED or DEPRECATED set
// (schema-proposal.md §9.3: "UpdateScoreScale for scale_kind, input_min/max").
var lockedScoreScaleFields = map[string]bool{
	"scale_kind": true,
	"input_min":  true,
	"input_max":  true,
}

// touchesLockedScoreScaleFields reports whether a protoToMap-shaped
// partial-update payload writes any of lockedScoreScaleFields, or deactivates
// the scale (active=false — the resolver requires ss.active, so deactivating a
// scale drops its published descriptors exactly like deleting it; codex impl2
// #1). active=true (a routine edit-form echo) does not trigger the guard.
func touchesLockedScoreScaleFields(data map[string]any) bool {
	for k := range lockedScoreScaleFields {
		if _, ok := data[k]; ok {
			return true
		}
	}
	if v, ok := data["active"]; ok {
		if b, isBool := v.(bool); !isBool || !b {
			return true
		}
	}
	return false
}

// existsLockedEntryForScale is the scale-level counterpart of
// existsLockedEntryForBand (score_scale_band.go): true when any band of this
// scale is referenced by a rating_description_set_entry of a PUBLISHED or
// DEPRECATED set.
func existsLockedEntryForScale(ctx context.Context, exec sqlexec.DBExecutor, scaleID string) (bool, error) {
	var locked bool
	if err := exec.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM `+ratingDescriptionSetEntryTable+` e
			JOIN `+ratingDescriptionSetTable+` s ON s.id = e.rating_description_set_id
			JOIN `+entityid.ScoreScaleBand+` b ON b.id = e.score_scale_band_id
			WHERE b.score_scale_id = $1
			  AND s.version_status IN ($2, $3)
		)`, scaleID, versionStatusPublishedValue, versionStatusDeprecatedValue).Scan(&locked); err != nil {
		return false, fmt.Errorf("existsLockedEntryForScale: %w", err)
	}
	return locked, nil
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ScoreScale, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres score_scale repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresScoreScaleRepository(dbOps, tableName), nil
	})
}

// PostgresScoreScaleRepository implements score_scale CRUD via PostgreSQL.
// ListByGroup and GetCurrentPublished are extra RPCs covered by the Unimplemented embedding.
type PostgresScoreScaleRepository struct {
	pb.UnimplementedScoreScaleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresScoreScaleRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ScoreScaleDomainServiceServer {
	if tableName == "" {
		tableName = "score_scale"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresScoreScaleRepository{dbOps: dbOps, db: db, tableName: tableName}
}

// executor returns the transaction-aware SQL executor: the active *sql.Tx when
// one is present on ctx, else the pooled *sql.DB. Mirrors
// PostgresScoreScaleBandRepository.executor.
func (r *PostgresScoreScaleRepository) executor(ctx context.Context) sqlexec.DBExecutor {
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

// Descriptor lock protocol (codex impl2 #1) — every writer that can change
// what a PUBLISHED/DEPRECATED rating description set resolves to acquires row
// locks in ONE global order:  score_scale → score_scale_band → rating_description_set.
//
//   - scale edit / delete / deactivate: score_scale FOR UPDATE (this guard).
//   - band edit / delete:               score_scale FOR UPDATE, then band FOR UPDATE
//     (guardScoreScaleBandLockedWrite).
//   - publish:                          score_scale FOR SHARE, then set FOR UPDATE
//     (LockScoreScaleForPublish, rating_description_set_publish_lock.go).
//   - entry create/update/delete:       set FOR SHARE only (never a scale/band lock).
//
// FOR SHARE (publish) conflicts with FOR UPDATE (scale/band editors), so a
// publish and a meaning-changing scale/band edit of the same scale serialize:
// whichever commits second observes the first's committed state (the editor
// sees the now-PUBLISHED reference and rejects with BAND_LOCKED; the publisher
// publishes the already-changed meaning). The single order makes the protocol
// deadlock-free.
//
// guardScoreScaleLockedWrite is the scale-level counterpart of
// PostgresScoreScaleBandRepository.guardScoreScaleBandLockedWrite
// (schema-proposal.md §9.3): once any band of this scale is referenced by a
// PUBLISHED or DEPRECATED set's entry, scale_kind / input_min / input_max can
// never change (BAND_LOCKED — the same code as the band-level guard: the
// scale edit is rejected for exactly the same underlying reason, a locked
// band's meaning). Requires an ambient transaction on ctx and fails closed
// otherwise.
func (r *PostgresScoreScaleRepository) guardScoreScaleLockedWrite(ctx context.Context, scaleID string) error {
	if scaleID == "" {
		return fmt.Errorf("score scale write guard: id is required (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return fmt.Errorf("score scale write guard: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return fmt.Errorf("score scale write guard: requires an ambient transaction (fail closed)")
	}
	var found string
	if err := exec.QueryRowContext(ctx, `SELECT id FROM `+entityid.ScoreScale+` WHERE id = $1 FOR UPDATE`, scaleID).Scan(&found); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("score scale write guard: scale not found — fail closed")
		}
		return fmt.Errorf("score scale write guard: lock scale FOR UPDATE: %w", err)
	}
	locked, err := existsLockedEntryForScale(ctx, exec, scaleID)
	if err != nil {
		return err
	}
	if locked {
		return fmt.Errorf("BAND_LOCKED: this scale has a level referenced by a published or deprecated rating description set and its meaning cannot change")
	}
	return nil
}

func (r *PostgresScoreScaleRepository) CreateScoreScale(ctx context.Context, req *pb.CreateScoreScaleRequest) (*pb.CreateScoreScaleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("score scale data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateScoreScaleResponse{Data: []*pb.ScoreScale{item}, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) ReadScoreScale(ctx context.Context, req *pb.ReadScoreScaleRequest) (*pb.ReadScoreScaleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadScoreScaleResponse{Data: []*pb.ScoreScale{item}, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) UpdateScoreScale(ctx context.Context, req *pb.UpdateScoreScaleRequest) (*pb.UpdateScoreScaleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	if touchesLockedScoreScaleFields(data) {
		if err := r.guardScoreScaleLockedWrite(ctx, req.Data.Id); err != nil {
			return nil, err
		}
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateScoreScaleResponse{Data: []*pb.ScoreScale{item}, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) DeleteScoreScale(ctx context.Context, req *pb.DeleteScoreScaleRequest) (*pb.DeleteScoreScaleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale ID is required")
	}
	// codex impl2 #1: deletion (generic Delete = active=false) is guarded like
	// a locked-field edit — scale FOR UPDATE first (lock protocol above), then
	// BAND_LOCKED when any band is referenced by a PUBLISHED/DEPRECATED set.
	if err := r.guardScoreScaleLockedWrite(ctx, req.Data.Id); err != nil {
		return nil, err
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete score scale: %w", err)
	}
	return &pb.DeleteScoreScaleResponse{Success: true}, nil
}

func (r *PostgresScoreScaleRepository) ListScoreScales(ctx context.Context, req *pb.ListScoreScalesRequest) (*pb.ListScoreScalesResponse, error) {
	// fix3-backend (codex-review-impl3 #2): forward search/sort/pagination
	// too, so the rating-description pickers can page past the adapter's
	// 100-row default cap (a request without them is unchanged: nil params).
	items, err := r.listWithParams(ctx, req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	return &pb.ListScoreScalesResponse{Data: items, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) GetScoreScaleListPageData(ctx context.Context, req *pb.GetScoreScaleListPageDataRequest) (*pb.GetScoreScaleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateScoreScale(all, req.GetPagination())
	_ = page
	return &pb.GetScoreScaleListPageDataResponse{ScoreScaleList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) GetScoreScaleItemPageData(ctx context.Context, req *pb.GetScoreScaleItemPageDataRequest) (*pb.GetScoreScaleItemPageDataResponse, error) {
	if req == nil || req.ScoreScaleId == "" {
		return nil, fmt.Errorf("score scale ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ScoreScaleId)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetScoreScaleItemPageDataResponse{ScoreScale: item, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ScoreScale, error) {
	return r.listWithParams(ctx, nil, filters, nil, nil)
}

func (r *PostgresScoreScaleRepository) listWithParams(ctx context.Context, search *commonpb.SearchRequest, filters *commonpb.FilterRequest, sort *commonpb.SortRequest, pagination *commonpb.PaginationRequest) ([]*pb.ScoreScale, error) {
	var params *interfaces.ListParams
	if search != nil || filters != nil || sort != nil || pagination != nil {
		params = &interfaces.ListParams{Search: search, Filters: filters, Sort: sort, Pagination: pagination}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list score scales: %w", err)
	}
	var items []*pb.ScoreScale
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ScoreScale{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func scoreScaleFromResult(result any) (*pb.ScoreScale, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ScoreScale{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateScoreScale(all []*pb.ScoreScale, p *commonpb.PaginationRequest) (int32, []*pb.ScoreScale, *commonpb.PaginationResponse) {
	limit, page := int32(50), int32(1)
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if off := p.GetOffset(); off != nil && off.Page > 0 {
			page = off.Page
		}
	}
	total := int32(len(all))
	start := (page - 1) * limit
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	totalPages := int32(0)
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return page, all[start:end], &commonpb.PaginationResponse{
		TotalItems:  total,
		CurrentPage: &page,
		TotalPages:  &totalPages,
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
}
