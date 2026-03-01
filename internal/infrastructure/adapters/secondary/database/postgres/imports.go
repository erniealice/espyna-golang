//go:build postgresql

package postgres

import (
	// Repository sub-packages - each registers its factory via init()
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/attribute_value"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/common"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/entity"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/event"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/integrations"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/inventory_attribute"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/inventory_depreciation"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/inventory_item"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/inventory_serial"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/inventory_serial_history"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/inventory_transaction"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/revenue"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/revenue_attribute"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/revenue_category"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/revenue_line_item"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/product"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/product_option"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/product_option_value"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/product_variant"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/product_variant_image"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/product_variant_option"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/subscription"
)
