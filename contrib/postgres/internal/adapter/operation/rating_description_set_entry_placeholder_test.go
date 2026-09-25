//go:build postgresql

package operation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	"github.com/erniealice/espyna-golang/shared/placeholder"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

// Authoring guard (schema-proposal.md §10): entry create/update reject any
// grammar-matching tag that is not allowlisted — BEFORE any DB access, so a
// zero-value repository (no dbOps) proves the rejection is not DB-dependent.
func TestRatingDescriptionSetEntry_PlaceholderAuthoringGuard(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		reject bool
	}{
		{"plain text", "Rarely states the problem.", false},
		{"allowlisted tag", "{client.user.first_name} rarely states the problem.", false},
		{"non-grammar braces are literal", "Uses {braces} and {Client.Name} literally.", false},
		{"unknown tag", "{client.user.last_name} rarely states the problem.", true},
		{"allowed + unknown", "{client.user.first_name} and {job.name}", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRatingDescriptionPlaceholders(tc.text)
			if !tc.reject {
				if err != nil {
					t.Fatalf("unexpected rejection: %v", err)
				}
				return
			}
			if err == nil || !strings.HasPrefix(err.Error(), placeholder.CodeUnknownPlaceholder+":") || !errors.Is(err, placeholder.ErrUnknownPlaceholder) {
				t.Fatalf("want UNKNOWN_PLACEHOLDER, got %v", err)
			}

			repo := &PostgresRatingDescriptionSetEntryRepository{}
			ctx := context.Background()
			if _, err := repo.CreateRatingDescriptionSetEntry(ctx, &entrypb.CreateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
				RatingDescriptionSetId: "s", ScoreScaleBandId: "b", Description: tc.text,
			}}); !errors.Is(err, placeholder.ErrUnknownPlaceholder) {
				t.Fatalf("create: want UNKNOWN_PLACEHOLDER, got %v", err)
			}
			if _, err := repo.UpdateRatingDescriptionSetEntry(ctx, &entrypb.UpdateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
				Id: "e", Description: tc.text,
			}}); !errors.Is(err, placeholder.ErrUnknownPlaceholder) {
				t.Fatalf("update: want UNKNOWN_PLACEHOLDER, got %v", err)
			}
		})
	}
}

// Integration (rolled back): on a DRAFT set, an allowlisted tag is accepted
// on create and update, an unknown tag is rejected on both and the stored
// text is unchanged.
func TestIntegration_RatingDescriptionSetEntry_PlaceholderAuthoringGuard(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	entryRepo := NewPostgresRatingDescriptionSetEntryRepository(wsOps, "rating_description_set_entry").(*PostgresRatingDescriptionSetEntryRepository)

	const ws = "rdph-guard-ws"
	const scale = "rdph-guard-scale"
	const parent = "rdph-guard-parent"
	const bandID = "rdph-guard-band"
	const criterionID = "rdph-guard-criterion"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, parent, ws, scale, ratingDescriptionSetVersionStatusDraft)
		if _, err := ex.ExecContext(txCtx, `INSERT INTO score_scale_band (id, workspace_id, score_scale_id, sequence_order, output_label) VALUES ($1, $2, $3, 1, 'Level 1')`, bandID, ws, scale); err != nil {
			return fmt.Errorf("seed band: %w", err)
		}
		if _, err := ex.ExecContext(txCtx, `INSERT INTO outcome_criteria (id, workspace_id, name, active) VALUES ($1, $2, 'RDPH Criterion', true)`, criterionID, ws); err != nil {
			return fmt.Errorf("seed criterion: %w", err)
		}
		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "rdph-user", WorkspaceID: ws})

		if _, err := entryRepo.CreateRatingDescriptionSetEntry(ctx, &entrypb.CreateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
			WorkspaceId: ws, RatingDescriptionSetId: parent, OutcomeCriteriaId: criterionID, ScoreScaleBandId: bandID,
			Description: "{client.user.last_name} states the problem.", Active: true,
		}}); !errors.Is(err, placeholder.ErrUnknownPlaceholder) {
			return fmt.Errorf("create with unknown tag: want UNKNOWN_PLACEHOLDER, got %v", err)
		}

		resp, err := entryRepo.CreateRatingDescriptionSetEntry(ctx, &entrypb.CreateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
			WorkspaceId: ws, RatingDescriptionSetId: parent, OutcomeCriteriaId: criterionID, ScoreScaleBandId: bandID,
			Description: "{client.user.first_name} states the problem.", Active: true,
		}})
		if err != nil || len(resp.GetData()) != 1 {
			return fmt.Errorf("create with allowlisted tag: want success, got %v", err)
		}
		entryID := resp.GetData()[0].GetId()

		if _, err := entryRepo.UpdateRatingDescriptionSetEntry(ctx, &entrypb.UpdateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
			Id: entryID, Description: "{job.name} states the problem.",
		}}); !errors.Is(err, placeholder.ErrUnknownPlaceholder) {
			return fmt.Errorf("update with unknown tag: want UNKNOWN_PLACEHOLDER, got %v", err)
		}
		var stored string
		if err := ex.QueryRowContext(txCtx, `SELECT description FROM rating_description_set_entry WHERE id = $1`, entryID).Scan(&stored); err != nil {
			return fmt.Errorf("read back: %w", err)
		}
		if stored != "{client.user.first_name} states the problem." {
			return fmt.Errorf("rejected update changed stored text to %q", stored)
		}
		if _, err := entryRepo.UpdateRatingDescriptionSetEntry(ctx, &entrypb.UpdateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
			Id: entryID, Description: "{client.user.first_name} clearly states the problem.",
		}}); err != nil {
			return fmt.Errorf("update with allowlisted tag: want success, got %w", err)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}
