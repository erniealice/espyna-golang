// Package middleware provides HTTP middleware implementations for espyna consumer
// apps. This is the internal implementation layer — the public API surface is
// consumer/http/middleware, which re-exports these implementations as thin wrappers.
package middleware

import "net/http"

// MiddlewareFunc is a function that wraps an http.Handler with middleware
// behaviour. This is the standard Go middleware signature.
type MiddlewareFunc func(http.Handler) http.Handler
