
package postgres

import (
	// Repository sub-packages - each registers its factory via init()
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/attribute_value"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/common"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/entity"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/event"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/expenditure"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/integrations"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/inventory_attribute"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/inventory_depreciation"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/inventory_item"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/inventory_serial"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/inventory_serial_history"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/inventory_transaction"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/revenue"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/revenue_attribute"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/revenue_category"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/revenue_line_item"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/product"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/product_option"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/product_option_value"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/product_variant"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/product_variant_image"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/product_variant_option"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/subscription"
)
