package audit

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
)

// UseCases aggregates every service-driven audit use case.
type UseCases struct {
	ListAuditEntries *ListAuditEntriesUseCase
}

// Repositories groups infrastructure dependencies. AuditService may be
// nil when no audit provider is registered — the use cases degrade
// gracefully in that case.
type Repositories struct {
	AuditService infraports.AuditService
}

// Services groups application services.
type Services struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// NewUseCases wires every audit service use case from shared
// dependencies.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		ListAuditEntries: NewListAuditEntriesUseCase(
			ListAuditEntriesRepositories{AuditService: repositories.AuditService},
			ListAuditEntriesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
