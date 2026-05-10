//go:build postgresql

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TaxClass, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres tax_class repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTaxClassRepository(db, dbOps, tableName), nil
	})
}

// PostgresTaxClassRepository implements tax_class read operations using PostgreSQL.
type PostgresTaxClassRepository struct {
	taxclasspb.UnimplementedTaxClassDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTaxClassRepository creates a new PostgreSQL tax_class repository.
func NewPostgresTaxClassRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxclasspb.TaxClassDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxClass
	}
	return &PostgresTaxClassRepository{
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
func (r *PostgresTaxClassRepository) ReadTaxClass(ctx context.Context, req *taxclasspb.ReadTaxClassRequest) (*taxclasspb.ReadTaxClassResponse, error) {
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
func (r *PostgresTaxClassRepository) ListTaxClasses(ctx context.Context, req *taxclasspb.ListTaxClassesRequest) (*taxclasspb.ListTaxClassesResponse, error) {
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
// Direction is "WITHHOLDING" for WHT classes; future directions include
// "OUTPUT", "EXCISE", "SALES_TAX", etc.
func (r *PostgresTaxClassRepository) FindByCode(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindByCode requires raw *sql.DB")
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT row_to_json(c) FROM tax_class c
		 WHERE code = $1 AND direction = $2 AND active = true
		 LIMIT 1`,
		code, direction,
	)
	var rawJSON []byte
	if err := row.Scan(&rawJSON); err == sql.ErrNoRows {
		return nil, fmt.Errorf("tax_class not found for code=%q direction=%q", code, direction)
	} else if err != nil {
		return nil, fmt.Errorf("FindByCode query: %w", err)
	}
	c := &taxclasspb.TaxClass{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, c); err != nil {
		return nil, fmt.Errorf("FindByCode unmarshal: %w", err)
	}
	return c, nil
}

var _ FindByCodeQueries = (*PostgresTaxClassRepository)(nil)
