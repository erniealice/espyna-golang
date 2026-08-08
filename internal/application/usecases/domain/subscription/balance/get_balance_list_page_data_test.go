package balance

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

type balancePageRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	listCalls      int
	pageCalls      int
	pageRequest    *balancepb.GetBalanceListPageDataRequest
	supportsPaging bool
}

func (r *balancePageRepository) ListBalances(context.Context, *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
	r.listCalls++
	return &balancepb.ListBalancesResponse{Data: []*balancepb.Balance{{Id: "balance-1"}}}, nil
}

func (r *balancePageRepository) GetBalanceListPageData(_ context.Context, req *balancepb.GetBalanceListPageDataRequest) (*balancepb.GetBalanceListPageDataResponse, error) {
	r.pageCalls++
	r.pageRequest = req
	return &balancepb.GetBalanceListPageDataResponse{Success: true}, nil
}

func (r *balancePageRepository) SupportsBalanceServerPagination() bool {
	return r.supportsPaging
}

type balanceFallbackRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	listCalls int
	pageCalls int
}

func (r *balanceFallbackRepository) ListBalances(context.Context, *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
	r.listCalls++
	return &balancepb.ListBalancesResponse{Data: []*balancepb.Balance{{Id: "balance-1"}}}, nil
}

func (r *balanceFallbackRepository) GetBalanceListPageData(context.Context, *balancepb.GetBalanceListPageDataRequest) (*balancepb.GetBalanceListPageDataResponse, error) {
	r.pageCalls++
	return &balancepb.GetBalanceListPageDataResponse{Success: true}, nil
}

func newBalanceListPageDataUseCase(repository balancepb.BalanceDomainServiceServer) *GetBalanceListPageDataUseCase {
	return NewGetBalanceListPageDataUseCase(
		GetBalanceListPageDataRepositories{Balance: repository},
		GetBalanceListPageDataServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func TestGetBalanceListPageDataUseCase_Execute_UsesOptInServerPagination(t *testing.T) {
	repository := &balancePageRepository{supportsPaging: true}
	req := &balancepb.GetBalanceListPageDataRequest{}

	response, err := newBalanceListPageDataUseCase(repository).Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response == nil || !response.Success {
		t.Fatalf("Execute() response = %#v, want successful server response", response)
	}
	if repository.pageCalls != 1 {
		t.Fatalf("GetBalanceListPageData calls = %d, want 1", repository.pageCalls)
	}
	if repository.pageRequest != req {
		t.Fatal("GetBalanceListPageData did not receive the exact request")
	}
	if repository.listCalls != 0 {
		t.Fatalf("ListBalances calls = %d, want 0", repository.listCalls)
	}
}

func TestGetBalanceListPageDataUseCase_Execute_FallsBackWithoutCapability(t *testing.T) {
	repository := &balanceFallbackRepository{}

	response, err := newBalanceListPageDataUseCase(repository).Execute(context.Background(), &balancepb.GetBalanceListPageDataRequest{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response == nil || !response.Success {
		t.Fatalf("Execute() response = %#v, want successful fallback response", response)
	}
	if repository.listCalls != 1 {
		t.Fatalf("ListBalances calls = %d, want 1", repository.listCalls)
	}
	if repository.pageCalls != 0 {
		t.Fatalf("GetBalanceListPageData calls = %d, want 0", repository.pageCalls)
	}
}

func TestGetBalanceListPageDataUseCase_Execute_FallsBackWhenCapabilityDeclines(t *testing.T) {
	repository := &balancePageRepository{supportsPaging: false}

	response, err := newBalanceListPageDataUseCase(repository).Execute(context.Background(), &balancepb.GetBalanceListPageDataRequest{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response == nil || !response.Success {
		t.Fatalf("Execute() response = %#v, want successful fallback response", response)
	}
	if repository.listCalls != 1 {
		t.Fatalf("ListBalances calls = %d, want 1", repository.listCalls)
	}
	if repository.pageCalls != 0 {
		t.Fatalf("GetBalanceListPageData calls = %d, want 0", repository.pageCalls)
	}
}
