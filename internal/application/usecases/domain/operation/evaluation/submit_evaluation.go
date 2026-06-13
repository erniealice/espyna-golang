package evaluation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	evaluationscore "github.com/erniealice/espyna-golang/internal/application/shared/evaluation_score"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	evaluationresponsepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_response"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// SubmitEvaluationRequest is the Go-shaped input (no proto request type exists for
// this state-transition op).
type SubmitEvaluationRequest struct {
	EvaluationID string
}

// SubmitEvaluationRepositories groups all repository dependencies.
type SubmitEvaluationRepositories struct {
	Evaluation         evaluationpb.EvaluationDomainServiceServer
	EvaluationResponse evaluationresponsepb.EvaluationResponseDomainServiceServer
	OutcomeCriteria    outcomecriteriapb.OutcomeCriteriaDomainServiceServer // snapshot source
}

// SubmitEvaluationServices groups all business service dependencies.
type SubmitEvaluationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SubmitEvaluationUseCase moves a DRAFT evaluation to SUBMITTED (Q-EVAL-SNAPSHOT-1).
//
// For each response it snapshots criteria_label / criteria_weight /
// criteria_version_id / criteria_type from the linked OutcomeCriteria, computes
// the overall_score via the pure shared/evaluation_score leaf (from the
// snapshots, NOT the live rubric), sets submitted_at, and re-asserts the IDOR
// invariants (client_id == acting_as_client_id + anchor ownership for
// CLIENT_TO_ASSOCIATE). The multi-write (response snapshots + header update) runs
// inside ExecuteInTransaction for atomicity.
type SubmitEvaluationUseCase struct {
	repositories SubmitEvaluationRepositories
	services     SubmitEvaluationServices
}

func NewSubmitEvaluationUseCase(repositories SubmitEvaluationRepositories, services SubmitEvaluationServices) *SubmitEvaluationUseCase {
	return &SubmitEvaluationUseCase{repositories: repositories, services: services}
}

func (uc *SubmitEvaluationUseCase) Execute(ctx context.Context, req *SubmitEvaluationRequest) (*evaluationpb.UpdateEvaluationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Evaluation,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.EvaluationID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.id_required", "Evaluation ID is required [DEFAULT]"))
	}

	// Load the header.
	readResp, err := uc.repositories.Evaluation.ReadEvaluation(ctx, &evaluationpb.ReadEvaluationRequest{
		Data: &evaluationpb.Evaluation{Id: req.EvaluationID},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.not_found", "Evaluation not found [DEFAULT]"))
	}
	eval := readResp.Data[0]

	// Only DRAFT may be submitted.
	if eval.Status != evaluationpb.EvaluationStatus_EVALUATION_STATUS_DRAFT {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.not_draft", "Only a draft evaluation can be submitted [DEFAULT]"))
	}

	// Re-assert IDOR: client-facing submit requires acting scope + ownership.
	actingClient := contextutil.GetActingAsClientIDFromContext(ctx)
	if actingClient == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.no_acting_client", "An acting client scope is required to submit an evaluation [DEFAULT]"))
	}
	if eval.ClientId != actingClient {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.not_owned", "This evaluation is not owned by the acting client [DEFAULT]"))
	}
	if eval.RelationshipType == evaluationpb.RelationshipType_RELATIONSHIP_TYPE_CLIENT_TO_ASSOCIATE &&
		(eval.SubscriptionSeatId == nil || *eval.SubscriptionSeatId == "") {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.seat_required", "A subscription seat anchor is required for this evaluation [DEFAULT]"))
	}

	// Load this evaluation's responses.
	respList, err := uc.repositories.EvaluationResponse.ListEvaluationResponses(ctx, &evaluationresponsepb.ListEvaluationResponsesRequest{})
	if err != nil {
		return nil, err
	}
	var responses []*evaluationresponsepb.EvaluationResponse
	if respList != nil {
		for _, r := range respList.Data {
			if r.EvaluationId == req.EvaluationID {
				responses = append(responses, r)
			}
		}
	}

	// Snapshot each response from its linked OutcomeCriteria + collect score inputs.
	var scoreInputs []evaluationscore.Response
	for _, r := range responses {
		if err := uc.snapshotResponse(ctx, r); err != nil {
			return nil, err
		}
		scoreInputs = append(scoreInputs, evaluationscore.Response{
			IsNumeric:    isNumericCriteriaType(r.CriteriaType),
			Weight:       r.CriteriaWeight,
			NumericValue: r.NumericValue,
		})
	}

	score, err := evaluationscore.ComputeEvaluationScore(scoreInputs)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.score_compute_failed", "Evaluation score could not be computed [DEFAULT]"))
	}

	now := time.Now()
	eval.Status = evaluationpb.EvaluationStatus_EVALUATION_STATUS_SUBMITTED
	eval.OverallScore = score
	eval.SubmittedAt = &[]int64{now.UnixMilli()}[0]
	eval.DateModified = &[]int64{now.UnixMilli()}[0]
	eval.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Atomic multi-write: persist each snapshotted response then the header.
	persist := func(c context.Context) error {
		for _, r := range responses {
			if _, uerr := uc.repositories.EvaluationResponse.UpdateEvaluationResponse(c, &evaluationresponsepb.UpdateEvaluationResponseRequest{Data: r}); uerr != nil {
				return uerr
			}
		}
		_, uerr := uc.repositories.Evaluation.UpdateEvaluation(c, &evaluationpb.UpdateEvaluationRequest{Data: eval})
		return uerr
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, persist); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.submit_failed", "Evaluation submit failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	} else {
		if err := persist(ctx); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.submit_failed", "Evaluation submit failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	}

	return &evaluationpb.UpdateEvaluationResponse{Data: []*evaluationpb.Evaluation{eval}, Success: true}, nil
}

// snapshotResponse copies criteria_label / criteria_weight / criteria_version_id /
// criteria_type from the linked OutcomeCriteria onto the response (the SNAPSHOT
// columns the score is computed from). A missing criterion leaves the existing
// snapshot in place (best-effort — the criterion may have been deleted).
func (uc *SubmitEvaluationUseCase) snapshotResponse(ctx context.Context, r *evaluationresponsepb.EvaluationResponse) error {
	if uc.repositories.OutcomeCriteria == nil || r.OutcomeCriteriaId == "" {
		return nil
	}
	ocResp, err := uc.repositories.OutcomeCriteria.ReadOutcomeCriteria(ctx, &outcomecriteriapb.ReadOutcomeCriteriaRequest{
		Data: &outcomecriteriapb.OutcomeCriteria{Id: r.OutcomeCriteriaId},
	})
	if err != nil {
		return err
	}
	if ocResp == nil || len(ocResp.Data) == 0 {
		return nil
	}
	oc := ocResp.Data[0]
	r.CriteriaLabel = oc.Name
	w := oc.Weight
	r.CriteriaWeight = &w
	r.CriteriaType = oc.CriteriaType
	if oc.Id != "" {
		id := oc.Id
		r.CriteriaVersionId = &id
	}
	return nil
}

// isNumericCriteriaType reports whether a snapshotted criteria_type contributes
// to the weighted score (numeric_range / numeric_score only).
func isNumericCriteriaType(t enumspb.CriteriaType) bool {
	return t == enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE ||
		t == enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE
}
