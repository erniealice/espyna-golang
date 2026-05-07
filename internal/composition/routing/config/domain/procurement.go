package domain

import (
	"fmt"

	procurementuc "github.com/erniealice/espyna-golang/internal/application/usecases/procurement"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// ConfigureProcurementDomain configures routes for the Procurement domain with use cases injected directly.
// Masterlists: CostSchedule, SupplierPlan, CostPlan, SupplierProductPlan, SupplierSubscription.
// SupplierProductCostPlan is an inline child editor — no standalone Masterlist routes.
func ConfigureProcurementDomain(procurementUseCases *procurementuc.ProcurementUseCases) contracts.DomainRouteConfiguration {
	if procurementUseCases == nil {
		fmt.Printf("WARNING: Procurement use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "procurement",
			Prefix:  "/procurement",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	fmt.Printf("Procurement use cases are properly initialized!\n")

	routes := []contracts.RouteConfiguration{}

	// CostSchedule module routes
	if procurementUseCases.CostSchedule != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-schedule/create",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostSchedule.CreateCostSchedule, &costschedulepb.CreateCostScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-schedule/read",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostSchedule.ReadCostSchedule, &costschedulepb.ReadCostScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-schedule/update",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostSchedule.UpdateCostSchedule, &costschedulepb.UpdateCostScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-schedule/delete",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostSchedule.DeleteCostSchedule, &costschedulepb.DeleteCostScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-schedule/list",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostSchedule.ListCostSchedules, &costschedulepb.ListCostSchedulesRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-schedule/get-list-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostSchedule.GetCostScheduleListPageData, &costschedulepb.GetCostScheduleListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-schedule/get-item-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostSchedule.GetCostScheduleItemPageData, &costschedulepb.GetCostScheduleItemPageDataRequest{}),
		})
	}

	// SupplierPlan module routes
	if procurementUseCases.SupplierPlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-plan/create",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierPlan.CreateSupplierPlan, &supplierplanpb.CreateSupplierPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-plan/read",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierPlan.ReadSupplierPlan, &supplierplanpb.ReadSupplierPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-plan/update",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierPlan.UpdateSupplierPlan, &supplierplanpb.UpdateSupplierPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-plan/delete",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierPlan.DeleteSupplierPlan, &supplierplanpb.DeleteSupplierPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-plan/list",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierPlan.ListSupplierPlans, &supplierplanpb.ListSupplierPlansRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierPlan.GetSupplierPlanListPageData, &supplierplanpb.GetSupplierPlanListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierPlan.GetSupplierPlanItemPageData, &supplierplanpb.GetSupplierPlanItemPageDataRequest{}),
		})
	}

	// CostPlan module routes
	if procurementUseCases.CostPlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-plan/create",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostPlan.CreateCostPlan, &costplanpb.CreateCostPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-plan/read",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostPlan.ReadCostPlan, &costplanpb.ReadCostPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-plan/update",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostPlan.UpdateCostPlan, &costplanpb.UpdateCostPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-plan/delete",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostPlan.DeleteCostPlan, &costplanpb.DeleteCostPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-plan/list",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostPlan.ListCostPlans, &costplanpb.ListCostPlansRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostPlan.GetCostPlanListPageData, &costplanpb.GetCostPlanListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/cost-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.CostPlan.GetCostPlanItemPageData, &costplanpb.GetCostPlanItemPageDataRequest{}),
		})
	}

	// SupplierProductPlan module routes
	if procurementUseCases.SupplierProductPlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-product-plan/create",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierProductPlan.CreateSupplierProductPlan, &supplierproductplanpb.CreateSupplierProductPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-product-plan/read",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierProductPlan.ReadSupplierProductPlan, &supplierproductplanpb.ReadSupplierProductPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-product-plan/update",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierProductPlan.UpdateSupplierProductPlan, &supplierproductplanpb.UpdateSupplierProductPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-product-plan/delete",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierProductPlan.DeleteSupplierProductPlan, &supplierproductplanpb.DeleteSupplierProductPlanRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-product-plan/list",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierProductPlan.ListSupplierProductPlans, &supplierproductplanpb.ListSupplierProductPlansRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-product-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierProductPlan.GetSupplierProductPlanListPageData, &supplierproductplanpb.GetSupplierProductPlanListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-product-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierProductPlan.GetSupplierProductPlanItemPageData, &supplierproductplanpb.GetSupplierProductPlanItemPageDataRequest{}),
		})
	}

	// SupplierSubscription module routes
	if procurementUseCases.SupplierSubscription != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/create",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.CreateSupplierSubscription, &suppliersubscriptionpb.CreateSupplierSubscriptionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/read",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.ReadSupplierSubscription, &suppliersubscriptionpb.ReadSupplierSubscriptionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/update",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.UpdateSupplierSubscription, &suppliersubscriptionpb.UpdateSupplierSubscriptionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/delete",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.DeleteSupplierSubscription, &suppliersubscriptionpb.DeleteSupplierSubscriptionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/list",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.ListSupplierSubscriptions, &suppliersubscriptionpb.ListSupplierSubscriptionsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/get-list-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.GetSupplierSubscriptionListPageData, &suppliersubscriptionpb.GetSupplierSubscriptionListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/get-item-page-data",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.GetSupplierSubscriptionItemPageData, &suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/count-active-by-supplier-ids",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.CountActiveBySupplierIds, &suppliersubscriptionpb.CountActiveBySupplierIdsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/procurement/supplier-subscription/list-by-cost-plan",
			Handler: contracts.NewGenericHandler(procurementUseCases.SupplierSubscription.ListSupplierSubscriptionsByCostPlan, &suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "procurement",
		Prefix:  "/procurement",
		Enabled: true,
		Routes:  routes,
	}
}
