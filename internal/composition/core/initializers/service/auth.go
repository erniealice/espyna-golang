package service

import (
	"os"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	serviceauth "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// initServiceAuth wires the service-driven Auth sub-aggregate. The package
// is single-layer: it consumes the Session + User proto repositories
// directly and exposes proto-shaped use cases (no entity-layer indirection).
// See docs/plan/20260524-usecases-auth-collapse/ for the rationale and
// docs/wiki/articles/proto-categories.md §2 for the architectural placement.
//
// Always returns a non-nil *serviceauth.UseCases. When entityRepos is nil
// (or its Session/User fields are nil) each use case's Execute fails closed
// with the `auth.errors.service_unavailable` translator key — the same UX
// the prior wrapper layer guaranteed.
func initServiceAuth(
	entityRepos *domain.EntityRepositories,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
) *serviceauth.UseCases {
	var repos serviceauth.Repositories
	if entityRepos != nil {
		repos.Session = entityRepos.Session
		repos.User = entityRepos.User
		// SessionSwitch is a narrow extension interface satisfied only by
		// concrete backends that implement SwitchPrincipal (Phase 2 postgres
		// adapter). Under mock-db / non-postgres builds the assertion fails
		// and SessionSwitch stays nil — the use case's body-entry nil-guard
		// returns auth.errors.service_unavailable. See
		// docs/plan/20260524-principal-switch-typed-stack/ §Phase 4.
		if adapter, ok := entityRepos.Session.(serviceauth.SessionSwitchAdapter); ok {
			repos.SessionSwitch = adapter
		}
		// PrincipalResolver is a narrow extension interface satisfied only by
		// concrete backends that implement ResolvePrincipals,
		// EnumerateBindingsInWorkspace, and LookupSessionPrincipal. Under
		// mock-db / non-postgres builds the assertion fails and
		// PrincipalResolver stays nil — the use case's body-entry nil-guard
		// returns auth.errors.service_unavailable.
		if adapter, ok := entityRepos.Session.(serviceauth.PrincipalResolverAdapter); ok {
			repos.PrincipalResolver = adapter
		}
	}
	services := serviceauth.Services{
		Transactor:    txSvc,
		Translator:    i18nSvc,
		IDGenerator:   idSvc,
		SessionExpiry: sessionExpiryFromEnv(),
	}
	return serviceauth.NewUseCases(repos, services)
}

// sessionExpiryFromEnv reads AUTH_PASSWORD_SESSION_EXPIRY (Go duration
// format, e.g. "168h"). A missing or malformed value leaves Duration at
// zero, which asks IssueSession to fall back to its package default.
func sessionExpiryFromEnv() serviceauth.SessionExpiryConfig {
	raw := os.Getenv("AUTH_PASSWORD_SESSION_EXPIRY")
	if raw == "" {
		return serviceauth.SessionExpiryConfig{}
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return serviceauth.SessionExpiryConfig{}
	}
	return serviceauth.SessionExpiryConfig{Duration: d}
}
