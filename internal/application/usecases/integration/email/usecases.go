// Package email provides use cases for email integration (Gmail, etc.)
//
// # Adding New Use Cases
//
// When adding a new use case to this package, remember to update:
//
//  1. UseCases struct - Add the new use case field
//  2. NewUseCases() - Initialize the new use case
//  3. Routing config - packages/espyna/internal/composition/routing/config/integration/email.go
//  4. Workflow registry - packages/espyna/internal/orchestration/workflow/integration/email.go
//
// # Use Case Types
//
// All email use cases are proto-based and can be exposed via HTTP routing AND workflow activities.
package email

import (
	"leapfor.xyz/espyna/internal/application/ports"
)

// EmailRepositories groups all repository dependencies for email use cases
type EmailRepositories struct {
	// No repositories needed for external email provider integration
}

// EmailServices groups all business service dependencies for email use cases
type EmailServices struct {
	Provider ports.EmailProvider
}

// UseCases contains all email integration use cases
type UseCases struct {
	SendEmail       *SendEmailUseCase
	CheckHealth     *CheckHealthUseCase
	GetCapabilities *GetCapabilitiesUseCase
}

// NewUseCases creates a new collection of email integration use cases
func NewUseCases(
	repositories EmailRepositories,
	services EmailServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	sendEmailRepos := SendEmailRepositories{}
	sendEmailServices := SendEmailServices{
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
		SendEmail:       NewSendEmailUseCase(sendEmailRepos, sendEmailServices),
		CheckHealth:     NewCheckHealthUseCase(checkHealthRepos, checkHealthServices),
		GetCapabilities: NewGetCapabilitiesUseCase(getCapabilitiesRepos, getCapabilitiesServices),
	}
}

// NewUseCasesFromProvider creates use cases directly from an email provider
// This is a convenience function for simple setups
func NewUseCasesFromProvider(provider ports.EmailProvider) *UseCases {
	if provider == nil {
		return nil
	}

	repositories := EmailRepositories{}
	services := EmailServices{
		Provider: provider,
	}

	return NewUseCases(repositories, services)
}
