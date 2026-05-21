//go:build mock_db && mock_auth

package event

import (
	"context"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/event"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

func TestGetEventListPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with some test data
	mockRepo := event.NewMockEventRepository("education")

	// Create test events with different times for time-based filtering tests
	now := time.Now()
	pastTime := now.Add(-2 * time.Hour)
	futureTime := now.Add(2 * time.Hour)

	// Past event
	_, err := mockRepo.CreateEvent(ctx, &eventpb.CreateEventRequest{
		Data: &eventpb.Event{
			Name:             "Past Event",
			Description:      nil,
			Active:           true,
			StartDateTimeUtc: pastTime.Unix(),
			EndDateTimeUtc:   pastTime.Add(1 * time.Hour).Unix(),
			Timezone:         "UTC",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test event 1: %v", err)
	}

	// Future event
	_, err = mockRepo.CreateEvent(ctx, &eventpb.CreateEventRequest{
		Data: &eventpb.Event{
			Name:             "Future Event",
			Description:      nil,
			Active:           true,
			StartDateTimeUtc: futureTime.Unix(),
			EndDateTimeUtc:   futureTime.Add(1 * time.Hour).Unix(),
			Timezone:         "America/New_York",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test event 2: %v", err)
	}

	// Current/ongoing event
	_, err = mockRepo.CreateEvent(ctx, &eventpb.CreateEventRequest{
		Data: &eventpb.Event{
			Name:             "Current Event",
			Description:      nil,
			Active:           true,
			StartDateTimeUtc: now.Add(-30 * time.Minute).Unix(),
			EndDateTimeUtc:   now.Add(30 * time.Minute).Unix(),
			Timezone:         "Europe/London",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test event 3: %v", err)
	}

	// Setup repositories and services
	repos := GetEventListPageDataRepositories{
		Event: mockRepo,
	}
	services := GetEventListPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetEventListPageDataUseCase(repos, services)

	// Test 1: Basic list without any filters
	t.Run("BasicList", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{
						Page: 1,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Expected success to be true")
		}

		if len(resp.EventList) != 3 {
			t.Fatalf("Expected 3 events, got: %d", len(resp.EventList))
		}

		if resp.Pagination == nil {
			t.Fatalf("Expected pagination response")
		}
	})

	// Test 2: Filter by active status
	t.Run("FilterByActive", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "active",
						FilterType: &commonpb.TypedFilter_BooleanFilter{
							BooleanFilter: &commonpb.BooleanFilter{
								Value: true,
							},
						},
					},
				},
			},
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{
						Page: 1,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Expected success to be true")
		}

		// All test events are active, so should get all 3
		if len(resp.EventList) != 3 {
			t.Fatalf("Expected 3 active events, got: %d", len(resp.EventList))
		}
	})

	// Test 3: Sort by start time
	t.Run("SortByStartTime", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "start_date_time_utc",
						Direction: commonpb.SortDirection_ASC,
					},
				},
			},
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{
						Page: 1,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Expected success to be true")
		}

		if len(resp.EventList) != 3 {
			t.Fatalf("Expected 3 events, got: %d", len(resp.EventList))
		}

		// Check that events are sorted by start time (ascending)
		for i := 1; i < len(resp.EventList); i++ {
			if resp.EventList[i-1].StartDateTimeUtc > resp.EventList[i].StartDateTimeUtc {
				t.Fatalf("Events not sorted correctly by start time")
			}
		}
	})

	// Test 4: Search by name
	t.Run("SearchByName", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Search: &commonpb.SearchRequest{
				Query: "Future",
				Options: &commonpb.SearchOptions{
					SearchFields: []string{"name"},
					MaxResults:   10,
				},
			},
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{
						Page: 1,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Expected success to be true")
		}

		// Should find 1 event with "Future" in the name
		if len(resp.EventList) != 1 {
			t.Fatalf("Expected 1 event matching search, got: %d", len(resp.EventList))
		}

		if resp.EventList[0].Name != "Future Event" {
			t.Fatalf("Expected 'Future Event', got: %s", resp.EventList[0].Name)
		}
	})

	// Test 5: Pagination
	t.Run("Pagination", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 2, // Limit to 2 events per page
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{
						Page: 1,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Expected success to be true")
		}

		if len(resp.EventList) != 2 {
			t.Fatalf("Expected 2 events on first page, got: %d", len(resp.EventList))
		}

		if resp.Pagination == nil {
			t.Fatalf("Expected pagination response")
		}

		if resp.Pagination.TotalItems != 3 {
			t.Fatalf("Expected 3 total items, got: %d", resp.Pagination.TotalItems)
		}
	})
}

func TestGetEventListPageDataUseCase_Execute_EmptyList(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with no data
	mockRepo := event.NewMockEventRepository("education")

	// Setup repositories and services
	repos := GetEventListPageDataRepositories{
		Event: mockRepo,
	}
	services := GetEventListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	// Create use case
	useCase := NewGetEventListPageDataUseCase(repos, services)

	// Execute
	req := &eventpb.GetEventListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{
			Limit: 10,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{
					Page: 1,
				},
			},
		},
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Expected success to be true")
	}

	if len(resp.EventList) != 0 {
		t.Fatalf("Expected 0 events, got: %d", len(resp.EventList))
	}

	if resp.Pagination == nil {
		t.Fatalf("Expected pagination response")
	}

	if resp.Pagination.TotalItems != 0 {
		t.Fatalf("Expected 0 total items, got: %d", resp.Pagination.TotalItems)
	}
}

func TestGetEventListPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := event.NewMockEventRepository("education")

	repos := GetEventListPageDataRepositories{
		Event: mockRepo,
	}
	services := GetEventListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetEventListPageDataUseCase(repos, services)

	// Test invalid request (nil)
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Fatalf("Expected error for nil request")
		}
	})

	// Test invalid pagination
	t.Run("InvalidPagination", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 101, // Over the limit
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{
						Page: 1,
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for invalid pagination limit")
		}
	})

	// Test invalid filter field
	t.Run("InvalidFilterField", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "invalid_field",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value:    "value",
								Operator: commonpb.StringOperator_STRING_EQUALS,
							},
						},
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for invalid filter field")
		}
	})

	// Test invalid sort field
	t.Run("InvalidSortField", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "invalid_field",
						Direction: commonpb.SortDirection_ASC,
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for invalid sort field")
		}
	})

	// Test empty search query
	t.Run("EmptySearchQuery", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Search: &commonpb.SearchRequest{
				Query: "", // Empty query
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for empty search query")
		}
	})
}

func TestGetEventListPageDataUseCase_TimeBasedValidation(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := event.NewMockEventRepository("education")

	repos := GetEventListPageDataRepositories{
		Event: mockRepo,
	}
	services := GetEventListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetEventListPageDataUseCase(repos, services)

	// Test valid time-based filters
	t.Run("ValidTimeRangeFilter", func(t *testing.T) {
		now := time.Now()
		req := &eventpb.GetEventListPageDataRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "start_date_time_utc",
						FilterType: &commonpb.TypedFilter_DateFilter{
							DateFilter: &commonpb.DateFilter{
								Value:    now.Add(-1 * time.Hour).Format(time.RFC3339),
								Operator: commonpb.DateOperator_DATE_BETWEEN,
								RangeEnd: stringPtr(now.Add(1 * time.Hour).Format(time.RFC3339)),
							},
						},
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error for valid time range filter, got: %v", err)
		}
	})

	// Test invalid time-based filters
	t.Run("InvalidTimeRangeFilter", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "start_date_time_utc",
						FilterType: &commonpb.TypedFilter_DateFilter{
							DateFilter: &commonpb.DateFilter{
								Value:    "invalid_date",
								Operator: commonpb.DateOperator_DATE_BETWEEN,
								// Missing range_end for BETWEEN operator
							},
						},
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for invalid time range filter")
		}
	})

	// Test duration filter validation
	t.Run("ValidDurationFilter", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "duration",
						FilterType: &commonpb.TypedFilter_NumberFilter{
							NumberFilter: &commonpb.NumberFilter{
								Value:    3600, // 1 hour in seconds
								Operator: commonpb.NumberOperator_NUMBER_GREATER_THAN,
							},
						},
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error for valid duration filter, got: %v", err)
		}
	})

	// Test weekday filter validation
	t.Run("ValidWeekdayFilter", func(t *testing.T) {
		req := &eventpb.GetEventListPageDataRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "weekday",
						FilterType: &commonpb.TypedFilter_ListFilter{
							ListFilter: &commonpb.ListFilter{
								Values:   []string{"1", "2", "3"}, // Monday, Tuesday, Wednesday
								Operator: commonpb.ListOperator_LIST_IN,
							},
						},
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error for valid weekday filter, got: %v", err)
		}
	})
}
