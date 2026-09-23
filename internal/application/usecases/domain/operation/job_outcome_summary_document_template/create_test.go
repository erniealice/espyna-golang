package job_outcome_summary_document_template

import (
	"context"
	"errors"
	"testing"

	phasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubPhaseCodeReader struct {
	response   *phasepb.ListPhaseCodesByPriceScheduleResponse
	err        error
	scheduleID string
}

func (s *stubPhaseCodeReader) ListPhaseCodesByPriceSchedule(_ context.Context, req *phasepb.ListPhaseCodesByPriceScheduleRequest) (*phasepb.ListPhaseCodesByPriceScheduleResponse, error) {
	s.scheduleID = req.GetPriceScheduleId()
	return s.response, s.err
}

func TestBindingPhaseCodeMustBelongToSchedule(t *testing.T) {
	reader := &stubPhaseCodeReader{response: &phasepb.ListPhaseCodesByPriceScheduleResponse{
		Success: true, Options: []*phasepb.PhaseCodeOption{{Code: "s1"}},
	}}
	uc := &CreateUseCase{phaseCodes: reader}
	if err := uc.validSchedulePhaseCode(context.Background(), "academic-year", "s1"); err != nil {
		t.Fatalf("known active code rejected: %v", err)
	}
	if reader.scheduleID != "academic-year" {
		t.Fatalf("queried schedule %q", reader.scheduleID)
	}
	if err := uc.validSchedulePhaseCode(context.Background(), "academic-year", "s2"); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unlisted code error = %v, want InvalidArgument", err)
	}
	reader.response.Success = false
	if err := uc.validSchedulePhaseCode(context.Background(), "academic-year", "s1"); status.Code(err) != codes.Internal {
		t.Fatalf("unsuccessful lookup error = %v, want Internal", err)
	}
	reader.response.Success = true
	reader.response = nil
	if err := uc.validSchedulePhaseCode(context.Background(), "academic-year", "s1"); status.Code(err) != codes.Internal {
		t.Fatalf("missing lookup result error = %v, want Internal", err)
	}
	reader.err = errors.New("lookup failed")
	if err := uc.validSchedulePhaseCode(context.Background(), "academic-year", "s1"); status.Code(err) != codes.Internal {
		t.Fatalf("lookup failure error = %v, want Internal", err)
	}
	uc.phaseCodes = nil
	if err := uc.validSchedulePhaseCode(context.Background(), "academic-year", "s1"); status.Code(err) != codes.Internal {
		t.Fatalf("unwired phase reader error = %v, want Internal", err)
	}
}

func TestValidBindingPhaseCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		code *string
		want bool
	}{
		{name: "whole year", want: true},
		{name: "phase", code: phaseCodePtr("progress_report"), want: true},
		{name: "uppercase", code: phaseCodePtr("S1")},
		{name: "path", code: phaseCodePtr("../s1")},
		{name: "space", code: phaseCodePtr("s1 ")},
		{name: "empty", code: phaseCodePtr("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validBindingPhaseCode(tc.code); got != tc.want {
				t.Fatalf("validBindingPhaseCode(%v) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func phaseCodePtr(s string) *string { return &s }
