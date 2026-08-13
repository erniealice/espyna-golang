//go:build postgresql

package entity

import (
	"context"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

type clientAttributeListDB struct {
	params *interfaces.ListParams
}

func (db *clientAttributeListDB) Create(context.Context, string, map[string]any) (map[string]any, error) {
	panic("unexpected Create")
}

func (db *clientAttributeListDB) Read(context.Context, string, string) (map[string]any, error) {
	panic("unexpected Read")
}

func (db *clientAttributeListDB) Update(context.Context, string, string, map[string]any) (map[string]any, error) {
	panic("unexpected Update")
}

func (db *clientAttributeListDB) Delete(context.Context, string, string) error {
	panic("unexpected Delete")
}

func (db *clientAttributeListDB) HardDelete(context.Context, string, string) error {
	panic("unexpected HardDelete")
}

func (db *clientAttributeListDB) List(_ context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	if tableName != "client_attribute" {
		panic("unexpected table")
	}
	db.params = params
	return &interfaces.ListResult{Data: []map[string]any{{
		"id":           "client-attribute-1",
		"client_id":    "client-1",
		"attribute_id": "attribute-1",
		"value":        "A",
		"active":       true,
	}}}, nil
}

func (db *clientAttributeListDB) Query(context.Context, string, interfaces.QueryBuilder) ([]map[string]any, error) {
	panic("unexpected Query")
}

func (db *clientAttributeListDB) QueryOne(context.Context, string, interfaces.QueryBuilder) (map[string]any, error) {
	panic("unexpected QueryOne")
}

func TestPostgresClientAttributeRepository_ListReturnsSuccessAndForwardsBounds(t *testing.T) {
	db := &clientAttributeListDB{}
	repository := NewPostgresClientAttributeRepository(db, "client_attribute")
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "attribute_id",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: "attribute-1",
		}},
	}}}
	pagination := &commonpb.PaginationRequest{Limit: 37}

	response, err := repository.ListClientAttributes(context.Background(), &clientattributepb.ListClientAttributesRequest{
		Filters:    filters,
		Pagination: pagination,
	})
	if err != nil {
		t.Fatalf("ListClientAttributes() error = %v", err)
	}
	if !response.GetSuccess() {
		t.Fatal("ListClientAttributes() success = false, want true")
	}
	if len(response.GetData()) != 1 || response.GetData()[0].GetClientId() != "client-1" || response.GetData()[0].GetAttributeId() != "attribute-1" {
		t.Fatalf("ListClientAttributes() data = %#v", response.GetData())
	}
	if db.params == nil || db.params.Filters != filters || db.params.Pagination != pagination {
		t.Fatalf("ListClientAttributes() params = %#v, want original filters and pagination", db.params)
	}
}
