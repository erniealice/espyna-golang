//go:build postgresql

package operation

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type updateCountingOps struct {
	interfaces.DatabaseOperation
	updates int
}

func (u *updateCountingOps) Update(context.Context, string, string, map[string]any) (map[string]any, error) {
	u.updates++
	return map[string]any{}, nil
}

func classCtx(staff string) context.Context {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: "u-" + staff, WorkspaceID: "ws", PrincipalType: principalscope.PrincipalTypeStaff, PrincipalID: staff,
	})
	return reviewedit.WithDecision(ctx, reviewedit.Decision{ClassOverride: true})
}

// A class override whose audit pre-read yields nothing must fail closed like the
// review edit: strict, refused, and no write reaches the database.
func TestClassOverride_NilPreReadIsStrictAndRefusesPlainUpdate(t *testing.T) {
	if !reviewEditStrict(classCtx("teacher"), nil) {
		t.Fatal("ClassOverride with a nil pre-read must be strict")
	}
	if reviewEditStrict(classCtx("teacher"), &pb.TaskOutcome{RecordedBy: "teacher"}) {
		t.Error("own row must stay non-strict")
	}
	if !reviewEditStrict(classCtx("teacher"), &pb.TaskOutcome{RecordedBy: "other"}) {
		t.Error("another recorder's row must be strict")
	}
	ops := &updateCountingOps{}
	repo := &PostgresTaskOutcomeRepository{dbOps: ops, tableName: "task_outcome"} // audit nil ⇒ pre-read is nil
	_, err := repo.UpdateTaskOutcome(classCtx("teacher"), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1"}})
	if err == nil || !strings.Contains(err.Error(), "fail closed") {
		t.Fatalf("want fail-closed refusal, got %v", err)
	}
	if ops.updates != 0 {
		t.Fatalf("no write may happen, got %d Update calls", ops.updates)
	}
}
