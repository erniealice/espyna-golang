//go:build postgresql

package entity

import (
	"context"
	"strings"
	"testing"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

func TestSearchClientsByNameRejectsOversizeLimitBeforeDatabaseAccess(t *testing.T) {
	limit := int32(101)
	_, err := (&PostgresClientRepository{}).SearchClientsByName(context.Background(), &clientpb.SearchClientsByNameRequest{Limit: &limit})
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("SearchClientsByName() error = %v, want bounded limit rejection", err)
	}
}
