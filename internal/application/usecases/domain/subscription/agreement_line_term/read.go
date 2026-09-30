package agreement_line_term

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// ReadAgreementLineTermUseCase reads one row by id (foreign-workspace and missing ids are both "not found").
type ReadAgreementLineTermUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ReadAgreementLineTermUseCase) Execute(ctx context.Context, req *agreementlinetermpb.ReadAgreementLineTermRequest) (*agreementlinetermpb.ReadAgreementLineTermResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repos.AgreementLineTerm == nil {
		return nil, fmt.Errorf("agreement_line_term: repository unavailable")
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("agreement_line_term: id is required")
	}
	return uc.repos.AgreementLineTerm.ReadAgreementLineTerm(ctx, req)
}
