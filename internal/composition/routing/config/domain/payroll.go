package domain

import (
	"fmt"

	payrolluc "github.com/erniealice/espyna-golang/internal/application/usecases/payroll"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// ConfigurePayrollDomain configures routes for the Payroll domain.
func ConfigurePayrollDomain(payrollUseCases *payrolluc.PayrollUseCases) contracts.DomainRouteConfiguration {
	if payrollUseCases == nil {
		fmt.Printf("Payroll use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "payroll",
			Prefix:  "/payroll",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// PayrollRun routes
	if payrollUseCases.PayrollRun != nil {
		if payrollUseCases.PayrollRun.CreatePayrollRun != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/payroll/payroll-run/create",
				Handler: contracts.NewGenericHandler(payrollUseCases.PayrollRun.CreatePayrollRun, &payrollrunpb.CreatePayrollRunRequest{}),
			})
		}
		if payrollUseCases.PayrollRun.ReadPayrollRun != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/payroll/payroll-run/read",
				Handler: contracts.NewGenericHandler(payrollUseCases.PayrollRun.ReadPayrollRun, &payrollrunpb.ReadPayrollRunRequest{}),
			})
		}
		if payrollUseCases.PayrollRun.ListPayrollRuns != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/payroll/payroll-run/list",
				Handler: contracts.NewGenericHandler(payrollUseCases.PayrollRun.ListPayrollRuns, &payrollrunpb.ListPayrollRunsRequest{}),
			})
		}
		if payrollUseCases.PayrollRun.GetPayrollRunListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/payroll/payroll-run/get-list-page-data",
				Handler: contracts.NewGenericHandler(payrollUseCases.PayrollRun.GetPayrollRunListPageData, &payrollrunpb.GetPayrollRunListPageDataRequest{}),
			})
		}
	}

	// PayrollRemittance routes
	if payrollUseCases.PayrollRemittance != nil {
		if payrollUseCases.PayrollRemittance.CreatePayrollRemittance != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/payroll/payroll-remittance/create",
				Handler: contracts.NewGenericHandler(payrollUseCases.PayrollRemittance.CreatePayrollRemittance, &payrollremittancepb.CreatePayrollRemittanceRequest{}),
			})
		}
		if payrollUseCases.PayrollRemittance.ListPayrollRemittances != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/payroll/payroll-remittance/list",
				Handler: contracts.NewGenericHandler(payrollUseCases.PayrollRemittance.ListPayrollRemittances, &payrollremittancepb.ListPayrollRemittancesRequest{}),
			})
		}
		if payrollUseCases.PayrollRemittance.GetPayrollRemittanceListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/payroll/payroll-remittance/get-list-page-data",
				Handler: contracts.NewGenericHandler(payrollUseCases.PayrollRemittance.GetPayrollRemittanceListPageData, &payrollremittancepb.GetPayrollRemittanceListPageDataRequest{}),
			})
		}
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "payroll",
		Prefix:  "/payroll",
		Enabled: len(routes) > 0,
		Routes:  routes,
	}
}
