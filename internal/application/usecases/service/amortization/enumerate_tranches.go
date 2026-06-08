package amortization

// NOTE: pnpm build must be run in packages/esqyma/ to generate the proto
// types before this package compiles. The import path
// github.com/erniealice/esqyma/pkg/schema/v1/service/amortization will not
// resolve until generation completes.

import (
	"context"
	"errors"

	amortizeschedule "github.com/erniealice/espyna-golang/internal/application/shared/amortize_schedule"

	amortizationpb "github.com/erniealice/esqyma/pkg/schema/v1/service/amortization"
)

// EnumerateTranchesUseCase is the proto-shaped wrapper over the pure-math
// amortize_schedule.EnumerateTranches function.
//
// It translates the proto Request/Response shape defined in
// proto/v1/service/amortization/amortization.proto into the Go-shaped
// Inputs/[]TrancheSpec the shared math package uses. The wrapper does NOT
// re-implement the algorithm — it delegates to the pure-math leaf.
type EnumerateTranchesUseCase struct{}

// NewEnumerateTranchesUseCase wires the wrapper. No deps — pure computation.
func NewEnumerateTranchesUseCase() *EnumerateTranchesUseCase {
	return &EnumerateTranchesUseCase{}
}

// Execute runs the pure-math EnumerateTranches with proto-shaped IO.
func (uc *EnumerateTranchesUseCase) Execute(
	_ context.Context,
	req *amortizationpb.EnumerateTranchesRequest,
) (*amortizationpb.EnumerateTranchesResponse, error) {
	if req == nil {
		return nil, errors.New("EnumerateTranchesRequest is nil")
	}

	tranches, err := amortizeschedule.EnumerateTranches(amortizeschedule.Inputs{
		StartDate:       req.GetStartDate(),
		EndDate:         req.GetEndDate(),
		PeriodCount:     int(req.GetPeriodCount()),
		PeriodUnit:      req.GetPeriodUnit(),
		TotalAmount:     req.GetTotalAmount(),
		ProrationPolicy: protoProrationToHelper(req.GetProrationPolicy()),
	})
	if err != nil {
		return nil, err
	}

	pbTranches := make([]*amortizationpb.TrancheSpec, len(tranches))
	for i, t := range tranches {
		pbTranches[i] = &amortizationpb.TrancheSpec{
			Index:       int32(t.Index),
			PeriodStart: t.PeriodStart,
			PeriodEnd:   t.PeriodEnd,
			Amount:      t.Amount,
		}
	}

	return &amortizationpb.EnumerateTranchesResponse{
		Tranches: pbTranches,
	}, nil
}
