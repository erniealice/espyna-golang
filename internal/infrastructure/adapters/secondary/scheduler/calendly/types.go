//go:build calendly

package calendly

// CalendlyWebhookPayload represents the incoming Calendly webhook structure
type CalendlyWebhookPayload struct {
	Event     string              `json:"event"`
	CreatedAt string              `json:"created_at"`
	Payload   CalendlyInviteeData `json:"payload"`
}

// CalendlyInviteeData contains the invitee information from webhook
type CalendlyInviteeData struct {
	URI                 string                   `json:"uri"`
	Name                string                   `json:"name"`
	Email               string                   `json:"email"`
	Status              string                   `json:"status"`
	Timezone            string                   `json:"timezone"`
	Event               string                   `json:"event"`
	CancelURL           string                   `json:"cancel_url"`
	RescheduleURL       string                   `json:"reschedule_url"`
	OldInvitee          string                   `json:"old_invitee"`
	ScheduledEvent      *CalendlyScheduledEvent  `json:"scheduled_event"`
	QuestionsAndAnswers []CalendlyQuestionAnswer `json:"questions_and_answers"`
	Tracking            *CalendlyTracking        `json:"tracking"`
}

// CalendlyScheduledEvent contains scheduled event details
type CalendlyScheduledEvent struct {
	URI             string `json:"uri"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	EventType       string `json:"event_type"`
	Location        string `json:"location"`
	MeetingNotes    string `json:"meeting_notes"`
	CancellationURL string `json:"cancellation_url"`
	RescheduleURL   string `json:"reschedule_url"`
}

// CalendlyQuestionAnswer represents a custom question and answer
type CalendlyQuestionAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Position int    `json:"position"`
}

// CalendlyTracking contains UTM tracking data
type CalendlyTracking struct {
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
	UTMContent  string `json:"utm_content"`
	UTMTerm     string `json:"utm_term"`
}

// CalendlyEvent represents a scheduled event
type CalendlyEvent struct {
	URI             string            `json:"uri"`
	Name            string            `json:"name"`
	Status          string            `json:"status"`
	StartTime       string            `json:"start_time"`
	EndTime         string            `json:"end_time"`
	EventType       string            `json:"event_type"`
	Location        *CalendlyLocation `json:"location"`
	CancellationURL string            `json:"cancellation"`
	RescheduleURL   string            `json:"reschedule"`
	Timezone        string            `json:"event_memberships"`
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
}

// CalendlyLocation represents event location
type CalendlyLocation struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	JoinURL  string `json:"join_url"`
	Status   string `json:"status"`
}

// CalendlyEventResponse wraps a single event response
type CalendlyEventResponse struct {
	Resource CalendlyEvent `json:"resource"`
}

// CalendlyListEventsResponse wraps a list of events
type CalendlyListEventsResponse struct {
	Collection []CalendlyEvent        `json:"collection"`
	Pagination CalendlyPaginationInfo `json:"pagination"`
}

// CalendlyPaginationInfo contains pagination details
type CalendlyPaginationInfo struct {
	Count         int    `json:"count"`
	NextPage      string `json:"next_page"`
	NextPageToken string `json:"next_page_token"`
	PreviousPage  string `json:"previous_page"`
}

// CalendlyEventTypeInfo represents an event type
type CalendlyEventTypeInfo struct {
	URI           string `json:"uri"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
	Slug          string `json:"slug"`
	Duration      int    `json:"duration"`
	SchedulingURL string `json:"scheduling_url"`
	Description   string `json:"description_plain"`
	Color         string `json:"color"`
	Secret        bool   `json:"secret"`
	Type          string `json:"type"`
}

// CalendlyEventTypesResponse wraps a list of event types
type CalendlyEventTypesResponse struct {
	Collection []CalendlyEventTypeInfo `json:"collection"`
}

// CalendlyEventTypeResponse wraps a single event type
type CalendlyEventTypeResponse struct {
	Resource CalendlyEventTypeInfo `json:"resource"`
}

// CalendlyAvailableTime represents an available time slot
type CalendlyAvailableTime struct {
	StartTime         string `json:"start_time"`
	Status            string `json:"status"`
	SchedulingURL     string `json:"scheduling_url"`
	InviteesRemaining int    `json:"invitees_remaining"`
	Duration          int    `json:"duration"`
}

// CalendlyAvailabilityResponse wraps availability results
type CalendlyAvailabilityResponse struct {
	Collection []CalendlyAvailableTime `json:"collection"`
}
