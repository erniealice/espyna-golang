package job_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jtphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jttaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
)

// Deprecated. As of 20260429
// (docs/plan/20260429-auto-spawn-jobs-from-subscription/), the canonical
// spawn use case is
// internal/application/usecases/subscription/subscription/MaterializeJobsForSubscription.
// This file is a thin compatibility shim retained while UI / centymo / fayna
// surfaces migrate to the new use case (plan §13 Phase D); the legacy
// ProductPlan.job_template_id field is no longer read.

// InstantiateJobsFromPlanRepositories is kept for legacy wiring symmetry.
type InstantiateJobsFromPlanRepositories struct {
	ProductPlan      productplanpb.ProductPlanDomainServiceServer
	JobTemplate      jobtemplatepb.JobTemplateDomainServiceServer
	JobTemplatePhase jtphasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask  jttaskpb.JobTemplateTaskDomainServiceServer
	Job              jobpb.JobDomainServiceServer
	JobPhase         jobphasepb.JobPhaseDomainServiceServer
	JobTask          jobtaskpb.JobTaskDomainServiceServer
}

// InstantiateJobsFromPlanServices is kept for legacy wiring symmetry.
type InstantiateJobsFromPlanServices struct {
	Transactor  ports.Transactor
	IDGenerator ports.IDGenerator
}

// InstantiateJobsFromPlanUseCase is a deprecated compatibility shim.
type InstantiateJobsFromPlanUseCase struct {
	repositories InstantiateJobsFromPlanRepositories
	services     InstantiateJobsFromPlanServices
}

// NewInstantiateJobsFromPlanUseCase returns a deprecated shim. The active
// use case lives at
// internal/application/usecases/subscription/subscription/MaterializeJobsForSubscriptionUseCase.
func NewInstantiateJobsFromPlanUseCase(
	repos InstantiateJobsFromPlanRepositories,
	services InstantiateJobsFromPlanServices,
) *InstantiateJobsFromPlanUseCase {
	return &InstantiateJobsFromPlanUseCase{repositories: repos, services: services}
}

// InstantiateJobsFromPlan is a no-op shim. Callers should migrate to
// MaterializeJobsForSubscriptionUseCase.Execute (subscription/subscription
// package).
func (uc *InstantiateJobsFromPlanUseCase) InstantiateJobsFromPlan(
	_ context.Context, _, _, _, _ string,
) error {
	return errors.New("instantiate_from_plan: deprecated; use MaterializeJobsForSubscriptionUseCase")
}
