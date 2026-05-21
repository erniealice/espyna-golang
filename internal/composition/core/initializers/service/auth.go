package service

import (
	"os"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	entityauth "github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	svcusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service"
	serviceauth "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// initServiceAuth wires the service-layer Auth sub-aggregate. Per
// Q-CR8 (Option B), it builds the entity-layer auth.UseCases internally
// from Session + User repos and wraps them in the proto-shaped service-
// layer use cases — collapsing what used to be two separate steps
// (composition/core/initializers/auth.go + composition/auth/wrapper.go).
//
// Returns a non-nil *serviceauth.UseCases even when entityRepos is nil —
// each wrapped use case carries a per-call nil-inner guard.
func initServiceAuth(
	entityRepos *domain.EntityRepositories,
	deps *svcusecases.Deps,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) *serviceauth.UseCases {
	// Step 1: build entity-layer auth use cases (orchestrates Session + User).
	var entityAuth *entityauth.UseCases
	if entityRepos != nil && entityRepos.Session != nil && entityRepos.User != nil {
		entityAuth = entityauth.NewUseCases(
			entityauth.Repositories{
				Session: entityRepos.Session,
				User:    entityRepos.User,
			},
			entityauth.Services{
				TransactionService: txSvc,
				TranslationService: i18nSvc,
				IDService:          idSvc,
				SessionExpiry:      sessionExpiryFromEnv(),
			},
		)
	}

	// Step 2: wrap entity-layer auth use cases in the service-layer proto
	// contract (nil-inner guards + per-call i18n).
	svc := serviceauth.Services{}
	if deps != nil {
		svc.TranslationService = deps.TranslationService
	}
	if entityAuth == nil {
		return &serviceauth.UseCases{
			AuthenticateSession: serviceauth.NewAuthenticateSessionUseCase(nil, svc),
			IssueSession:        serviceauth.NewIssueSessionUseCase(nil, svc),
			InvalidateSession:   serviceauth.NewInvalidateSessionUseCase(nil, svc),
		}
	}
	return &serviceauth.UseCases{
		AuthenticateSession: serviceauth.NewAuthenticateSessionUseCase(entityAuth.AuthenticateSession, svc),
		IssueSession:        serviceauth.NewIssueSessionUseCase(entityAuth.IssueSession, svc),
		InvalidateSession:   serviceauth.NewInvalidateSessionUseCase(entityAuth.InvalidateSession, svc),
	}
}

// sessionExpiryFromEnv reads PASSWORD_AUTH_SESSION_EXPIRY (Go duration
// format, e.g. "168h"). A missing or malformed value leaves Duration at
// zero, which asks IssueSession to fall back to its package default.
func sessionExpiryFromEnv() entityauth.SessionExpiryConfig {
	raw := os.Getenv("PASSWORD_AUTH_SESSION_EXPIRY")
	if raw == "" {
		return entityauth.SessionExpiryConfig{}
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return entityauth.SessionExpiryConfig{}
	}
	return entityauth.SessionExpiryConfig{Duration: d}
}
