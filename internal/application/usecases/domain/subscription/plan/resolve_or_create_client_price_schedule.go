package plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ResolveOrCreateClientScheduleRepos is the minimal repository surface the
// shared helper needs. It is consumed by CustomizePlanForClient as well as
// CreatePricePlan / UpdatePricePlan when an operator submits an empty
// price_schedule_id for a client-scoped parent Plan (see plan §3.2 / §4.4 of
// 20260427-plan-client-scope).
type ResolveOrCreateClientScheduleRepos struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer
}

// ResolveOrCreateClientPriceSchedule looks up an existing client-scoped
// PriceSchedule for `(workspaceID, locationID, clientID, active=true)`. When
// found it returns the row with `reused=true`; when missing it creates a new
// schedule from the supplied template + derived name and returns
// `reused=false`.
//
// Caller passes a pre-built `derivedName` (e.g. "Cruz Engineering - Rate
// Cards"); helper does NOT read lyngua. The original implementation lived as
// a private function inside customize_plan_for_client.go; it was extracted
// 2026-04-28 so the price_plan create/update flows can reuse the same
// resolve-or-create semantics.
func ResolveOrCreateClientPriceSchedule(
	ctx context.Context,
	repos *ResolveOrCreateClientScheduleRepos,
	idSvc ports.IDGenerator,
	workspaceID, locationID, clientID, derivedName string,
	template *priceschedulepb.PriceSchedule,
) (*priceschedulepb.PriceSchedule, bool, error) {
	if repos == nil || repos.PriceSchedule == nil {
		return nil, false, errors.New("price_schedule repository unavailable")
	}

	// Look up matching client schedule (workspace + location + client + active).
	filters := []*commonpb.TypedFilter{
		stringEqFilterShared("client_id", clientID),
		boolEqFilterShared("active", true),
	}
	if locationID != "" {
		filters = append(filters, stringEqFilterShared("location_id", locationID))
	}
	if workspaceID != "" {
		filters = append(filters, stringEqFilterShared("workspace_id", workspaceID))
	}

	listResp, err := repos.PriceSchedule.ListPriceSchedules(ctx, &priceschedulepb.ListPriceSchedulesRequest{
		Filters: &commonpb.FilterRequest{Filters: filters},
	})
	if err != nil {
		return nil, false, err
	}
	if listResp != nil {
		for _, ps := range listResp.GetData() {
			// Belt-and-braces: even if the adapter doesn't honour the
			// client_id filter, the use case must not reuse a master
			// schedule for a client.
			if ps.GetClientId() != clientID {
				continue
			}
			if locationID != "" && ps.GetLocationId() != locationID {
				continue
			}
			return ps, true, nil
		}
	}

	// Not found — create.
	now := time.Now()
	newID := ""
	if idSvc != nil {
		newID = idSvc.GenerateID()
	} else {
		newID = fmt.Sprintf("ps-%d", now.UnixNano())
	}
	clientCopy := clientID
	// 2026-05-03 — DateCreated/ModifiedString stamped in UTC so the round-trip
	// is unambiguous regardless of the server's local tz. The int64 millis
	// field is the canonical pair; the string is for human/debug.
	create := &priceschedulepb.PriceSchedule{
		Id:                 newID,
		Name:               derivedName,
		Active:             true,
		ClientId:           &clientCopy,
		DateCreated:        ptrInt64Shared(now.UnixMilli()),
		DateCreatedString:  ptrStringShared(now.UTC().Format(time.RFC3339)),
		DateModified:       ptrInt64Shared(now.UnixMilli()),
		DateModifiedString: ptrStringShared(now.UTC().Format(time.RFC3339)),
	}
	if locationID != "" {
		locCopy := locationID
		create.LocationId = &locCopy
	}
	if template != nil {
		if template.GetDateTimeStart() != nil {
			create.DateTimeStart = template.GetDateTimeStart()
		}
		if template.GetDateTimeEnd() != nil {
			create.DateTimeEnd = template.GetDateTimeEnd()
		}
	}
	// date_time_start is NOT NULL on the column; default to "now" when no
	// template provides one. date_time_end stays nil for an open-ended
	// schedule — the column is nullable. Operator can adjust later via the
	// schedule edit drawer.
	if create.DateTimeStart == nil {
		create.DateTimeStart = timestamppb.New(now)
	}
	createResp, err := repos.PriceSchedule.CreatePriceSchedule(ctx, &priceschedulepb.CreatePriceScheduleRequest{
		Data: create,
	})
	if err != nil {
		return nil, false, err
	}
	if createResp == nil || len(createResp.GetData()) == 0 {
		return nil, false, errors.New("create price schedule returned no data")
	}
	return createResp.GetData()[0], false, nil
}

// stringEqFilterShared returns a STRING_EQUALS TypedFilter — small helper to
// keep ResolveOrCreate readable. Named with `Shared` suffix to avoid
// colliding with the private helper of the same shape inside
// customize_plan_for_client.go.
func stringEqFilterShared(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    value,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}
}

// boolEqFilterShared returns a BOOLEAN_EQUALS TypedFilter for the active flag.
func boolEqFilterShared(field string, value bool) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_BooleanFilter{
			BooleanFilter: &commonpb.BooleanFilter{
				Value: value,
			},
		},
	}
}

func ptrStringShared(s string) *string { v := s; return &v }
func ptrInt64Shared(v int64) *int64    { c := v; return &c }
