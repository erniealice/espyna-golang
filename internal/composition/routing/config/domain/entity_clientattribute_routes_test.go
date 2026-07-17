package domain

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity"
	adminUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/admin"
	clientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client"
	clientAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_attribute"
	userUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/user"
	workspaceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/workspace"
)

// TestConfigureEntityDomain_ClientAttributeReadSurfaceUnregistered proves the
// W3-HIGH-4 fix: the generic client_attribute READ surface (read / list /
// get-list-page-data / get-item-page-data) is NOT registered — those endpoints
// are a cross-tenant IDOR because client_attribute has no workspace_id column and
// cannot be parent-JOIN scoped at the generic adapter layer. The write endpoints
// remain. Before the fix all seven endpoints were registered; this test would
// have failed on the four read paths.
func TestConfigureEntityDomain_ClientAttributeReadSurfaceUnregistered(t *testing.T) {
	// The route builder early-returns "disabled" unless Admin/Client/User/Workspace
	// are non-nil; the handler closures are method VALUES (never invoked here), so
	// zero-value use-case wrappers are sufficient to exercise registration.
	uc := &entity.EntityUseCases{
		Admin:           &adminUseCases.UseCases{},
		Client:          &clientUseCases.UseCases{},
		User:            &userUseCases.UseCases{},
		Workspace:       &workspaceUseCases.UseCases{},
		ClientAttribute: &clientAttributeUseCases.UseCases{},
	}

	cfg := ConfigureEntityDomain(uc)
	if !cfg.Enabled {
		t.Fatalf("expected entity domain routes to be enabled, got disabled")
	}

	registered := make(map[string]bool)
	for _, r := range cfg.Routes {
		registered[r.Path] = true
	}

	// The read/list/page-data surface MUST be gone (IDOR closed).
	forbidden := []string{
		"/api/entity/client-attribute/read",
		"/api/entity/client-attribute/list",
		"/api/entity/client-attribute/get-list-page-data",
		"/api/entity/client-attribute/get-item-page-data",
	}
	for _, p := range forbidden {
		if registered[p] {
			t.Errorf("cross-tenant IDOR surface still registered: %s", p)
		}
	}

	// The write endpoints remain (not the disclosure surface this finding concerns).
	for _, p := range []string{
		"/api/entity/client-attribute/create",
		"/api/entity/client-attribute/update",
		"/api/entity/client-attribute/delete",
	} {
		if !registered[p] {
			t.Errorf("expected write endpoint to remain registered: %s", p)
		}
	}
}
