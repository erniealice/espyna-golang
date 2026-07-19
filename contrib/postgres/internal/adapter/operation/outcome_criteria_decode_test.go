//go:build postgresql

package operation

import (
	"context"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// decodeFailDBOps serves a single row whose "version" field is a non-numeric
// string, which cannot decode into OutcomeCriteria.version (int32). It exists to
// prove ListOutcomeCriterias now propagates the decode failure (fail-closed)
// instead of the old log-and-continue that silently DROPPED the row — a dropped
// row could hide an established lineage code or a collision from the use-case
// pre-checks. Only List is exercised; the other methods satisfy the interface.
type decodeFailDBOps struct{}

func (decodeFailDBOps) Create(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (decodeFailDBOps) Read(context.Context, string, string) (map[string]any, error) {
	return nil, nil
}
func (decodeFailDBOps) Update(context.Context, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (decodeFailDBOps) Delete(context.Context, string, string) error     { return nil }
func (decodeFailDBOps) HardDelete(context.Context, string, string) error { return nil }
func (decodeFailDBOps) List(context.Context, string, *interfaces.ListParams) (*interfaces.ListResult, error) {
	return &interfaces.ListResult{Data: []map[string]any{
		{"id": "oc-corrupt", "version": "not-an-int"},
	}}, nil
}
func (decodeFailDBOps) Query(context.Context, string, interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, nil
}
func (decodeFailDBOps) QueryOne(context.Context, string, interfaces.QueryBuilder) (map[string]any, error) {
	return nil, nil
}

func TestListOutcomeCriterias_DecodeFailurePropagates(t *testing.T) {
	r := NewPostgresOutcomeCriteriaRepository(decodeFailDBOps{}, "outcome_criteria")

	_, err := r.ListOutcomeCriterias(context.Background(), &pb.ListOutcomeCriteriasRequest{})
	if err == nil {
		t.Fatal("an undecodable outcome_criteria row must fail the whole list read (fail-closed), not be silently dropped")
	}
}
