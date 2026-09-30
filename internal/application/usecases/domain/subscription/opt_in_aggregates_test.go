package subscription

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// C32: the S1 aggregates are nil unless their own repository is wired (subscription is in the
// Firestore set, where the S1 tables do not exist).
func TestKnownCostRecoveryAggregatesAreNilWithoutTheirRepositories(t *testing.T) {
	gate := actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
	uc := NewUseCases(SubscriptionRepositories{}, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate, nil, nil, "")
	if uc.AgreementLineTerm != nil || uc.BillableCharge != nil || uc.ChargeComponent != nil {
		t.Fatal("S1 aggregates must be nil without repositories")
	}
	uc = NewUseCases(SubscriptionRepositories{
		AgreementLineTerm: agreementlinetermpb.UnimplementedAgreementLineTermDomainServiceServer{},
		BillableCharge:    billablechargepb.UnimplementedBillableChargeDomainServiceServer{},
		ChargeComponent:   chargecomponentpb.UnimplementedChargeComponentDomainServiceServer{},
	}, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate, nil, nil, "")
	if uc.AgreementLineTerm == nil || uc.BillableCharge == nil || uc.ChargeComponent == nil {
		t.Fatal("S1 aggregates must be built when their repositories are wired")
	}
}
