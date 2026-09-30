// Package charge_effect writes the append-only accounting effects of charge events
// (20260927-usage-and-pass-through-charges, build-spec §6.3: ISSUE / APPLICATION / REVERSAL).
// Effects are not journal entries; reports read operational entities. Callers (issuance,
// application) invoke RecordChargeEffects INSIDE their own transaction so effects commit or
// roll back with the event.
package charge_effect

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// Refusals are usecaseerr.Error values (build-spec §7 C2, §7c C29): ErrorCode() = <code>, the
// message is translated from the Lyngua key <entity>.errors.<code> by usecaseerr.Localize.

// fallbackMessages are the English defaults of the general-tier keys charge_effect.errors.<code>.
var fallbackMessages = map[string]string{
	"missing_posting": "The charge policy has no account mapping for this step.",
	"validation":      "The charge effect is not valid.",
}

func newErr(code string) *usecaseerr.Error {
	return usecaseerr.New("charge_effect: ", code, fallbackMessages[code])
}

var (
	errMissingPosting = newErr("missing_posting")
	errValidation     = newErr("validation")
)

// Repos are the collaborators. Both are workspace-scoped repositories.
type Repos struct {
	ChargeEffect        chargeeffectpb.ChargeEffectDomainServiceServer
	ChargePolicyPosting postingpb.ChargePolicyPostingDomainServiceServer
	// NewID optionally supplies row ids (the wired IDGenerator); nil falls back to a local UUIDv7.
	NewID func() string
	// Translator optionally translates refusal messages; nil keeps the English defaults.
	Translator ports.Translator
}

// RecordChargeEffects writes the balanced DR/CR rows of one charge event.
//
//	ISSUE:       DR RECEIVABLE / CR CLEARING
//	APPLICATION: DR CASH       / CR RECEIVABLE
//	REVERSAL:    mirror of the reversed APPLICATION (DR RECEIVABLE / CR CASH); use
//	             RecordReversalEffects to mirror an ISSUE instead (e.g. a credit).
//
// amount is signed centavos; a negative amount (credit note) swaps the directions and posts
// |amount|. A zero amount writes nothing. The accounts come from the version's postings for the
// event; a missing mapping refuses with errMissingPosting. Idempotent on
// (event_kind, event_id, posting_role, direction): existing rows are never duplicated.
func RecordChargeEffects(ctx context.Context, repos Repos, versionID string, event enumspb.ChargePostingEvent,
	eventID, eventDate, currency string, amount int64, sourceRef string) error {
	reversed := event
	if event == enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_REVERSAL {
		reversed = enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION
	}
	return usecaseerr.Localize(ctx, repos.Translator, "charge_effect", record(ctx, repos, versionID, event, reversed, eventID, eventDate, currency, amount, sourceRef))
}

// RecordReversalEffects records a REVERSAL event that mirrors the effects of reversedEvent
// (ISSUE or APPLICATION), using that event's account mapping.
func RecordReversalEffects(ctx context.Context, repos Repos, versionID string, reversedEvent enumspb.ChargePostingEvent,
	eventID, eventDate, currency string, amount int64, sourceRef string) error {
	switch reversedEvent {
	case enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION:
	default:
		return usecaseerr.Localize(ctx, repos.Translator, "charge_effect", fmt.Errorf("%w: reversed event must be ISSUE or APPLICATION", errValidation))
	}
	return usecaseerr.Localize(ctx, repos.Translator, "charge_effect", record(ctx, repos, versionID, enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_REVERSAL, reversedEvent, eventID, eventDate, currency, amount, sourceRef))
}

type leg struct {
	role enumspb.ChargePostingRole
	dir  chargeeffectpb.EffectDirection
}

func legsFor(mapping enumspb.ChargePostingEvent, reversal bool) (debit, credit enumspb.ChargePostingRole, ok bool) {
	const (
		recv  = enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE
		clear = enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CLEARING
		cash  = enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH
	)
	switch mapping {
	case enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE:
		debit, credit = recv, clear
	case enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION:
		debit, credit = cash, recv
	default:
		return 0, 0, false
	}
	if reversal {
		debit, credit = credit, debit
	}
	return debit, credit, true
}

func record(ctx context.Context, repos Repos, versionID string, event, mapping enumspb.ChargePostingEvent,
	eventID, eventDate, currency string, amount int64, sourceRef string) error {
	if repos.ChargeEffect == nil || repos.ChargePolicyPosting == nil {
		return fmt.Errorf("%w: repositories not wired", errValidation)
	}
	if versionID == "" || eventID == "" || currency == "" {
		return fmt.Errorf("%w: version, event id and currency are required", errValidation)
	}
	debitRole, creditRole, ok := legsFor(mapping, event == enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_REVERSAL)
	if !ok {
		return fmt.Errorf("%w: unsupported event %s", errValidation, event)
	}
	if amount == 0 {
		return nil
	}
	if amount < 0 { // credit: mirror the direction, post the magnitude
		debitRole, creditRole = creditRole, debitRole
		amount = -amount
	}

	// Accounts: the version's postings for the mapped event (role -> account id).
	pr, err := repos.ChargePolicyPosting.ListChargePolicyPostings(ctx, &postingpb.ListChargePolicyPostingsRequest{Filters: listdata.EqFilter("charge_policy_version_id", versionID)})
	if err != nil {
		return fmt.Errorf("charge_effect: list postings: %w", err)
	}
	rows := pr.GetData()
	sort.Slice(rows, func(i, j int) bool { return rows[i].GetId() < rows[j].GetId() })
	acct := map[enumspb.ChargePostingRole]string{}
	for _, p := range rows {
		if p.GetEvent() != mapping || !p.GetActive() || p.GetAccountId() == "" {
			continue
		}
		if _, dup := acct[p.GetPostingRole()]; !dup {
			acct[p.GetPostingRole()] = p.GetAccountId()
		}
	}
	debitAcct, creditAcct := acct[debitRole], acct[creditRole]
	if debitAcct == "" || creditAcct == "" {
		return errMissingPosting
	}

	// Idempotency: skip legs already written for this event.
	er, err := repos.ChargeEffect.ListChargeEffects(ctx, &chargeeffectpb.ListChargeEffectsRequest{Filters: listdata.EqFilter("event_id", eventID)})
	if err != nil {
		return fmt.Errorf("charge_effect: list effects: %w", err)
	}
	have := map[leg]bool{}
	for _, e := range er.GetData() {
		if e.GetEventKind() == event {
			have[leg{e.GetPostingRole(), e.GetDirection()}] = true
		}
	}
	for _, l := range []struct {
		role enumspb.ChargePostingRole
		dir  chargeeffectpb.EffectDirection
		acct string
	}{
		{debitRole, chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT, debitAcct},
		{creditRole, chargeeffectpb.EffectDirection_EFFECT_DIRECTION_CREDIT, creditAcct},
	} {
		if have[leg{l.role, l.dir}] {
			continue
		}
		row := &chargeeffectpb.ChargeEffect{
			Id:                    newID(repos),
			EventKind:             event,
			EventId:               eventID,
			PostingRole:           l.role,
			AccountId:             l.acct,
			Direction:             l.dir,
			Amount:                amount,
			Currency:              currency,
			ChargePolicyVersionId: strp(versionID),
			Active:                true,
		}
		if eventDate != "" {
			row.EventDate = strp(eventDate)
		}
		if sourceRef != "" {
			row.SourceRef = strp(sourceRef)
		}
		if _, err := repos.ChargeEffect.CreateChargeEffect(ctx, &chargeeffectpb.CreateChargeEffectRequest{Data: row}); err != nil {
			return fmt.Errorf("charge_effect: write %s/%s: %w", l.role, l.dir, err)
		}
	}
	return nil
}

func strp(s string) *string { return &s }

func newID(r Repos) string {
	if r.NewID != nil {
		if id := r.NewID(); id != "" {
			return id
		}
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	ms := uint64(time.Now().UnixMilli())
	b[0], b[1], b[2], b[3], b[4], b[5] = byte(ms>>40), byte(ms>>32), byte(ms>>24), byte(ms>>16), byte(ms>>8), byte(ms)
	b[6] = b[6]&0x0f | 0x70
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
