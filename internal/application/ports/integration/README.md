# ports/integration

Third-party provider contracts. This package is mixed: some interfaces are genuine
Go-lifecycle ports; others are message shapes that belong in proto.

## Residents

| Interface | Type | Note |
|-----------|------|------|
| `EmailProvider` | **Genuine port** | Lifecycle (`Initialize`, `Close`), capability discovery, `GetInboxMessages` streaming concern. HTTP-specific behavior stays here. |
| `PaymentProvider` | **Genuine port** | Lifecycle + webhook HTTP handling (`ProcessWebhook`) is HTTP-specific and cannot be a simple proto RPC. |
| `IntegrationPaymentRepository` | **Migrating** | `LogWebhook(ctx, *paymentpb.LogWebhookRequest)` — pure request/response with proto types. Should move to a proto service. |
| `SchedulerProvider` | **Genuine port** | Lifecycle + scheduling-service callback handling. |
| `FulfillmentProvider` | **Genuine port** | Logistics provider lifecycle. Uses plain Go structs today because esqyma has no `integration/fulfillment` proto yet; migrate types when the proto is authored. |

## Criteria for staying here

An integration interface stays in this package when it:

1. Manages provider lifecycle (`Initialize`, `Close`, `IsHealthy`) — Go-specific.
2. Handles HTTP-specific webhook/callback mechanics — no proto equivalent.
3. Declares capability discovery (`GetCapabilities`) — runtime Go interface concern.

Pure request/response message shapes (no lifecycle, no HTTP specifics) should be
expressed as proto services under `proto/v1/integration/` or
`proto/v1/service/<X>/`.

## When to add a file here

When integrating a new third-party provider that requires lifecycle management or
HTTP-callback handling. Evaluate whether the message shapes alone can live in
`proto/v1/integration/` and only put the lifecycle contract here.
