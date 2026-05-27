//go:build sqlserver

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TaxClass, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver tax_class repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTaxClassRepository(db, dbOps, tableName), nil
	})
}

// SQLServerTaxClassRepository implements tax_class read operations using SQL Server.
type SQLServerTaxClassRepository struct {
	taxclasspb.UnimplementedTaxClassDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerTaxClassRepository creates a new SQL Server tax_class repository.
func NewSQLServerTaxClassRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxclasspb.TaxClassDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxClass
	}
	return &SQLServerTaxClassRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxClass(raw map[string]any) (*taxclasspb.TaxClass, error) {
	js, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
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
func (r *SQLServerTaxClassRepository) ReadTaxClass(ctx context.Context, req *taxclasspb.ReadTaxClassRequest) (*taxclasspb.ReadTaxClassResponse, error) {
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
func (r *SQLServerTaxClassRepository) ListTaxClasses(ctx context.Context, req *taxclasspb.ListTaxClassesRequest) (*taxclasspb.ListTaxClassesResponse, error) {
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
// SQL Server differences from the postgres gold standard:
//   - $1/$2 → @p1/@p2.
//   - active = true → active = 1 (SQL Server BIT).
//   - row_to_json() not available — we select all columns and unmarshal via the
//     generic DenormalizeKeys path used by all other SQL Server adapters.
//   - LIMIT 1 → SELECT TOP 1.
func (r *SQLServerTaxClassRepository) FindByCode(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindByCode requires raw *sql.DB")
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT TOP 1 id, code, direction, name, description, active
		 FROM tax_class
		 WHERE code = @p1 AND direction = @p2 AND active = 1`,
		code, direction,
	)
	var (
		id           string
		codeVal      string
		directionVal string
		name         string
		description  *string
		active       bool
	)
	if err := row.Scan(&id, &codeVal, &directionVal, &name, &description, &active); err == sql.ErrNoRows {
		return nil, fmt.Errorf("tax_class not found for code=%q direction=%q", code, direction)
	} else if err != nil {
		return nil, fmt.Errorf("FindByCode query: %w", err)
	}
	c := &taxclasspb.TaxClass{
		Id:     id,
		Code:   codeVal,
		Name:   name,
		Active: active,
	}
	if val, ok := taxclasspb.TaxClassDirection_value[directionVal]; ok {
		c.Direction = taxclasspb.TaxClassDirection(val)
	}
	if description != nil {
		c.Description = description
	}
	return c, nil
}

var _ FindByCodeQueries = (*SQLServerTaxClassRepository)(nil)
