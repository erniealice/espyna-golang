package shared

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// CreateRequestMessage creates a protobuf request message from HTTP request data
func CreateRequestMessage(requestTypeField reflect.Value, requestData *RequestData) (proto.Message, error) {
	// Get the concrete type of the request message
	requestType := requestTypeField.Type()

	// Create a new instance of the request message
	newRequest := reflect.New(requestType.Elem()).Interface()

	// Type assert to proto.Message
	requestMsg, ok := newRequest.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("request type is not a proto.Message")
	}

	// Parse JSON body into protobuf message if body is present
	if len(requestData.Body) > 0 {
		// Use protojson for unmarshaling
		if err := protojson.Unmarshal(requestData.Body, requestMsg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
		}
	}

	// Handle path parameters and query parameters
	if err := populateRequestFromParams(requestMsg, requestData); err != nil {
		return nil, fmt.Errorf("failed to populate request from parameters: %w", err)
	}

	return requestMsg, nil
}

// CallUseCase calls the use case function using reflection
func CallUseCase(useCaseField reflect.Value, ctx any, requestMsg proto.Message) (any, error) {
	// Prepare arguments for the use case call
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(requestMsg),
	}

	// Call the use case function
	results := useCaseField.Call(args)
	if len(results) != 2 {
		return nil, fmt.Errorf("expected 2 return values from use case, got %d", len(results))
	}

	// Check for error
	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		return nil, err
	}

	// Return the response
	return results[0].Interface(), nil
}

// WriteProtobufJSONResponse writes a protobuf message response as JSON to the HTTP response writer
func WriteProtobufJSONResponse(w http.ResponseWriter, response any) error {
	w.Header().Set("Content-Type", "application/json")

	// If the response is a proto message, use protojson for marshaling
	if protoMsg, ok := response.(proto.Message); ok {
		jsonBytes, err := protojson.Marshal(protoMsg)
		if err != nil {
			return fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
		}
		_, err = w.Write(jsonBytes)
		return err
	}

	// Otherwise, use standard JSON marshaling
	return WriteJSONResponse(w, response)
}

// ExtractRouteFields extracts route fields from a route interface using reflection
func ExtractRouteFields(routeInterface any) (method, path string, useCaseField, requestTypeField reflect.Value, err error) {
	// Use reflection to determine the concrete type of the route
	routeValue := reflect.ValueOf(routeInterface)

	// Extract route information using reflection
	methodField := routeValue.FieldByName("Method")
	pathFieldValue := routeValue.FieldByName("Path")
	useCaseFieldValue := routeValue.FieldByName("UseCase")
	requestTypeFieldValue := routeValue.FieldByName("RequestType")

	if !methodField.IsValid() || !pathFieldValue.IsValid() || !useCaseFieldValue.IsValid() || !requestTypeFieldValue.IsValid() {
		return "", "", reflect.Value{}, reflect.Value{}, fmt.Errorf("invalid route structure")
	}

	method = methodField.String()
	path = pathFieldValue.String()
	useCaseField = useCaseFieldValue
	requestTypeField = requestTypeFieldValue

	return method, path, useCaseField, requestTypeField, nil
}

// populateRequestFromParams populates protobuf request message fields from path and query parameters
func populateRequestFromParams(requestMsg proto.Message, requestData *RequestData) error {
	// Get protobuf reflection descriptor
	msgReflect := requestMsg.ProtoReflect()
	msgDesc := msgReflect.Descriptor()

	// Iterate through all fields in the protobuf message
	fields := msgDesc.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		// Try to get value from path parameters first
		if value, exists := requestData.PathParams[fieldName]; exists {
			if err := setFieldValue(msgReflect, field, value); err != nil {
				return fmt.Errorf("failed to set path parameter %s: %w", fieldName, err)
			}
			continue
		}

		// Try to get value from query parameters
		if value, exists := requestData.QueryParams[fieldName]; exists {
			if err := setFieldValue(msgReflect, field, value); err != nil {
				return fmt.Errorf("failed to set query parameter %s: %w", fieldName, err)
			}
			continue
		}

		// Handle nested message fields (like 'data' field in most requests)
		if field.Kind() == protoreflect.MessageKind {
			nestedMsg := msgReflect.Get(field).Message()
			if nestedMsg.IsValid() {
				if err := populateNestedMessage(nestedMsg, requestData); err != nil {
					return fmt.Errorf("failed to populate nested message %s: %w", fieldName, err)
				}
			}
		}
	}

	return nil
}

// populateNestedMessage populates fields in a nested protobuf message from parameters
func populateNestedMessage(nestedMsg protoreflect.Message, requestData *RequestData) error {
	msgDesc := nestedMsg.Descriptor()
	fields := msgDesc.Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		// Try path parameters first
		if value, exists := requestData.PathParams[fieldName]; exists {
			if err := setFieldValue(nestedMsg, field, value); err != nil {
				return fmt.Errorf("failed to set nested path parameter %s: %w", fieldName, err)
			}
			continue
		}

		// Try query parameters
		if value, exists := requestData.QueryParams[fieldName]; exists {
			if err := setFieldValue(nestedMsg, field, value); err != nil {
				return fmt.Errorf("failed to set nested query parameter %s: %w", fieldName, err)
			}
			continue
		}
	}

	return nil
}

// setFieldValue sets a protobuf field value from a string parameter
func setFieldValue(msgReflect protoreflect.Message, field protoreflect.FieldDescriptor, value string) error {
	if value == "" {
		return nil // Skip empty values
	}

	switch field.Kind() {
	case protoreflect.StringKind:
		msgReflect.Set(field, protoreflect.ValueOfString(value))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		intValue, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid int32 value %s: %w", value, err)
		}
		msgReflect.Set(field, protoreflect.ValueOfInt32(int32(intValue)))
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int64 value %s: %w", value, err)
		}
		msgReflect.Set(field, protoreflect.ValueOfInt64(intValue))
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		uintValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid uint32 value %s: %w", value, err)
		}
		msgReflect.Set(field, protoreflect.ValueOfUint32(uint32(uintValue)))
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint64 value %s: %w", value, err)
		}
		msgReflect.Set(field, protoreflect.ValueOfUint64(uintValue))
	case protoreflect.BoolKind:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool value %s: %w", value, err)
		}
		msgReflect.Set(field, protoreflect.ValueOfBool(boolValue))
	case protoreflect.FloatKind:
		floatValue, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return fmt.Errorf("invalid float value %s: %w", value, err)
		}
		msgReflect.Set(field, protoreflect.ValueOfFloat32(float32(floatValue)))
	case protoreflect.DoubleKind:
		doubleValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid double value %s: %w", value, err)
		}
		msgReflect.Set(field, protoreflect.ValueOfFloat64(doubleValue))
	case protoreflect.BytesKind:
		msgReflect.Set(field, protoreflect.ValueOfBytes([]byte(value)))
	case protoreflect.EnumKind:
		// For enums, try to match by name or number
		enumDesc := field.Enum()
		enumValue := enumDesc.Values().ByName(protoreflect.Name(value))
		if enumValue != nil {
			msgReflect.Set(field, protoreflect.ValueOfEnum(enumValue.Number()))
		} else {
			// Try parsing as number
			enumNumber, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid enum value %s: %w", value, err)
			}
			msgReflect.Set(field, protoreflect.ValueOfEnum(protoreflect.EnumNumber(enumNumber)))
		}
	default:
		// For complex types like messages, we'll skip them for now
		// They should be handled by JSON body parsing
		return nil
	}

	return nil
}
