package initializers

import (
	"os"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeAuth wires identity-lifecycle use cases from the already-
// constructed EntityRepositories. Session and User proto repos are reused —
// we do NOT open a second database path here.
//
// Note: auth flows establish/terminate identity, so no AuthorizationService
// is threaded in. See usecases/auth/usecases.go for the invariant.
func InitializeAuth(
	entityRepos *domain.EntityRepositories,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*auth.UseCases, error) {
	if entityRepos == nil || entityRepos.Session == nil || entityRepos.User == nil {
		// Without Session + User repos, auth orchestration is not viable.
		// Return an empty struct for graceful degradation.
		return &auth.UseCases{}, nil
	}

	return auth.NewUseCases(
		auth.Repositories{
			Session: entityRepos.Session,
			User:    entityRepos.User,
		},
		auth.Services{
			TransactionService: txSvc,
			TranslationService: i18nSvc,
			IDService:          idSvc,
			SessionExpiry:      sessionExpiryFromEnv(),
		},
	), nil
}

// sessionExpiryFromEnv reads PASSWORD_AUTH_SESSION_EXPIRY (Go duration
// format, e.g. "168h"). A missing or malformed value leaves Duration at
// zero, which asks IssueSession to fall back to its package default.
func sessionExpiryFromEnv() auth.SessionExpiryConfig {
	raw := os.Getenv("PASSWORD_AUTH_SESSION_EXPIRY")
	if raw == "" {
		return auth.SessionExpiryConfig{}
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return auth.SessionExpiryConfig{}
	}
	return auth.SessionExpiryConfig{Duration: d}
}
