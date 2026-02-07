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
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/noop"

	// --- Database (mock) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"

	// --- Email (mock) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/email/mock"

	// --- ID (noop + uuidv7 — both have stub.go, zero-cost) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/id/noop"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/id/uuidv7"

	// --- Payment (mock) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/payment/mock"

	// --- Scheduler (mock) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/scheduler/mock"

	// --- Storage (mock + local) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/local"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/mock"

	// --- Tabular (mock — no build tag, always compiles) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/tabular/mock"

	// --- Translation (noop + file + mock — all have stubs) ---
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation/file"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation/mock"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation/noop"
)
