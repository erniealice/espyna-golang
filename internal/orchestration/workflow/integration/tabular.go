package integration

import (
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases"
	"leapfor.xyz/espyna/internal/orchestration/workflow/executor"
)

// RegisterTabularIntegrationUseCases registers all tabular integration use cases with the registry.
// Tabular integration includes: ReadRecords, WriteRecords, UpdateRecords, DeleteRecords,
// SearchRecords, GetSchema, GetSource, ListTables, BatchExecute, CheckHealth, GetCapabilities.
func RegisterTabularIntegrationUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Integration == nil {
		log.Printf("[WorkflowRegistry] ⚠️ Skipping tabular registration: Integration is nil")
		return
	}
	if useCases.Integration.Tabular == nil {
		log.Printf("[WorkflowRegistry] ⚠️ Skipping tabular registration: Integration.Tabular is nil")
		return
	}
	log.Printf("[WorkflowRegistry] ✅ Registering tabular integration use cases (WriteRecordSimple: %v)", useCases.Integration.Tabular.WriteRecordSimple != nil)

	// Read records from tabular source
	if useCases.Integration.Tabular.ReadRecords != nil {
		register("integration.tabular.read_records", executor.New(useCases.Integration.Tabular.ReadRecords.Execute))
	}

	// Write records to tabular source (INSERT)
	if useCases.Integration.Tabular.WriteRecords != nil {
		register("integration.tabular.write_records", executor.New(useCases.Integration.Tabular.WriteRecords.Execute))
	}

	// Write single record with flat input (workflow-friendly, proto-based)
	if useCases.Integration.Tabular.WriteRecordSimple != nil {
		register("integration.tabular.write_record_simple", executor.New(useCases.Integration.Tabular.WriteRecordSimple.Execute))
	}

	// Update records in tabular source
	if useCases.Integration.Tabular.UpdateRecords != nil {
		register("integration.tabular.update_records", executor.New(useCases.Integration.Tabular.UpdateRecords.Execute))
	}

	// Delete records from tabular source
	if useCases.Integration.Tabular.DeleteRecords != nil {
		register("integration.tabular.delete_records", executor.New(useCases.Integration.Tabular.DeleteRecords.Execute))
	}

	// Search records in tabular source
	if useCases.Integration.Tabular.SearchRecords != nil {
		register("integration.tabular.search_records", executor.New(useCases.Integration.Tabular.SearchRecords.Execute))
	}

	// Get schema of tabular source
	if useCases.Integration.Tabular.GetSchema != nil {
		register("integration.tabular.get_schema", executor.New(useCases.Integration.Tabular.GetSchema.Execute))
	}

	// Get source configuration
	if useCases.Integration.Tabular.GetSource != nil {
		register("integration.tabular.get_source", executor.New(useCases.Integration.Tabular.GetSource.Execute))
	}

	// List available tables
	if useCases.Integration.Tabular.ListTables != nil {
		register("integration.tabular.list_tables", executor.New(useCases.Integration.Tabular.ListTables.Execute))
	}

	// Batch execute multiple operations
	if useCases.Integration.Tabular.BatchExecute != nil {
		register("integration.tabular.batch_execute", executor.New(useCases.Integration.Tabular.BatchExecute.Execute))
	}

	// Check health of tabular provider
	if useCases.Integration.Tabular.CheckHealth != nil {
		register("integration.tabular.check_health", executor.New(useCases.Integration.Tabular.CheckHealth.Execute))
	}

	// Get capabilities of tabular provider
	if useCases.Integration.Tabular.GetCapabilities != nil {
		register("integration.tabular.get_capabilities", executor.New(useCases.Integration.Tabular.GetCapabilities.Execute))
	}
}
