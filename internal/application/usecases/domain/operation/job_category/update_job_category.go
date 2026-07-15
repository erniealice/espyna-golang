package job_category

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
)

type UpdateJobCategoryRepositories struct {
	JobCategory pb.JobCategoryDomainServiceServer
}

type UpdateJobCategoryServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateJobCategoryUseCase struct {
	repositories UpdateJobCategoryRepositories
	services     UpdateJobCategoryServices
}

func NewUpdateJobCategoryUseCase(r UpdateJobCategoryRepositories, s UpdateJobCategoryServices) *UpdateJobCategoryUseCase {
	return &UpdateJobCategoryUseCase{repositories: r, services: s}
}

func (uc *UpdateJobCategoryUseCase) Execute(ctx context.Context, req *pb.UpdateJobCategoryRequest) (*pb.UpdateJobCategoryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobCategory, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_category.validation.data_required", "Data is required [DEFAULT]"))
	}
	// workspace_id is the immutable tenant anchor; never honor a client-supplied
	// workspace on update (gate H1).
	req.Data.WorkspaceId = nil
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.JobCategory.UpdateJobCategory(ctx, req)
}
