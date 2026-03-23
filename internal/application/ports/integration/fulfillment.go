package integration

import "context"

// FulfillmentProvider defines the interface for delivery/logistics integrations.
// This interface abstracts delivery services like Lalamove, GrabExpress, etc.
// following the hexagonal architecture pattern established for PaymentProvider and SchedulerProvider.
//
// Note: Request/response types are defined as plain Go structs in this file
// because esqyma does not yet have a fulfillment integration proto package.
// When esqyma/pkg/schema/v1/integration/fulfillment is created, migrate these types.
type FulfillmentProvider interface {
	// Name returns the provider name (e.g., "lalamove", "grabexpress")
	Name() string

	// IsEnabled returns true if the provider is configured and ready
	IsEnabled() bool

	// IsHealthy checks if the provider API is reachable
	IsHealthy(ctx context.Context) error

	// Close releases any resources held by the provider
	Close() error

	// GetCapabilities returns what this provider supports
	GetCapabilities() []string

	// GetQuote requests a delivery price estimate
	GetQuote(ctx context.Context, req *FulfillmentQuoteRequest) (*FulfillmentQuoteResponse, error)

	// CreateDelivery creates a new delivery order
	CreateDelivery(ctx context.Context, req *CreateDeliveryRequest) (*CreateDeliveryResponse, error)

	// CancelDelivery cancels an existing delivery
	CancelDelivery(ctx context.Context, req *CancelDeliveryRequest) (*CancelDeliveryResponse, error)

	// TrackDelivery gets the current status of a delivery
	TrackDelivery(ctx context.Context, req *TrackDeliveryRequest) (*TrackDeliveryResponse, error)

	// ProcessWebhook handles incoming webhook events from the provider
	ProcessWebhook(ctx context.Context, req *FulfillmentWebhookRequest) (*FulfillmentWebhookResponse, error)
}

// FulfillmentQuoteRequest contains delivery quote parameters
type FulfillmentQuoteRequest struct {
	PickupAddress   Address `json:"pickup_address"`
	DeliveryAddress Address `json:"delivery_address"`
	ItemDescription string  `json:"item_description"`
	ItemWeight      float64 `json:"item_weight_kg"`
	ServiceType     string  `json:"service_type"` // e.g., "standard", "express", "same_day"
}

// FulfillmentQuoteResponse contains the delivery quote result
type FulfillmentQuoteResponse struct {
	QuoteID       string  `json:"quote_id"`
	ProviderName  string  `json:"provider_name"`
	PriceAmount   float64 `json:"price_amount"`
	PriceCurrency string  `json:"price_currency"`
	EstimatedTime string  `json:"estimated_time"` // e.g., "45 mins"
	ExpiresAt     string  `json:"expires_at"`     // ISO8601
}

// CreateDeliveryRequest contains delivery creation parameters
type CreateDeliveryRequest struct {
	QuoteID         string  `json:"quote_id,omitempty"` // optional, from GetQuote
	PickupAddress   Address `json:"pickup_address"`
	DeliveryAddress Address `json:"delivery_address"`
	SenderName      string  `json:"sender_name"`
	SenderPhone     string  `json:"sender_phone"`
	RecipientName   string  `json:"recipient_name"`
	RecipientPhone  string  `json:"recipient_phone"`
	ItemDescription string  `json:"item_description"`
	ItemWeight      float64 `json:"item_weight_kg"`
	ServiceType     string  `json:"service_type"`
	Remarks         string  `json:"remarks,omitempty"`
}

// CreateDeliveryResponse contains the created delivery details
type CreateDeliveryResponse struct {
	DeliveryID    string `json:"delivery_id"`
	ProviderName  string `json:"provider_name"`
	Status        string `json:"status"`
	TrackingURL   string `json:"tracking_url,omitempty"`
	DriverName    string `json:"driver_name,omitempty"`
	DriverPhone   string `json:"driver_phone,omitempty"`
	EstimatedTime string `json:"estimated_time,omitempty"`
}

// CancelDeliveryRequest contains delivery cancellation parameters
type CancelDeliveryRequest struct {
	DeliveryID string `json:"delivery_id"`
	Reason     string `json:"reason,omitempty"`
}

// CancelDeliveryResponse contains the cancellation result
type CancelDeliveryResponse struct {
	DeliveryID string `json:"delivery_id"`
	Status     string `json:"status"`
	Refunded   bool   `json:"refunded"`
}

// TrackDeliveryRequest contains delivery tracking parameters
type TrackDeliveryRequest struct {
	DeliveryID string `json:"delivery_id"`
}

// TrackDeliveryResponse contains the delivery tracking result
type TrackDeliveryResponse struct {
	DeliveryID  string  `json:"delivery_id"`
	Status      string  `json:"status"` // pending, assigned, picked_up, in_transit, delivered, cancelled
	DriverName  string  `json:"driver_name,omitempty"`
	DriverPhone string  `json:"driver_phone,omitempty"`
	DriverLat   float64 `json:"driver_lat,omitempty"`
	DriverLng   float64 `json:"driver_lng,omitempty"`
	UpdatedAt   string  `json:"updated_at"`
}

// FulfillmentWebhookRequest contains raw webhook data from the provider
type FulfillmentWebhookRequest struct {
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// FulfillmentWebhookResponse contains parsed webhook data
type FulfillmentWebhookResponse struct {
	EventType  string `json:"event_type"` // delivery.created, delivery.updated, delivery.completed, etc.
	DeliveryID string `json:"delivery_id"`
	Status     string `json:"status"`
}

// Address represents a physical address for pickup/delivery
type Address struct {
	Line1     string  `json:"line1"`
	Line2     string  `json:"line2,omitempty"`
	City      string  `json:"city"`
	State     string  `json:"state,omitempty"`
	PostCode  string  `json:"post_code,omitempty"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}
