package domain

import (
	"errors"
	"fmt"

	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// Typed refusals of the job_phase approval ladder. They only classify an
// existing refusal so a presentation layer can answer precisely; they never
// change which actors or states are refused. Messages stay free of identifiers
// and callers must not echo wrapped error text to end users.
var (
	// ErrJobPhaseApprovalStateConflict: the sheet is not in the state the verb needs.
	ErrJobPhaseApprovalStateConflict = errors.New("job_phase approval: state conflict")
	// ErrJobPhaseApprovalOperatorLogin: an operator login lacks the workspace approval scope.
	ErrJobPhaseApprovalOperatorLogin = errors.New("job_phase approval: operator login lacks workspace approval scope")
	// ErrJobPhaseApprovalNotAllowed: the caller lacks the verb or is outside the sheet's scope.
	ErrJobPhaseApprovalNotAllowed = errors.New("job_phase approval: not allowed")
	// ErrJobPhaseApprovalSelfVerify: separation of duties forbids verifying one's own submission.
	ErrJobPhaseApprovalSelfVerify = errors.New("job_phase approval: self-verify denied")
	// ErrJobPhaseApprovalFrozen: the sheet is hard-frozen and cannot change.
	ErrJobPhaseApprovalFrozen = errors.New("job_phase approval: sheet frozen")
)

// JobPhaseApprovalStateError reports a refused transition because the sheet is
// not uniformly in the wanted source state. Found is the observed approval
// status (UNSPECIFIED when Mixed). It matches ErrJobPhaseApprovalStateConflict
// via errors.Is.
type JobPhaseApprovalStateError struct {
	Want  jobphasepb.PhaseApprovalStatus
	Found jobphasepb.PhaseApprovalStatus
	Mixed bool
}

func (e *JobPhaseApprovalStateError) Error() string {
	if e.Mixed {
		return fmt.Sprintf("job_phase approval: sheet is not uniformly %s (mixed states)", e.Want)
	}
	return fmt.Sprintf("job_phase approval: sheet is not uniformly %s (found %s)", e.Want, e.Found)
}

// Is makes errors.Is(err, ErrJobPhaseApprovalStateConflict) true.
func (e *JobPhaseApprovalStateError) Is(target error) bool {
	return target == ErrJobPhaseApprovalStateConflict
}
