//go:build postgresql

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
			cfg.ExpenditureLineItem = getStringField(v, "ExpenditureLineItem")
			cfg.ExpenditureCategory = getStringField(v, "ExpenditureCategory")
			cfg.Supplier = getStringField(v, "Supplier")
			cfg.ProductCollection = getStringField(v, "ProductCollection")
			cfg.Collection = getStringField(v, "Collection")
			cfg.Line = getStringField(v, "Line")
			cfg.LocationArea = getStringField(v, "LocationArea")
			cfg.SupplierCategory = getStringField(v, "SupplierCategory")
			cfg.TreasuryDisbursement = getStringField(v, "TreasuryDisbursement")
			cfg.DisbursementMethod = getStringField(v, "DisbursementMethod")
			cfg.Client = getStringField(v, "Client")
			cfg.ClientCategory = getStringField(v, "ClientCategory")
			cfg.Category = getStringField(v, "Category")
			cfg.TreasuryCollection = getStringField(v, "TreasuryCollection")
			cfg.CollectionMethod = getStringField(v, "CollectionMethod")
			cfg.PaymentTerm = getStringField(v, "PaymentTerm")
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
