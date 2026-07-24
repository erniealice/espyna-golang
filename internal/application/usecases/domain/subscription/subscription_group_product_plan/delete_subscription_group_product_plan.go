package subscription_group_product_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	espynaports "github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

type DeleteSubscriptionGroupProductPlanRepositories struct {
	SubscriptionGroupProductPlan pb.SubscriptionGroupProductPlanDomainServiceServer
}

type DeleteSubscriptionGroupProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// ReferenceChecker is the fail-closed in-use guard (plan.md §2). See
	// in_use_guard.go — a nil checker degrades to un-guarded.
	ReferenceChecker espynaports.Checker
}

type DeleteSubscriptionGroupProductPlanUseCase struct {
	repositories DeleteSubscriptionGroupProductPlanRepositories
	services     DeleteSubscriptionGroupProductPlanServices
}

func NewDeleteSubscriptionGroupProductPlanUseCase(r DeleteSubscriptionGroupProductPlanRepositories, s DeleteSubscriptionGroupProductPlanServices) *DeleteSubscriptionGroupProductPlanUseCase {
	return &DeleteSubscriptionGroupProductPlanUseCase{repositories: r, services: s}
}

func (uc *DeleteSubscriptionGroupProductPlanUseCase) Execute(ctx context.Context, req *pb.DeleteSubscriptionGroupProductPlanRequest) (*pb.DeleteSubscriptionGroupProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlan, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.GetData() == nil || req.GetData().GetId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}

	// Referential guard: fail closed while active assignments (or, once wired,
	// live realized courses) reference the class (plan.md §2, section-delete-
	// guard precedent).
	if err := guardNotInUse(ctx, uc.services.ReferenceChecker, uc.services.Translator, req.GetData().GetId()); err != nil {
		return nil, err
	}

	return uc.repositories.SubscriptionGroupProductPlan.DeleteSubscriptionGroupProductPlan(ctx, req)
}
