//go:build postgresql

// Package expenditure — attempt-row methods for the expense_recognition_run
// adapter.
//
// Attempt CRUD is NOT on the ExpenseRecognitionRunDomainServiceServer
// interface; the run engine reaches these methods via the concrete
// `*PostgresExpenseRecognitionRunRepository` type. This mirrors the
// revenue_run pattern (attempt methods live on the same repository struct
// because they share the same workspace/tx scope as the parent run).
//
// Schema notes (per migration 20260517160000_expense_run_tables.sql):
//   - source_kind + outcome are INTEGER columns; protojson emits the enum
//     names which the helper folds to numeric wire values.
//   - attempted_at is `bigint` (epoch millis), not postgres timestamp; no
//     time.Time conversion is needed on write.
//   - period_start / period_end / period_marker are `text`; passed through.
package expenditure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_run"
)

// foldExpenseRunAttemptEnumStringsToInt collapses protojson enum-name strings
// on the attempt row to numeric wire values for the INTEGER-typed
// source_kind / outcome columns.
func foldExpenseRunAttemptEnumStringsToInt(data map[string]any) {
	if v, ok := data["sourceKind"].(string); ok {
		if num, ok := pb.ExpenseRecognitionRunSourceKind_value[v]; ok {
			data["sourceKind"] = int32(num)
		}
	}
	if v, ok := data["outcome"].(string); ok {
		if num, ok := pb.ExpenseRecognitionRunAttemptOutcome_value[v]; ok {
			data["outcome"] = int32(num)
		}
	}
}

func unmarshalExpenseRecognitionRunAttempt(raw map[string]any) (*pb.ExpenseRecognitionRunAttempt, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw expense_recognition_run_attempt: %w", err)
	}
	a := &pb.ExpenseRecognitionRunAttempt{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, a); err != nil {
		return nil, fmt.Errorf("unmarshal expense_recognition_run_attempt proto: %w", err)
	}
	return a, nil
}

// CreateExpenseRecognitionRunAttempt inserts one attempt row. Called from
// inside the per-selection loop of GenerateExpenseRun. Errors are tolerated
// by the caller (it logs and continues); this method itself returns the row
// on success.
//
// NOTE: this method is NOT on the proto domain-service interface. The run
// engine invokes it via the concrete `*PostgresExpenseRecognitionRunRepository`
// type, mirroring the revenue_run.CreateRevenueRunAttempt pattern.
func (r *PostgresExpenseRecognitionRunRepository) CreateExpenseRecognitionRunAttempt(ctx context.Context, req *pb.CreateExpenseRecognitionRunAttemptRequest) (*pb.CreateExpenseRecognitionRunAttemptResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("expense_recognition_run_attempt data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expense_recognition_run_attempt to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expense_recognition_run_attempt JSON to map: %w", err)
	}
	foldExpenseRunAttemptEnumStringsToInt(data)

	result, err := r.dbOps.Create(ctx, r.attemptTable, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense_recognition_run_attempt: %w", err)
	}
	attempt, err := unmarshalExpenseRecognitionRunAttempt(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateExpenseRecognitionRunAttemptResponse{
		Success: true,
		Data:    []*pb.ExpenseRecognitionRunAttempt{attempt},
	}, nil
}

// ListExpenseRecognitionRunAttempts returns every attempt for a given run, in
// attempt order. RunId on the request is required and is the only filter
// callers currently use; any user-supplied Filters are merged in alongside it.
//
// NOTE: this method is NOT on the proto domain-service interface — same
// repository-level pattern as CreateExpenseRecognitionRunAttempt above.
func (r *PostgresExpenseRecognitionRunRepository) ListExpenseRecognitionRunAttempts(ctx context.Context, req *pb.ListExpenseRecognitionRunAttemptsRequest) (*pb.ListExpenseRecognitionRunAttemptsResponse, error) {
	if req == nil || req.RunId == "" {
		return nil, fmt.Errorf("expense_recognition_run_attempt run_id is required")
	}

	// Build a StringFilter on run_id and merge with any caller-supplied filters.
	runFilter := &commonpb.TypedFilter{
		Field: "run_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    req.RunId,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{runFilter}}
	if req.Filters != nil {
		filters.Filters = append(filters.Filters, req.Filters.Filters...)
	}

	// Default sort: attempted_at ASC (chronological per-run insert order).
	sort := req.Sort
	if sort == nil || len(sort.Fields) == 0 {
		sort = &commonpb.SortRequest{
			Fields: []*commonpb.SortField{{
				Field:     "attempted_at",
				Direction: commonpb.SortDirection_ASC,
			}},
		}
	}

	params := &interfaces.ListParams{Filters: filters, Sort: sort}
	listResult, err := r.dbOps.List(ctx, r.attemptTable, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expense_recognition_run_attempts: %w", err)
	}
	attempts := make([]*pb.ExpenseRecognitionRunAttempt, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		a, err := unmarshalExpenseRecognitionRunAttempt(raw)
		if err != nil {
			log.Printf("WARN: unmarshal expense_recognition_run_attempt row: %v", err)
			continue
		}
		attempts = append(attempts, a)
	}
	return &pb.ListExpenseRecognitionRunAttemptsResponse{
		Success: true,
		Data:    attempts,
	}, nil
}
