package initializers

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service"
	auditusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/audit"
	audithelpers "github.com/erniealice/espyna-golang/internal/composition/audit"
)

// InitializeService wires every service-driven use case sub-aggregate.
//
// Per Q7 (LOCKED), the `service/` namespace hosts cross-cutting concerns
// with proto contracts but no entity-driven CRUD (audit, reporting,
// auth, security). Phase 1 anchors this with the audit sub-aggregate;
// follow-up plans add reporting, auth, security.
//
// db may be nil when no SQL provider is in play; in that case the
// audit AuditService resolves to nil and the use cases degrade
// gracefully (return empty responses for ListByEntity).
func InitializeService(
	db *sql.DB,
	authSvc ports.AuthorizationService,
	i18nSvc ports.TranslationService,
) (*service.ServiceUseCases, error) {
	auditSvc := audithelpers.NewAuditServiceFromDB(db)

	auditUC := auditusecases.NewUseCases(
		auditusecases.Repositories{AuditService: auditSvc},
		auditusecases.Services{
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
		},
	)

	return service.NewServiceUseCases(auditUC), nil
}
