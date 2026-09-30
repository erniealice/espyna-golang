package agreement_line_term

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// GetAgreementLineTermListPageDataUseCase returns a page of rows with pagination.
type GetAgreementLineTermListPageDataUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *GetAgreementLineTermListPageDataUseCase) Execute(ctx context.Context, req *agreementlinetermpb.GetAgreementLineTermListPageDataRequest) (*agreementlinetermpb.GetAgreementLineTermListPageDataResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repos.AgreementLineTerm == nil {
		return nil, fmt.Errorf("agreement_line_term: repository unavailable")
	}
	if req == nil {
		req = &agreementlinetermpb.GetAgreementLineTermListPageDataRequest{}
	}
	return uc.repos.AgreementLineTerm.GetAgreementLineTermListPageData(ctx, req)
}
