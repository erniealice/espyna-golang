// Package scheduler provides use cases for scheduler integration (Calendly, etc.)
//
// # Adding New Use Cases
//
// When adding a new use case to this package, remember to update:
//
//  1. UseCases struct - Add the new use case field
//  2. NewUseCases() - Initialize the new use case
//  3. Routing config - packages/espyna/internal/composition/routing/config/integration/scheduler.go (create if needed)
//  4. Workflow registry - packages/espyna/internal/orchestration/workflow/integration/scheduler.go (create if needed)
//
// # Use Case Types
//
// All scheduler use cases are proto-based and can be exposed via HTTP routing AND workflow activities.
package scheduler

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// SchedulerRepositories groups all repository dependencies for scheduler use cases
type SchedulerRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// SchedulerServices groups all business service dependencies for scheduler use cases
type SchedulerServices struct {
	Provider ports.SchedulerProvider
}

// UseCases contains all scheduler integration use cases
type UseCases struct {
	CreateSchedule    *CreateScheduleUseCase
	CancelSchedule    *CancelScheduleUseCase
	GetSchedule       *GetScheduleUseCase
	ListSchedules     *ListSchedulesUseCase
	CheckAvailability *CheckAvailabilityUseCase
	ProcessWebhook    *ProcessWebhookUseCase
	ListEventTypes    *ListEventTypesUseCase
	GetEventType      *GetEventTypeUseCase
	CheckHealth       *CheckHealthUseCase
	GetCapabilities   *GetCapabilitiesUseCase
}

// NewUseCases creates a new collection of scheduler integration use cases
func NewUseCases(
	repositories SchedulerRepositories,
	services SchedulerServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createScheduleRepos := CreateScheduleRepositories{}
	createScheduleServices := CreateScheduleServices{
		Provider: services.Provider,
	}

	cancelScheduleRepos := CancelScheduleRepositories{}
	cancelScheduleServices := CancelScheduleServices{
		Provider: services.Provider,
	}

	getScheduleRepos := GetScheduleRepositories{}
	getScheduleServices := GetScheduleServices{
		Provider: services.Provider,
	}

	listSchedulesRepos := ListSchedulesRepositories{}
	listSchedulesServices := ListSchedulesServices{
		Provider: services.Provider,
	}

	checkAvailabilityRepos := CheckAvailabilityRepositories{}
	checkAvailabilityServices := CheckAvailabilityServices{
		Provider: services.Provider,
	}

	processWebhookRepos := ProcessWebhookRepositories{}
	processWebhookServices := ProcessWebhookServices{
		Provider: services.Provider,
	}

	listEventTypesRepos := ListEventTypesRepositories{}
	listEventTypesServices := ListEventTypesServices{
		Provider: services.Provider,
	}

	getEventTypeRepos := GetEventTypeRepositories{}
	getEventTypeServices := GetEventTypeServices{
		Provider: services.Provider,
	}

	checkHealthRepos := CheckHealthRepositories{}
	checkHealthServices := CheckHealthServices{
		Provider: services.Provider,
	}

	getCapabilitiesRepos := GetCapabilitiesRepositories{}
	getCapabilitiesServices := GetCapabilitiesServices{
		Provider: services.Provider,
	}

	return &UseCases{
		CreateSchedule:    NewCreateScheduleUseCase(createScheduleRepos, createScheduleServices),
		CancelSchedule:    NewCancelScheduleUseCase(cancelScheduleRepos, cancelScheduleServices),
		GetSchedule:       NewGetScheduleUseCase(getScheduleRepos, getScheduleServices),
		ListSchedules:     NewListSchedulesUseCase(listSchedulesRepos, listSchedulesServices),
		CheckAvailability: NewCheckAvailabilityUseCase(checkAvailabilityRepos, checkAvailabilityServices),
		ProcessWebhook:    NewProcessWebhookUseCase(processWebhookRepos, processWebhookServices),
		ListEventTypes:    NewListEventTypesUseCase(listEventTypesRepos, listEventTypesServices),
		GetEventType:      NewGetEventTypeUseCase(getEventTypeRepos, getEventTypeServices),
		CheckHealth:       NewCheckHealthUseCase(checkHealthRepos, checkHealthServices),
		GetCapabilities:   NewGetCapabilitiesUseCase(getCapabilitiesRepos, getCapabilitiesServices),
	}
}

// NewUseCasesFromProvider creates use cases directly from a scheduler provider
// This is a convenience function for simple setups
func NewUseCasesFromProvider(provider ports.SchedulerProvider) *UseCases {
	if provider == nil {
		return nil
	}

	repositories := SchedulerRepositories{}
	services := SchedulerServices{
		Provider: provider,
	}

	return NewUseCases(repositories, services)
}
