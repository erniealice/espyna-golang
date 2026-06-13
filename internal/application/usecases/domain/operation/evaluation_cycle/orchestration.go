package evaluation_cycle

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle"
	evaluationcyclememberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle_member"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// OpenUseCase transitions a cycle to OPEN and freezes its denominator by snapshot.
type OpenUseCase struct {
	r Repositories
	s Services
}

// OpenRequest is the Go-shaped input.
type OpenRequest struct {
	EvaluationCycleID string
}

func (uc *OpenUseCase) Execute(ctx context.Context, req *OpenRequest) (*pb.UpdateEvaluationCycleResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.EvaluationCycleID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.id_required", "Evaluation cycle ID is required [DEFAULT]"))
	}

	read, err := uc.r.EvaluationCycle.ReadEvaluationCycle(ctx, &pb.ReadEvaluationCycleRequest{Data: &pb.EvaluationCycle{Id: req.EvaluationCycleID}})
	if err != nil {
		return nil, err
	}
	if read == nil || len(read.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.errors.not_found", "Evaluation cycle not found [DEFAULT]"))
	}
	cycle := read.Data[0]

	// Multi-aggregate write tx: set status OPEN + insert member snapshot rows.
	open := func(c context.Context) error {
		cycle.Status = pb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_OPEN
		cycle.Active = true
		if _, uerr := uc.r.EvaluationCycle.UpdateEvaluationCycle(c, &pb.UpdateEvaluationCycleRequest{Data: cycle}); uerr != nil {
			return uerr
		}
		return uc.freezeDenominator(c, cycle)
	}

	if uc.s.Transactor != nil && uc.s.Transactor.SupportsTransactions() {
		if err := uc.s.Transactor.ExecuteInTransaction(ctx, open); err != nil {
			return nil, err
		}
	} else if err := open(ctx); err != nil {
		return nil, err
	}

	return &pb.UpdateEvaluationCycleResponse{Data: []*pb.EvaluationCycle{cycle}, Success: true}, nil
}

// freezeDenominator inserts one evaluation_cycle_member per in-scope
// cadence-bearing ACTIVE seat for the cycle's subscription. It is idempotent: a
// member that already exists for (cycle, staff, client) is skipped, so re-opening
// the cycle is a no-op (EVCYC-10). On-demand / cadence-null seats are excluded
// (EVCYC-9). The seat list is filtered by the cycle's subscription_id.
func (uc *OpenUseCase) freezeDenominator(ctx context.Context, cycle *pb.EvaluationCycle) error {
	if uc.r.SubscriptionSeat == nil || uc.r.EvaluationCycleMember == nil {
		// Best-effort: without the seat/member repos the cycle still opens but
		// the denominator stays empty (degrade, do not panic).
		return nil
	}

	seatsResp, err := uc.r.SubscriptionSeat.ListSubscriptionSeats(ctx, &subscriptionseatpb.ListSubscriptionSeatsRequest{})
	if err != nil {
		return err
	}
	if seatsResp == nil {
		return nil
	}

	// Existing members for this cycle, keyed by (staff, client) for dedupe.
	existing := map[string]bool{}
	memResp, err := uc.r.EvaluationCycleMember.ListEvaluationCycleMembers(ctx, &evaluationcyclememberpb.ListEvaluationCycleMembersRequest{})
	if err != nil {
		return err
	}
	if memResp != nil {
		for _, m := range memResp.Data {
			if m.EvaluationCycleId == cycle.Id {
				existing[memberKey(m.SubjectStaffId, m.ClientId)] = true
			}
		}
	}

	for _, seat := range seatsResp.Data {
		// Scope to this cycle's subscription.
		if seat.SubscriptionId != cycle.SubscriptionId {
			continue
		}
		// Cadence-bearing ACTIVE seats only (EVCYC-9): exclude on-demand /
		// cadence-null seats and non-active seats.
		if seat.Status != subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE {
			continue
		}
		if seat.ReviewCadenceValue == nil || seat.ReviewCadenceUnit == nil || *seat.ReviewCadenceUnit == "" {
			continue
		}
		if seat.StaffId == "" {
			continue
		}
		key := memberKey(seat.StaffId, seat.ClientId)
		if existing[key] {
			continue // ON CONFLICT DO NOTHING — re-open is a no-op.
		}
		existing[key] = true

		member := &evaluationcyclememberpb.EvaluationCycleMember{
			Id:                uc.s.IDGenerator.GenerateID(),
			WorkspaceId:       cycle.WorkspaceId,
			EvaluationCycleId: cycle.Id,
			ClientId:          seat.ClientId,
			SubjectStaffId:    seat.StaffId,
			IsProbation:       false,
			Active:            true,
		}
		if _, err := uc.r.EvaluationCycleMember.CreateEvaluationCycleMember(ctx, &evaluationcyclememberpb.CreateEvaluationCycleMemberRequest{Data: member}); err != nil {
			return err
		}
	}
	return nil
}

func memberKey(staffID, clientID string) string {
	return staffID + "\x00" + clientID
}

// CloseUseCase transitions a cycle to CLOSED.
type CloseUseCase struct {
	r Repositories
	s Services
}

// CloseRequest is the Go-shaped input.
type CloseRequest struct {
	EvaluationCycleID string
}

func (uc *CloseUseCase) Execute(ctx context.Context, req *CloseRequest) (*pb.UpdateEvaluationCycleResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.EvaluationCycleID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.id_required", "Evaluation cycle ID is required [DEFAULT]"))
	}
	read, err := uc.r.EvaluationCycle.ReadEvaluationCycle(ctx, &pb.ReadEvaluationCycleRequest{Data: &pb.EvaluationCycle{Id: req.EvaluationCycleID}})
	if err != nil {
		return nil, err
	}
	if read == nil || len(read.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.errors.not_found", "Evaluation cycle not found [DEFAULT]"))
	}
	cycle := read.Data[0]
	cycle.Status = pb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_CLOSED
	// active = (status NOT IN {CLOSED}); a closed cycle is inactive.
	cycle.Active = false
	return uc.r.EvaluationCycle.UpdateEvaluationCycle(ctx, &pb.UpdateEvaluationCycleRequest{Data: cycle})
}
