package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"leapfor.xyz/espyna/internal/application/ports"
)

// GenericExecutor wraps a standard use case Execute method to implement
// the UseCaseExecutor interface. It handles the conversion between
// map[string]interface{} and protobuf messages.
type GenericExecutor[Req proto.Message, Res proto.Message] struct {
	ExecuteFunc func(context.Context, Req) (Res, error)
}

// Execute converts a map input to proto, executes the use case,
// and converts the result back to a map.
func (e *GenericExecutor[Req, Res]) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// 1. Convert Map -> JSON -> Proto
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input map: %w", err)
	}

	// Create a new request instance of type Req
	req := reflectNew[Req]()

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(jsonBytes, req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to proto request: %w", err)
	}

	// 2. Execute Use Case
	res, err := e.ExecuteFunc(ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. Convert Proto -> JSON -> Map
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}
	resBytes, err := marshaler.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proto response to JSON: %w", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(resBytes, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to output map: %w", err)
	}

	return output, nil
}

// reflectNew uses reflection to create a new instance of a generic type.
// This is necessary because Go generics don't support `new(T)` for
// constrained types like proto.Message which are typically pointers.
func reflectNew[T any]() T {
	var v T
	// Check if T is a pointer
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		// Create a new instance of the element type and return a pointer to it
		return reflect.New(t.Elem()).Interface().(T)
	}
	// Return zero value for non-pointers (might not work well for proto.Message)
	return v
}

// New is a convenience function to create a GenericExecutor.
// It simplifies the registration of use cases in domain-specific files.
func New[Req proto.Message, Res proto.Message](
	execute func(context.Context, Req) (Res, error),
) ports.ActivityExecutor {
	return &GenericExecutor[Req, Res]{
		ExecuteFunc: execute,
	}
}

// RawExecutor wraps a use case that already uses map[string]interface{} input/output.
// This is useful for workflow-friendly adapters that don't need protobuf conversion.
type RawExecutor struct {
	ExecuteFunc func(context.Context, map[string]interface{}) (map[string]interface{}, error)
}

// Execute passes through the map input directly to the use case.
func (e *RawExecutor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return e.ExecuteFunc(ctx, input)
}

// NewRaw creates an executor for use cases that already use map[string]interface{}.
// This bypasses protobuf conversion for workflow-friendly adapters.
func NewRaw(
	execute func(context.Context, map[string]interface{}) (map[string]interface{}, error),
) ports.ActivityExecutor {
	return &RawExecutor{
		ExecuteFunc: execute,
	}
}
