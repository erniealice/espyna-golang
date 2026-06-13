//go:build mock_tabular

package consumer

// Activates the mock tabular adapter under -tags mock_tabular.
import _ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/tabular/mock"
