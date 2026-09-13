//go:build postgresql

package asset

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	"strings"
	"testing"
)

type assignmentExec struct {
	sqlexec.DBExecutor
	rows  int64
	err   error
	query string
	args  []any
}

func (e *assignmentExec) ExecContext(_ context.Context, q string, args ...any) (sql.Result, error) {
	e.query = q
	e.args = args
	return driver.RowsAffected(e.rows), e.err
}

type assignmentOps struct {
	interfaces.DatabaseOperation
	exec *assignmentExec
}

func (o *assignmentOps) GetExecutor(context.Context) sqlexec.DBExecutor { return o.exec }

func TestAssignAssetProductConditionalWrite(t *testing.T) {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "ws"})
	for _, rows := range []int64{0, 1} {
		exec := &assignmentExec{rows: rows}
		r := &PostgresAssetRepository{dbOps: &assignmentOps{exec: exec}, tableName: "asset", productTableName: "product"}
		err := r.AssignAssetProduct(ctx, "a", "p")
		if (err == nil) != (rows == 1) {
			t.Fatalf("rows=%d error=%v", rows, err)
		}
		if len(exec.args) != 3 || exec.args[0] != "p" || exec.args[1] != "a" || exec.args[2] != "ws" {
			t.Fatalf("scope args=%v", exec.args)
		}
		for _, constraint := range []string{"a.workspace_id = $3", "COALESCE(a.product_id, '') = ''", "p.workspace_id = $3"} {
			if !strings.Contains(exec.query, constraint) {
				t.Fatalf("missing assignment guard %s", constraint)
			}
		}
		set := strings.Split(strings.Split(exec.query, "SET ")[1], "WHERE")[0]
		if strings.Contains(set, "acquisition_cost") || strings.Contains(set, "location_id") || strings.Contains(set, "active") {
			t.Fatalf("assignment rewrites asset fields: %s", set)
		}
	}
	exec := &assignmentExec{rows: 1}
	r := &PostgresAssetRepository{dbOps: &assignmentOps{exec: exec}, tableName: "asset"}
	if r.AssignAssetProduct(context.Background(), "a", "p") == nil || exec.query != "" {
		t.Fatal("unscoped assignment reached SQL")
	}
	exec.err = errors.New("database unavailable")
	if r.AssignAssetProduct(ctx, "a", "p") == nil {
		t.Fatal("write failure hidden")
	}
}

type captureAssetUpdate struct {
	interfaces.DatabaseOperation
	data map[string]any
}

func (o *captureAssetUpdate) Update(_ context.Context, _ string, id string, data map[string]any) (map[string]any, error) {
	o.data = data
	return map[string]any{"id": id, "product_id": "existing"}, nil
}
func TestUpdateAssetPreservesConcurrentProductAssignment(t *testing.T) {
	ops := &captureAssetUpdate{}
	r := &PostgresAssetRepository{dbOps: ops, tableName: "asset"}
	changed := "different"
	resp, err := r.UpdateAsset(context.Background(), &assetpb.UpdateAssetRequest{Data: &assetpb.Asset{Id: "a", Name: "Updated name", ProductId: &changed}})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"productId", "product_id"} {
		if _, ok := ops.data[key]; ok {
			t.Fatalf("generic edit overwrites link via %s", key)
		}
	}
	if ops.data["name"] != "Updated name" || resp.Data[0].GetProductId() != "existing" {
		t.Fatalf("unrelated edit or link lost: %+v", ops.data)
	}
}
