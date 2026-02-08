//go:build paypal

package paypal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterPaymentProvider(
		"paypal",
		func() ports.PaymentProvider {
			return NewPayPalProvider()
		},
		transformConfig,
	)
	registry.RegisterPaymentBuildFromEnv("paypal", buildFromEnv)
}

// buildFromEnv creates and initializes a PayPal provider from environment variables.
func buildFromEnv() (ports.PaymentProvider, error) {
	clientID := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_PAYPAL_CLIENT_ID")
	clientSecret := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_PAYPAL_CLIENT_SECRET")
	sandboxMode := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_PAYPAL_SANDBOX") == "true"
	baseURL := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_PAYPAL_BASE_URL")

	if clientID == "" {
		return nil, fmt.Errorf("paypal: LEAPFOR_INTEGRATION_PAYMENT_PAYPAL_CLIENT_ID is required")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("paypal: LEAPFOR_INTEGRATION_PAYMENT_PAYPAL_CLIENT_SECRET is required")
	}

	protoConfig := &paymentpb.PaymentProviderConfig{
		ProviderId:   "paypal",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_GATEWAY,
		Enabled:      true,
		SandboxMode:  sandboxMode,
		RedirectUrls: &paymentpb.RedirectUrls{
			BaseUrl: baseURL,
		},
		Auth: &paymentpb.PaymentProviderConfig_Oauth2Auth{
			Oauth2Auth: &paymentpb.OAuth2Auth{
				ClientId:     clientID,
				ClientSecret: clientSecret,
			},
		},
	}

	p := NewPayPalProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("paypal: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to PayPal proto config.
func transformConfig(rawConfig map[string]any) (*paymentpb.PaymentProviderConfig, error) {
	protoConfig := &paymentpb.PaymentProviderConfig{
		ProviderId:   "paypal",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_GATEWAY,
		Enabled:      true,
	}

	oauth2Auth := &paymentpb.OAuth2Auth{}

	if clientID, ok := rawConfig["client_id"].(string); ok && clientID != "" {
		oauth2Auth.ClientId = clientID
	} else {
		return nil, fmt.Errorf("paypal: client_id is required")
	}
	if clientSecret, ok := rawConfig["client_secret"].(string); ok && clientSecret != "" {
		oauth2Auth.ClientSecret = clientSecret
	} else {
		return nil, fmt.Errorf("paypal: client_secret is required")
	}

	protoConfig.Auth = &paymentpb.PaymentProviderConfig_Oauth2Auth{Oauth2Auth: oauth2Auth}

	if sandboxMode, ok := rawConfig["sandbox_mode"].(bool); ok {
		protoConfig.SandboxMode = sandboxMode
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// PayPal API endpoints
const (
	paypalProductionURL = "https://api-m.paypal.com"
	paypalSandboxURL    = "https://api-m.sandbox.paypal.com"
	ordersPath          = "/v2/checkout/orders"
	tokenPath           = "/v1/oauth2/token"
)

// PayPalProvider implements the PaymentProvider interface for PayPal payment gateway
type PayPalProvider struct {
	enabled      bool
	clientID     string
	clientSecret string
	sandboxMode  bool
	apiEndpoint  string
	baseURL      string
	successPath  string
	failurePath  string
	cancelPath   string
	webhookPath  string
	timeout      time.Duration
	httpClient   *http.Client

	// OAuth2 token management
	accessToken    string
	tokenExpiresAt time.Time
	tokenMu        sync.RWMutex
}

// PayPalOrderRequest represents the order creation request
type PayPalOrderRequest struct {
	Intent             string                    `json:"intent"`
	PurchaseUnits      []PayPalPurchaseUnit      `json:"purchase_units"`
	PaymentSource      *PayPalPaymentSource      `json:"payment_source,omitempty"`
	ApplicationContext *PayPalApplicationContext `json:"application_context,omitempty"`
}

// PayPalPurchaseUnit represents a purchase unit in the order
type PayPalPurchaseUnit struct {
	ReferenceID string       `json:"reference_id,omitempty"`
	Description string       `json:"description,omitempty"`
	CustomID    string       `json:"custom_id,omitempty"`
	InvoiceID   string       `json:"invoice_id,omitempty"`
	Amount      PayPalAmount `json:"amount"`
	Items       []PayPalItem `json:"items,omitempty"`
	Payee       *PayPalPayee `json:"payee,omitempty"`
}

// PayPalAmount represents amount in PayPal's format
type PayPalAmount struct {
	CurrencyCode string                 `json:"currency_code"`
	Value        string                 `json:"value"`
	Breakdown    *PayPalAmountBreakdown `json:"breakdown,omitempty"`
}

// PayPalAmountBreakdown represents the breakdown of an amount
type PayPalAmountBreakdown struct {
	ItemTotal *PayPalMoney `json:"item_total,omitempty"`
}

// PayPalMoney represents a simple money object
type PayPalMoney struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

// PayPalItem represents a line item
type PayPalItem struct {
	Name        string      `json:"name"`
	Quantity    string      `json:"quantity"`
	UnitAmount  PayPalMoney `json:"unit_amount"`
	Description string      `json:"description,omitempty"`
	SKU         string      `json:"sku,omitempty"`
	Category    string      `json:"category,omitempty"`
}

// PayPalPayee represents the payee (merchant)
type PayPalPayee struct {
	EmailAddress string `json:"email_address,omitempty"`
	MerchantID   string `json:"merchant_id,omitempty"`
}

// PayPalPaymentSource represents the payment source
type PayPalPaymentSource struct {
	PayPal *PayPalExperienceContext `json:"paypal,omitempty"`
}

// PayPalExperienceContext represents the PayPal experience context
type PayPalExperienceContext struct {
	BrandName          string `json:"brand_name,omitempty"`
	Locale             string `json:"locale,omitempty"`
	LandingPage        string `json:"landing_page,omitempty"`
	ShippingPreference string `json:"shipping_preference,omitempty"`
	UserAction         string `json:"user_action,omitempty"`
	ReturnURL          string `json:"return_url"`
	CancelURL          string `json:"cancel_url"`
}

// PayPalApplicationContext represents application context (deprecated but still used)
type PayPalApplicationContext struct {
	BrandName          string `json:"brand_name,omitempty"`
	Locale             string `json:"locale,omitempty"`
	LandingPage        string `json:"landing_page,omitempty"`
	ShippingPreference string `json:"shipping_preference,omitempty"`
	UserAction         string `json:"user_action,omitempty"`
	ReturnURL          string `json:"return_url"`
	CancelURL          string `json:"cancel_url"`
}

// PayPalOrderResponse represents the order creation response
type PayPalOrderResponse struct {
	ID            string                       `json:"id"`
	Status        string                       `json:"status"`
	Links         []PayPalLink                 `json:"links"`
	CreateTime    string                       `json:"create_time,omitempty"`
	UpdateTime    string                       `json:"update_time,omitempty"`
	PurchaseUnits []PayPalPurchaseUnitResponse `json:"purchase_units,omitempty"`
}

// PayPalPurchaseUnitResponse represents a purchase unit in the response
type PayPalPurchaseUnitResponse struct {
	ReferenceID string          `json:"reference_id,omitempty"`
	Payments    *PayPalPayments `json:"payments,omitempty"`
}

// PayPalPayments represents payments in a purchase unit
type PayPalPayments struct {
	Captures       []PayPalCapture       `json:"captures,omitempty"`
	Authorizations []PayPalAuthorization `json:"authorizations,omitempty"`
}

// PayPalCapture represents a capture
type PayPalCapture struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Amount PayPalMoney `json:"amount"`
}

// PayPalAuthorization represents an authorization
type PayPalAuthorization struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Amount PayPalMoney `json:"amount"`
}

// PayPalLink represents a HATEOAS link
type PayPalLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method,omitempty"`
}

// PayPalWebhookEvent represents a webhook event from PayPal
type PayPalWebhookEvent struct {
	ID           string          `json:"id"`
	EventVersion string          `json:"event_version"`
	CreateTime   string          `json:"create_time"`
	ResourceType string          `json:"resource_type"`
	EventType    string          `json:"event_type"`
	Summary      string          `json:"summary"`
	Resource     json.RawMessage `json:"resource"`
	Links        []PayPalLink    `json:"links"`
}

// PayPalWebhookResource represents the resource in a webhook event
type PayPalWebhookResource struct {
	ID            string                       `json:"id"`
	Status        string                       `json:"status"`
	Amount        *PayPalMoney                 `json:"amount,omitempty"`
	CustomID      string                       `json:"custom_id,omitempty"`
	InvoiceID     string                       `json:"invoice_id,omitempty"`
	PurchaseUnits []PayPalPurchaseUnitResponse `json:"purchase_units,omitempty"`
}

// PayPalTokenResponse represents the OAuth token response
type PayPalTokenResponse struct {
	Scope       string `json:"scope"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	AppID       string `json:"app_id"`
	ExpiresIn   int    `json:"expires_in"`
	Nonce       string `json:"nonce"`
}

// PayPalErrorResponse represents an error from PayPal API
type PayPalErrorResponse struct {
	Name    string              `json:"name"`
	Message string              `json:"message"`
	DebugID string              `json:"debug_id"`
	Details []PayPalErrorDetail `json:"details,omitempty"`
	Links   []PayPalLink        `json:"links,omitempty"`
}

// PayPalErrorDetail represents an error detail
type PayPalErrorDetail struct {
	Issue       string `json:"issue"`
	Description string `json:"description"`
	Field       string `json:"field,omitempty"`
	Value       string `json:"value,omitempty"`
	Location    string `json:"location,omitempty"`
}

func NewPayPalProvider() ports.PaymentProvider {
	return &PayPalProvider{
		enabled:     false,
		timeout:     30 * time.Second,
		successPath: "/payment/success",
		failurePath: "/payment/fail",
		cancelPath:  "/payment/cancel",
		webhookPath: "/integration/payment/webhook",
	}
}

func (p *PayPalProvider) Name() string {
	return "paypal"
}

func (p *PayPalProvider) Initialize(config *paymentpb.PaymentProviderConfig) error {
	oauth2Auth := config.GetOauth2Auth()
	if oauth2Auth == nil {
		return fmt.Errorf("OAuth2 authentication configuration is required for PayPal")
	}

	if oauth2Auth.ClientId == "" {
		return fmt.Errorf("client_id is required for PayPal provider")
	}
	p.clientID = oauth2Auth.ClientId

	if oauth2Auth.ClientSecret == "" {
		return fmt.Errorf("client_secret is required for PayPal provider")
	}
	p.clientSecret = oauth2Auth.ClientSecret

	p.sandboxMode = config.SandboxMode
	if p.sandboxMode {
		p.apiEndpoint = paypalSandboxURL
	} else {
		p.apiEndpoint = paypalProductionURL
	}

	if config.ApiEndpoint != "" {
		p.apiEndpoint = config.ApiEndpoint
	} else if p.sandboxMode && config.SandboxEndpoint != "" {
		p.apiEndpoint = config.SandboxEndpoint
	}

	if redirects := config.RedirectUrls; redirects != nil {
		p.baseURL = redirects.BaseUrl
		if redirects.SuccessPath != "" {
			p.successPath = redirects.SuccessPath
		}
		if redirects.FailurePath != "" {
			p.failurePath = redirects.FailurePath
		}
		if redirects.CancelPath != "" {
			p.cancelPath = redirects.CancelPath
		}
		if redirects.WebhookPath != "" {
			p.webhookPath = redirects.WebhookPath
		}
	}

	if config.TimeoutSeconds > 0 {
		p.timeout = time.Duration(config.TimeoutSeconds) * time.Second
	}

	p.httpClient = &http.Client{
		Timeout: p.timeout,
	}

	p.enabled = config.Enabled
	log.Printf("✅ PayPal payment provider initialized (Sandbox: %v)", p.sandboxMode)
	return nil
}

func (p *PayPalProvider) CreateCheckoutSession(ctx context.Context, req *paymentpb.CreateCheckoutSessionRequest) (*paymentpb.CreateCheckoutSessionResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("PayPal provider is not initialized")
	}

	data := req.Data
	if data == nil {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "INVALID_REQUEST",
				Description: "Request data is required",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	if data.Amount <= 0 {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "INVALID_AMOUNT",
				Description: "Amount must be greater than zero",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	merchantRef := data.OrderRef
	if merchantRef == "" {
		merchantRef = data.PaymentId
	}
	if merchantRef == "" {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "MISSING_REFERENCE",
				Description: "OrderRef or PaymentId is required",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	currency := data.Currency
	if currency == "" {
		currency = "USD"
	}

	// Build redirect URLs
	// If SuccessUrl is provided but doesn't include the path, append it
	successURL := data.SuccessUrl
	if successURL == "" && p.baseURL != "" {
		successURL = p.baseURL + p.successPath
	} else if successURL != "" && !strings.Contains(successURL, p.successPath) && p.successPath != "" {
		// URL provided but missing the success path - append it
		successURL = strings.TrimSuffix(successURL, "/") + p.successPath
	}
	cancelURL := data.CancelUrl
	if cancelURL == "" && p.baseURL != "" {
		cancelURL = p.baseURL + p.cancelPath
	} else if cancelURL != "" && !strings.Contains(cancelURL, p.cancelPath) && p.cancelPath != "" {
		// URL provided but missing the cancel path - append it
		cancelURL = strings.TrimSuffix(cancelURL, "/") + p.cancelPath
	}

	// PayPal requires return URL, add order reference as query param
	if successURL != "" {
		if strings.Contains(successURL, "?") {
			successURL += "&ref=" + merchantRef
		} else {
			successURL += "?ref=" + merchantRef
		}
	}

	// Debug logging for return URLs
	log.Printf("[PayPal] 🔗 Return URL Config: baseURL=%q, successPath=%q, cancelPath=%q", p.baseURL, p.successPath, p.cancelPath)
	log.Printf("[PayPal] 🔗 Built URLs: successURL=%q, cancelURL=%q", successURL, cancelURL)

	// Format amount as string with 2 decimal places
	amountStr := fmt.Sprintf("%.2f", data.Amount)

	// Create order request
	orderReq := PayPalOrderRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []PayPalPurchaseUnit{
			{
				ReferenceID: merchantRef,
				Description: data.Description,
				CustomID:    data.PaymentId,
				InvoiceID:   data.SubscriptionId,
				Amount: PayPalAmount{
					CurrencyCode: currency,
					Value:        amountStr,
				},
			},
		},
		PaymentSource: &PayPalPaymentSource{
			PayPal: &PayPalExperienceContext{
				ReturnURL:          successURL,
				CancelURL:          cancelURL,
				UserAction:         "PAY_NOW",
				ShippingPreference: "NO_SHIPPING",
				LandingPage:        "LOGIN",
			},
		},
	}

	// Add item details if description is provided
	if data.Description != "" {
		orderReq.PurchaseUnits[0].Items = []PayPalItem{
			{
				Name:     data.Description,
				Quantity: "1",
				UnitAmount: PayPalMoney{
					CurrencyCode: currency,
					Value:        amountStr,
				},
				Description: data.Description,
				Category:    "DIGITAL_GOODS",
			},
		}
		orderReq.PurchaseUnits[0].Amount.Breakdown = &PayPalAmountBreakdown{
			ItemTotal: &PayPalMoney{
				CurrencyCode: currency,
				Value:        amountStr,
			},
		}
	}

	// Make API request
	orderResp, err := p.createOrder(ctx, &orderReq)
	if err != nil {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "PAYPAL_API_ERROR",
				Description: fmt.Sprintf("Failed to create PayPal order: %v", err),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
			},
		}, nil
	}

	// Find the approval URL from links
	var checkoutURL string
	for _, link := range orderResp.Links {
		if link.Rel == "payer-action" || link.Rel == "approve" {
			checkoutURL = link.Href
			break
		}
	}

	if checkoutURL == "" {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "NO_CHECKOUT_URL",
				Description: "PayPal did not return an approval URL",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
			},
		}, nil
	}

	now := timestamppb.Now()
	expiresAt := timestamppb.New(time.Now().Add(3 * time.Hour)) // PayPal orders expire in 3 hours by default
	if data.ExpiresInMinutes > 0 {
		expiresAt = timestamppb.New(time.Now().Add(time.Duration(data.ExpiresInMinutes) * time.Minute))
	}

	session := &paymentpb.CheckoutSession{
		Id:                fmt.Sprintf("paypal_%s", merchantRef),
		ProviderSessionId: orderResp.ID,
		ProviderId:        "paypal",
		ProviderType:      paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_GATEWAY,
		Amount:            data.Amount,
		Currency:          currency,
		Status:            paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING,
		CheckoutUrl:       checkoutURL,
		SuccessUrl:        successURL,
		FailureUrl:        data.FailureUrl,
		CancelUrl:         cancelURL,
		PaymentId:         data.PaymentId,
		SubscriptionId:    data.SubscriptionId,
		ClientId:          data.ClientId,
		OrderRef:          merchantRef,
		Description:       data.Description,
		Metadata:          data.Metadata,
		ExpiresAt:         expiresAt,
		CreatedAt:         now,
		UpdatedAt:         now,
		ProviderData: map[string]string{
			"order_id":     orderResp.ID,
			"order_status": orderResp.Status,
			"sandbox_mode": fmt.Sprintf("%v", p.sandboxMode),
		},
	}

	log.Printf("📦 PayPal order created: %s (order_id: %s, status: %s)", session.Id, orderResp.ID, orderResp.Status)
	return &paymentpb.CreateCheckoutSessionResponse{Success: true, Data: []*paymentpb.CheckoutSession{session}}, nil
}

// getAccessToken obtains or refreshes the OAuth2 access token
func (p *PayPalProvider) getAccessToken(ctx context.Context) (string, error) {
	p.tokenMu.RLock()
	if p.accessToken != "" && time.Now().Before(p.tokenExpiresAt.Add(-30*time.Second)) {
		token := p.accessToken
		p.tokenMu.RUnlock()
		return token, nil
	}
	p.tokenMu.RUnlock()

	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if p.accessToken != "" && time.Now().Before(p.tokenExpiresAt.Add(-30*time.Second)) {
		return p.accessToken, nil
	}

	// Request new token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiEndpoint+tokenPath, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	// Basic Auth with client ID and secret
	auth := base64.StdEncoding.EncodeToString([]byte(p.clientID + ":" + p.clientSecret))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp PayPalErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return "", fmt.Errorf("PayPal token error [%s]: %s", errResp.Name, errResp.Message)
		}
		return "", fmt.Errorf("PayPal token error status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp PayPalTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	p.accessToken = tokenResp.AccessToken
	p.tokenExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	log.Printf("🔑 PayPal access token obtained (expires in %d seconds)", tokenResp.ExpiresIn)
	return p.accessToken, nil
}

func (p *PayPalProvider) createOrder(ctx context.Context, req *PayPalOrderRequest) (*PayPalOrderResponse, error) {
	token, err := p.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiEndpoint+ordersPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Prefer", "return=representation")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp PayPalErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			details := ""
			if len(errResp.Details) > 0 {
				details = fmt.Sprintf(" - %s: %s", errResp.Details[0].Issue, errResp.Details[0].Description)
			}
			return nil, fmt.Errorf("PayPal API error [%s]: %s%s", errResp.Name, errResp.Message, details)
		}
		return nil, fmt.Errorf("PayPal API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var orderResp PayPalOrderResponse
	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order response: %w", err)
	}

	return &orderResp, nil
}

func (p *PayPalProvider) ProcessWebhook(ctx context.Context, req *paymentpb.ProcessWebhookRequest) (*paymentpb.ProcessWebhookResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("PayPal provider is not initialized")
	}

	data := req.Data
	if data == nil {
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "INVALID_REQUEST",
				Description: "Request data is required",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	var webhookEvent PayPalWebhookEvent
	if err := json.Unmarshal(data.Payload, &webhookEvent); err != nil {
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "WEBHOOK_PARSE_ERROR",
				Description: fmt.Sprintf("Failed to parse webhook payload: %v", err),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	// Parse the resource
	var resource PayPalWebhookResource
	if err := json.Unmarshal(webhookEvent.Resource, &resource); err != nil {
		log.Printf("⚠️ PayPal webhook: Failed to parse resource: %v", err)
	}

	// Map PayPal event types to our status
	var status paymentpb.PaymentStatus
	var action string

	switch webhookEvent.EventType {
	case "CHECKOUT.ORDER.APPROVED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
		action = "authorized"
	case "PAYMENT.CAPTURE.COMPLETED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS
		action = "success"
	case "PAYMENT.CAPTURE.DENIED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED
		action = "failure"
	case "PAYMENT.CAPTURE.REFUNDED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_REFUNDED
		action = "refunded"
	case "CHECKOUT.ORDER.COMPLETED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS
		action = "success"
	case "PAYMENT.CAPTURE.PENDING":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_PROCESSING
		action = "processing"
	default:
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_PROCESSING
		action = "processing"
	}

	// Extract payment ID from custom_id or invoice_id
	paymentID := resource.CustomID
	if paymentID == "" && len(resource.PurchaseUnits) > 0 {
		paymentID = resource.PurchaseUnits[0].ReferenceID
	}

	// Extract order reference from purchase units
	orderRef := ""
	if len(resource.PurchaseUnits) > 0 {
		orderRef = resource.PurchaseUnits[0].ReferenceID
	}
	if orderRef == "" {
		orderRef = resource.ID // fallback to resource ID
	}

	var amount int64
	if resource.Amount != nil {
		// Parse amount string to float then convert to cents
		var amountFloat float64
		fmt.Sscanf(resource.Amount.Value, "%f", &amountFloat)
		amount = int64(amountFloat * 100)
	}

	currency := "USD"
	if resource.Amount != nil && resource.Amount.CurrencyCode != "" {
		currency = resource.Amount.CurrencyCode
	}

	transaction := &paymentpb.PaymentTransaction{
		Id:                 webhookEvent.ID,
		ProviderRef:        resource.ID,
		ProviderPaymentRef: resource.ID,
		ProviderId:         "paypal",
		Status:             status,
		Amount:             amount,
		Currency:           currency,
		PaymentMethod:      "paypal",
		PaymentId:          paymentID,
		OrderRef:           orderRef,
		ProcessedAt:        timestamppb.Now(),
		RawData: map[string]string{
			"event_id":     webhookEvent.ID,
			"event_type":   webhookEvent.EventType,
			"resource_id":  resource.ID,
			"order_status": resource.Status,
		},
	}

	log.Printf("📨 PayPal webhook processed: %s -> %s (event: %s)", webhookEvent.ID, action, webhookEvent.EventType)
	return &paymentpb.ProcessWebhookResponse{
		Success: true,
		Data: []*paymentpb.WebhookResult{{
			Transaction: transaction,
			Status:      status,
			Action:      action,
			PaymentId:   paymentID,
		}},
	}, nil
}

func (p *PayPalProvider) GetPaymentStatus(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("PayPal provider is not initialized")
	}

	data := req.Data
	if data == nil || data.ProviderRef == "" {
		return &paymentpb.GetPaymentStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "INVALID_REQUEST",
				Description: "Provider reference (order ID) is required",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	// Get order details
	token, err := p.getAccessToken(ctx)
	if err != nil {
		return &paymentpb.GetPaymentStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "AUTH_ERROR",
				Description: fmt.Sprintf("Failed to authenticate: %v", err),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
			},
		}, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.apiEndpoint+ordersPath+"/"+data.ProviderRef, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return &paymentpb.GetPaymentStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "HTTP_ERROR",
				Description: fmt.Sprintf("Request failed: %v", err),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
			},
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &paymentpb.GetPaymentStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "API_ERROR",
				Description: fmt.Sprintf("PayPal returned status %d", resp.StatusCode),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
			},
		}, nil
	}

	var order PayPalOrderResponse
	if err := json.Unmarshal(respBody, &order); err != nil {
		return nil, err
	}

	// Map status
	var status paymentpb.PaymentStatus
	switch order.Status {
	case "COMPLETED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS
	case "APPROVED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
	case "CREATED", "SAVED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING
	case "VOIDED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_CANCELLED
	default:
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_PROCESSING
	}

	return &paymentpb.GetPaymentStatusResponse{
		Success: true,
		Data: []*paymentpb.PaymentStatusData{
			{
				Status: status,
				Transaction: &paymentpb.PaymentTransaction{
					ProviderRef: order.ID,
					ProviderId:  "paypal",
					Status:      status,
				},
			},
		},
	}, nil
}

func (p *PayPalProvider) RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("PayPal provider is not initialized")
	}

	// PayPal refunds require the capture ID
	// POST /v2/payments/captures/{capture_id}/refund
	return &paymentpb.RefundPaymentResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:        "NOT_IMPLEMENTED",
			Description: "Refund API not yet implemented - use PayPal dashboard",
			Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
		},
	}, nil
}

func (p *PayPalProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("PayPal provider is not initialized")
	}
	if p.clientID == "" || p.clientSecret == "" {
		return fmt.Errorf("PayPal provider is not properly configured")
	}
	// Test token endpoint
	_, err := p.getAccessToken(ctx)
	return err
}

func (p *PayPalProvider) Close() error {
	p.enabled = false
	p.tokenMu.Lock()
	p.accessToken = ""
	p.tokenExpiresAt = time.Time{}
	p.tokenMu.Unlock()
	log.Printf("🔌 PayPal provider closed")
	return nil
}

func (p *PayPalProvider) IsEnabled() bool {
	return p.enabled
}

func (p *PayPalProvider) GetCapabilities() []paymentpb.PaymentCapability {
	return []paymentpb.PaymentCapability{
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_ONE_TIME,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_WEBHOOKS,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_REFUND,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_VOID,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_RECURRING,
	}
}

func (p *PayPalProvider) GetSupportedCurrencies() []string {
	return []string{"USD", "EUR", "GBP", "CAD", "AUD", "PHP", "SGD", "HKD", "JPY"}
}

var _ ports.PaymentProvider = (*PayPalProvider)(nil)
