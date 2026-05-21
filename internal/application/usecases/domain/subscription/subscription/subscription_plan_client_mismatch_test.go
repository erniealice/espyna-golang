package subscription

// Tests for the §3.3 plan_client_mismatch guard on CreateSubscription /
// UpdateSubscription. Uses lightweight in-package mocks so the tests run
// without the mock_db / mock_auth build tags.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ----- mocks ---------------------------------------------------------------

type mockSubRepo struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	existing *subscriptionpb.Subscription
}

func (m *mockSubRepo) ReadSubscription(_ context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if m.existing == nil {
		return &subscriptionpb.ReadSubscriptionResponse{Data: nil, Success: true}, nil
	}
	if req.GetData() != nil && req.GetData().GetId() != m.existing.GetId() && m.existing.GetId() != "" {
		return &subscriptionpb.ReadSubscriptionResponse{Data: nil, Success: true}, nil
	}
	return &subscriptionpb.ReadSubscriptionResponse{Data: []*subscriptionpb.Subscription{m.existing}, Success: true}, nil
}

func (m *mockSubRepo) CreateSubscription(_ context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
	return &subscriptionpb.CreateSubscriptionResponse{Data: []*subscriptionpb.Subscription{req.GetData()}, Success: true}, nil
}

func (m *mockSubRepo) UpdateSubscription(_ context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	return &subscriptionpb.UpdateSubscriptionResponse{Data: []*subscriptionpb.Subscription{req.GetData()}, Success: true}, nil
}

type mockClientRepoSub struct {
	clientpb.UnimplementedClientDomainServiceServer
	c *clientpb.Client
}

func (m *mockClientRepoSub) ReadClient(_ context.Context, _ *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if m.c == nil {
		return &clientpb.ReadClientResponse{Data: nil, Success: true}, nil
	}
	return &clientpb.ReadClientResponse{Data: []*clientpb.Client{m.c}, Success: true}, nil
}

type mockPricePlanRepoSub struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	pp *priceplanpb.PricePlan
}

func (m *mockPricePlanRepoSub) ReadPricePlan(_ context.Context, _ *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if m.pp == nil {
		return &priceplanpb.ReadPricePlanResponse{Data: nil, Success: true}, nil
	}
	return &priceplanpb.ReadPricePlanResponse{Data: []*priceplanpb.PricePlan{m.pp}, Success: true}, nil
}

// ----- helpers --------------------------------------------------------------

type stubIDForSub struct{}

func (stubIDForSub) GenerateID() string                   { return "sub-new" }
func (stubIDForSub) GenerateIDWithPrefix(p string) string { return p + "-new" }
func (stubIDForSub) IsEnabled() bool                      { return true }
func (stubIDForSub) GetProviderInfo() string              { return "stub" }

type noTxnSub struct{}

func (noTxnSub) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (noTxnSub) SupportsTransactions() bool               { return false }
func (noTxnSub) IsTransactionActive(context.Context) bool { return false }

func makePricePlan(id, clientID, currency string) *priceplanpb.PricePlan {
	pp := &priceplanpb.PricePlan{Id: id, BillingCurrency: currency, Active: true}
	if clientID != "" {
		c := clientID
		pp.ClientId = &c
	}
	return pp
}

func newCreateSubUC(t *testing.T, ppRepo *mockPricePlanRepoSub, clientRepo *mockClientRepoSub) *CreateSubscriptionUseCase {
	t.Helper()
	return NewCreateSubscriptionUseCase(
		CreateSubscriptionRepositories{
			Subscription: &mockSubRepo{},
			Client:       clientRepo,
			PricePlan:    ppRepo,
		},
		CreateSubscriptionServices{
			Authorizer:              ports.NewNoOpAuthorizer(),
			Transactor:              noTxnSub{},
			Translator:              ports.NewNoOpTranslator(),
			IDGenerator:             stubIDForSub{},
			JobTemplateInstantiator: nil,
		},
	)
}

func newUpdateSubUC(t *testing.T, ppRepo *mockPricePlanRepoSub, clientRepo *mockClientRepoSub, subRepo *mockSubRepo) *UpdateSubscriptionUseCase {
	t.Helper()
	return NewUpdateSubscriptionUseCase(
		UpdateSubscriptionRepositories{
			Subscription: subRepo,
			Client:       clientRepo,
			PricePlan:    ppRepo,
		},
		UpdateSubscriptionServices{
			Authorizer: ports.NewNoOpAuthorizer(),
			Transactor: noTxnSub{},
			Translator: ports.NewNoOpTranslator(),
		},
	)
}

func makeClient(id string) *clientpb.Client {
	return &clientpb.Client{Id: id, Active: true}
}

func newSubscription(name, clientID, pricePlanID string) *subscriptionpb.Subscription {
	return &subscriptionpb.Subscription{
		Name:        name,
		ClientId:    clientID,
		PricePlanId: pricePlanID,
	}
}

// ----- tests: Create --------------------------------------------------------

func TestCreateSubscription_ClientMatch_Success(t *testing.T) {
	ppRepo := &mockPricePlanRepoSub{pp: makePricePlan("pp-1", "client-cruz", "PHP")}
	clientRepo := &mockClientRepoSub{c: makeClient("client-cruz")}
	uc := newCreateSubUC(t, ppRepo, clientRepo)

	_, err := uc.Execute(context.Background(), &subscriptionpb.CreateSubscriptionRequest{
		Data: newSubscription("New Engagement", "client-cruz", "pp-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSubscription_MasterPricePlan_Success(t *testing.T) {
	ppRepo := &mockPricePlanRepoSub{pp: makePricePlan("pp-1", "", "PHP")}
	clientRepo := &mockClientRepoSub{c: makeClient("client-cruz")}
	uc := newCreateSubUC(t, ppRepo, clientRepo)

	_, err := uc.Execute(context.Background(), &subscriptionpb.CreateSubscriptionRequest{
		Data: newSubscription("New Engagement", "client-cruz", "pp-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSubscription_OtherClientPricePlan_Rejects(t *testing.T) {
	ppRepo := &mockPricePlanRepoSub{pp: makePricePlan("pp-1", "client-other", "PHP")}
	clientRepo := &mockClientRepoSub{c: makeClient("client-cruz")}
	uc := newCreateSubUC(t, ppRepo, clientRepo)

	_, err := uc.Execute(context.Background(), &subscriptionpb.CreateSubscriptionRequest{
		Data: newSubscription("New Engagement", "client-cruz", "pp-1"),
	})
	if err == nil {
		t.Fatalf("expected planClientMismatch error")
	}
	// Use translated message — fallback "[DEFAULT]" suffix is present.
	if !strings.Contains(err.Error(), "different client") {
		t.Errorf("expected mismatch wording in error, got %q", err.Error())
	}
}

// ----- tests: Update --------------------------------------------------------

func TestUpdateSubscription_RepointToOtherClientPricePlan_Rejects(t *testing.T) {
	ppRepo := &mockPricePlanRepoSub{pp: makePricePlan("pp-other", "client-other", "PHP")}
	clientRepo := &mockClientRepoSub{c: makeClient("client-cruz")}
	subRepo := &mockSubRepo{existing: &subscriptionpb.Subscription{
		Id:          "sub-1",
		ClientId:    "client-cruz",
		PricePlanId: "pp-original",
		Active:      true,
	}}
	uc := newUpdateSubUC(t, ppRepo, clientRepo, subRepo)

	// Partial update: only price plan changing — client_id absent in body, so
	// the use case must resolve from the existing record.
	_, err := uc.Execute(context.Background(), &subscriptionpb.UpdateSubscriptionRequest{
		Data: &subscriptionpb.Subscription{
			Id:          "sub-1",
			PricePlanId: "pp-other",
		},
	})
	if err == nil {
		t.Fatalf("expected planClientMismatch error on cross-client repoint")
	}
}

func TestUpdateSubscription_RepointWithinSameClient_Success(t *testing.T) {
	ppRepo := &mockPricePlanRepoSub{pp: makePricePlan("pp-2", "client-cruz", "PHP")}
	clientRepo := &mockClientRepoSub{c: makeClient("client-cruz")}
	subRepo := &mockSubRepo{existing: &subscriptionpb.Subscription{
		Id:          "sub-1",
		ClientId:    "client-cruz",
		PricePlanId: "pp-original",
		Active:      true,
	}}
	uc := newUpdateSubUC(t, ppRepo, clientRepo, subRepo)

	_, err := uc.Execute(context.Background(), &subscriptionpb.UpdateSubscriptionRequest{
		Data: &subscriptionpb.Subscription{
			Id:          "sub-1",
			ClientId:    "client-cruz",
			PricePlanId: "pp-2",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
