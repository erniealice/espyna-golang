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

type GetJobCategoryListPageDataRepositories struct {
	JobCategory pb.JobCategoryDomainServiceServer
}

type GetJobCategoryListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetJobCategoryListPageDataUseCase struct {
	repositories GetJobCategoryListPageDataRepositories
	services     GetJobCategoryListPageDataServices
}

func NewGetJobCategoryListPageDataUseCase(r GetJobCategoryListPageDataRepositories, s GetJobCategoryListPageDataServices) *GetJobCategoryListPageDataUseCase {
	return &GetJobCategoryListPageDataUseCase{repositories: r, services: s}
}

func (uc *GetJobCategoryListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobCategoryListPageDataRequest) (*pb.GetJobCategoryListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobCategory, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_category.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.JobCategory.GetJobCategoryListPageData(ctx, req)
}
