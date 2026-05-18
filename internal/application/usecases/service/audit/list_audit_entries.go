// Package audit hosts the service-driven Audit use cases.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// worked example) Audit is a *service-driven* domain — cross-cutting,
// append-only, no entity-driven CRUD. Its proto contract lives at
// `proto/v1/service/audit/audit_query.proto`. This use case is the read
// surface (ListByEntity) consumed by the audit-trail History tab.
package audit

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	auditquerypb "github.com/erniealice/esqyma/pkg/schema/v1/service/audit"
)

// ListAuditEntriesRepositories groups the infrastructure dependencies of
// the use case. AuditService is the audit infrastructure port — there are
// no proto domain repositories because audit_trail is not an entity-driven
// domain (per Q7).
type ListAuditEntriesRepositories struct {
	AuditService infraports.AuditService
}

// ListAuditEntriesServices groups application services. No
// TransactionService — the use case is read-only.
type ListAuditEntriesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListAuditEntriesUseCase resolves a paginated audit entry list for an
// entity, gated by ActionList authorization against the "audit_trail"
// resource.
type ListAuditEntriesUseCase struct {
	repositories ListAuditEntriesRepositories
	services     ListAuditEntriesServices
}

// NewListAuditEntriesUseCase wires the use case from grouped dependencies.
func NewListAuditEntriesUseCase(
	repositories ListAuditEntriesRepositories,
	services ListAuditEntriesServices,
) *ListAuditEntriesUseCase {
	return &ListAuditEntriesUseCase{repositories: repositories, services: services}
}

// Execute performs the ListByEntity audit query.
//
// Translation flow: the proto-shaped request is rewritten into the
// infrastructure port's request struct, then the port response is
// rewritten back into the proto-shaped response. The two shapes are kept
// in sync manually (matches the old consumer/adapter_audit.go pattern).
func (uc *ListAuditEntriesUseCase) Execute(
	ctx context.Context,
	req *auditquerypb.ListAuditEntriesRequest,
) (*auditquerypb.ListAuditEntriesResponse, error) {
	// Authorization — "audit_trail" + ActionList.
	if err := authcheck.Check(
		ctx,
		uc.services.AuthorizationService,
		uc.services.TranslationService,
		"audit_trail",
		ports.ActionList,
	); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"audit.validation.request_required", "request is required"))
	}
	if uc.repositories.AuditService == nil {
		// No audit provider — return empty response rather than error so
		// the History tab degrades gracefully on non-postgres builds.
		return &auditquerypb.ListAuditEntriesResponse{}, nil
	}

	innerReq := &infraports.ListAuditRequest{
		WorkspaceID: req.GetWorkspaceId(),
		EntityType:  req.GetEntityType(),
		EntityID:    req.GetEntityId(),
		Limit:       int(req.GetLimit()),
		CursorToken: req.GetCursorToken(),
	}
	resp, err := uc.repositories.AuditService.ListByEntity(ctx, innerReq)
	if err != nil {
		return nil, fmt.Errorf(
			contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.TranslationService,
				"audit.errors.list_failed", "failed to list audit entries: %w"),
			err,
		)
	}
	if resp == nil {
		return &auditquerypb.ListAuditEntriesResponse{}, nil
	}

	out := &auditquerypb.ListAuditEntriesResponse{
		HasNext:    resp.HasNext,
		NextCursor: resp.NextCursor,
		Entries:    make([]*auditquerypb.AuditEntry, len(resp.Entries)),
	}
	for i, e := range resp.Entries {
		changes := make([]*auditquerypb.AuditFieldChange, len(e.FieldChanges))
		for j, fc := range e.FieldChanges {
			changes[j] = &auditquerypb.AuditFieldChange{
				FieldName: fc.FieldName,
				FieldType: fc.FieldType,
				OldValue:  fc.OldValue,
				NewValue:  fc.NewValue,
			}
		}
		out.Entries[i] = &auditquerypb.AuditEntry{
			Id:             e.ID,
			ActorId:        e.ActorID,
			ActorType:      e.ActorType,
			Action:         e.Action,
			PermissionCode: e.PermissionCode,
			UseCase:        e.UseCase,
			FieldCount:     e.FieldCount,
			OccurredAt:     e.OccurredAt,
			FieldChanges:   changes,
		}
	}
	return out, nil
}
