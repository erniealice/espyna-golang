//go:build mock_db && mock_auth

package event

import (
	"context"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/event"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

func TestGetEventItemPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with test data
	mockRepo := event.NewMockEventRepository("education")

	// Create a test event
	now := time.Now()
	startTime := now.Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	createResp, err := mockRepo.CreateEvent(ctx, &eventpb.CreateEventRequest{
		Data: &eventpb.Event{
			Name:             "Test Event",
			Description:      stringPtr("This is a test event for educational purposes"),
			Active:           true,
			StartDateTimeUtc: startTime.Unix(),
			EndDateTimeUtc:   endTime.Unix(),
			Timezone:         "America/New_York",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	testEventId := createResp.Data[0].Id

	// Setup repositories and services
	repos := GetEventItemPageDataRepositories{
		Event: mockRepo,
	}
	services := GetEventItemPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetEventItemPageDataUseCase(repos, services)

	// Test successful retrieval
	t.Run("SuccessfulRetrieval", func(t *testing.T) {
		req := &eventpb.GetEventItemPageDataRequest{
			EventId: testEventId,
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Expected success to be true")
		}

		if resp.Event == nil {
			t.Fatalf("Expected event to be returned")
		}

		if resp.Event.Id != testEventId {
			t.Fatalf("Expected event ID %s, got: %s", testEventId, resp.Event.Id)
		}

		if resp.Event.Name != "Test Event" {
			t.Fatalf("Expected event name 'Test Event', got: %s", resp.Event.Name)
		}

		if resp.Event.Description == nil || *resp.Event.Description != "This is a test event for educational purposes" {
			t.Fatalf("Expected correct description")
		}

		if resp.Event.StartDateTimeUtc != startTime.Unix() {
			t.Fatalf("Expected start time %d, got: %d", startTime.Unix(), resp.Event.StartDateTimeUtc)
		}

		if resp.Event.EndDateTimeUtc != endTime.Unix() {
			t.Fatalf("Expected end time %d, got: %d", endTime.Unix(), resp.Event.EndDateTimeUtc)
		}

		if resp.Event.Timezone != "America/New_York" {
			t.Fatalf("Expected timezone 'America/New_York', got: %s", resp.Event.Timezone)
		}

		// Check that string representations are enhanced
		if resp.Event.StartDateTimeUtcString == nil {
			t.Fatalf("Expected start time string to be set")
		}

		if resp.Event.EndDateTimeUtcString == nil {
			t.Fatalf("Expected end time string to be set")
		}
	})

	// Test event not found
	t.Run("EventNotFound", func(t *testing.T) {
		req := &eventpb.GetEventItemPageDataRequest{
			EventId: "non-existent-id",
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for non-existent event")
		}
	})
}

func TestGetEventItemPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := event.NewMockEventRepository("education")

	repos := GetEventItemPageDataRepositories{
		Event: mockRepo,
	}
	services := GetEventItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetEventItemPageDataUseCase(repos, services)

	// Test nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Fatalf("Expected error for nil request")
		}
	})

	// Test empty event ID
	t.Run("EmptyEventId", func(t *testing.T) {
		req := &eventpb.GetEventItemPageDataRequest{
			EventId: "",
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for empty event ID")
		}
	})

	// Test too short event ID
	t.Run("TooShortEventId", func(t *testing.T) {
		req := &eventpb.GetEventItemPageDataRequest{
			EventId: "ab", // Less than 3 characters
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Fatalf("Expected error for too short event ID")
		}
	})
}

func TestGetEventItemPageDataUseCase_TimezoneHandling(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := event.NewMockEventRepository("education")

	// Test with various timezones
	testCases := []struct {
		name     string
		timezone string
	}{
		{"UTC", "UTC"},
		{"NewYork", "America/New_York"},
		{"London", "Europe/London"},
		{"Tokyo", "Asia/Tokyo"},
		{"Empty", ""},                   // Should default to UTC
		{"Invalid", "Invalid/Timezone"}, // Should fallback to UTC
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create event with specific timezone
			now := time.Now()
			startTime := now.Add(1 * time.Hour)
			endTime := startTime.Add(1 * time.Hour)

			createResp, err := mockRepo.CreateEvent(ctx, &eventpb.CreateEventRequest{
				Data: &eventpb.Event{
					Name:             "Timezone Test Event",
					Active:           true,
					StartDateTimeUtc: startTime.Unix(),
					EndDateTimeUtc:   endTime.Unix(),
					Timezone:         tc.timezone,
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test event: %v", err)
			}

			testEventId := createResp.Data[0].Id

			// Setup use case
			repos := GetEventItemPageDataRepositories{
				Event: mockRepo,
			}
			services := GetEventItemPageDataServices{
				TransactionService: nil,
				TranslationService: nil,
			}
			useCase := NewGetEventItemPageDataUseCase(repos, services)

			// Execute
			req := &eventpb.GetEventItemPageDataRequest{
				EventId: testEventId,
			}

			resp, err := useCase.Execute(ctx, req)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if !resp.Success {
				t.Fatalf("Expected success to be true")
			}

			// Verify timezone handling
			if resp.Event.Timezone != tc.timezone {
				t.Fatalf("Expected timezone %s, got: %s", tc.timezone, resp.Event.Timezone)
			}

			// Verify that string representations are properly formatted
			if resp.Event.StartDateTimeUtcString != nil {
				// Should be a valid RFC3339 format
				_, parseErr := time.Parse(time.RFC3339, *resp.Event.StartDateTimeUtcString)
				if parseErr != nil {
					t.Fatalf("Expected valid RFC3339 format for start time string, got: %s", *resp.Event.StartDateTimeUtcString)
				}
			}

			if resp.Event.EndDateTimeUtcString != nil {
				// Should be a valid RFC3339 format
				_, parseErr := time.Parse(time.RFC3339, *resp.Event.EndDateTimeUtcString)
				if parseErr != nil {
					t.Fatalf("Expected valid RFC3339 format for end time string, got: %s", *resp.Event.EndDateTimeUtcString)
				}
			}
		})
	}
}

func TestGetEventItemPageDataUseCase_SchedulingEnhancement(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := event.NewMockEventRepository("education")

	// Create events with different timing scenarios
	now := time.Now()

	testCases := []struct {
		name      string
		startTime time.Time
		endTime   time.Time
		timezone  string
	}{
		{
			name:      "PastEvent",
			startTime: now.Add(-2 * time.Hour),
			endTime:   now.Add(-1 * time.Hour),
			timezone:  "UTC",
		},
		{
			name:      "OngoingEvent",
			startTime: now.Add(-30 * time.Minute),
			endTime:   now.Add(30 * time.Minute),
			timezone:  "America/New_York",
		},
		{
			name:      "FutureEvent",
			startTime: now.Add(1 * time.Hour),
			endTime:   now.Add(2 * time.Hour),
			timezone:  "Europe/London",
		},
		{
			name:      "AllDayEvent",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			endTime:   time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC),
			timezone:  "UTC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create event
			createResp, err := mockRepo.CreateEvent(ctx, &eventpb.CreateEventRequest{
				Data: &eventpb.Event{
					Name:             tc.name,
					Active:           true,
					StartDateTimeUtc: tc.startTime.Unix(),
					EndDateTimeUtc:   tc.endTime.Unix(),
					Timezone:         tc.timezone,
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test event: %v", err)
			}

			testEventId := createResp.Data[0].Id

			// Setup use case
			repos := GetEventItemPageDataRepositories{
				Event: mockRepo,
			}
			services := GetEventItemPageDataServices{
				TransactionService: nil,
				TranslationService: nil,
			}
			useCase := NewGetEventItemPageDataUseCase(repos, services)

			// Execute
			req := &eventpb.GetEventItemPageDataRequest{
				EventId: testEventId,
			}

			resp, err := useCase.Execute(ctx, req)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if !resp.Success {
				t.Fatalf("Expected success to be true")
			}

			// Verify basic event data
			if resp.Event.Id != testEventId {
				t.Fatalf("Expected event ID %s, got: %s", testEventId, resp.Event.Id)
			}

			if resp.Event.Name != tc.name {
				t.Fatalf("Expected event name %s, got: %s", tc.name, resp.Event.Name)
			}

			// Verify time data consistency
			if resp.Event.StartDateTimeUtc != tc.startTime.Unix() {
				t.Fatalf("Expected start time %d, got: %d", tc.startTime.Unix(), resp.Event.StartDateTimeUtc)
			}

			if resp.Event.EndDateTimeUtc != tc.endTime.Unix() {
				t.Fatalf("Expected end time %d, got: %d", tc.endTime.Unix(), resp.Event.EndDateTimeUtc)
			}

			// Verify timezone handling
			if resp.Event.Timezone != tc.timezone {
				t.Fatalf("Expected timezone %s, got: %s", tc.timezone, resp.Event.Timezone)
			}

			// Verify enhanced string fields are present
			if resp.Event.StartDateTimeUtcString == nil {
				t.Fatalf("Expected start time string to be enhanced")
			}

			if resp.Event.EndDateTimeUtcString == nil {
				t.Fatalf("Expected end time string to be enhanced")
			}

			// TODO: In future iterations, test additional scheduling enhancements:
			// - Duration calculations
			// - Conflict detection
			// - Calendar context (is_past, is_future, etc.)
			// - Related event loading
			// - Attendee information
			// - Resource availability
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
