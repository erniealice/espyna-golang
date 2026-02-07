# options/integrations/

**External service integration configuration** using Go's Functional Options pattern.

## Purpose

Configures connections to third-party services organized by capability domain:
- **messaging/** - Communication services (email, SMS, push)
- **payment/** - Financial transaction services

## Structure

```
integrations/
├── messaging/
│   └── email.go      # Gmail, Microsoft Graph, Mock
└── payment/
    └── payment.go    # Stripe, AsiaPay, Mock
```

## Messaging Providers

| Provider | Config | Environment Variables |
|----------|--------|-----------------------|
| Gmail | `GmailConfig` | `GMAIL_*` |
| Microsoft | `MicrosoftConfig` | `MICROSOFT_*` |
| Mock | - | - |

## Payment Providers

| Provider | Config | Environment Variables |
|----------|--------|-----------------------|
| AsiaPay | `AsiaPayConfig` | `ASIAPAY_*` |
| Stripe | `StripeConfig` | `STRIPE_*` |
| Mock | - | - |

## Usage

```go
import (
    "leapfor.xyz/espyna/internal/composition/options/integrations/messaging"
    "leapfor.xyz/espyna/internal/composition/options/integrations/payment"
)

// Apply options to container
messaging.WithEmailFromEnv()(container)
payment.WithPaymentFromEnv()(container)
```

## Future Extensions

Additional integration domains can be added:
- `sms/` - SMS providers (Twilio, etc.)
- `push/` - Push notification providers (FCM, APNS)
- `analytics/` - Analytics providers (Segment, etc.)
