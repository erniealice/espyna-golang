package subscription_group_product_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	espynaports "github.com/erniealice/espyna-golang/ports"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

// UseCases aggregates the full CRUD+List set for subscription_group_product_plan
// (THE CLASS), mirroring the subscription_group_product_plan_staff UseCases
// shape (docs/plan/20260724-section-assignment-merged/espyna.md §1).
type UseCases struct {
	CreateSubscriptionGroupProductPlan *CreateSubscriptionGroupProductPlanUseCase
	ReadSubscriptionGroupProductPlan   *ReadSubscriptionGroupProductPlanUseCase
	UpdateSubscriptionGroupProductPlan *UpdateSubscriptionGroupProductPlanUseCase
	DeleteSubscriptionGroupProductPlan *DeleteSubscriptionGroupProductPlanUseCase
	ListSubscriptionGroupProductPlans  *ListSubscriptionGroupProductPlansUseCase
}

// Repositories groups every repository dependency across the class UC set,
// including the class-invariant guard anchors (cross-domain: product +
// operation).
type Repositories struct {
	SubscriptionGroupProductPlan pb.SubscriptionGroupProductPlanDomainServiceServer
	ProductPlan                  productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup            subscriptiongrouppb.SubscriptionGroupDomainServiceServer
	// JobTemplate is best-effort (see validation.go).
	JobTemplate jobtemplatepb.JobTemplateDomainServiceServer
}

// Services groups every service dependency, including the optional in-use
// guard checker (Update's exclude transition + Delete).
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
	ReferenceChecker espynaports.Checker
}

func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		CreateSubscriptionGroupProductPlan: NewCreateSubscriptionGroupProductPlanUseCase(CreateSubscriptionGroupProductPlanRepositories{
			SubscriptionGroupProductPlan: r.SubscriptionGroupProductPlan,
			ProductPlan:                  r.ProductPlan,
			SubscriptionGroup:            r.SubscriptionGroup,
			JobTemplate:                  r.JobTemplate,
		}, CreateSubscriptionGroupProductPlanServices{
			Authorizer:       s.Authorizer,
			Transactor:       s.Transactor,
			Translator:       s.Translator,
			IDGenerator:      s.IDGenerator,
			ActionGatekeeper: s.ActionGatekeeper,
		}),
		ReadSubscriptionGroupProductPlan: NewReadSubscriptionGroupProductPlanUseCase(ReadSubscriptionGroupProductPlanRepositories{
			SubscriptionGroupProductPlan: r.SubscriptionGroupProductPlan,
		}, ReadSubscriptionGroupProductPlanServices{
			Authorizer:       s.Authorizer,
			Transactor:       s.Transactor,
			Translator:       s.Translator,
			IDGenerator:      s.IDGenerator,
			ActionGatekeeper: s.ActionGatekeeper,
		}),
		UpdateSubscriptionGroupProductPlan: NewUpdateSubscriptionGroupProductPlanUseCase(UpdateSubscriptionGroupProductPlanRepositories{
			SubscriptionGroupProductPlan: r.SubscriptionGroupProductPlan,
			ProductPlan:                  r.ProductPlan,
			SubscriptionGroup:            r.SubscriptionGroup,
			JobTemplate:                  r.JobTemplate,
		}, UpdateSubscriptionGroupProductPlanServices{
			Authorizer:       s.Authorizer,
			Transactor:       s.Transactor,
			Translator:       s.Translator,
			IDGenerator:      s.IDGenerator,
			ActionGatekeeper: s.ActionGatekeeper,
			ReferenceChecker: s.ReferenceChecker,
		}),
		DeleteSubscriptionGroupProductPlan: NewDeleteSubscriptionGroupProductPlanUseCase(DeleteSubscriptionGroupProductPlanRepositories{
			SubscriptionGroupProductPlan: r.SubscriptionGroupProductPlan,
		}, DeleteSubscriptionGroupProductPlanServices{
			Authorizer:       s.Authorizer,
			Transactor:       s.Transactor,
			Translator:       s.Translator,
			IDGenerator:      s.IDGenerator,
			ActionGatekeeper: s.ActionGatekeeper,
			ReferenceChecker: s.ReferenceChecker,
		}),
		ListSubscriptionGroupProductPlans: NewListSubscriptionGroupProductPlansUseCase(ListSubscriptionGroupProductPlansRepositories{
			SubscriptionGroupProductPlan: r.SubscriptionGroupProductPlan,
		}, ListSubscriptionGroupProductPlansServices{
			Authorizer:       s.Authorizer,
			Transactor:       s.Transactor,
			Translator:       s.Translator,
			IDGenerator:      s.IDGenerator,
			ActionGatekeeper: s.ActionGatekeeper,
		}),
	}
}
