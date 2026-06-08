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

// ComputeNextDueTrancheUseCase is the proto-shaped wrapper over the pure-math
// amortize_schedule.ComputeNextDueTranche function.
//
// It translates the proto Request/Response shape defined in
// proto/v1/service/amortization/amortization.proto into the Go-shaped
// Inputs/TrancheSpec the shared math package uses.
type ComputeNextDueTrancheUseCase struct{}

// NewComputeNextDueTrancheUseCase wires the wrapper. No deps — pure computation.
func NewComputeNextDueTrancheUseCase() *ComputeNextDueTrancheUseCase {
	return &ComputeNextDueTrancheUseCase{}
}

// Execute runs the pure-math ComputeNextDueTranche with proto-shaped IO.
func (uc *ComputeNextDueTrancheUseCase) Execute(
	_ context.Context,
	req *amortizationpb.ComputeNextDueTrancheRequest,
) (*amortizationpb.ComputeNextDueTrancheResponse, error) {
	if req == nil {
		return nil, errors.New("ComputeNextDueTrancheRequest is nil")
	}

	tranche, found, err := amortizeschedule.ComputeNextDueTranche(amortizeschedule.Inputs{
		StartDate:       req.GetStartDate(),
		EndDate:         req.GetEndDate(),
		PeriodCount:     int(req.GetPeriodCount()),
		PeriodUnit:      req.GetPeriodUnit(),
		TotalAmount:     req.GetTotalAmount(),
		ProrationPolicy: protoProrationToHelper(req.GetProrationPolicy()),
		AsOfDate:        req.GetAsOfDate(),
	})
	if err != nil {
		return nil, err
	}

	resp := &amortizationpb.ComputeNextDueTrancheResponse{
		Found: found,
	}
	if found {
		resp.Tranche = &amortizationpb.TrancheSpec{
			Index:       int32(tranche.Index),
			PeriodStart: tranche.PeriodStart,
			PeriodEnd:   tranche.PeriodEnd,
			Amount:      tranche.Amount,
		}
	}

	return resp, nil
}
