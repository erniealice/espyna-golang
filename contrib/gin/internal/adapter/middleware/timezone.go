//go:build gin

package middleware

// timezone.go — per-request timezone resolution for Gin.
//
// TODO: Implement TimezoneMiddleware for Gin. This middleware should mirror the
// vanilla net/http implementation
// (apps/service-admin/internal/infrastructure/input/http/middleware/timezone.go):
//
//  1. Skip static assets (/assets/*, /favicon.ico).
//  2. Extract user ID from context (set by authentication middleware).
//  3. Look up user.timezone from the User use case (with per-uid LRU cache).
//  4. Resolve timezone to *time.Location via pyezatypes.LoadLocationOrDefault.
//  5. Stash the location on the request context via pyezatypes.WithLocation.
//
// Downstream views call pyezatypes.LocationFromContext(ctx) to render dates
// in the operator's preferred timezone.
//
// Public surface needed:
//   - NewTimezoneMiddleware(useCases *consumer.UseCases) *TimezoneMiddleware
//   - (m *TimezoneMiddleware) Handle() gin.HandlerFunc
//   - (m *TimezoneMiddleware) InvalidateUserCache(userID string)
//
// Blocked on: Server API — requires the consumer.UseCases to be wired through
// the Gin adapter's Initialize method, and the pyeza types package for
// location context propagation.
