package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Payroll domain
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// PayrollRepositories contains all payroll domain repositories
type PayrollRepositories struct {
	PayrollRun        payrollrunpb.PayrollRunDomainServiceServer
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
}

// NewPayrollRepositories creates and returns a new set of PayrollRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewPayrollRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*PayrollRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &PayrollRepositories{}
	var skipped []string

	// Helper: try to create a repository, log and skip on failure
	tryCreate := func(entity string) interface{} {
		repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
		if err != nil {
			skipped = append(skipped, entity)
			return nil
		}
		return repo
	}

	if r := tryCreate(entityid.PayrollRun); r != nil {
		repos.PayrollRun = r.(payrollrunpb.PayrollRunDomainServiceServer)
	}
	if r := tryCreate(entityid.PayrollRemittance); r != nil {
		repos.PayrollRemittance = r.(payrollremittancepb.PayrollRemittanceDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Payroll repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
