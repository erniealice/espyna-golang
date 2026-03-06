// Package operations re-exports internal database operation helpers for use by contrib sub-modules.
package operations

import (
	internal "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	"google.golang.org/protobuf/proto"
)

// Protobuf helpers
type (
	ProtobufTimestamp = internal.ProtobufTimestamp
	ProtobufMapper    = internal.ProtobufMapper
)

var (
	NewProtobufTimestamp   = internal.NewProtobufTimestamp
	ConvertToProtobufMap   = internal.ConvertToProtobufMap
	NewProtobufMapper      = internal.NewProtobufMapper
	TimestampFromTime      = internal.TimestampFromTime
	ParseTimestamp         = internal.ParseTimestamp
)

// Generic protobuf conversion wrappers (generic funcs cannot be assigned to vars).

// ConvertMapToProtobuf converts a map[string]any to any protobuf message using protojson.
func ConvertMapToProtobuf[T proto.Message](data map[string]any, target T) (T, error) {
	return internal.ConvertMapToProtobuf(data, target)
}

// ConvertSliceToProtobuf converts a slice of map[string]any to a slice of protobuf messages.
func ConvertSliceToProtobuf[T proto.Message](dataSlice []map[string]any, targetFactory func() T) ([]T, []error) {
	return internal.ConvertSliceToProtobuf(dataSlice, targetFactory)
}

// Query builder (operations package has its own SimpleQueryBuilder that wraps interfaces)
type SimpleQueryBuilder = internal.SimpleQueryBuilder

var NewQueryBuilder = internal.NewQueryBuilder

// Transaction context helpers
var (
	WithTransaction                   = internal.WithTransaction
	GetTransactionFromContext         = internal.GetTransactionFromContext
	WithTransactionManager            = internal.WithTransactionManager
	GetTransactionManagerFromContext   = internal.GetTransactionManagerFromContext
	IsTransactionContext              = internal.IsTransactionContext
)
