package expenditure

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// C32: the S1 aggregates are nil unless their own repository is wired.
func TestKnownCostRecoveryAggregatesAreNilWithoutTheirRepositories(t *testing.T) {
	gate := actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
	uc := NewUseCases(ExpenditureRepositories{}, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate, nil)
	if uc.CostSourceComponent != nil || uc.AllocationBatch != nil || uc.AllocationShare != nil {
		t.Fatal("S1 aggregates must be nil without repositories")
	}
	uc = NewUseCases(ExpenditureRepositories{
		CostSourceComponent: costsourcecomponentpb.UnimplementedCostSourceComponentDomainServiceServer{},
		AllocationBatch:     allocationbatchpb.UnimplementedAllocationBatchDomainServiceServer{},
		AllocationShare:     allocationsharepb.UnimplementedAllocationShareDomainServiceServer{},
	}, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate, nil)
	if uc.CostSourceComponent == nil || uc.CostSourceComponent.ReconcileCostSource == nil || uc.AllocationBatch == nil || uc.AllocationShare == nil {
		t.Fatal("S1 aggregates must be built when their repositories are wired")
	}
}
