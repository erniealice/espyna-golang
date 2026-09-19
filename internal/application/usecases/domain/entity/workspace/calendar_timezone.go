package workspace

import (
	"context"
	"errors"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	"strings"
	"time"
)

// CalendarTimezone returns only the calendar setting of the current trusted
// workspace context. HTTP calls it after session/workspace binding validation.
// It intentionally does not require workspace:read: using business forms is not
// workspace administration. Callers cannot supply a different workspace ID.
func (uc *ReadWorkspaceUseCase) CalendarTimezone(ctx context.Context) (string, error) {
	id := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if id == "" {
		return "", nil
	}
	if uc.repositories.Workspace == nil {
		return "", errors.New("workspace timezone repository unavailable")
	}
	resp, err := uc.repositories.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{Data: &workspacepb.Workspace{Id: id}})
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.GetData()) != 1 || resp.GetData()[0].GetId() != id {
		return "", errors.New("workspace not found")
	}
	zone := strings.TrimSpace(resp.GetData()[0].GetTimezone())
	if zone == "" {
		zone = "UTC"
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return "", err
	}
	return zone, nil
}
