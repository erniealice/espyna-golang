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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	"google.golang.org/protobuf/encoding/protojson"
)

// lockedScoreScaleBandFields are the score_scale_band columns whose meaning
// must not change once any rating_description_set_entry of a PUBLISHED or
// DEPRECATED rating_description_set references the band (schema-proposal.md
// §9.3 "Immutable meaning"). Label/sequence edits (output_label,
// sequence_order, determination, output_value, band_role) stay allowed even
// when the band is locked — only these five keys trigger the guard.
var lockedScoreScaleBandFields = map[string]bool{
	"input_match":    true,
	"input_min":      true,
	"input_max":      true,
	"score_scale_id": true,
	"active":         true,
}

// touchesLockedScoreScaleBandFields reports whether a protoToMap-shaped
// partial-update payload writes any of lockedScoreScaleBandFields.
func touchesLockedScoreScaleBandFields(data map[string]any) bool {
	for k := range lockedScoreScaleBandFields {
		if _, ok := data[k]; ok {
			return true
		}
	}
	return false
}

// existsLockedEntryForBand is the typed adapter check schema-proposal.md
// §9.3 calls ExistsLockedEntryForBand(band_id): true when any
// rating_description_set_entry of a PUBLISHED or DEPRECATED
// rating_description_set references bandID. Shared by score_scale_band.go
// (band-level guard) and score_scale.go (scale-level guard, via
// existsLockedEntryForScale).
func existsLockedEntryForBand(ctx context.Context, exec sqlexec.DBExecutor, bandID string) (bool, error) {
	var locked bool
	if err := exec.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM `+ratingDescriptionSetEntryTable+` e
			JOIN `+ratingDescriptionSetTable+` s ON s.id = e.rating_description_set_id
			WHERE e.score_scale_band_id = $1
			  AND s.version_status IN ($2, $3)
		)`, bandID, versionStatusPublishedValue, versionStatusDeprecatedValue).Scan(&locked); err != nil {
		return false, fmt.Errorf("existsLockedEntryForBand: %w", err)
	}
	return locked, nil
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ScoreScaleBand, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres score_scale_band repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresScoreScaleBandRepository(dbOps, tableName), nil
	})
}

// PostgresScoreScaleBandRepository implements score_scale_band CRUD via PostgreSQL.
type PostgresScoreScaleBandRepository struct {
	pb.UnimplementedScoreScaleBandDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresScoreScaleBandRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ScoreScaleBandDomainServiceServer {
	if tableName == "" {
		tableName = "score_scale_band"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresScoreScaleBandRepository{dbOps: dbOps, db: db, tableName: tableName}
}

// executor returns the transaction-aware SQL executor: the active *sql.Tx when
// one is present on ctx, else the pooled *sql.DB. Mirrors
// PostgresRatingDescriptionSetEntryRepository.executor.
func (r *PostgresScoreScaleBandRepository) executor(ctx context.Context) sqlexec.DBExecutor {
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

// guardScoreScaleBandLockedWrite enforces schema-proposal.md §9.3: once any
// rating_description_set_entry of a PUBLISHED or DEPRECATED
// rating_description_set references this band, input_match / input_min /
// input_max / score_scale_id / active can never change, and the band can
// never be deleted (BAND_LOCKED). The band row is locked FOR UPDATE first so
// a concurrent write (or a concurrent entry publish that would newly
// reference this band) serializes against this check rather than racing it —
// same "SELECT ... FOR UPDATE then check" shape schema-proposal.md §9.3
// explicitly allows. Requires an ambient transaction on ctx
// (services.Transactor.ExecuteInTransaction) and fails closed otherwise,
// mirroring PostgresRatingDescriptionSetEntryRepository's write guard.
func (r *PostgresScoreScaleBandRepository) guardScoreScaleBandLockedWrite(ctx context.Context, bandID string) error {
	if bandID == "" {
		return fmt.Errorf("score scale band write guard: id is required (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return fmt.Errorf("score scale band write guard: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return fmt.Errorf("score scale band write guard: requires an ambient transaction (fail closed)")
	}
	// Lock protocol (score_scale.go "Descriptor lock protocol"): scale → band
	// → set. The owning score_scale row is locked FOR UPDATE BEFORE the band,
	// so a concurrent publish (which holds that scale FOR SHARE while it flips
	// status) and this edit serialize — a publish can no longer commit between
	// this guard's reference check and the band write (codex impl2 #1).
	var scaleID sql.NullString
	if err := exec.QueryRowContext(ctx, `SELECT score_scale_id FROM `+entityid.ScoreScaleBand+` WHERE id = $1`, bandID).Scan(&scaleID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("score scale band write guard: band not found — fail closed")
		}
		return fmt.Errorf("score scale band write guard: read band scale: %w", err)
	}
	if scaleID.Valid && scaleID.String != "" {
		var lockedScale string
		if err := exec.QueryRowContext(ctx, `SELECT id FROM `+entityid.ScoreScale+` WHERE id = $1 FOR UPDATE`, scaleID.String).Scan(&lockedScale); err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("score scale band write guard: lock scale FOR UPDATE: %w", err)
		}
	}
	var found string
	var lockedBandScale sql.NullString
	if err := exec.QueryRowContext(ctx, `SELECT id, score_scale_id FROM `+entityid.ScoreScaleBand+` WHERE id = $1 FOR UPDATE`, bandID).Scan(&found, &lockedBandScale); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("score scale band write guard: band not found — fail closed")
		}
		return fmt.Errorf("score scale band write guard: lock band FOR UPDATE: %w", err)
	}
	if lockedBandScale != scaleID {
		// The band moved scales between the unlocked read and the lock — the
		// scale lock above protects the wrong scale. Fail closed; retry.
		return fmt.Errorf("score scale band write guard: band scale changed concurrently — retry (fail closed)")
	}
	locked, err := existsLockedEntryForBand(ctx, exec, bandID)
	if err != nil {
		return err
	}
	if locked {
		return fmt.Errorf("BAND_LOCKED: this level is referenced by a published or deprecated rating description set and its meaning cannot change")
	}
	return nil
}

func (r *PostgresScoreScaleBandRepository) CreateScoreScaleBand(ctx context.Context, req *pb.CreateScoreScaleBandRequest) (*pb.CreateScoreScaleBandResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("score scale band data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateScoreScaleBandResponse{Data: []*pb.ScoreScaleBand{item}, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) ReadScoreScaleBand(ctx context.Context, req *pb.ReadScoreScaleBandRequest) (*pb.ReadScoreScaleBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale band ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadScoreScaleBandResponse{Data: []*pb.ScoreScaleBand{item}, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) UpdateScoreScaleBand(ctx context.Context, req *pb.UpdateScoreScaleBandRequest) (*pb.UpdateScoreScaleBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale band ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	if touchesLockedScoreScaleBandFields(data) {
		if err := r.guardScoreScaleBandLockedWrite(ctx, req.Data.Id); err != nil {
			return nil, err
		}
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateScoreScaleBandResponse{Data: []*pb.ScoreScaleBand{item}, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) DeleteScoreScaleBand(ctx context.Context, req *pb.DeleteScoreScaleBandRequest) (*pb.DeleteScoreScaleBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale band ID is required")
	}
	if err := r.guardScoreScaleBandLockedWrite(ctx, req.Data.Id); err != nil {
		return nil, err
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete score scale band: %w", err)
	}
	return &pb.DeleteScoreScaleBandResponse{Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) ListScoreScaleBands(ctx context.Context, req *pb.ListScoreScaleBandsRequest) (*pb.ListScoreScaleBandsResponse, error) {
	// fix3-backend (codex-review-impl3 #2): forward search/sort/pagination
	// too, so the rating-description pickers can page past the adapter's
	// 100-row default cap (a request without them is unchanged: nil params).
	items, err := r.listWithParams(ctx, req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	return &pb.ListScoreScaleBandsResponse{Data: items, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) GetScoreScaleBandListPageData(ctx context.Context, req *pb.GetScoreScaleBandListPageDataRequest) (*pb.GetScoreScaleBandListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateScoreScaleBand(all, req.GetPagination())
	_ = page
	return &pb.GetScoreScaleBandListPageDataResponse{ScoreScaleBandList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) GetScoreScaleBandItemPageData(ctx context.Context, req *pb.GetScoreScaleBandItemPageDataRequest) (*pb.GetScoreScaleBandItemPageDataResponse, error) {
	if req == nil || req.ScoreScaleBandId == "" {
		return nil, fmt.Errorf("score scale band ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ScoreScaleBandId)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetScoreScaleBandItemPageDataResponse{ScoreScaleBand: item, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ScoreScaleBand, error) {
	return r.listWithParams(ctx, nil, filters, nil, nil)
}

func (r *PostgresScoreScaleBandRepository) listWithParams(ctx context.Context, search *commonpb.SearchRequest, filters *commonpb.FilterRequest, sort *commonpb.SortRequest, pagination *commonpb.PaginationRequest) ([]*pb.ScoreScaleBand, error) {
	var params *interfaces.ListParams
	if search != nil || filters != nil || sort != nil || pagination != nil {
		params = &interfaces.ListParams{Search: search, Filters: filters, Sort: sort, Pagination: pagination}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list score scale bands: %w", err)
	}
	var items []*pb.ScoreScaleBand
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ScoreScaleBand{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func scoreScaleBandFromResult(result any) (*pb.ScoreScaleBand, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ScoreScaleBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateScoreScaleBand(all []*pb.ScoreScaleBand, p *commonpb.PaginationRequest) (int32, []*pb.ScoreScaleBand, *commonpb.PaginationResponse) {
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
