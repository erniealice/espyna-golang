package disbursement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// ListDisbursementsRepositories groups all repository dependencies
type ListDisbursementsRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// ListDisbursementsServices groups all business service dependencies
type ListDisbursementsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListDisbursementsUseCase handles the business logic for listing disbursements
type ListDisbursementsUseCase struct {
	repositories ListDisbursementsRepositories
	services     ListDisbursementsServices
}

// NewListDisbursementsUseCase creates a new ListDisbursementsUseCase
func NewListDisbursementsUseCase(
	repositories ListDisbursementsRepositories,
	services ListDisbursementsServices,
) *ListDisbursementsUseCase {
	return &ListDisbursementsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list disbursements operation
func (uc *ListDisbursementsUseCase) Execute(ctx context.Context, req *disbursementpb.ListDisbursementsRequest) (*disbursementpb.ListDisbursementsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityDisbursement,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Disbursement.ListDisbursements(ctx, req)
}
