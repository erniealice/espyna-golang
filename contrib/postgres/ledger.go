package postgres

import (
	"database/sql"
	"reflect"

	ledgeradapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/ledger"
	"github.com/erniealice/espyna-golang/registry"
)

func init() {
	registry.RegisterLedgerReportingFactory(func(db any, config any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}

		cfg := ledgeradapter.TableConfig{}
		v := reflect.ValueOf(config)
		if v.Kind() == reflect.Struct {
			cfg.Revenue = getStringField(v, "Revenue")
			cfg.RevenueLineItem = getStringField(v, "RevenueLineItem")
			cfg.InventoryTransaction = getStringField(v, "InventoryTransaction")
			cfg.InventoryItem = getStringField(v, "InventoryItem")
			cfg.Product = getStringField(v, "Product")
			cfg.Location = getStringField(v, "Location")
			cfg.RevenueCategory = getStringField(v, "RevenueCategory")
			cfg.Expenditure = getStringField(v, "Expenditure")
			cfg.ProductCollection = getStringField(v, "ProductCollection")
			cfg.Collection = getStringField(v, "Collection")
			cfg.LocationArea = getStringField(v, "LocationArea")
		}
		return ledgeradapter.NewLedgerReportingAdapter(sqlDB, cfg)
	})
}

func getStringField(v reflect.Value, name string) string {
	f := v.FieldByName(name)
	if f.IsValid() && f.Kind() == reflect.String {
		return f.String()
	}
	return ""
}
