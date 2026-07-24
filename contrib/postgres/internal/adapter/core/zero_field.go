//go:build postgresql

package core

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ForceBoolField re-inserts a proto3 no-presence bool field into a
// protojson-derived column map even when it holds its zero value (false).
//
// The storage bridge marshals a proto with default protojson options, which
// follow proto3 semantics and OMIT a no-presence scalar at its zero value
// (false / 0 / ""). The generic UPDATE path (PostgresOperations.Update) then
// SETs only the columns whose keys are present in the map, so an omitted key
// means the column is never written. A deactivation therefore silently no-ops:
// the row keeps active=true even though the caller sent active=false.
//
// This helper re-inserts a single named bool field under its protojson
// (camelCase) key so the column IS written, WITHOUT changing default marshal
// behaviour for any other field — the blast radius is exactly the named field.
// Callers pass the same *proto.Message they marshaled and the map produced from
// it (e.g. "active").
//
// Presence rules:
//   - no-presence bool (plain proto3 bool): always written (this is the point —
//     false must reach the column).
//   - presence-supporting bool (proto3 optional) that is genuinely unset: left
//     absent, because a nil must not overwrite a stored column value.
//
// The key spelling matches protojson's JSON name, so the downstream snake_case
// normalization in Create/Update maps it to the right column.
func ForceBoolField(msg proto.Message, data map[string]any, name protoreflect.Name) {
	if msg == nil || data == nil {
		return
	}
	m := msg.ProtoReflect()
	fd := m.Descriptor().Fields().ByName(name)
	if fd == nil || fd.Kind() != protoreflect.BoolKind {
		return
	}
	if fd.HasPresence() && !m.Has(fd) {
		return
	}
	data[fd.JSONName()] = m.Get(fd).Bool()
}
