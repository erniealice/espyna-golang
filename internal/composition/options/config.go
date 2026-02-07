package options

import (
	infra "leapfor.xyz/espyna/internal/composition/options/infrastructure"
	"leapfor.xyz/espyna/internal/composition/options/integrations/messaging"
	"leapfor.xyz/espyna/internal/composition/options/integrations/payment"
)

// ManagerConfig holds configuration for all providers.
// This aggregates infrastructure and integration configs for unified initialization.
type ManagerConfig struct {
	// Infrastructure providers
	Database *infra.DatabaseConfig `json:"database,omitempty"`
	Auth     *infra.AuthConfig     `json:"auth,omitempty"`
	Storage  *infra.StorageConfig  `json:"storage,omitempty"`
	ID       *infra.IDConfig       `json:"id,omitempty"`

	// Integration providers
	Email   *messaging.EmailConfig `json:"email,omitempty"`
	Payment *payment.PaymentConfig `json:"payment,omitempty"`
}
