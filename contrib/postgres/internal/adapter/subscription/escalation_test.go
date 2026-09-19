//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

type escalationWriteFixture struct {
	interfaces.DatabaseOperation
	createData map[string]any
	updateData map[string]any
}

func cloneEscalationMap(data map[string]any) map[string]any {
	result := make(map[string]any, len(data)+1)
	for key, value := range data {
		result[key] = value
	}
	result["id"] = "escalation-record-1"
	return result
}

func (f *escalationWriteFixture) Create(_ context.Context, _ string, data map[string]any) (map[string]any, error) {
	f.createData = data
	return cloneEscalationMap(data), nil
}

func (f *escalationWriteFixture) Update(_ context.Context, _ string, _ string, data map[string]any) (map[string]any, error) {
	f.updateData = data
	return cloneEscalationMap(data), nil
}

func TestWriteEscalationTupleUsesDatabaseTokensAndClearsDependents(t *testing.T) {
	mode := priceplanpb.EscalationMode_ESCALATION_MODE_NONE
	data := map[string]any{}
	writeEscalationTuple(data, "default", &mode, nil, nil, nil, nil)

	if data["defaultEscalationMode"] != "none" {
		t.Fatalf("mode = %v", data["defaultEscalationMode"])
	}
	for _, key := range []string{"defaultEscalationScope", "defaultEscalationRateBps", "defaultEscalationFirstAfterMonths", "defaultEscalationEveryMonths"} {
		if value, ok := data[key]; !ok || value != nil {
			t.Fatalf("%s = %v, present = %v; want present nil", key, value, ok)
		}
	}
}

func TestApplySubscriptionEscalationPreservesOptionalPresence(t *testing.T) {
	record := &subscriptionpb.Subscription{}
	applySubscriptionEscalation(record,
		sql.NullString{String: "fixed_percentage", Valid: true},
		sql.NullString{String: "within_agreement", Valid: true},
		sql.NullInt32{Int32: 425, Valid: true},
		sql.NullInt32{Int32: 12, Valid: true},
		sql.NullInt32{Int32: 12, Valid: true},
	)

	if record.EscalationMode == nil || record.GetEscalationMode() != priceplanpb.EscalationMode_ESCALATION_MODE_FIXED_PERCENTAGE {
		t.Fatalf("mode = %v", record.EscalationMode)
	}
	if record.EscalationRateBps == nil || record.GetEscalationRateBps() != 425 {
		t.Fatalf("rate = %v", record.EscalationRateBps)
	}
}

func TestEscalationPostgresRoundTrip(t *testing.T) {
	fixed := priceplanpb.EscalationMode_ESCALATION_MODE_FIXED_PERCENTAGE
	none := priceplanpb.EscalationMode_ESCALATION_MODE_NONE
	renewal := priceplanpb.EscalationScope_ESCALATION_SCOPE_ON_RENEWAL
	rate := int32(425)

	t.Run("price plan create and clear update", func(t *testing.T) {
		fixture := &escalationWriteFixture{}
		repo := NewPostgresPricePlanRepository(fixture, "price_plan").(*PostgresPricePlanRepository)
		created, err := repo.CreatePricePlan(context.Background(), &priceplanpb.CreatePricePlanRequest{Data: &priceplanpb.PricePlan{
			DefaultEscalationMode:    &fixed,
			DefaultEscalationScope:   &renewal,
			DefaultEscalationRateBps: &rate,
		}})
		if err != nil {
			t.Fatal(err)
		}
		if fixture.createData["defaultEscalationMode"] != "fixed_percentage" || created.GetData()[0].GetDefaultEscalationRateBps() != 425 {
			t.Fatalf("create payload=%#v response=%+v", fixture.createData, created.GetData()[0])
		}

		_, err = repo.UpdatePricePlan(context.Background(), &priceplanpb.UpdatePricePlanRequest{Data: &priceplanpb.PricePlan{
			Id:                    "escalation-record-1",
			DefaultEscalationMode: &none,
		}})
		if err != nil {
			t.Fatal(err)
		}
		if fixture.updateData["defaultEscalationMode"] != "none" || fixture.updateData["defaultEscalationRateBps"] != nil {
			t.Fatalf("clear update payload=%#v", fixture.updateData)
		}
	})

	t.Run("agreement create", func(t *testing.T) {
		fixture := &escalationWriteFixture{}
		repo := NewPostgresSubscriptionRepository(fixture, "subscription").(*PostgresSubscriptionRepository)
		created, err := repo.CreateSubscription(context.Background(), &subscriptionpb.CreateSubscriptionRequest{Data: &subscriptionpb.Subscription{
			EscalationMode:    &fixed,
			EscalationScope:   &renewal,
			EscalationRateBps: &rate,
		}})
		if err != nil {
			t.Fatal(err)
		}
		if fixture.createData["escalationMode"] != "fixed_percentage" || created.GetData()[0].GetEscalationRateBps() != 425 {
			t.Fatalf("create payload=%#v response=%+v", fixture.createData, created.GetData()[0])
		}
	})
}
