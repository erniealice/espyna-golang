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

type ReadJobCategoryRepositories struct {
	JobCategory pb.JobCategoryDomainServiceServer
}

type ReadJobCategoryServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadJobCategoryUseCase struct {
	repositories ReadJobCategoryRepositories
	services     ReadJobCategoryServices
}

func NewReadJobCategoryUseCase(r ReadJobCategoryRepositories, s ReadJobCategoryServices) *ReadJobCategoryUseCase {
	return &ReadJobCategoryUseCase{repositories: r, services: s}
}

func (uc *ReadJobCategoryUseCase) Execute(ctx context.Context, req *pb.ReadJobCategoryRequest) (*pb.ReadJobCategoryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobCategory, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_category.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.JobCategory.ReadJobCategory(ctx, req)
}
