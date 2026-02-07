package scheduler

import (
	"fmt"
	"os"
	"strings"
)

// =============================================================================
// CONTAINER INTERFACE
// =============================================================================

// Container defines the interface for container configuration
type Container interface {
	GetConfig() interface{}
	SetConfig(interface{})
}

// ContainerOption defines a function that can configure a Container
type ContainerOption func(Container) error

// SchedulerConfigSetter defines methods for setting scheduler configuration
type SchedulerConfigSetter interface {
	SetSchedulerConfig(config interface{})
}

// =============================================================================
// SCHEDULER CONFIGURATION TYPES
// =============================================================================

// SchedulerConfig holds configuration for scheduler providers
type SchedulerConfig struct {
	Calendly       *CalendlyConfig       `json:"calendly,omitempty"`
	GoogleCalendar *GoogleCalendarConfig `json:"google_calendar,omitempty"`
	Mock           bool                  `json:"mock,omitempty"`
}

// CalendlyConfig holds configuration for Calendly scheduler provider
type CalendlyConfig struct {
	AccessToken        string `json:"access_token"`
	DefaultEventTypeID string `json:"default_event_type_id,omitempty"`
	UserURI            string `json:"user_uri,omitempty"`
	OrganizationURI    string `json:"organization_uri,omitempty"`
	WebhookSecret      string `json:"webhook_secret,omitempty"`
	APIBaseURL         string `json:"api_base_url,omitempty"`
}

// Validate validates the Calendly configuration
func (c CalendlyConfig) Validate() error {
	if c.AccessToken == "" {
		return fmt.Errorf("calendly access token is required")
	}
	return nil
}

// GoogleCalendarConfig holds configuration for Google Calendar scheduler provider
type GoogleCalendarConfig struct {
	CredentialsPath string `json:"credentials_path"`
	TokenPath       string `json:"token_path,omitempty"`
	CalendarID      string `json:"calendar_id,omitempty"`
}

// Validate validates the Google Calendar configuration
func (c GoogleCalendarConfig) Validate() error {
	if c.CredentialsPath == "" {
		return fmt.Errorf("google calendar credentials path is required")
	}
	return nil
}

// =============================================================================
// ENVIRONMENT CONFIGURATION LOADERS
// =============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createCalendlyConfigFromEnv() CalendlyConfig {
	return CalendlyConfig{
		AccessToken:        getEnv("CALENDLY_PERSONAL_ACCESS_TOKEN", ""),
		DefaultEventTypeID: getEnv("CALENDLY_DEFAULT_EVENT_TYPE_ID", ""),
		UserURI:            getEnv("CALENDLY_USER_URI", ""),
		OrganizationURI:    getEnv("CALENDLY_ORGANIZATION_URI", ""),
		WebhookSecret:      getEnv("CALENDLY_WEBHOOK_SECRET", ""),
		APIBaseURL:         getEnv("CALENDLY_API_BASE_URL", ""),
	}
}

func createGoogleCalendarConfigFromEnv() GoogleCalendarConfig {
	return GoogleCalendarConfig{
		CredentialsPath: getEnv("GOOGLE_CALENDAR_CREDENTIALS_PATH", ""),
		TokenPath:       getEnv("GOOGLE_CALENDAR_TOKEN_PATH", ""),
		CalendarID:      getEnv("GOOGLE_CALENDAR_ID", "primary"),
	}
}

// =============================================================================
// SCHEDULER PROVIDER OPTIONS
// =============================================================================

// WithSchedulerFromEnv dynamically selects scheduler provider based on CONFIG_SCHEDULER_PROVIDER
func WithSchedulerFromEnv() ContainerOption {
	return func(c Container) error {
		schedulerProvider := strings.ToLower(getEnv("CONFIG_SCHEDULER_PROVIDER", "mock"))

		switch schedulerProvider {
		case "calendly":
			return WithCalendly(createCalendlyConfigFromEnv())(c)
		case "google_calendar":
			return WithGoogleCalendar(createGoogleCalendarConfigFromEnv())(c)
		case "mock", "mock_scheduler", "":
			return WithMockScheduler()(c)
		default:
			return fmt.Errorf("unsupported scheduler provider: %s", schedulerProvider)
		}
	}
}

// WithCalendly configures Calendly as scheduler provider
func WithCalendly(config CalendlyConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid calendly configuration: %w", err)
		}

		if setter, ok := c.(SchedulerConfigSetter); ok {
			setter.SetSchedulerConfig(SchedulerConfig{Calendly: &config})
		} else {
			return fmt.Errorf("container does not implement SetSchedulerConfig method")
		}

		fmt.Printf("📅 Configured Calendly scheduler\n")
		return nil
	}
}

// WithGoogleCalendar configures Google Calendar as scheduler provider
func WithGoogleCalendar(config GoogleCalendarConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid google calendar configuration: %w", err)
		}

		if setter, ok := c.(SchedulerConfigSetter); ok {
			setter.SetSchedulerConfig(SchedulerConfig{GoogleCalendar: &config})
		} else {
			return fmt.Errorf("container does not implement SetSchedulerConfig method")
		}

		fmt.Printf("📅 Configured Google Calendar scheduler\n")
		return nil
	}
}

// WithMockScheduler configures mock scheduler for testing/development
func WithMockScheduler() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(SchedulerConfigSetter); ok {
			setter.SetSchedulerConfig(SchedulerConfig{Mock: true})
		} else {
			return fmt.Errorf("container does not implement SetSchedulerConfig method")
		}

		fmt.Printf("🧪 Configured mock scheduler\n")
		return nil
	}
}
