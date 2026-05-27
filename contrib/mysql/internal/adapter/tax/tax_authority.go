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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.TaxAuthority, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql tax_authority repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLTaxAuthorityRepository(dbOps, tableName), nil
	})
}

// MySQLTaxAuthorityRepository implements tax_authority read operations using MySQL 8.0+.
type MySQLTaxAuthorityRepository struct {
	taxauthoritypb.UnimplementedTaxAuthorityDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLTaxAuthorityRepository creates a new MySQL tax_authority repository.
func NewMySQLTaxAuthorityRepository(dbOps interfaces.DatabaseOperation, tableName string) taxauthoritypb.TaxAuthorityDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxAuthority
	}
	return &MySQLTaxAuthorityRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func unmarshalTaxAuthority(raw map[string]any) (*taxauthoritypb.TaxAuthority, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	ta := &taxauthoritypb.TaxAuthority{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, ta); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return ta, nil
}

// ReadTaxAuthority retrieves a tax_authority record by ID.
func (r *MySQLTaxAuthorityRepository) ReadTaxAuthority(ctx context.Context, req *taxauthoritypb.ReadTaxAuthorityRequest) (*taxauthoritypb.ReadTaxAuthorityResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_authority ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tax_authority: %w", err)
	}
	ta, err := unmarshalTaxAuthority(result)
	if err != nil {
		return nil, err
	}
	return &taxauthoritypb.ReadTaxAuthorityResponse{Success: true, Data: []*taxauthoritypb.TaxAuthority{ta}}, nil
}

// ListTaxAuthorities lists all tax_authority records.
func (r *MySQLTaxAuthorityRepository) ListTaxAuthorities(ctx context.Context, req *taxauthoritypb.ListTaxAuthoritiesRequest) (*taxauthoritypb.ListTaxAuthoritiesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tax_authorities: %w", err)
	}
	var items []*taxauthoritypb.TaxAuthority
	for _, raw := range listResult.Data {
		ta, err := unmarshalTaxAuthority(raw)
		if err != nil {
			log.Printf("WARN: unmarshal tax_authority: %v", err)
			continue
		}
		items = append(items, ta)
	}
	return &taxauthoritypb.ListTaxAuthoritiesResponse{Success: true, Data: items}, nil
}
