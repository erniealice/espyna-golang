// Package schema is the dialect-neutral source of column truth for espyna's
// reflectionless CRUD path (Plan 2, docs/plan/20260530-reflectionless-crud/).
//
// It walks the protobuf descriptors registered in protoregistry.GlobalTypes and
// derives, per table-annotated message, the exact set of persisted columns and
// their write-time serialization kind. The result is stored in a dialect-neutral
// Registry that the postgres operations layer consumes read-only, replacing the
// two per-call information_schema round-trips with a single boot-time walk.
//
// This package contains NO SQL and NO information_schema access. SQL reconciliation
// (the boot-shot validator) lives in the per-dialect contrib adapters
// (contrib/postgres/internal/adapter/core/schema_validator.go), keeping mysql /
// sqlserver able to add a ~20-line sibling without duplicating the classifier.
//
// Locked decisions (docs/plan/20260530-reflectionless-crud/decisions.md):
//   - Q-DD1=C  column classification = proto-kind-derive + (db).ignore annotation
//   - Q-DD2=A  registry populated from a protoregistry.GlobalTypes walk
//   - Q-DD6=A  dialect-neutral schema package (no god object)
package schema

import (
	optionsv1 "github.com/erniealice/esqyma/pkg/schema/v1/options"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// wellKnownTimestamp is the full proto type name of google.protobuf.Timestamp.
const wellKnownTimestamp = "google.protobuf.Timestamp"

// wellKnownStruct is the full proto type name of google.protobuf.Struct, which
// maps to a jsonb metadata column rather than a relation.
const wellKnownStruct = "google.protobuf.Struct"

// auditTimestampNames are the two int64 unix-millis audit columns. They are the
// only fields for which IsBigintMillis is asserted, mirroring the operational
// convention (213 int64 date_created + 195 int64 date_modified, zero
// google.protobuf.Timestamp date_created in the tree). Other int64 date fields
// (due_date, planned_start, ...) are plain bigint columns; they are not
// auto-stamped by the write path, so they do not need the IsBigintMillis flag.
var auditTimestampNames = map[string]bool{
	"date_created":  true,
	"date_modified": true,
}

// ColumnInfo is the dialect-neutral description of a single persisted column,
// derived from one proto field (Q-DD1=C). The flags below feed the write path's
// serialization decision (autoTimestampValue) and the boot-shot reconcile without
// any further proto reflection at call time.
type ColumnInfo struct {
	// Name is the DB column name == the field's proto snake_case TextName
	// (== camelToSnake(jsonName)). This is what the post-normalizeKeys map keys
	// are compared against in operations.go.
	Name string

	// ProtoKind is the underlying proto kind of the field (StringKind, Int64Kind,
	// BoolKind, EnumKind, BytesKind, MessageKind for the two well-knowns, ...).
	ProtoKind protoreflect.Kind

	// IsTimestamp is true for a google.protobuf.Timestamp business-timestamp column
	// (scheduled_at, delivered_at, ...). These serialize as a time.Time / TIMESTAMPTZ.
	IsTimestamp bool

	// IsBigintMillis is true for the int64 audit columns date_created / date_modified
	// (unix-ms convention). autoTimestampValue writes now.UnixMilli() for these.
	IsBigintMillis bool

	// IsActive marks the bool `active` soft-delete flag.
	IsActive bool

	// IsMetadata is true for a google.protobuf.Struct column mapped to jsonb.
	IsMetadata bool
}

// Classify derives the persisted-column set for one message descriptor per the
// Q-DD1=C rules. It performs NO SQL and NO information_schema access — column
// truth is read entirely from the proto descriptor and its (options.v1.db)
// annotations.
//
// Rules (in field order):
//  1. SKIP message-kind fields that are relations (singular or repeated nested
//     message) EXCEPT google.protobuf.Struct, which becomes a jsonb metadata column.
//  2. SKIP repeated scalar/enum fields (array / join-table backed; no parent column).
//  3. SKIP any field carrying (options.v1.db).ignore = true (the computed *_string
//     display mirrors). CRITICAL: ignore is consumed from the annotation ONLY — it
//     is never re-derived from a *_string suffix — so EventRecurrence.rrule_string /
//     exdate_string (real columns without the annotation) are preserved.
//  4. INCLUDE scalar / enum / bytes / well-known google.protobuf.Timestamp as columns.
//
// The column name is the field's proto snake_case TextName, which equals the DB
// column name and equals camelToSnake(jsonName).
func Classify(md protoreflect.MessageDescriptor) []ColumnInfo {
	fields := md.Fields()
	cols := make([]ColumnInfo, 0, fields.Len())

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		// (3) Annotation-driven ignore — consumed ONLY from the annotation, never
		// derived from the *_string suffix. Checked first so an ignored mirror is
		// dropped regardless of its kind.
		if fieldIgnored(fd) {
			continue
		}

		if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
			full := string(fd.Message().FullName())

			// (1-exception) google.protobuf.Struct -> jsonb metadata column.
			// A repeated Struct would be array-of-jsonb; not a convention in this
			// tree, so only the singular case is treated as a metadata column.
			if full == wellKnownStruct && !fd.IsList() {
				cols = append(cols, ColumnInfo{
					Name:       string(fd.Name()),
					ProtoKind:  fd.Kind(),
					IsMetadata: true,
				})
				continue
			}

			// (4) google.protobuf.Timestamp -> business-timestamp column.
			if full == wellKnownTimestamp && !fd.IsList() {
				cols = append(cols, ColumnInfo{
					Name:        string(fd.Name()),
					ProtoKind:   fd.Kind(),
					IsTimestamp: true,
				})
				continue
			}

			// (1) Any other nested message (singular or repeated) is a relation. SKIP.
			continue
		}

		// (2) Repeated scalar / enum -> array or join-table backed. SKIP.
		if fd.IsList() {
			continue
		}

		// (4) Scalar / enum / bytes column.
		name := string(fd.Name())
		cols = append(cols, ColumnInfo{
			Name:           name,
			ProtoKind:      fd.Kind(),
			IsBigintMillis: fd.Kind() == protoreflect.Int64Kind && auditTimestampNames[name],
			IsActive:       name == "active" && fd.Kind() == protoreflect.BoolKind,
		})
	}

	return cols
}

// fieldIgnored reports whether a field carries (options.v1.db).ignore = true.
// This is the sole source of ignore truth (Q-DD1=C): the classifier never
// inspects the field name to guess that a *_string field is a computed mirror.
func fieldIgnored(fd protoreflect.FieldDescriptor) bool {
	opts := fd.Options()
	if opts == nil {
		return false
	}
	ext := proto.GetExtension(opts, optionsv1.E_Db)
	dbOpts, ok := ext.(*optionsv1.FieldOptions)
	if !ok || dbOpts == nil {
		return false
	}
	return dbOpts.GetIgnore()
}
