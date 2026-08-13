//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	adaptercore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/shared/identity"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

type outcomeLandingSQL struct {
	statement   string
	args        []any
	staffScoped bool
}

// ListSubscriptionGroupOutcomeLanding implements the generated transport
// contract with the narrow scope. Only the application use case may derive and
// pass a workspace-wide scope through the in-process method below.
func (q *PostgresSubscriptionGroupOutcomeExportQuery) ListSubscriptionGroupOutcomeLanding(
	ctx context.Context,
	req *exportpb.ListSubscriptionGroupOutcomeLandingRequest,
) (*exportpb.ListSubscriptionGroupOutcomeLandingResponse, error) {
	return q.ListSubscriptionGroupOutcomeLandingScoped(ctx, req, ports.SubscriptionGroupOutcomeExportScope{})
}

// ListSubscriptionGroupOutcomeLandingScoped returns the report landing rows in
// one bounded statement. Workspace and active-binding identity come only from
// the session; request data can only select the optional schedule predicate.
func (q *PostgresSubscriptionGroupOutcomeExportQuery) ListSubscriptionGroupOutcomeLandingScoped(
	ctx context.Context,
	req *exportpb.ListSubscriptionGroupOutcomeLandingRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.ListSubscriptionGroupOutcomeLandingResponse, error) {
	empty := func() *exportpb.ListSubscriptionGroupOutcomeLandingResponse {
		return &exportpb.ListSubscriptionGroupOutcomeLandingResponse{Success: true, Rows: []*exportpb.SubscriptionGroupOutcomeLandingRow{}}
	}
	if req == nil {
		return nil, fmt.Errorf("subscription group outcome landing request is required")
	}
	if q == nil || q.db == nil {
		return nil, fmt.Errorf("subscription group outcome landing requires PostgreSQL")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || strings.TrimSpace(id.WorkspaceID) == "" {
		return empty(), nil
	}

	built := buildSubscriptionGroupOutcomeLandingSQL(ctx, id, req, scope)
	rows, err := adaptercore.ExecutorFromContext(ctx, q.db).QueryContext(ctx, built.statement, built.args...)
	if err != nil {
		return nil, fmt.Errorf("subscription group outcome landing query: %w", err)
	}
	defer rows.Close()

	response := empty()
	for rows.Next() {
		var row exportpb.SubscriptionGroupOutcomeLandingRow
		var sortOrder sql.NullInt64
		if err := rows.Scan(
			&row.PriceScheduleId, &row.PriceScheduleName, &row.PriceScheduleActive,
			&sortOrder, &row.SubscriptionGroupId, &row.SubscriptionGroupName,
			&row.SubscriptionGroupActive, &row.MemberCount, &row.JobTemplateCount,
		); err != nil {
			return nil, fmt.Errorf("subscription group outcome landing scan: %w", err)
		}
		if sortOrder.Valid {
			value := int32(sortOrder.Int64)
			row.PriceScheduleSortOrder = &value
		}
		response.Rows = append(response.Rows, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("subscription group outcome landing rows: %w", err)
	}
	return response, nil
}

func buildSubscriptionGroupOutcomeLandingSQL(
	ctx context.Context,
	id *identity.RequestIdentity,
	req *exportpb.ListSubscriptionGroupOutcomeLandingRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) outcomeLandingSQL {
	active := any(nil)
	if req != nil && req.PriceScheduleActive != nil {
		active = *req.PriceScheduleActive
	}

	staffScoped := false
	staffClause := ""
	var staffArgs []any
	if !scope.WorkspaceWide {
		if _, applies := principalscope.StaffRowScope(ctx); applies {
			staffScoped = true
			staffClause, staffArgs = principalscope.StaffReachableJobClause(ctx, "j", 6)
		}
	}

	args := []any{id.WorkspaceID, active, id.WorkspaceUserID, id.UserID, scope.WorkspaceWide}
	args = append(args, staffArgs...)
	return outcomeLandingSQL{
		statement:   renderOutcomeExportTables(outcomeLandingStatement(staffClause)),
		args:        args,
		staffScoped: staffScoped,
	}
}

func outcomeLandingStatement(staffClause string) string {
	return `
WITH visible_groups AS (
  SELECT ps.id AS price_schedule_id,
         COALESCE(ps.name, '') AS price_schedule_name,
         ps.active AS price_schedule_active,
         ps.sort_order AS price_schedule_sort_order,
         sg.id AS subscription_group_id,
         COALESCE(sg.name, '') AS subscription_group_name,
         sg.active AS subscription_group_active
    FROM {{subscription_group}} sg
    JOIN {{price_schedule}} ps
      ON ps.id = sg.price_schedule_id
     AND ps.workspace_id = $1
   WHERE sg.workspace_id = $1
     AND ($2::boolean IS NULL OR ps.active = $2::boolean)
     AND (
       $5::boolean
       OR EXISTS (
         SELECT 1
           FROM {{workspace_user}} wu
           JOIN {{subscription_group_workspace_user}} sgwu
             ON sgwu.workspace_user_id = wu.id
            AND sgwu.workspace_id = $1
            AND sgwu.subscription_group_id = sg.id
            AND sgwu.active = true
          WHERE wu.id = $3
            AND wu.user_id = $4
            AND wu.workspace_id = $1
            AND wu.active = true
       )
     )
), eligible_members AS (
  SELECT g.subscription_group_id, m.id AS member_id, m.subscription_id,
         m.client_id, g.subscription_group_active
    FROM visible_groups g
    JOIN {{subscription_group_member}} m
      ON m.subscription_group_id = g.subscription_group_id
     AND m.workspace_id = $1
    JOIN {{subscription}} s
      ON s.id = m.subscription_id
     AND s.workspace_id = $1
     AND s.client_id = m.client_id
   WHERE g.subscription_group_active = false
      OR (m.active = true AND s.active = true)
), eligible_jobs AS (
  SELECT em.subscription_group_id, j.job_template_id
    FROM eligible_members em
    JOIN {{job}} j
      ON j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'
     AND j.origin_id = em.subscription_id
     AND j.client_id = em.client_id
     AND j.workspace_id = $1
    LEFT JOIN {{job_template}} jt
      ON jt.id = j.job_template_id
     AND jt.workspace_id = $1
   WHERE (
       (em.subscription_group_active = true
        AND j.active = true
        AND jt.id IS NOT NULL
        AND jt.active = true)
       OR
       (em.subscription_group_active = false
        AND (jt.id IS NOT NULL OR NOT EXISTS (
          SELECT 1
            FROM {{job_template}} historical_jt
           WHERE historical_jt.id = j.job_template_id
             AND historical_jt.workspace_id = $1
        )))
     )` + staffClause + `
)
SELECT g.price_schedule_id,
       g.price_schedule_name,
       g.price_schedule_active,
       g.price_schedule_sort_order,
       g.subscription_group_id,
       g.subscription_group_name,
       g.subscription_group_active,
       COUNT(DISTINCT em.member_id) AS member_count,
       COUNT(DISTINCT ej.job_template_id) AS job_template_count
  FROM visible_groups g
  LEFT JOIN eligible_members em
    ON em.subscription_group_id = g.subscription_group_id
  LEFT JOIN eligible_jobs ej
    ON ej.subscription_group_id = g.subscription_group_id
 GROUP BY g.price_schedule_id, g.price_schedule_name,
          g.price_schedule_active, g.price_schedule_sort_order,
          g.subscription_group_id, g.subscription_group_name,
          g.subscription_group_active
 ORDER BY g.price_schedule_sort_order NULLS LAST,
          lower(btrim(g.price_schedule_name)), g.price_schedule_id,
          lower(btrim(g.subscription_group_name)), g.subscription_group_id
`
}
