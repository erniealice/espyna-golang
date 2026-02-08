package consumer

// register.go — Blank imports for mock/noop/lightweight adapters.
//
// These adapters are SAFE to always compile: they either have no external
// dependencies, or their source files carry build tags that compile to empty
// stubs when the tags are absent (stub.go pattern).
//
// Non-mock adapters with external dependencies live in their own
// register_{category}_{adapter}.go files with matching //go:build tags.

import (
	// --- Auth (mock + noop) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/noop"

	// --- Database (mock) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"

	// --- Email (mock) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/email/mock"

	// --- ID (noop + uuidv7 — both have stub.go, zero-cost) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/id/noop"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/id/uuidv7"

	// --- Payment (mock) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/payment/mock"

	// --- Scheduler (mock) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/scheduler/mock"

	// --- Storage (mock + local) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/storage/local"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/storage/mock"

	// --- Tabular (mock — no build tag, always compiles) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/tabular/mock"

	// --- Translation (noop + file + mock — all have stubs) ---
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/translation/file"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/translation/mock"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/translation/noop"
)
