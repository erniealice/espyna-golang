//go:build postgresql

package workflow

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
)

func init() {
	registry.RegisterAssigneeQueryFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok || sqlDB == nil {
			return nil
		}
		return NewPostgresAssigneeQueryRepository(sqlDB)
	})
}

// defaultAssigneeQueryLimit is the max rows returned when the caller supplies
// zero or a negative limit.
const defaultAssigneeQueryLimit = 50

// PostgresAssigneeQueryRepository implements the bridge SQL for
// WorkflowAssigneeQueryService. It resolves pending engine activities
// assigned to a workspace user by JOINing:
//
//	activity.assigned_to  →  workspace_user.user_id  (THE BRIDGE)
//	stage → workflow → work_request                  (scope to requests domain)
//	work_request.workspace_id = $2                   (tenant isolation)
//
// No new column or migration — workspace_user.user_id (proto f5) is already
// indexed and the join chain uses only existing FK relationships.
type PostgresAssigneeQueryRepository struct {
	db *sql.DB
}

// NewPostgresAssigneeQueryRepository creates a new assignee query repository.
// Requires a *sql.DB for executing the bridge CTE (the generic dbOps
// interface cannot express this multi-table join shape).
func NewPostgresAssigneeQueryRepository(db *sql.DB) *PostgresAssigneeQueryRepository {
	return &PostgresAssigneeQueryRepository{db: db}
}

// ListPendingActivitiesForAssignee executes the identity bridge query.
//
// The SQL joins through workspace_user.user_id to resolve engine-level
// Activity.assigned_to (a global user.id) back to the session's
// workspace_user_id. The work_request join narrows to the requests domain
// (Q-EIB-SCOPE) and supplies workspace_id for tenant isolation (Q-EIB-SCOPE-WS).
//
// Fail-closed: empty WorkspaceUserID or WorkspaceID returns empty, no SQL.
// NULL assigned_to never leaks (explicit IS NOT NULL guard).
func (r *PostgresAssigneeQueryRepository) ListPendingActivitiesForAssignee(
	ctx context.Context,
	req *domain.ListPendingActivitiesForAssigneeRequest,
) (*domain.ListPendingActivitiesForAssigneeResponse, error) {
	// ── Fail-closed: empty identity ⇒ no SQL ──
	if req == nil || req.WorkspaceUserID == "" || req.WorkspaceID == "" {
		return &domain.ListPendingActivitiesForAssigneeResponse{
			Activities: make([]*activitypb.Activity, 0),
			Total:      0,
		}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultAssigneeQueryLimit
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// ── Count query (same join, no LIMIT/OFFSET) ──
	countQuery := `
		SELECT COUNT(*)
		FROM activity a
		JOIN stage st        ON a.stage_id    = st.id
		JOIN workflow wf     ON st.workflow_id = wf.id
		JOIN work_request wr ON wf.id         = wr.workflow_id
		JOIN workspace_user wu ON wu.id        = $1
		WHERE a.assigned_to    = wu.user_id
		  AND a.assigned_to IS NOT NULL
		  AND a.status NOT IN ('completed', 'skipped', 'cancelled')
		  AND wr.workspace_id  = $2
	`

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, req.WorkspaceUserID, req.WorkspaceID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count pending activities for assignee: %w", err)
	}

	if total == 0 {
		return &domain.ListPendingActivitiesForAssigneeResponse{
			Activities: make([]*activitypb.Activity, 0),
			Total:      0,
		}, nil
	}

	// ── Data query ──
	// Fixed ORDER BY allowlist (date_created DESC) — not caller-supplied.
	// The join chain: activity → stage → workflow → work_request → workspace_user
	// mirrors the plan's bridge SQL exactly.
	dataQuery := `
		SELECT
			a.id,
			a.stage_id,
			a.activity_template_id,
			a.name,
			a.description,
			a.status,
			a.priority,
			a.assigned_to,
			a.date_created,
			a.date_modified
		FROM activity a
		JOIN stage st        ON a.stage_id    = st.id
		JOIN workflow wf     ON st.workflow_id = wf.id
		JOIN work_request wr ON wf.id         = wr.workflow_id
		JOIN workspace_user wu ON wu.id        = $1
		WHERE a.assigned_to    = wu.user_id
		  AND a.assigned_to IS NOT NULL
		  AND a.status NOT IN ('completed', 'skipped', 'cancelled')
		  AND wr.workspace_id  = $2
		ORDER BY a.date_created DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, dataQuery, req.WorkspaceUserID, req.WorkspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending activities for assignee: %w", err)
	}
	defer rows.Close()

	activities := make([]*activitypb.Activity, 0, limit)
	for rows.Next() {
		var (
			id, stageID, activityTemplateID string
			name                            string
			description                     sql.NullString
			status, priority                string
			assignedTo                      sql.NullString
			dateCreated, dateModified       sql.NullInt64
		)

		if err := rows.Scan(
			&id,
			&stageID,
			&activityTemplateID,
			&name,
			&description,
			&status,
			&priority,
			&assignedTo,
			&dateCreated,
			&dateModified,
		); err != nil {
			return nil, fmt.Errorf("failed to scan activity row: %w", err)
		}

		activity := &activitypb.Activity{
			Id:                 id,
			StageId:            stageID,
			ActivityTemplateId: activityTemplateID,
			Name:               name,
			Status:             status,
			Priority:           priority,
		}

		if description.Valid {
			activity.Description = &description.String
		}
		if assignedTo.Valid {
			activity.AssignedTo = &assignedTo.String
		}
		if dateCreated.Valid {
			activity.DateCreated = &dateCreated.Int64
		}
		if dateModified.Valid {
			activity.DateModified = &dateModified.Int64
		}

		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activity rows: %w", err)
	}

	return &domain.ListPendingActivitiesForAssigneeResponse{
		Activities: activities,
		Total:      total,
	}, nil
}
