package registry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

// =============================================================================
// Scheduler Factory Registry Instance
// =============================================================================

var schedulerRegistry = NewFactoryRegistry[integration.SchedulerProvider, *schedulerpb.SchedulerProviderConfig]("scheduler")

// =============================================================================
// Scheduler Provider Functions
// =============================================================================

func RegisterSchedulerProviderFactory(name string, factory func() integration.SchedulerProvider) {
	schedulerRegistry.RegisterFactory(name, factory)
}

func GetSchedulerProviderFactory(name string) (func() integration.SchedulerProvider, bool) {
	return schedulerRegistry.GetFactory(name)
}

func ListAvailableSchedulerProviderFactories() []string {
	return schedulerRegistry.ListFactories()
}

type SchedulerConfigTransformer func(rawConfig map[string]any) (*schedulerpb.SchedulerProviderConfig, error)

func RegisterSchedulerConfigTransformer(name string, transformer SchedulerConfigTransformer) {
	schedulerRegistry.RegisterConfigTransformer(name, transformer)
}

func GetSchedulerConfigTransformer(name string) (SchedulerConfigTransformer, bool) {
	return schedulerRegistry.GetConfigTransformer(name)
}

func TransformSchedulerConfig(name string, rawConfig map[string]any) (*schedulerpb.SchedulerProviderConfig, error) {
	return schedulerRegistry.TransformConfig(name, rawConfig)
}

func RegisterSchedulerBuildFromEnv(name string, builder func() (integration.SchedulerProvider, error)) {
	schedulerRegistry.RegisterBuildFromEnv(name, builder)
}

func GetSchedulerBuildFromEnv(name string) (func() (integration.SchedulerProvider, error), bool) {
	return schedulerRegistry.GetBuildFromEnv(name)
}

func BuildSchedulerProviderFromEnv(name string) (integration.SchedulerProvider, error) {
	return schedulerRegistry.BuildFromEnv(name)
}

func ListAvailableSchedulerBuildFromEnv() []string {
	return schedulerRegistry.ListBuildFromEnv()
}

func RegisterSchedulerProvider(name string, factory func() integration.SchedulerProvider, transformer SchedulerConfigTransformer) {
	RegisterSchedulerProviderFactory(name, factory)
	if transformer != nil {
		RegisterSchedulerConfigTransformer(name, transformer)
	}
}
