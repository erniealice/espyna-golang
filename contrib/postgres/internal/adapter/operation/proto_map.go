//go:build postgresql

package operation

import (
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// protoToMap marshals any proto message to a column-shaped map[string]any for the
// generic dbOps write path. It is a pure proto→map utility (protojson.Marshal +
// json.Unmarshal) that then CANONICALIZES every top-level key to snake_case, so
// the keys the write path sees are already the persisted column spelling.
//
// Why canonicalize here (gate H1 — cross-tenant write). protojson.Marshal emits
// lowerCamelCase JSON names (e.g. workspaceId). The downstream normalizeKeys
// collapses camel→snake over a randomly-ordered map, so if a camelCase key and
// its snake_case twin ever coexist in one map, one nondeterministically overwrites
// the other. That was the tenant-reassignment vector: a client-supplied camelCase
// workspaceId could ride alongside the trusted snake workspace_id the decorator
// injects and win the collapse. By snake-casing FIRST and rejecting any collision,
// the map handed to the tenancy-aware decorator carries exactly one canonical
// spelling per column, so the decorator's trusted workspace_id overwrites
// deterministically instead of racing a twin key.
//
// Only TOP-LEVEL keys are canonicalized (those map to columns); nested message
// values keep their protojson camelCase spelling and round-trip as opaque JSONB
// unchanged.
func protoToMap(msg proto.Message) (map[string]any, error) {
	jsonData, err := protojson.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	return canonicalizeColumnKeys(data)
}

// canonicalizeColumnKeys snake_cases every top-level key and fails loudly if two
// distinct source keys canonicalize to the same column, rather than letting one
// nondeterministically overwrite the other (gate H1). It is factored out of
// protoToMap so the collision guard is unit-testable without a proto fixture.
func canonicalizeColumnKeys(in map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(in))
	claimed := make(map[string]string, len(in)) // canonical column → source key that claimed it
	for key, value := range in {
		col := postgresCore.CamelToSnake(key)
		if prev, dup := claimed[col]; dup {
			return nil, fmt.Errorf("proto→column key collision: %q and %q both canonicalize to %q", prev, key, col)
		}
		claimed[col] = key
		out[col] = value
	}
	return out, nil
}

// stripClientWorkspaceKeys removes any client-supplied workspace identifier — the
// canonical snake_case column AND the protojson camelCase spelling — from a write
// payload. On Create the workspace-aware decorator re-injects the trusted
// workspace_id from the request context; on Update the workspace_id is the
// immutable tenant anchor. Stripping both spellings at the adapter is the belt to
// the decorator/core suspenders (gate H1).
func stripClientWorkspaceKeys(data map[string]any) {
	delete(data, "workspace_id")
	delete(data, "workspaceId")
}
