package revenuepayment

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// Legacy-payment guard (20260927-usage-and-pass-through-charges, build-spec §6.1, AC-UC-33).
//
// A revenue is "participating" in known-cost recovery when its subscription has any agreement line
// term (a term exists exactly for each line that opted in to a charge policy) OR when the revenue
// already has a collection_application row (any status). Legacy revenue_payment writes against a
// participating revenue are refused: settlement goes through "Receive & apply" so cash is never
// counted twice. Revenues of every other kind keep the legacy behaviour byte for byte.

const codeLegacyPaymentRefusedParticipating = "legacy_payment_refused_participating"

// refusedParticipating builds the refusal with its message translated from
// revenue_payment.errors.legacy_payment_refused_participating (C2); views branch on ErrorCode().
func refusedParticipating(ctx context.Context, tr ports.Translator) error {
	return usecaseerr.New("", codeLegacyPaymentRefusedParticipating,
		contextutil.GetTranslatedMessageWithContext(ctx, tr, "revenue_payment.errors."+codeLegacyPaymentRefusedParticipating,
			"This invoice is settled through applications. Use Receive and apply instead."))
}

// ParticipationGuard holds the read repositories of the guard. The zero value (no Revenue
// repository) disables it.
type ParticipationGuard struct {
	Revenue               revenuepb.RevenueDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	AgreementLineTerm     agreementlinetermpb.AgreementLineTermDomainServiceServer
}

func guardFrom(r RevenuePaymentRepositories) ParticipationGuard {
	return ParticipationGuard{Revenue: r.Revenue, CollectionApplication: r.CollectionApplication, AgreementLineTerm: r.AgreementLineTerm}
}

// refuseIfParticipating returns ErrLegacyPaymentRefusedParticipating when the revenue participates.
// Lookup failures fail closed (the write is refused with the underlying error).
func (g ParticipationGuard) refuseIfParticipating(ctx context.Context, tr ports.Translator, revenueID string) error {
	if g.Revenue == nil || revenueID == "" {
		return nil
	}
	rr, err := g.Revenue.ReadRevenue(ctx, &revenuepb.ReadRevenueRequest{Data: &revenuepb.Revenue{Id: revenueID}})
	if err != nil {
		log.Printf("revenue_payment: guard could not read revenue (id=%s): %v", revenueID, err)
		return fmt.Errorf("revenue_payment: guard could not read the revenue: %w", err)
	}
	if len(rr.GetData()) == 0 {
		return fmt.Errorf("revenue_payment: guard could not read the revenue: not found")
	}
	if g.CollectionApplication != nil {
		ar, err := g.CollectionApplication.ListCollectionApplications(ctx, &collectionapplicationpb.ListCollectionApplicationsRequest{Filters: listdata.EqFilter("revenue_id", revenueID)})
		if err != nil {
			return fmt.Errorf("revenue_payment: guard could not list applications: %w", err)
		}
		if len(ar.GetData()) > 0 {
			return refusedParticipating(ctx, tr)
		}
	}
	if sub := rr.Data[0].GetSubscriptionId(); sub != "" && g.AgreementLineTerm != nil {
		termsResp, err := g.AgreementLineTerm.ListAgreementLineTerms(ctx, &agreementlinetermpb.ListAgreementLineTermsRequest{Filters: listdata.EqFilter("subscription_id", sub)})
		if err != nil {
			return fmt.Errorf("revenue_payment: guard could not list agreement terms: %w", err)
		}
		if len(termsResp.GetData()) > 0 {
			return refusedParticipating(ctx, tr)
		}
	}
	return nil
}

// revenueOfPayment resolves the revenue id of a stored payment. A payment the repository reports as
// missing yields "" (the repository call that follows reports the missing row exactly as before); any
// OTHER read failure is returned so the guard fails closed instead of skipping (C12).
func revenueOfPayment(ctx context.Context, repo pb.RevenuePaymentDomainServiceServer, id string) (string, error) {
	resp, err := repo.ReadRevenuePayment(ctx, &pb.ReadRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: id}})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return "", nil
		}
		log.Printf("revenue_payment: guard could not read payment (id=%s): %v", id, err)
		return "", fmt.Errorf("revenue_payment: guard could not read the payment: %w", err)
	}
	if len(resp.GetData()) == 0 {
		return "", nil
	}
	return resp.Data[0].GetRevenueId(), nil
}
