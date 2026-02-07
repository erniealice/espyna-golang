package tabular

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/protobuf/types/known/structpb"
	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// WriteRecordSimpleRepositories groups all repository dependencies
type WriteRecordSimpleRepositories struct {
	// No repositories needed for external tabular provider integration
}

// WriteRecordSimpleServices groups all service dependencies
type WriteRecordSimpleServices struct {
	Provider integration.TabularSourceProvider
}

// WriteRecordSimpleUseCase handles writing a single record with flat field input.
// This is a workflow-friendly wrapper that accepts a Struct of key-value pairs
// and constructs the proper WriteRecordsRequest internally.
//
// Input format:
//
//	{
//	  "data": {
//	    "source_id": "spreadsheet_id",
//	    "table": "Sheet1",
//	    "fields": {
//	      "first_name": "John",
//	      "last_name": "Doe",
//	      "amount": 100
//	    }
//	  }
//	}
//
// The use case extracts source_id and table, then converts the fields Struct
// to Record.named_values for the underlying WriteRecords call.
type WriteRecordSimpleUseCase struct {
	repositories WriteRecordSimpleRepositories
	services     WriteRecordSimpleServices
}

// NewWriteRecordSimpleUseCase creates a new WriteRecordSimpleUseCase
func NewWriteRecordSimpleUseCase(
	repositories WriteRecordSimpleRepositories,
	services WriteRecordSimpleServices,
) *WriteRecordSimpleUseCase {
	return &WriteRecordSimpleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute writes a single record using the proto-based request format
func (uc *WriteRecordSimpleUseCase) Execute(ctx context.Context, req *tabularpb.WriteRecordSimpleRequest) (*tabularpb.WriteRecordSimpleResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.WriteRecordSimpleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.WriteRecordSimpleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Extract source_id (required)
	sourceID := req.Data.SourceId
	if sourceID == "" {
		return &tabularpb.WriteRecordSimpleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "source_id is required",
			},
		}, nil
	}

	// Extract table (optional, default to "Sheet1")
	table := req.Data.Table
	if table == "" {
		table = "Sheet1"
	}

	// Convert Struct fields to named_values
	namedValues := make(map[string]*tabularpb.FieldValue)
	if req.Data.Fields != nil {
		for key, value := range req.Data.Fields.Fields {
			namedValues[key] = structValueToFieldValue(value)
		}
	}

	if len(namedValues) == 0 {
		return &tabularpb.WriteRecordSimpleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "At least one field value is required in 'fields'",
			},
		}, nil
	}

	log.Printf("[WriteRecordSimple] Writing to %s/%s with fields: %v", sourceID, table, getFieldNames(namedValues))

	// Build the proper WriteRecordsRequest
	writeReq := &tabularpb.WriteRecordsRequest{
		Data: &tabularpb.WriteRecordsData{
			SourceId: sourceID,
			Table:    table,
			InsertAt: -1, // Append to end
			Records: []*tabularpb.Record{
				{
					NamedValues: namedValues,
				},
			},
			Options: &tabularpb.WriteOptions{
				ValueInputOption: "USER_ENTERED", // Parse values (dates, numbers, etc.)
			},
		},
	}

	// Execute via provider
	response, err := uc.services.Provider.WriteRecords(ctx, writeReq)
	if err != nil {
		log.Printf("[WriteRecordSimple] Failed to write record: %v", err)
		return &tabularpb.WriteRecordSimpleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "WRITE_RECORD_FAILED",
				Message: fmt.Sprintf("Failed to write record: %v", err),
			},
		}, nil
	}

	if !response.Success {
		errMsg := "Write failed"
		if response.Error != nil {
			errMsg = response.Error.Message
		}
		log.Printf("[WriteRecordSimple] Provider returned failure: %s", errMsg)
		return &tabularpb.WriteRecordSimpleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "WRITE_RECORD_FAILED",
				Message: errMsg,
			},
		}, nil
	}

	// Extract result
	recordsWritten := int32(0)
	location := ""
	if len(response.Data) > 0 {
		recordsWritten = response.Data[0].RecordsWritten
		location = response.Data[0].Location
	}

	log.Printf("[WriteRecordSimple] Successfully wrote %d record(s) to %s", recordsWritten, location)

	return &tabularpb.WriteRecordSimpleResponse{
		Success: true,
		Data: []*tabularpb.WriteRecordSimpleResult{
			{
				RecordsWritten: recordsWritten,
				Location:       location,
			},
		},
	}, nil
}

// structValueToFieldValue converts a protobuf Struct Value to a tabularpb.FieldValue
func structValueToFieldValue(v *structpb.Value) *tabularpb.FieldValue {
	if v == nil {
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_NULL,
		}
	}

	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_NULL,
		}
	case *structpb.Value_StringValue:
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_STRING,
			Value:     &tabularpb.FieldValue_StringValue{StringValue: kind.StringValue},
		}
	case *structpb.Value_NumberValue:
		// Check if it's an integer or float
		if kind.NumberValue == float64(int64(kind.NumberValue)) {
			return &tabularpb.FieldValue{
				FieldType: tabularpb.FieldType_FIELD_TYPE_INTEGER,
				Value:     &tabularpb.FieldValue_IntegerValue{IntegerValue: int64(kind.NumberValue)},
			}
		}
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_FLOAT,
			Value:     &tabularpb.FieldValue_FloatValue{FloatValue: kind.NumberValue},
		}
	case *structpb.Value_BoolValue:
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_BOOLEAN,
			Value:     &tabularpb.FieldValue_BooleanValue{BooleanValue: kind.BoolValue},
		}
	default:
		// For structs and lists, convert to string representation
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_STRING,
			Value:     &tabularpb.FieldValue_StringValue{StringValue: fmt.Sprintf("%v", v.AsInterface())},
		}
	}
}

// getFieldNames extracts field names from namedValues map for logging
func getFieldNames(namedValues map[string]*tabularpb.FieldValue) []string {
	names := make([]string, 0, len(namedValues))
	for k := range namedValues {
		names = append(names, k)
	}
	return names
}
