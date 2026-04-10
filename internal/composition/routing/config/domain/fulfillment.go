package domain

import (
	"fmt"

	fulfillmentuc "github.com/erniealice/espyna-golang/internal/application/usecases/fulfillment"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ConfigureFulfillmentDomain configures routes for the Fulfillment domain.
func ConfigureFulfillmentDomain(fulfillmentUseCases *fulfillmentuc.UseCases) contracts.DomainRouteConfiguration {
	if fulfillmentUseCases == nil {
		fmt.Printf("Fulfillment use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "fulfillment",
			Prefix:  "/fulfillment",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	if fulfillmentUseCases.CreateFulfillment != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/create",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.CreateFulfillment, &pb.CreateFulfillmentRequest{}),
		})
	}
	if fulfillmentUseCases.GetFulfillment != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/read",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.GetFulfillment, &pb.GetFulfillmentRequest{}),
		})
	}
	if fulfillmentUseCases.UpdateFulfillment != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/update",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.UpdateFulfillment, &pb.UpdateFulfillmentRequest{}),
		})
	}
	if fulfillmentUseCases.DeleteFulfillment != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/delete",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.DeleteFulfillment, &pb.DeleteFulfillmentRequest{}),
		})
	}
	if fulfillmentUseCases.ListFulfillments != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/list",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.ListFulfillments, &pb.ListFulfillmentsRequest{}),
		})
	}
	if fulfillmentUseCases.GetFulfillmentListPageData != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/get-list-page-data",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.GetFulfillmentListPageData, &pb.GetFulfillmentListPageDataRequest{}),
		})
	}
	if fulfillmentUseCases.GetFulfillmentItemPageData != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/get-item-page-data",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.GetFulfillmentItemPageData, &pb.GetFulfillmentItemPageDataRequest{}),
		})
	}
	if fulfillmentUseCases.TransitionStatus != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/transition-status",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.TransitionStatus, &pb.TransitionStatusRequest{}),
		})
	}
	if fulfillmentUseCases.ListStatusEvents != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/fulfillment/list-status-events",
			Handler: contracts.NewGenericHandler(fulfillmentUseCases.ListStatusEvents, &pb.ListStatusEventsRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "fulfillment",
		Prefix:  "/fulfillment",
		Enabled: len(routes) > 0,
		Routes:  routes,
	}
}
