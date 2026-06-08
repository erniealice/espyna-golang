package evaluation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// CreateEvaluationRepositories groups all repository dependencies.
type CreateEvaluationRepositories struct {
	Evaluation       evaluationpb.EvaluationDomainServiceServer
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer // anchor-ownership validation
}

// CreateEvaluationServices groups all business service dependencies.
type CreateEvaluationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateEvaluationUseCase creates a DRAFT evaluation.
//
// IDOR gate (Q-EVAL-IDOR-1, enforced in THIS use-case predicate, not the view):
//   - reject nil/empty acting_as_client_id with deny BEFORE any SQL
//   - stamp client_id from acting_as_client_id (NEVER from the form) and assert
//     equality
//   - for relationship_type = CLIENT_TO_ASSOCIATE REQUIRE subscription_seat_id AND
//     assert its client_id == acting_as_client_id (anchor ownership; pairs with
//     the DB anti-phantom CHECK)
type CreateEvaluationUseCase struct {
	repositories CreateEvaluationRepositories
	services     CreateEvaluationServices
}

func NewCreateEvaluationUseCase(repositories CreateEvaluationRepositories, services CreateEvaluationServices) *CreateEvaluationUseCase {
	return &CreateEvaluationUseCase{repositories: repositories, services: services}
}

func (uc *CreateEvaluationUseCase) Execute(ctx context.Context, req *evaluationpb.CreateEvaluationRequest) (*evaluationpb.CreateEvaluationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Evaluation, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.data_required", "Evaluation data is required [DEFAULT]"))
	}

	// IDOR gate 1 (fail-closed): a client-facing create REQUIRES an acting-as
	// client scope; deny before any SQL.
	actingClient := contextutil.GetActingAsClientIDFromContext(ctx)
	if actingClient == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.no_acting_client", "An acting client scope is required to create an evaluation [DEFAULT]"))
	}

	// IDOR gate 2: stamp client_id from the session scope, never the form.
	req.Data.ClientId = actingClient

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// IDOR gate 3 (anchor ownership): for CLIENT_TO_ASSOCIATE the seat is required
	// and its client_id must equal the acting client.
	if req.Data.RelationshipType == evaluationpb.RelationshipType_RELATIONSHIP_TYPE_CLIENT_TO_ASSOCIATE {
		if req.Data.SubscriptionSeatId == nil || *req.Data.SubscriptionSeatId == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.seat_required", "A subscription seat anchor is required for this evaluation [DEFAULT]"))
		}
		if err := uc.assertSeatOwnership(ctx, *req.Data.SubscriptionSeatId, actingClient); err != nil {
			return nil, err
		}
	}

	uc.enrich(req.Data)

	resp, err := uc.repositories.Evaluation.CreateEvaluation(ctx, req)
	if err != nil {
		translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.creation_failed", "Evaluation creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translated, err)
	}
	return resp, nil
}

func (uc *CreateEvaluationUseCase) validateBusinessRules(ctx context.Context, e *evaluationpb.Evaluation) error {
	if e.EvaluationType == evaluationpb.EvaluationType_EVALUATION_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.evaluation_type_required", "Evaluation type is required [DEFAULT]"))
	}
	if e.RelationshipType == evaluationpb.RelationshipType_RELATIONSHIP_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.relationship_type_required", "Relationship type is required [DEFAULT]"))
	}
	if e.SubjectType == evaluationpb.SubjectType_SUBJECT_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.subject_type_required", "Subject type is required [DEFAULT]"))
	}
	if e.EvaluatorType == evaluationpb.EvaluatorType_EVALUATOR_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.evaluator_type_required", "Evaluator type is required [DEFAULT]"))
	}
	if e.PeriodStart == "" || e.PeriodEnd == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.period_required", "Evaluation period start and end are required [DEFAULT]"))
	}
	// Template required for PERFORMANCE_REVIEW (pairs with the DB CHECK).
	if e.EvaluationType == evaluationpb.EvaluationType_EVALUATION_TYPE_PERFORMANCE_REVIEW &&
		(e.EvaluationTemplateId == nil || *e.EvaluationTemplateId == "") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.template_required", "A template is required for a performance review [DEFAULT]"))
	}
	return nil
}

// assertSeatOwnership reads the seat and verifies its client_id equals the acting
// client — closing the anchor-ownership IDOR hole at the use-case layer.
func (uc *CreateEvaluationUseCase) assertSeatOwnership(ctx context.Context, seatID, actingClient string) error {
	if uc.repositories.SubscriptionSeat == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.seat_validation_unavailable", "Seat ownership cannot be validated [DEFAULT]"))
	}
	resp, err := uc.repositories.SubscriptionSeat.ReadSubscriptionSeat(ctx, &subscriptionseatpb.ReadSubscriptionSeatRequest{
		Data: &subscriptionseatpb.SubscriptionSeat{Id: seatID},
	})
	if err != nil {
		return err
	}
	if resp == nil || len(resp.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.seat_not_found", "Subscription seat not found [DEFAULT]"))
	}
	if resp.Data[0].ClientId != actingClient {
		// Fail-closed: the seat belongs to a different client — IDOR rejection.
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.seat_not_owned", "The seat anchor is not owned by the acting client [DEFAULT]"))
	}
	return nil
}

func (uc *CreateEvaluationUseCase) enrich(e *evaluationpb.Evaluation) {
	now := time.Now()
	if e.Id == "" {
		e.Id = uc.services.IDGenerator.GenerateID()
	}
	// New evaluations start as DRAFT (never NULL — entity-status-conventions).
	if e.Status == evaluationpb.EvaluationStatus_EVALUATION_STATUS_UNSPECIFIED {
		e.Status = evaluationpb.EvaluationStatus_EVALUATION_STATUS_DRAFT
	}
	if e.VisibilityType == evaluationpb.VisibilityType_VISIBILITY_TYPE_UNSPECIFIED {
		e.VisibilityType = evaluationpb.VisibilityType_VISIBILITY_TYPE_INTERNAL_ONLY
	}
	e.Active = true
	e.DateCreated = &[]int64{now.UnixMilli()}[0]
	e.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	e.DateModified = &[]int64{now.UnixMilli()}[0]
	e.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}
