package domain

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases"
	"leapfor.xyz/espyna/internal/orchestration/workflow/executor"
)

// RegisterCommonUseCases registers all common domain use cases with the registry.
// Common domain includes: Attribute (cross-domain dependency).
func RegisterCommonUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Common == nil {
		return
	}

	// Attribute use cases
	registerAttributeUseCases(useCases, register)
}

// registerAttributeUseCases registers attribute CRUD use cases.
// Attributes are used across multiple domains for flexible key-value metadata.
func registerAttributeUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Common.Attribute == nil {
		return
	}

	if useCases.Common.Attribute.CreateAttribute != nil {
		register("common.attribute.create", executor.New(useCases.Common.Attribute.CreateAttribute.Execute))
	}
	if useCases.Common.Attribute.ReadAttribute != nil {
		register("common.attribute.read", executor.New(useCases.Common.Attribute.ReadAttribute.Execute))
	}
	if useCases.Common.Attribute.UpdateAttribute != nil {
		register("common.attribute.update", executor.New(useCases.Common.Attribute.UpdateAttribute.Execute))
	}
	if useCases.Common.Attribute.DeleteAttribute != nil {
		register("common.attribute.delete", executor.New(useCases.Common.Attribute.DeleteAttribute.Execute))
	}
	if useCases.Common.Attribute.ListAttributes != nil {
		register("common.attribute.list", executor.New(useCases.Common.Attribute.ListAttributes.Execute))
	}
	// Note: Attribute entity doesn't have GetListPageData or GetItemPageData use cases
}
