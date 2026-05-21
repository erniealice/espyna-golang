package fulfillment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- ListFulfillments ----

type ListFulfillmentsRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type ListFulfillmentsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}
type ListFulfillmentsUseCase struct {
	repositories ListFulfillmentsRepositories
	services     ListFulfillmentsServices
}

// ---- GetFulfillmentListPageData ----

type GetFulfillmentListPageDataRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type GetFulfillmentListPageDataServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}
type GetFulfillmentListPageDataUseCase struct {
	repositories GetFulfillmentListPageDataRepositories
	services     GetFulfillmentListPageDataServices
}

func (uc *ListFulfillmentsUseCase) Execute(ctx context.Context, req *pb.ListFulfillmentsRequest) (*pb.ListFulfillmentsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator, "fulfillment", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.request_required", "request is required [DEFAULT]"))
	}
	result, err := uc.repositories.Fulfillment.ListFulfillments(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.errors.list_failed", "fulfillment listing failed [DEFAULT]"))
	}
	return result, nil
}

// ---- GetFulfillmentListPageData ----

func (uc *GetFulfillmentListPageDataUseCase) Execute(ctx context.Context, req *pb.GetFulfillmentListPageDataRequest) (*pb.GetFulfillmentListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator, "fulfillment", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.request_required", "request is required [DEFAULT]"))
	}
	return uc.repositories.Fulfillment.GetFulfillmentListPageData(ctx, req)
}
