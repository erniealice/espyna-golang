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

type CreateJobCategoryRepositories struct {
	JobCategory pb.JobCategoryDomainServiceServer
}

type CreateJobCategoryServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateJobCategoryUseCase struct {
	repositories CreateJobCategoryRepositories
	services     CreateJobCategoryServices
}

func NewCreateJobCategoryUseCase(r CreateJobCategoryRepositories, s CreateJobCategoryServices) *CreateJobCategoryUseCase {
	return &CreateJobCategoryUseCase{repositories: r, services: s}
}

func (uc *CreateJobCategoryUseCase) Execute(ctx context.Context, req *pb.CreateJobCategoryRequest) (*pb.CreateJobCategoryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobCategory, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_category.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.JobCategory.CreateJobCategory(ctx, req)
}

func (uc *CreateJobCategoryUseCase) enrich(data *pb.JobCategory) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
