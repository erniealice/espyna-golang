package job_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
)

type DeleteJobCategoryRepositories struct {
	JobCategory pb.JobCategoryDomainServiceServer
}

type DeleteJobCategoryServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteJobCategoryUseCase struct {
	repositories DeleteJobCategoryRepositories
	services     DeleteJobCategoryServices
}

func NewDeleteJobCategoryUseCase(r DeleteJobCategoryRepositories, s DeleteJobCategoryServices) *DeleteJobCategoryUseCase {
	return &DeleteJobCategoryUseCase{repositories: r, services: s}
}

func (uc *DeleteJobCategoryUseCase) Execute(ctx context.Context, req *pb.DeleteJobCategoryRequest) (*pb.DeleteJobCategoryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobCategory, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_category.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.JobCategory.DeleteJobCategory(ctx, req)
}
