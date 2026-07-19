//go:build postgresql

package operation

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

func wsCtx(ws string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: ws})
}

// TestJobWorkspaceScope_BindsUnconditionally pins the FIX-4 contract for the three
// explicit projections (list/item/ListByJob): when a trusted workspace is present
// the parent-ancestry JOIN is ALWAYS emitted with the correct placeholder + bind —
// regardless of the global AUTHZ_ENFORCE (shadow/enforce) flag. jobWorkspaceScope
// never reads that flag: it depends only on identity, so the join is
// shadow-independent.
func TestJobWorkspaceScope_BindsUnconditionally(t *testing.T) {
	// Prove shadow-independence: neither of these env states may change the result.
	for _, enforce := range []string{"", "false", "true", "1"} {
		t.Run("AUTHZ_ENFORCE="+enforce, func(t *testing.T) {
			old, had := os.LookupEnv("AUTHZ_ENFORCE")
			if enforce == "" {
				os.Unsetenv("AUTHZ_ENFORCE")
			} else {
				os.Setenv("AUTHZ_ENFORCE", enforce)
			}
			defer func() {
				if had {
					os.Setenv("AUTHZ_ENFORCE", old)
				} else {
					os.Unsetenv("AUTHZ_ENFORCE")
				}
			}()

			join, arg, scoped := jobWorkspaceScope(wsCtx("ws-42"), 4)
			if !scoped {
				t.Fatal("a trusted workspace must always be scoped (shadow-independent)")
			}
			if arg != "ws-42" {
				t.Fatalf("bind arg got %q want ws-42", arg)
			}
			if !strings.Contains(join, "JOIN job j ON j.id = jp.job_id AND j.workspace_id = $4") {
				t.Fatalf("join fragment must bind job.workspace_id at $4, got %q", join)
			}
		})
	}
}

// TestJobWorkspaceScope_PlaceholderRespected proves each projection can request its
// own $N (item/ListByJob use $2, the paged list uses $4).
func TestJobWorkspaceScope_PlaceholderRespected(t *testing.T) {
	join, _, scoped := jobWorkspaceScope(wsCtx("ws-1"), 2)
	if !scoped || !strings.Contains(join, "j.workspace_id = $2") {
		t.Fatalf("expected $2 bind, got scoped=%v join=%q", scoped, join)
	}
}

// TestJobWorkspaceScope_NoWorkspacePassThrough proves a service-to-service / CLI
// context (no identity) is pass-through: no join, no bind — system callers are
// unaffected (mirrors the decorator's wsID=="" short-circuit).
func TestJobWorkspaceScope_NoWorkspacePassThrough(t *testing.T) {
	join, arg, scoped := jobWorkspaceScope(context.Background(), 4)
	if scoped || join != "" || arg != "" {
		t.Fatalf("no-identity context must be pass-through, got scoped=%v join=%q arg=%q", scoped, join, arg)
	}
	// Identity present but empty workspace → still pass-through.
	join2, _, scoped2 := jobWorkspaceScope(wsCtx(""), 4)
	if scoped2 || join2 != "" {
		t.Fatal("empty workspace must be pass-through")
	}
}

// TestFilterPhasesByWorkspace_PassThroughAndEmpty pins the generic-List FIX-4 seam's
// no-DB branches: a no-workspace context passes phases through unchanged, and an
// empty input returns empty without touching the DB.
func TestFilterPhasesByWorkspace_PassThroughAndEmpty(t *testing.T) {
	r := &PostgresJobPhaseRepository{} // no db needed for the pass-through/empty branches
	phases := []*pb.JobPhase{{Id: "p1", JobId: "j1"}, {Id: "p2", JobId: "j2"}}

	// No trusted workspace → pass-through unchanged (no probe).
	out, err := r.filterPhasesByWorkspace(context.Background(), phases)
	if err != nil {
		t.Fatalf("pass-through must not error, got %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("no-workspace context must pass all phases through, got %d", len(out))
	}

	// Empty input with a workspace → empty out, no DB call.
	out2, err := r.filterPhasesByWorkspace(wsCtx("ws-1"), nil)
	if err != nil {
		t.Fatalf("empty input must not error, got %v", err)
	}
	if len(out2) != 0 {
		t.Fatalf("empty input must return empty, got %d", len(out2))
	}
}
