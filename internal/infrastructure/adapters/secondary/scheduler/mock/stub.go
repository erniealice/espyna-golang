//go:build !mock_scheduler

package mock

// Stub — when mock_scheduler tag is absent, this file provides the package
// declaration so Go tooling can resolve the package. The real adapter lives
// in adapter.go (gated by //go:build mock_scheduler).
