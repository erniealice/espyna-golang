package handlers

import (
	"context"
	"fmt"
	"time"
)

// IntegrationHandlers contains handlers for integration-related routes
type IntegrationHandlers struct {
	WebhookHandler     *WebhookHandler
	ExternalAPIHandler *ExternalAPIHandler
	CallbackHandler    *CallbackHandler
}

// NewIntegrationHandlers creates a new instance of integration handlers
func NewIntegrationHandlers(container interface{}) *IntegrationHandlers {
	return &IntegrationHandlers{
		WebhookHandler:     NewWebhookHandler(container),
		ExternalAPIHandler: NewExternalAPIHandler(container),
		CallbackHandler:    NewCallbackHandler(container),
	}
}

// WebhookHandler handles webhook endpoints
type WebhookHandler struct {
	container  interface{}
	webhooks   map[string]*WebhookConfig
	validators map[string]WebhookValidator
	processors map[string]WebhookProcessor
}

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	Name        string            `json:"name"`
	Endpoint    string            `json:"endpoint"`
	Secret      string            `json:"secret"`
	Events      []string          `json:"events"`
	Enabled     bool              `json:"enabled"`
	RateLimit   int               `json:"rateLimit"`
	Headers     map[string]string `json:"headers"`
	RetryPolicy *RetryPolicy      `json:"retryPolicy"`
}

// RetryPolicy represents retry configuration
type RetryPolicy struct {
	MaxRetries int           `json:"maxRetries"`
	Delay      time.Duration `json:"delay"`
	Backoff    BackoffType   `json:"backoff"`
}

type BackoffType string

const (
	BackoffLinear      BackoffType = "linear"
	BackoffExponential BackoffType = "exponential"
	BackoffFixed       BackoffType = "fixed"
)

// WebhookValidator represents a webhook signature validator
type WebhookValidator func(payload []byte, signature string, secret string) error

// WebhookProcessor represents a webhook event processor
type WebhookProcessor func(ctx context.Context, event *WebhookEvent) error

// WebhookEvent represents an incoming webhook event
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Headers   map[string]string      `json:"headers"`
	Signature string                 `json:"signature"`
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(container interface{}) *WebhookHandler {
	return &WebhookHandler{
		container:  container,
		webhooks:   make(map[string]*WebhookConfig),
		validators: make(map[string]WebhookValidator),
		processors: make(map[string]WebhookProcessor),
	}
}

// RegisterWebhook registers a new webhook
func (h *WebhookHandler) RegisterWebhook(config *WebhookConfig) error {
	if config.Name == "" {
		return fmt.Errorf("webhook name cannot be empty")
	}

	if config.Endpoint == "" {
		return fmt.Errorf("webhook endpoint cannot be empty")
	}

	h.webhooks[config.Name] = config
	return nil
}

// RegisterValidator registers a webhook validator
func (h *WebhookHandler) RegisterValidator(source string, validator WebhookValidator) {
	h.validators[source] = validator
}

// RegisterProcessor registers a webhook processor
func (h *WebhookHandler) RegisterProcessor(eventType string, processor WebhookProcessor) {
	h.processors[eventType] = processor
}

// ProcessWebhook processes an incoming webhook
func (h *WebhookHandler) ProcessWebhook(ctx context.Context, source string, event *WebhookEvent) (*WebhookResponse, error) {
	// Find webhook configuration
	webhookConfig, exists := h.webhooks[source]
	if !exists {
		return &WebhookResponse{
			Success: false,
			Message: "Webhook not found",
		}, fmt.Errorf("webhook not found: %s", source)
	}

	if !webhookConfig.Enabled {
		return &WebhookResponse{
			Success: false,
			Message: "Webhook is disabled",
		}, fmt.Errorf("webhook is disabled: %s", source)
	}

	// Validate webhook signature
	if validator, exists := h.validators[source]; exists {
		if err := validator([]byte(fmt.Sprintf("%v", event.Data)), event.Signature, webhookConfig.Secret); err != nil {
			return &WebhookResponse{
				Success: false,
				Message: "Invalid webhook signature",
			}, fmt.Errorf("invalid webhook signature: %w", err)
		}
	}

	// Process the event
	if processor, exists := h.processors[event.Type]; exists {
		if err := processor(ctx, event); err != nil {
			return &WebhookResponse{
				Success: false,
				Message: "Failed to process webhook event",
			}, fmt.Errorf("failed to process webhook event: %w", err)
		}
	}

	return &WebhookResponse{
		Success: true,
		Message: "Webhook processed successfully",
		EventID: event.ID,
	}, nil
}

// ListWebhooks returns all registered webhooks
func (h *WebhookHandler) ListWebhooks(ctx context.Context) []*WebhookConfig {
	var webhooks []*WebhookConfig
	for _, config := range h.webhooks {
		webhooks = append(webhooks, config)
	}
	return webhooks
}

// GetWebhook returns a specific webhook configuration
func (h *WebhookHandler) GetWebhook(ctx context.Context, name string) (*WebhookConfig, error) {
	config, exists := h.webhooks[name]
	if !exists {
		return nil, fmt.Errorf("webhook not found: %s", name)
	}
	return config, nil
}

// WebhookResponse represents webhook processing response
type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	EventID string `json:"eventId,omitempty"`
}

// ExternalAPIHandler handles external API integration endpoints
type ExternalAPIHandler struct {
	container   interface{}
	apis        map[string]*ExternalAPIConfig
	connections map[string]ExternalAPIConnection
}

// ExternalAPIConfig represents external API configuration
type ExternalAPIConfig struct {
	Name        string            `json:"name"`
	BaseURL     string            `json:"baseUrl"`
	AuthType    AuthType          `json:"authType"`
	AuthConfig  map[string]string `json:"authConfig"`
	Headers     map[string]string `json:"headers"`
	Timeout     time.Duration     `json:"timeout"`
	RateLimit   int               `json:"rateLimit"`
	RetryPolicy *RetryPolicy      `json:"retryPolicy"`
	Enabled     bool              `json:"enabled"`
}

type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeAPIKey AuthType = "apikey"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeBasic  AuthType = "basic"
	AuthTypeOAuth2 AuthType = "oauth2"
)

// ExternalAPIConnection represents a connection to an external API
type ExternalAPIConnection interface {
	Request(ctx context.Context, method, endpoint string, data interface{}) (*ExternalAPIResponse, error)
	IsHealthy(ctx context.Context) bool
}

// ExternalAPIResponse represents response from external API
type ExternalAPIResponse struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Data      interface{}       `json:"data"`
	RawBody   []byte            `json:"rawBody"`
	Duration  time.Duration     `json:"duration"`
	RequestID string            `json:"requestId"`
	Timestamp time.Time         `json:"timestamp"`
}

// NewExternalAPIHandler creates a new external API handler
func NewExternalAPIHandler(container interface{}) *ExternalAPIHandler {
	return &ExternalAPIHandler{
		container:   container,
		apis:        make(map[string]*ExternalAPIConfig),
		connections: make(map[string]ExternalAPIConnection),
	}
}

// RegisterAPI registers a new external API
func (h *ExternalAPIHandler) RegisterAPI(config *ExternalAPIConfig) error {
	if config.Name == "" {
		return fmt.Errorf("API name cannot be empty")
	}

	if config.BaseURL == "" {
		return fmt.Errorf("API base URL cannot be empty")
	}

	h.apis[config.Name] = config
	return nil
}

// RegisterConnection registers an external API connection
func (h *ExternalAPIHandler) RegisterConnection(name string, connection ExternalAPIConnection) {
	h.connections[name] = connection
}

// CallAPI makes a request to an external API
func (h *ExternalAPIHandler) CallAPI(ctx context.Context, apiName, method, endpoint string, data interface{}) (*ExternalAPIResponse, error) {
	connection, exists := h.connections[apiName]
	if !exists {
		return nil, fmt.Errorf("API connection not found: %s", apiName)
	}

	return connection.Request(ctx, method, endpoint, data)
}

// CheckAPIHealth checks the health of an external API
func (h *ExternalAPIHandler) CheckAPIHealth(ctx context.Context, apiName string) (bool, error) {
	connection, exists := h.connections[apiName]
	if !exists {
		return false, fmt.Errorf("API connection not found: %s", apiName)
	}

	return connection.IsHealthy(ctx), nil
}

// ListAPIs returns all registered external APIs
func (h *ExternalAPIHandler) ListAPIs(ctx context.Context) []*ExternalAPIConfig {
	var apis []*ExternalAPIConfig
	for _, config := range h.apis {
		apis = append(apis, config)
	}
	return apis
}

// CallbackHandler handles callback endpoints
type CallbackHandler struct {
	container  interface{}
	callbacks  map[string]*CallbackConfig
	processors map[string]CallbackProcessor
}

// CallbackConfig represents callback configuration
type CallbackConfig struct {
	Name       string            `json:"name"`
	Endpoint   string            `json:"endpoint"`
	Timeout    time.Duration     `json:"timeout"`
	MaxRetries int               `json:"maxRetries"`
	Headers    map[string]string `json:"headers"`
	Enabled    bool              `json:"enabled"`
}

// CallbackProcessor represents a callback processor
type CallbackProcessor func(ctx context.Context, callback *CallbackRequest) (*CallbackResponse, error)

// CallbackRequest represents an incoming callback
type CallbackRequest struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Headers   map[string]string      `json:"headers"`
}

// CallbackResponse represents callback processing response
type CallbackResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	RequestID string                 `json:"requestId"`
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(container interface{}) *CallbackHandler {
	return &CallbackHandler{
		container:  container,
		callbacks:  make(map[string]*CallbackConfig),
		processors: make(map[string]CallbackProcessor),
	}
}

// RegisterCallback registers a new callback
func (h *CallbackHandler) RegisterCallback(config *CallbackConfig) error {
	if config.Name == "" {
		return fmt.Errorf("callback name cannot be empty")
	}

	if config.Endpoint == "" {
		return fmt.Errorf("callback endpoint cannot be empty")
	}

	h.callbacks[config.Name] = config
	return nil
}

// RegisterProcessor registers a callback processor
func (h *CallbackHandler) RegisterProcessor(callbackType string, processor CallbackProcessor) {
	h.processors[callbackType] = processor
}

// ProcessCallback processes an incoming callback
func (h *CallbackHandler) ProcessCallback(ctx context.Context, callbackType string, request *CallbackRequest) (*CallbackResponse, error) {
	processor, exists := h.processors[callbackType]
	if !exists {
		return &CallbackResponse{
			Success: false,
			Message: "Callback processor not found",
		}, fmt.Errorf("callback processor not found: %s", callbackType)
	}

	return processor(ctx, request)
}

// ListCallbacks returns all registered callbacks
func (h *CallbackHandler) ListCallbacks(ctx context.Context) []*CallbackConfig {
	var callbacks []*CallbackConfig
	for _, config := range h.callbacks {
		callbacks = append(callbacks, config)
	}
	return callbacks
}
