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

type ListJobCategoriesRepositories struct {
	JobCategory pb.JobCategoryDomainServiceServer
}

type ListJobCategoriesServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListJobCategoriesUseCase struct {
	repositories ListJobCategoriesRepositories
	services     ListJobCategoriesServices
}

func NewListJobCategoriesUseCase(r ListJobCategoriesRepositories, s ListJobCategoriesServices) *ListJobCategoriesUseCase {
	return &ListJobCategoriesUseCase{repositories: r, services: s}
}

func (uc *ListJobCategoriesUseCase) Execute(ctx context.Context, req *pb.ListJobCategoriesRequest) (*pb.ListJobCategoriesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobCategory, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_category.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.JobCategory.ListJobCategories(ctx, req)
}
