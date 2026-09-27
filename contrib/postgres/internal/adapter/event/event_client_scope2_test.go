//go:build postgresql

package event

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
	_ "github.com/lib/pq"
)

func TestEventClientScope2RequiresWorkspace(t *testing.T) {
	repo := &PostgresEventClientRepository{}
	for _, tc := range []struct {
		ctx  context.Context
		want error
	}{
		{context.Background(), identity.ErrIdentityNotInContext},
		{identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{}), identity.ErrWorkspaceNotSelected},
	} {
		if _, err := repo.GetEventClientListPageData(tc.ctx, &eventclientpb.GetEventClientListPageDataRequest{}); !errors.Is(err, tc.want) {
			t.Fatalf("missing workspace: got %v want %v", err, tc.want)
		}
	}
}

func TestEventClientScope2SyntheticDB(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	query := `WITH event(id,workspace_id) AS (VALUES ('pa','A'),('pb','B')),
        event_client(id,event_id,active) AS (VALUES ('a','pa',true),('b','pb',true),('n',NULL,true),('d','missing',true))
        SELECT ec.id FROM event_client ec WHERE ec.active AND $1::text='' AND $2::int=50 AND $3::int=0 AND ` + eventClientListWorkspacePredicate + ` ORDER BY ec.id`
	for _, tc := range []struct {
		ws   string
		want []string
	}{{"A", []string{"a"}}, {"B", []string{"b"}}, {"missing", nil}} {
		rows, err := db.QueryContext(ctx, query, "", 50, 0, tc.ws)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			got = append(got, id)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("workspace %s: got %v want %v", tc.ws, got, tc.want)
		}
	}
	repo := NewPostgresEventClientRepository(core.NewWorkspaceAwareOperations(db), "event_client").(*PostgresEventClientRepository)
	bound := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: "scope2-no-such-workspace"})
	resp, err := repo.GetEventClientListPageData(bound, &eventclientpb.GetEventClientListPageDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetEventClientList()) != 0 {
		t.Fatalf("unexpected rows for absent workspace: %d", len(resp.GetEventClientList()))
	}
}
