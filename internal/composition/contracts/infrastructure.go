package contracts

import (
	"context"
)

// ============================================================================
// Health Check Interfaces
// ============================================================================

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	// Name returns the health checker name
	Name() string

	// Check performs health check
	Check(ctx context.Context) HealthStatus
}

// HealthStatus represents the health status
type HealthStatus struct {
	Name     string            `json:"name"`
	Status   string            `json:"status"` // "healthy", "unhealthy", "degraded"
	Message  string            `json:"message,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
	Duration string            `json:"duration"`
}

// ============================================================================
// Metrics Interfaces
// ============================================================================

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// Counter increments a counter metric
	Counter(name string, tags map[string]string) Counter

	// Gauge records a gauge metric
	Gauge(name string, tags map[string]string) Gauge

	// Histogram records a histogram metric
	Histogram(name string, tags map[string]string) Histogram

	// Summary records a summary metric
	Summary(name string, tags map[string]string) Summary
}

// Counter defines the interface for counter metrics
type Counter interface {
	Add(value float64)
	With(tags map[string]string) Counter
}

// Gauge defines the interface for gauge metrics
type Gauge interface {
	Set(value float64)
	Add(value float64)
	With(tags map[string]string) Gauge
}

// Histogram defines the interface for histogram metrics
type Histogram interface {
	Observe(value float64)
	With(tags map[string]string) Histogram
}

// Summary defines the interface for summary metrics
type Summary interface {
	Observe(value float64)
	With(tags map[string]string) Summary
}

// ============================================================================
// Logging Interface
// ============================================================================

// Logger defines the interface for logging
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...interface{})

	// Info logs an info message
	Info(msg string, fields ...interface{})

	// Warn logs a warning message
	Warn(msg string, fields ...interface{})

	// Error logs an error message
	Error(msg string, fields ...interface{})

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...interface{})
}

// ============================================================================
// Event Bus Interfaces
// ============================================================================

// EventBus defines the interface for event bus
type EventBus interface {
	// Publish publishes an event
	Publish(ctx context.Context, event Event) error

	// Subscribe subscribes to events
	Subscribe(eventType string, handler EventHandler) error

	// Unsubscribe unsubscribes from events
	Unsubscribe(eventType string, handler EventHandler) error
}

// Event represents an application event
type Event interface {
	// Type returns the event type
	Type() string

	// Data returns the event data
	Data() interface{}

	// Timestamp returns the event timestamp
	Timestamp() int64

	// ID returns the event ID
	ID() string
}

// EventHandler defines the interface for event handlers
type EventHandler interface {
	// Handle handles an event
	Handle(ctx context.Context, event Event) error
}

// ============================================================================
// Cache Interface
// ============================================================================

// Cache defines the interface for caching
type Cache interface {
	// Get gets a value from cache
	Get(key string) (interface{}, error)

	// Set sets a value in cache
	Set(key string, value interface{}, ttl int64) error

	// Delete deletes a value from cache
	Delete(key string) error

	// Clear clears all values from cache
	Clear() error
}

// ============================================================================
// Distributed Locking Interface
// ============================================================================

// Locker defines the interface for distributed locking
type Locker interface {
	// Lock acquires a lock
	Lock(ctx context.Context, key string, ttl int64) (Lock, error)

	// TryLock tries to acquire a lock without blocking
	TryLock(ctx context.Context, key string, ttl int64) (Lock, error)
}

// Lock defines the interface for a distributed lock
type Lock interface {
	// Unlock releases the lock
	Unlock() error

	// Refresh refreshes the lock TTL
	Refresh(ttl int64) error

	// IsLocked returns whether the lock is still active
	IsLocked() bool
}
