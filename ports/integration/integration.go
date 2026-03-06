// Package integration re-exports internal integration port types for use by contrib sub-modules.
package integration

import (
	internal "github.com/erniealice/espyna-golang/internal/application/ports/integration"
)

// Tabular types
type (
	TabularSourceProvider = internal.TabularSourceProvider
	SpreadsheetExtensions = internal.SpreadsheetExtensions
	TabularOptions        = internal.TabularOptions
	TabularRecord         = internal.TabularRecord
	TabularSelection      = internal.TabularSelection
	TabularTable          = internal.TabularTable
	TabularSchema         = internal.TabularSchema
)

// Payment types
type (
	IntegrationPaymentRepository = internal.IntegrationPaymentRepository
	PaymentProvider              = internal.PaymentProvider
	PaymentWebhookResult         = internal.PaymentWebhookResult
	CheckoutSessionParams        = internal.CheckoutSessionParams
)

// Email types
type (
	EmailProvider = internal.EmailProvider
)

// Scheduler types
type (
	SchedulerProvider = internal.SchedulerProvider
)
