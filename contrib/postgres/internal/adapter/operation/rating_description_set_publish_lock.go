//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// fix2-backend (codex-review-impl2 #1): publish side of the descriptor lock
// protocol. Global order (documented in score_scale.go "Descriptor lock
// protocol"): score_scale → score_scale_band → rating_description_set.
// Publish takes score_scale FOR SHARE here, BEFORE the use case locks the set
// row FOR UPDATE; scale/band meaning editors take score_scale FOR UPDATE
// first. Kept in its own file so the rating_description_set.go adapter (audit
// wiring, fix2-audit) is untouched.

var _ portsdomain.RatingDescriptionSetPublishLocker = (*PostgresRatingDescriptionSetRepository)(nil)

func (r *PostgresRatingDescriptionSetRepository) publishLockTx(ctx context.Context, op string) (string, sqlexec.DBExecutor, error) {
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return "", nil, fmt.Errorf("rating description set %s: no trusted workspace in context (fail closed)", op)
	}
	exec := r.executor(ctx)
	if exec == nil {
		return "", nil, fmt.Errorf("rating description set %s: no SQL executor available", op)
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return "", nil, fmt.Errorf("rating description set %s: requires an ambient transaction (fail closed)", op)
	}
	return idn.WorkspaceID, exec, nil
}

// LockScoreScaleForPublish reads the set's score_scale_id (unlocked — the set
// row is locked only AFTER the scale, per the lock order) and takes FOR SHARE
// on that score_scale row. The scale must be owned by the caller's workspace
// or global (workspace_id IS NULL — the score_scale convention already used by
// outcome_matrix_query.go); otherwise INVALID_CONFIG.
func (r *PostgresRatingDescriptionSetRepository) LockScoreScaleForPublish(ctx context.Context, setID string) (string, error) {
	if setID == "" {
		return "", fmt.Errorf("rating description set ID is required")
	}
	wsID, exec, err := r.publishLockTx(ctx, "publish lock")
	if err != nil {
		return "", err
	}
	var scaleID sql.NullString
	if err := exec.QueryRowContext(ctx,
		`SELECT score_scale_id FROM `+entityid.RatingDescriptionSet+` WHERE id = $1 AND workspace_id = $2`,
		setID, wsID).Scan(&scaleID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("rating description set publish lock: not found — fail closed")
		}
		return "", fmt.Errorf("rating description set publish lock: read scale: %w", err)
	}
	if !scaleID.Valid || scaleID.String == "" {
		return "", fmt.Errorf("INVALID_CONFIG: rating description set has no score scale")
	}
	var locked string
	if err := exec.QueryRowContext(ctx,
		`SELECT id FROM `+entityid.ScoreScale+`
		  WHERE id = $1 AND active AND (workspace_id = $2 OR workspace_id IS NULL)
		  FOR SHARE`,
		scaleID.String, wsID).Scan(&locked); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("INVALID_CONFIG: rating description set score scale is missing, inactive or owned by another workspace")
		}
		return "", fmt.Errorf("rating description set publish lock: lock scale FOR SHARE: %w", err)
	}
	return scaleID.String, nil
}

// VerifyRatingDescriptionSetPublishScale runs AFTER the set row is locked FOR
// UPDATE: the set must still declare the scale locked by
// LockScoreScaleForPublish (a DRAFT header edit could have changed it between
// the unlocked read and the set lock → fail closed, retry), and every active
// entry's band must be active-owned (caller workspace or global) and belong to
// that scale — so the FOR SHARE scale lock covers every band the published
// set will resolve through (codex impl2 #1 + #4).
func (r *PostgresRatingDescriptionSetRepository) VerifyRatingDescriptionSetPublishScale(ctx context.Context, setID, scoreScaleID string) error {
	wsID, exec, err := r.publishLockTx(ctx, "publish verify")
	if err != nil {
		return err
	}
	var current sql.NullString
	if err := exec.QueryRowContext(ctx,
		`SELECT score_scale_id FROM `+entityid.RatingDescriptionSet+` WHERE id = $1 AND workspace_id = $2`,
		setID, wsID).Scan(&current); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("rating description set publish verify: not found — fail closed")
		}
		return fmt.Errorf("rating description set publish verify: %w", err)
	}
	if !current.Valid || current.String != scoreScaleID {
		return fmt.Errorf("CONFLICT: rating description set score scale changed concurrently — retry")
	}
	// fix3-backend (codex-review-impl3 #6): every active entry's band must
	// also be ACTIVE and USABLE — i.e. exactly what the resolver
	// (outcome_matrix_rating_resolution_query.go resolveRatingEntries: JOIN
	// score_scale_band ON ... AND ssb.active) will read, and never a
	// band_role = 'no_description' level (the entry write guard's
	// BAND_NO_DESCRIPTION rule, re-checked here because the band's role could
	// have changed after the DRAFT entry was written). A DRAFT entry whose band
	// was deleted (soft: active = false) while the set was still DRAFT would
	// otherwise count toward the publication minimum yet supply no descriptor.
	// Race-free: the scale is held FOR SHARE (LockScoreScaleForPublish) and
	// every band write/delete takes that scale FOR UPDATE first
	// (guardScoreScaleBandLockedWrite), so no band can be deactivated between
	// this check and the status flip.
	var bad, inactive int
	if err := exec.QueryRowContext(ctx, `
		SELECT
		  count(*) FILTER (WHERE b.id IS NULL
		       OR b.score_scale_id IS DISTINCT FROM $2
		       OR NOT (b.workspace_id = $3 OR b.workspace_id IS NULL)),
		  count(*) FILTER (WHERE b.id IS NOT NULL
		       AND (NOT b.active OR b.band_role IS NOT DISTINCT FROM $4))
		FROM `+entityid.RatingDescriptionSetEntry+` e
		LEFT JOIN `+entityid.ScoreScaleBand+` b ON b.id = e.score_scale_band_id
		WHERE e.rating_description_set_id = $1 AND e.active`,
		setID, scoreScaleID, wsID, noDescriptionBandRole).Scan(&bad, &inactive); err != nil {
		return fmt.Errorf("rating description set publish verify: entry bands: %w", err)
	}
	if bad > 0 {
		return fmt.Errorf("INVALID_CONFIG: %d rating description entries reference a level outside the set's score scale or workspace", bad)
	}
	if inactive > 0 {
		return fmt.Errorf("INVALID_CONFIG: %d rating description entries reference a level that is inactive or cannot carry a description", inactive)
	}
	return nil
}

var _ portsdomain.RatingDescriptionSetScaleValidator = (*PostgresRatingDescriptionSetRepository)(nil)

// ValidateRatingDescriptionSetScoreScale (codex impl2 #4): the set's
// score_scale must be active and owned by the caller's workspace or global
// (workspace_id IS NULL); else INVALID_CONFIG. Runs on the ambient tx when
// present, otherwise the pool (a read-only ownership check).
func (r *PostgresRatingDescriptionSetRepository) ValidateRatingDescriptionSetScoreScale(ctx context.Context, scoreScaleID string) error {
	if scoreScaleID == "" {
		return fmt.Errorf("INVALID_CONFIG: score scale is required")
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return fmt.Errorf("rating description set scale validation: no trusted workspace in context (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return fmt.Errorf("rating description set scale validation: no SQL executor available")
	}
	var owned bool
	if err := exec.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM `+entityid.ScoreScale+`
		  WHERE id = $1 AND active AND (workspace_id = $2 OR workspace_id IS NULL))`,
		scoreScaleID, idn.WorkspaceID).Scan(&owned); err != nil {
		return fmt.Errorf("rating description set scale validation: %w", err)
	}
	if !owned {
		return fmt.Errorf("INVALID_CONFIG: score scale is missing, inactive or owned by another workspace")
	}
	return nil
}
