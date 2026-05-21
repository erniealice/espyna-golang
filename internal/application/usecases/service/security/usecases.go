package security

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
)

// UseCases aggregates every service-driven security use case.
type UseCases struct {
	GetUserPermissionCodes *GetUserPermissionCodesUseCase
}

// Repositories groups infrastructure dependencies. PermissionQuery may be
// nil when no RBAC provider is registered — the use cases degrade
// gracefully in that case (return empty permission sets).
type Repositories struct {
	PermissionQuery securityports.PermissionQuery
}

// Services groups application services.
type Services struct {
	Translator ports.Translator
}

// NewUseCases wires every security service use case from shared
// dependencies.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		GetUserPermissionCodes: NewGetUserPermissionCodesUseCase(
			GetUserPermissionCodesRepositories{PermissionQuery: repositories.PermissionQuery},
			GetUserPermissionCodesServices{Translator: services.Translator},
		),
	}
}
