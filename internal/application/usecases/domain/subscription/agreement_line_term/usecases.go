// Package agreement_line_term holds the read use cases of the agreement_line_term entity (domain
// subscription; 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.4) and the
// pure agreement-term rules (terms.go: Covers / Overlaps / FindCoveringTerm / ListBySubscription)
// shared by subscription create and allocation publish. The rows are written by subscription create
// (COPIED terms) and are never hard-deleted (C6, adapter). The "terms of one subscription" read is
// ListAgreementLineTerms with its subscription_id field (no custom-verb request, C1).
// Authorization: subscription:read (fail closed).
package agreement_line_term

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// Repositories groups the repository dependencies.
type Repositories struct {
	AgreementLineTerm agreementlinetermpb.AgreementLineTermDomainServiceServer
}

// Services groups the shared service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func (s Services) gate(ctx context.Context, action string) error {
	return s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Subscription, Action: action})
}

// UseCases aggregates the agreement_line_term read use cases.
type UseCases struct {
	ListAgreementLineTerms           *ListAgreementLineTermsUseCase
	GetAgreementLineTermListPageData *GetAgreementLineTermListPageDataUseCase
	ReadAgreementLineTerm            *ReadAgreementLineTermUseCase
}

// NewUseCases wires the agreement_line_term read use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		ListAgreementLineTerms:           &ListAgreementLineTermsUseCase{r, s},
		GetAgreementLineTermListPageData: &GetAgreementLineTermListPageDataUseCase{r, s},
		ReadAgreementLineTerm:            &ReadAgreementLineTermUseCase{r, s},
	}
}
