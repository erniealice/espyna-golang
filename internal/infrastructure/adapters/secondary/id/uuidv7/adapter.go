//go:build uuidv7

// Package uuidv7 provides the pure-standard-library UUIDv7 ID adapter.
package uuidv7

import (
	"fmt"
	"uuid"

	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
)

const providerName = "uuidv7"

func init() {
	registry.RegisterIDProvider(
		providerName,
		func() ports.IDGenerator { return NewService() },
		transformConfig,
	)
	registry.RegisterIDBuildFromEnv(providerName, buildFromEnv)
}

func buildFromEnv() (ports.IDGenerator, error) {
	return NewService(), nil
}

func transformConfig(map[string]any) (*registry.IDProviderConfig, error) {
	return &registry.IDProviderConfig{Provider: providerName, Enabled: true}, nil
}

// Service generates time-ordered RFC 9562 UUIDv7 identifiers.
type Service struct {
	enabled bool
}

// NewService returns an enabled UUIDv7 service.
func NewService() ports.IDGenerator {
	return &Service{enabled: true}
}

func (s *Service) Name() string { return providerName }

func (s *Service) GenerateID() string {
	if !s.enabled {
		return ports.NewNoOpIDGenerator().GenerateID()
	}
	return uuid.NewV7().String()
}

func (s *Service) GenerateIDWithPrefix(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, s.GenerateID())
}

func (s *Service) IsEnabled() bool { return s.enabled }

func (s *Service) GetProviderInfo() string { return "Go standard library UUID v7 service" }

var _ ports.IDGenerator = (*Service)(nil)
