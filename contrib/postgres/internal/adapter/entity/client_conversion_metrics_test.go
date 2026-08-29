//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

type clientConversionFixture struct {
	listResult *interfaces.ListResult
	queryErr   error
}

func (f *clientConversionFixture) Create(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("unexpected Create")
}
func (f *clientConversionFixture) Read(context.Context, string, string) (map[string]any, error) {
	return nil, fmt.Errorf("unexpected Read")
}
func (f *clientConversionFixture) Update(context.Context, string, string, map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("unexpected Update")
}
func (f *clientConversionFixture) Delete(context.Context, string, string) error {
	return fmt.Errorf("unexpected Delete")
}
func (f *clientConversionFixture) HardDelete(context.Context, string, string) error {
	return fmt.Errorf("unexpected HardDelete")
}
func (f *clientConversionFixture) List(context.Context, string, *interfaces.ListParams) (*interfaces.ListResult, error) {
	return f.listResult, nil
}
func (f *clientConversionFixture) Query(context.Context, string, interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, fmt.Errorf("unexpected Query")
}
func (f *clientConversionFixture) QueryOne(context.Context, string, interfaces.QueryBuilder) (map[string]any, error) {
	return nil, fmt.Errorf("unexpected QueryOne")
}
func (f *clientConversionFixture) GetExecutor(context.Context) sqlexec.DBExecutor { return f }
func (f *clientConversionFixture) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("unexpected ExecContext")
}
func (f *clientConversionFixture) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, f.queryErr
}
func (f *clientConversionFixture) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func TestListClientsConversionMetric(t *testing.T) {
	fx := &clientConversionFixture{listResult: &interfaces.ListResult{Data: []map[string]any{
		{"id": "client-1", "active": true, "internal_id": "S-001"},
		// A function cannot be represented in JSON, so existing behavior skips it.
		{"id": "client-2", "active": true, "invalid": func() {}},
	}}}
	repo := NewPostgresClientRepository(fx, "client").(*PostgresClientRepository)
	var samples []postgresCore.ConversionMetricSample
	ctx := postgresCore.WithConversionMetricSink(context.Background(), func(sample postgresCore.ConversionMetricSample) {
		samples = append(samples, sample)
	})

	response, err := repo.ListClients(ctx, &clientpb.ListClientsRequest{})
	if err != nil {
		t.Fatalf("ListClients() error = %v", err)
	}
	if len(response.GetData()) != 1 || response.GetData()[0].GetId() != "client-1" {
		t.Fatalf("ListClients() data = %#v", response.GetData())
	}
	if len(samples) != 1 {
		t.Fatalf("samples = %#v, want one", samples)
	}
	sample := samples[0]
	if sample.Table != "client" || sample.Path != "generic-proto" || sample.Operation != "list" {
		t.Fatalf("sample identity = %#v", sample)
	}
	if sample.Outcome != "partial" || sample.Rows != 2 || sample.Failures != 1 {
		t.Fatalf("sample result = %#v", sample)
	}
	if sample.Phases["json_marshal"] <= 0 || sample.Phases["protojson_unmarshal"] <= 0 {
		t.Fatalf("sample phases = %#v", sample.Phases)
	}
}

func TestClientPageConversionMetricClassifiesQueryError(t *testing.T) {
	fx := &clientConversionFixture{queryErr: errors.New("query-hit")}
	repo := NewPostgresClientRepository(fx, "client").(*PostgresClientRepository)
	var samples []postgresCore.ConversionMetricSample
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "workspace-1"})
	ctx = postgresCore.WithConversionMetricSink(ctx, func(sample postgresCore.ConversionMetricSample) {
		samples = append(samples, sample)
	})

	_, err := repo.GetClientListPageData(ctx, &clientpb.GetClientListPageDataRequest{})
	if err == nil {
		t.Fatal("GetClientListPageData() error = nil, want query error")
	}
	if len(samples) != 1 {
		t.Fatalf("samples = %#v, want one", samples)
	}
	sample := samples[0]
	if sample.Path != "typed-client-page" || sample.Operation != "list" || sample.Outcome != "error" {
		t.Fatalf("sample = %#v", sample)
	}
	if sample.FailureStage != "query_open" || sample.Columns != 33 {
		t.Fatalf("sample failure = %#v", sample)
	}
	if sample.Phases["query_open"] <= 0 {
		t.Fatalf("sample phases = %#v", sample.Phases)
	}
}
