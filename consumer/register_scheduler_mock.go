//go:build mock_scheduler

package consumer

// Activates the mock scheduler adapter under -tags mock_scheduler.
import _ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/scheduler/mock"
