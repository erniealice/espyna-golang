//go:build google && googlesheets

// Package integration provides HTTP routing configuration for integration use cases.
//
// # Tabular Integration Routes
//
// This file configures HTTP endpoints for tabular data operations (Google Sheets, etc.)
// All use cases are proto-based and exposed via HTTP routing.
//
// # Available Endpoints
//
//   - POST /integration/tabular/write - Full WriteRecords with complex Record structure
//   - POST /integration/tabular/write-simple - Simplified single record write with flat fields
//
// # Keeping in Sync
//
// When adding new tabular use cases, update:
//   - This file (for HTTP routing)
//   - packages/espyna/internal/orchestration/workflow/integration/tabular.go (for workflows)
//
// When adding a NEW integration type, also update:
//   - packages/espyna/internal/composition/routing/config/config.go (to register the integration)
package integration

import (
	"leapfor.xyz/espyna/internal/application/ports"
	integrationuc "leapfor.xyz/espyna/internal/application/usecases/integration"
	"leapfor.xyz/espyna/internal/composition/contracts"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// Ensure ports is used (for interface compatibility)
var _ ports.TabularSourceProvider = nil

// ConfigureTabularIntegration configures routes for tabular integration
// This is only compiled when both 'google' and 'googlesheets' build tags are present
//
// Note: WriteRecordSimple is NOT exposed here - it's a workflow-only adapter.
// See packages/espyna/internal/orchestration/workflow/integration/tabular.go
func ConfigureTabularIntegration(
	_ ports.TabularSourceProvider, // Kept for backward compatibility
	integration *integrationuc.IntegrationUseCases,
) contracts.DomainRouteConfiguration {
	// Check if tabular use cases are available (not just the provider)
	if integration == nil || integration.Tabular == nil {
		return contracts.DomainRouteConfiguration{
			Domain:  "tabular_integration",
			Prefix:  "/integration/tabular",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// Read records endpoint
	if integration.Tabular.ReadRecords != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/read",
			Handler: contracts.NewGenericHandler(integration.Tabular.ReadRecords, &tabularpb.ReadRecordsRequest{}),
		})
	}

	// Write records endpoint
	if integration.Tabular.WriteRecords != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/write",
			Handler: contracts.NewGenericHandler(integration.Tabular.WriteRecords, &tabularpb.WriteRecordsRequest{}),
		})
	}

	// Write single record endpoint (workflow-friendly flat input)
	if integration.Tabular.WriteRecordSimple != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/write-simple",
			Handler: contracts.NewGenericHandler(integration.Tabular.WriteRecordSimple, &tabularpb.WriteRecordSimpleRequest{}),
		})
	}

	// Update records endpoint
	if integration.Tabular.UpdateRecords != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/update",
			Handler: contracts.NewGenericHandler(integration.Tabular.UpdateRecords, &tabularpb.UpdateRecordsRequest{}),
		})
	}

	// Delete records endpoint
	if integration.Tabular.DeleteRecords != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/delete",
			Handler: contracts.NewGenericHandler(integration.Tabular.DeleteRecords, &tabularpb.DeleteRecordsRequest{}),
		})
	}

	// Search records endpoint
	if integration.Tabular.SearchRecords != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/search",
			Handler: contracts.NewGenericHandler(integration.Tabular.SearchRecords, &tabularpb.SearchRecordsRequest{}),
		})
	}

	// Get schema endpoint
	if integration.Tabular.GetSchema != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/schema",
			Handler: contracts.NewGenericHandler(integration.Tabular.GetSchema, &tabularpb.GetSchemaRequest{}),
		})
	}

	// Get source endpoint
	if integration.Tabular.GetSource != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/source",
			Handler: contracts.NewGenericHandler(integration.Tabular.GetSource, &tabularpb.GetSourceRequest{}),
		})
	}

	// List tables endpoint
	if integration.Tabular.ListTables != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/tables",
			Handler: contracts.NewGenericHandler(integration.Tabular.ListTables, &tabularpb.ListTablesRequest{}),
		})
	}

	// Batch execute endpoint
	if integration.Tabular.BatchExecute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/tabular/batch",
			Handler: contracts.NewGenericHandler(integration.Tabular.BatchExecute, &tabularpb.BatchExecuteRequest{}),
		})
	}

	// Health check endpoint
	if integration.Tabular.CheckHealth != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "GET",
			Path:    "/integration/tabular/health",
			Handler: contracts.NewGenericHandler(integration.Tabular.CheckHealth, &tabularpb.CheckHealthRequest{}),
		})
	}

	// Get capabilities endpoint
	if integration.Tabular.GetCapabilities != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "GET",
			Path:    "/integration/tabular/capabilities",
			Handler: contracts.NewGenericHandler(integration.Tabular.GetCapabilities, &tabularpb.GetCapabilitiesRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "tabular_integration",
		Prefix:  "/integration/tabular",
		Enabled: true,
		Routes:  routes,
	}
}
