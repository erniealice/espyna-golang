//go:build maya

package maya

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
		"maya",
		func() ports.PaymentProvider {
			return NewMayaProvider()
		},
		transformConfig,
	)
	registry.RegisterPaymentBuildFromEnv("maya", buildFromEnv)
}

// buildFromEnv creates and initializes a Maya provider from environment variables.
func buildFromEnv() (ports.PaymentProvider, error) {
	publicKey := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_MAYA_PUBLIC_KEY")
	secretKey := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_MAYA_SECRET_KEY")
	sandboxMode := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_MAYA_SANDBOX") == "true"
	baseURL := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_MAYA_BASE_URL")

	if publicKey == "" {
		return nil, fmt.Errorf("maya: LEAPFOR_INTEGRATION_PAYMENT_MAYA_PUBLIC_KEY is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("maya: LEAPFOR_INTEGRATION_PAYMENT_MAYA_SECRET_KEY is required")
	}

	protoConfig := &paymentpb.PaymentProviderConfig{
		ProviderId:   "maya",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_WALLET,
		Enabled:      true,
		SandboxMode:  sandboxMode,
		RedirectUrls: &paymentpb.RedirectUrls{
			BaseUrl: baseURL,
		},
		Auth: &paymentpb.PaymentProviderConfig_ApiKeyAuth{
			ApiKeyAuth: &paymentpb.ApiKeyAuth{
				ApiKey:    publicKey,
				ApiSecret: secretKey,
			},
		},
	}

	p := NewMayaProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("maya: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to Maya proto config.
func transformConfig(rawConfig map[string]any) (*paymentpb.PaymentProviderConfig, error) {
	protoConfig := &paymentpb.PaymentProviderConfig{
		ProviderId:   "maya",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_WALLET,
		Enabled:      true,
	}

	apiKeyAuth := &paymentpb.ApiKeyAuth{}

	if publicKey, ok := rawConfig["public_key"].(string); ok && publicKey != "" {
		apiKeyAuth.ApiKey = publicKey
	} else {
		return nil, fmt.Errorf("maya: public_key is required")
	}
	if secretKey, ok := rawConfig["secret_key"].(string); ok && secretKey != "" {
		apiKeyAuth.ApiSecret = secretKey
	} else {
		return nil, fmt.Errorf("maya: secret_key is required")
	}

	protoConfig.Auth = &paymentpb.PaymentProviderConfig_ApiKeyAuth{ApiKeyAuth: apiKeyAuth}

	if sandboxMode, ok := rawConfig["sandbox_mode"].(bool); ok {
		protoConfig.SandboxMode = sandboxMode
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// Maya API endpoints
const (
	mayaProductionURL = "https://pg.maya.ph"
	mayaSandboxURL    = "https://pg-sandbox.paymaya.com"
	checkoutPath      = "/checkout/v1/checkouts"
)

// MayaProvider implements the PaymentProvider interface for Maya payment gateway
type MayaProvider struct {
	enabled     bool
	publicKey   string
	secretKey   string
	sandboxMode bool
	apiEndpoint string
	baseURL     string
	successPath string
	failurePath string
	cancelPath  string
	webhookPath string
	timeout     time.Duration
	httpClient  *http.Client
}

// MayaCheckoutRequest represents the checkout creation request
type MayaCheckoutRequest struct {
	TotalAmount        MayaAmount       `json:"totalAmount"`
	Buyer              *MayaBuyer       `json:"buyer,omitempty"`
	Items              []MayaItem       `json:"items,omitempty"`
	RedirectUrl        MayaRedirectUrls `json:"redirectUrl"`
	RequestReferenceNo string           `json:"requestReferenceNumber"`
	Metadata           map[string]any   `json:"metadata,omitempty"`
}

// MayaAmount represents amount in Maya's format
type MayaAmount struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
}

// MayaBuyer represents buyer information
type MayaBuyer struct {
	FirstName    string       `json:"firstName,omitempty"`
	MiddleName   string       `json:"middleName,omitempty"`
	LastName     string       `json:"lastName,omitempty"`
	Contact      *MayaContact `json:"contact,omitempty"`
	BillingAddr  *MayaAddress `json:"billingAddress,omitempty"`
	ShippingAddr *MayaAddress `json:"shippingAddress,omitempty"`
}

// MayaContact represents contact information
type MayaContact struct {
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

// MayaAddress represents address information
type MayaAddress struct {
	Line1       string `json:"line1,omitempty"`
	Line2       string `json:"line2,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	ZipCode     string `json:"zipCode,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// MayaItem represents a line item
type MayaItem struct {
	Name        string     `json:"name"`
	Quantity    int        `json:"quantity"`
	Code        string     `json:"code,omitempty"`
	Description string     `json:"description,omitempty"`
	Amount      MayaAmount `json:"amount,omitempty"`
	TotalAmount MayaAmount `json:"totalAmount"`
}

// MayaRedirectUrls represents redirect URLs
type MayaRedirectUrls struct {
	Success string `json:"success"`
	Failure string `json:"failure"`
	Cancel  string `json:"cancel"`
}

// MayaCheckoutResponse represents the checkout creation response
type MayaCheckoutResponse struct {
	CheckoutId  string `json:"checkoutId"`
	RedirectUrl string `json:"redirectUrl"`
}

// MayaWebhookPayload represents the webhook payload from Maya
type MayaWebhookPayload struct {
	ID                     string          `json:"id"`
	IsPaid                 bool            `json:"isPaid"`
	Status                 string          `json:"status"`
	Amount                 float64         `json:"amount"`
	Currency               string          `json:"currency"`
	CanVoid                bool            `json:"canVoid"`
	CanRefund              bool            `json:"canRefund"`
	CanCapture             bool            `json:"canCapture"`
	CreatedAt              string          `json:"createdAt"`
	UpdatedAt              string          `json:"updatedAt"`
	Description            string          `json:"description"`
	PaymentStatus          string          `json:"paymentStatus"`
	RequestReferenceNumber string          `json:"requestReferenceNumber"`
	ReceiptNumber          string          `json:"receiptNumber,omitempty"`
	Metadata               map[string]any  `json:"metadata,omitempty"`
	FundSource             *MayaFundSource `json:"fundSource,omitempty"`
}

// MayaFundSource represents the payment fund source details
type MayaFundSource struct {
	ID          string           `json:"id,omitempty"`
	Type        string           `json:"type"`
	Description string           `json:"description,omitempty"`
	Details     *MayaCardDetails `json:"details,omitempty"`
}

// MayaCardDetails represents card details (masked)
type MayaCardDetails struct {
	Scheme       string   `json:"scheme,omitempty"`
	Last4        string   `json:"last4,omitempty"`
	First6       string   `json:"first6,omitempty"`
	Masked       string   `json:"masked,omitempty"`
	Issuer       string   `json:"issuer,omitempty"`
	CardType     string   `json:"cardType,omitempty"`
	ThreeDSecure *Maya3DS `json:"threeDSecure,omitempty"`
}

// Maya3DS represents 3D Secure authentication result
type Maya3DS struct {
	ID            string `json:"id,omitempty"`
	Status        string `json:"status,omitempty"`
	Eci           string `json:"eci,omitempty"`
	Cavv          string `json:"cavv,omitempty"`
	TransactionId string `json:"transactionId,omitempty"`
}

// MayaErrorResponse represents an error from Maya API
type MayaErrorResponse struct {
	Code       string           `json:"code"`
	Message    string           `json:"message"`
	Parameters []MayaErrorParam `json:"parameters,omitempty"`
}

// MayaErrorParam represents an error parameter
type MayaErrorParam struct {
	Field       string `json:"field"`
	Description string `json:"description"`
}

func NewMayaProvider() ports.PaymentProvider {
	return &MayaProvider{
		enabled:     false,
		timeout:     30 * time.Second,
		successPath: "/payment/success",
		failurePath: "/payment/fail",
		cancelPath:  "/payment/cancel",
		webhookPath: "/integration/payment/webhook",
	}
}

func (p *MayaProvider) Name() string {
	return "maya"
}

func (p *MayaProvider) Initialize(config *paymentpb.PaymentProviderConfig) error {
	apiKeyAuth := config.GetApiKeyAuth()
	if apiKeyAuth == nil {
		return fmt.Errorf("API key authentication configuration is required for Maya")
	}

	if apiKeyAuth.ApiKey == "" {
		return fmt.Errorf("public_key (api_key) is required for Maya provider")
	}
	p.publicKey = apiKeyAuth.ApiKey

	if apiKeyAuth.ApiSecret == "" {
		return fmt.Errorf("secret_key (api_secret) is required for Maya provider")
	}
	p.secretKey = apiKeyAuth.ApiSecret

	p.sandboxMode = config.SandboxMode
	if p.sandboxMode {
		p.apiEndpoint = mayaSandboxURL
	} else {
		p.apiEndpoint = mayaProductionURL
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
	log.Printf("✅ Maya payment provider initialized (Sandbox: %v)", p.sandboxMode)
	return nil
}

func (p *MayaProvider) CreateCheckoutSession(ctx context.Context, req *paymentpb.CreateCheckoutSessionRequest) (*paymentpb.CreateCheckoutSessionResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Maya provider is not initialized")
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
		currency = "PHP"
	}

	// Build redirect URLs
	successURL := data.SuccessUrl
	if successURL == "" && p.baseURL != "" {
		successURL = p.baseURL + p.successPath
	}
	failureURL := data.FailureUrl
	if failureURL == "" && p.baseURL != "" {
		failureURL = p.baseURL + p.failurePath
	}
	cancelURL := data.CancelUrl
	if cancelURL == "" && p.baseURL != "" {
		cancelURL = p.baseURL + p.cancelPath
	}

	// Create checkout request
	checkoutReq := MayaCheckoutRequest{
		TotalAmount: MayaAmount{
			Value:    float64(data.Amount) / 100.0, // Convert cents to major units
			Currency: currency,
		},
		RedirectUrl: MayaRedirectUrls{
			Success: successURL,
			Failure: failureURL,
			Cancel:  cancelURL,
		},
		RequestReferenceNo: merchantRef,
	}

	// Add description as item if provided
	if data.Description != "" {
		checkoutReq.Items = []MayaItem{
			{
				Name:        data.Description,
				Quantity:    1,
				TotalAmount: checkoutReq.TotalAmount,
			},
		}
	}

	// Add metadata
	if data.Metadata != nil {
		checkoutReq.Metadata = make(map[string]any)
		for k, v := range data.Metadata {
			checkoutReq.Metadata[k] = v
		}
	}
	// Always include payment_id and subscription_id in metadata for webhook correlation
	if checkoutReq.Metadata == nil {
		checkoutReq.Metadata = make(map[string]any)
	}
	if data.PaymentId != "" {
		checkoutReq.Metadata["payment_id"] = data.PaymentId
	}
	if data.SubscriptionId != "" {
		checkoutReq.Metadata["subscription_id"] = data.SubscriptionId
	}
	if data.ClientId != "" {
		checkoutReq.Metadata["client_id"] = data.ClientId
	}

	// Make API request
	checkoutResp, err := p.createCheckout(ctx, &checkoutReq)
	if err != nil {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "MAYA_API_ERROR",
				Description: fmt.Sprintf("Failed to create Maya checkout: %v", err),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
			},
		}, nil
	}

	now := timestamppb.Now()
	expiresAt := timestamppb.New(time.Now().Add(30 * time.Minute))
	if data.ExpiresInMinutes > 0 {
		expiresAt = timestamppb.New(time.Now().Add(time.Duration(data.ExpiresInMinutes) * time.Minute))
	}

	session := &paymentpb.CheckoutSession{
		Id:                fmt.Sprintf("maya_%s", merchantRef),
		ProviderSessionId: checkoutResp.CheckoutId,
		ProviderId:        "maya",
		ProviderType:      paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_WALLET,
		Amount:            data.Amount,
		Currency:          currency,
		Status:            paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING,
		CheckoutUrl:       checkoutResp.RedirectUrl,
		SuccessUrl:        successURL,
		FailureUrl:        failureURL,
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
			"checkout_id":  checkoutResp.CheckoutId,
			"sandbox_mode": fmt.Sprintf("%v", p.sandboxMode),
		},
	}

	log.Printf("📦 Maya checkout session created: %s (checkout_id: %s)", session.Id, checkoutResp.CheckoutId)
	return &paymentpb.CreateCheckoutSessionResponse{Success: true, Data: []*paymentpb.CheckoutSession{session}}, nil
}

func (p *MayaProvider) createCheckout(ctx context.Context, req *MayaCheckoutRequest) (*MayaCheckoutResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal checkout request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiEndpoint+checkoutPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Maya uses Basic Auth with public key for checkout creation
	auth := base64.StdEncoding.EncodeToString([]byte(p.publicKey + ":"))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp MayaErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, fmt.Errorf("Maya API error [%s]: %s", errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("Maya API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var checkoutResp MayaCheckoutResponse
	if err := json.Unmarshal(respBody, &checkoutResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkout response: %w", err)
	}

	return &checkoutResp, nil
}

func (p *MayaProvider) ProcessWebhook(ctx context.Context, req *paymentpb.ProcessWebhookRequest) (*paymentpb.ProcessWebhookResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Maya provider is not initialized")
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

	var webhook MayaWebhookPayload
	if err := json.Unmarshal(data.Payload, &webhook); err != nil {
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "WEBHOOK_PARSE_ERROR",
				Description: fmt.Sprintf("Failed to parse webhook payload: %v", err),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	// Map Maya status to our status
	var status paymentpb.PaymentStatus
	var action string

	switch webhook.PaymentStatus {
	case "PAYMENT_SUCCESS":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS
		action = "success"
	case "PAYMENT_FAILED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED
		action = "failure"
	case "PAYMENT_EXPIRED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_EXPIRED
		action = "expired"
	case "PAYMENT_CANCELLED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_CANCELLED
		action = "cancelled"
	case "AUTHORIZED":
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
		action = "authorized"
	default:
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_PROCESSING
		action = "processing"
	}

	// Extract payment method details
	methodDetails := &paymentpb.PaymentMethodDetails{}
	paymentMethod := "unknown"
	if webhook.FundSource != nil {
		paymentMethod = webhook.FundSource.Type
		methodDetails.Type = webhook.FundSource.Type
		if webhook.FundSource.Details != nil {
			methodDetails.LastFour = webhook.FundSource.Details.Last4
			if webhook.FundSource.Details.ThreeDSecure != nil {
				methodDetails.Eci = webhook.FundSource.Details.ThreeDSecure.Eci
				methodDetails.PayerAuth = webhook.FundSource.Details.ThreeDSecure.Status
			}
		}
	}

	// Extract payment_id from metadata if available
	paymentId := webhook.RequestReferenceNumber
	if webhook.Metadata != nil {
		if pid, ok := webhook.Metadata["payment_id"].(string); ok && pid != "" {
			paymentId = pid
		}
	}

	transaction := &paymentpb.PaymentTransaction{
		Id:                 webhook.ID,
		ProviderRef:        webhook.RequestReferenceNumber,
		ProviderPaymentRef: webhook.ReceiptNumber,
		ProviderId:         "maya",
		Status:             status,
		Amount:             int64(webhook.Amount * 100), // Convert to cents
		Currency:           webhook.Currency,
		PaymentMethod:      paymentMethod,
		MethodDetails:      methodDetails,
		PaymentId:          paymentId,
		OrderRef:           webhook.RequestReferenceNumber,
		ProcessedAt:        timestamppb.Now(),
		RawData: map[string]string{
			"id":             webhook.ID,
			"payment_status": webhook.PaymentStatus,
			"is_paid":        fmt.Sprintf("%v", webhook.IsPaid),
		},
	}

	log.Printf("📨 Maya webhook processed: %s -> %s", webhook.ID, action)
	return &paymentpb.ProcessWebhookResponse{
		Success: true,
		Data: []*paymentpb.WebhookResult{{
			Transaction: transaction,
			Status:      status,
			Action:      action,
			PaymentId:   paymentId,
		}},
	}, nil
}

func (p *MayaProvider) GetPaymentStatus(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Maya provider is not initialized")
	}

	// Maya supports payment status queries via GET /payments/v1/payments/{paymentId}
	// For now, return not supported - can be implemented later
	return &paymentpb.GetPaymentStatusResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:        "NOT_SUPPORTED",
			Description: "Direct status query not implemented - use webhook callbacks",
			Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
		},
	}, nil
}

func (p *MayaProvider) RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Maya provider is not initialized")
	}

	// Maya supports refunds via POST /payments/v1/payments/{paymentId}/refunds
	// Requires secret key authentication
	return &paymentpb.RefundPaymentResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:        "NOT_SUPPORTED",
			Description: "Refund API not implemented - contact support or use Maya dashboard",
			Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
		},
	}, nil
}

func (p *MayaProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("Maya provider is not initialized")
	}
	if p.publicKey == "" || p.secretKey == "" {
		return fmt.Errorf("Maya provider is not properly configured")
	}
	return nil
}

func (p *MayaProvider) Close() error {
	p.enabled = false
	log.Printf("🔌 Maya provider closed")
	return nil
}

func (p *MayaProvider) IsEnabled() bool {
	return p.enabled
}

func (p *MayaProvider) GetCapabilities() []paymentpb.PaymentCapability {
	return []paymentpb.PaymentCapability{
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_ONE_TIME,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_WEBHOOKS,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_3DS,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_REFUND,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_VOID,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_TOKENIZATION,
	}
}

func (p *MayaProvider) GetSupportedCurrencies() []string {
	return []string{"PHP", "USD"}
}

var _ ports.PaymentProvider = (*MayaProvider)(nil)
