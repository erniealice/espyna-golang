//go:build mysql

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.TaxClass, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql tax_class repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLTaxClassRepository(db, dbOps, tableName), nil
	})
}

// MySQLTaxClassRepository implements tax_class read operations using MySQL 8.0+.
type MySQLTaxClassRepository struct {
	taxclasspb.UnimplementedTaxClassDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLTaxClassRepository creates a new MySQL tax_class repository.
func NewMySQLTaxClassRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxclasspb.TaxClassDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxClass
	}
	return &MySQLTaxClassRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxClass(raw map[string]any) (*taxclasspb.TaxClass, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	c := &taxclasspb.TaxClass{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, c); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return c, nil
}

// ReadTaxClass retrieves a tax_class record by ID.
func (r *MySQLTaxClassRepository) ReadTaxClass(ctx context.Context, req *taxclasspb.ReadTaxClassRequest) (*taxclasspb.ReadTaxClassResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_class ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tax_class: %w", err)
	}
	c, err := unmarshalTaxClass(result)
	if err != nil {
		return nil, err
	}
	return &taxclasspb.ReadTaxClassResponse{Success: true, Data: []*taxclasspb.TaxClass{c}}, nil
}

// ListTaxClasses lists all tax_class records, optionally filtered by direction.
func (r *MySQLTaxClassRepository) ListTaxClasses(ctx context.Context, req *taxclasspb.ListTaxClassesRequest) (*taxclasspb.ListTaxClassesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tax_classes: %w", err)
	}
	var items []*taxclasspb.TaxClass
	for _, raw := range listResult.Data {
		c, err := unmarshalTaxClass(raw)
		if err != nil {
			log.Printf("WARN: unmarshal tax_class: %v", err)
			continue
		}
		items = append(items, c)
	}
	return &taxclasspb.ListTaxClassesResponse{Success: true, Data: items}, nil
}

// FindByCodeQueries is the interface for FindByCode.
type FindByCodeQueries interface {
	FindByCode(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error)
}

// FindByCode returns the tax_class matching the given code and direction.
//
// Dialect changes from postgres gold standard:
//   - row_to_json(c) → explicit column scan + manual struct population (TODO: full scan)
//   - active = true → active = 1 (MySQL TINYINT boolean)
//   - LIMIT 1 stays; ? placeholders.
//
// NOTE: MySQL has no JSON_OBJECT equivalent for row_to_json; we use the generic
// dbOps.List with a filter approach, or a raw SELECT returning JSON_OBJECT.
// For now we query via the generic ops layer with a code+direction compound filter
// and pick the first active result. TODO: implement raw SQL scan for performance.
func (r *MySQLTaxClassRepository) FindByCode(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindByCode requires raw *sql.DB")
	}
	// Dialect: active = 1 (MySQL TINYINT); LIKE not ILIKE; ? placeholders.
	// We select all columns individually and scan them. For simplicity we use
	// a JSON_OBJECT aggregate that MySQL 8.0+ supports.
	//
	// MySQL equivalent of postgres `SELECT row_to_json(c) FROM tax_class c WHERE ...`:
	// SELECT JSON_OBJECT('id', c.id, 'code', c.code, ...) FROM tax_class c WHERE ...
	// However, rather than enumerate every column in the proto, we use a simpler
	// approach: scan the relevant columns and build the proto manually.
	//
	// Simplified: query id only then use dbOps.Read. This avoids enumerating all columns.
	var id string
	row := r.db.QueryRowContext(ctx,
		"SELECT id FROM tax_class WHERE code = ? AND direction = ? AND active = 1 LIMIT 1",
		code, direction,
	)
	if err := row.Scan(&id); err == sql.ErrNoRows {
		return nil, fmt.Errorf("tax_class not found for code=%q direction=%q", code, direction)
	} else if err != nil {
		return nil, fmt.Errorf("FindByCode query: %w", err)
	}

	raw, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("FindByCode read: %w", err)
	}
	return unmarshalTaxClass(raw)
}

var _ FindByCodeQueries = (*MySQLTaxClassRepository)(nil)
