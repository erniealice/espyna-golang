//go:build asiapay

package asiapay

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	paymentpb "leapfor.xyz/esqyma/golang/v1/integration/payment"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterPaymentProvider(
		"asiapay",
		func() ports.PaymentProvider {
			return NewAsiaPayProvider()
		},
		transformConfig,
	)
	registry.RegisterPaymentBuildFromEnv("asiapay", buildFromEnv)
}

// buildFromEnv creates and initializes an AsiaPay provider from environment variables.
func buildFromEnv() (ports.PaymentProvider, error) {
	merchantID := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_ASIAPAY_MERCHANT_ID")
	secureSecret := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_ASIAPAY_SECURE_SECRET")
	currencyCode := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_ASIAPAY_CURRENCY_CODE")
	sandboxMode := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_ASIAPAY_SANDBOX") == "true"
	baseURL := os.Getenv("LEAPFOR_INTEGRATION_PAYMENT_ASIAPAY_BASE_URL")

	if merchantID == "" {
		return nil, fmt.Errorf("asiapay: LEAPFOR_INTEGRATION_PAYMENT_ASIAPAY_MERCHANT_ID is required")
	}
	if secureSecret == "" {
		return nil, fmt.Errorf("asiapay: LEAPFOR_INTEGRATION_PAYMENT_ASIAPAY_SECURE_SECRET is required")
	}
	if currencyCode == "" {
		currencyCode = "608" // Default PHP
	}

	protoConfig := &paymentpb.PaymentProviderConfig{
		ProviderId:   "asiapay",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_GATEWAY,
		Enabled:      true,
		SandboxMode:  sandboxMode,
		RedirectUrls: &paymentpb.RedirectUrls{
			BaseUrl: baseURL,
		},
		Auth: &paymentpb.PaymentProviderConfig_AsiapayAuth{
			AsiapayAuth: &paymentpb.AsiaPayAuth{
				MerchantId:   merchantID,
				SecureSecret: secureSecret,
				CurrencyCode: currencyCode,
			},
		},
	}

	p := NewAsiaPayProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("asiapay: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to AsiaPay proto config.
func transformConfig(rawConfig map[string]any) (*paymentpb.PaymentProviderConfig, error) {
	protoConfig := &paymentpb.PaymentProviderConfig{
		ProviderId:   "asiapay",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_GATEWAY,
		Enabled:      true,
	}

	asiaPayAuth := &paymentpb.AsiaPayAuth{}

	if merchantID, ok := rawConfig["merchant_id"].(string); ok && merchantID != "" {
		asiaPayAuth.MerchantId = merchantID
	} else {
		return nil, fmt.Errorf("asiapay: merchant_id is required")
	}
	if secureSecret, ok := rawConfig["secure_secret"].(string); ok && secureSecret != "" {
		asiaPayAuth.SecureSecret = secureSecret
	} else {
		return nil, fmt.Errorf("asiapay: secure_secret is required")
	}
	if currencyCode, ok := rawConfig["currency_code"].(string); ok {
		asiaPayAuth.CurrencyCode = currencyCode
	}

	protoConfig.Auth = &paymentpb.PaymentProviderConfig_AsiapayAuth{AsiapayAuth: asiaPayAuth}

	if sandboxMode, ok := rawConfig["sandbox_mode"].(bool); ok {
		protoConfig.SandboxMode = sandboxMode
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// AsiaPay endpoints
const (
	asiaPayProductionURL = "https://www.paydollar.com/b2cDemo/eng/payment/payForm.jsp"
	asiaPaySandboxURL    = "https://test.pesopay.com/b2cDemo/eng/payment/payForm.jsp"
)

// AsiaPayProvider implements the PaymentProvider interface for AsiaPay payment gateway
type AsiaPayProvider struct {
	enabled       bool
	merchantID    string
	secureSecret  string
	currencyCode  string
	paymentType   string
	language      string
	paymentMethod string
	sandboxMode   bool
	apiEndpoint   string
	baseURL       string
	successPath   string
	failurePath   string
	cancelPath    string
	webhookPath   string
	timeout       time.Duration
}

// PayMayaWebhookRequest represents the JSON webhook payload from PayMaya/AsiaPay
type PayMayaWebhookRequest struct {
	Body struct {
		RequestReferenceNumber string         `json:"requestReferenceNumber"`
		Status                 string         `json:"status"`
		Metadata               map[string]any `json:"metadata"`
		FundSource             map[string]any `json:"fundSource"`
		PaymentDetails         map[string]any `json:"paymentDetails"`
	} `json:"body"`
	Headers map[string]string `json:"headers"`
	Params  map[string]string `json:"params"`
	Query   map[string]string `json:"query"`
}

// AsiaPayDatafeedRequest represents the datafeed/webhook payload from AsiaPay
type AsiaPayDatafeedRequest struct {
	PRC                string `form:"prc"`
	SRC                string `form:"src"`
	SuccessCode        string `form:"successcode"`
	Ord                string `form:"Ord"`
	Ref                string `form:"Ref"`
	PayRef             string `form:"PayRef"`
	Amt                string `form:"Amt"`
	Cur                string `form:"Cur"`
	Holder             string `form:"Holder"`
	AuthId             string `form:"AuthId"`
	PayMethod          string `form:"payMethod"`
	CardIssuingCountry string `form:"cardIssuingCountry"`
	ChannelType        string `form:"channelType"`
	ECI                string `form:"eci"`
	PayerAuth          string `form:"payerAuth"`
	SourceIP           string `form:"sourceIp"`
	IPCountry          string `form:"ipCountry"`
	AlertCode          string `form:"AlertCode"`
	Remark             string `form:"remark"`
}

func (r *AsiaPayDatafeedRequest) IsSuccess() bool {
	return r.SuccessCode == "0"
}

func NewAsiaPayProvider() ports.PaymentProvider {
	return &AsiaPayProvider{
		enabled:       false,
		paymentType:   "N",
		language:      "E",
		paymentMethod: "ALL",
		currencyCode:  "608",
		timeout:       30 * time.Second,
	}
}

func (p *AsiaPayProvider) Name() string {
	return "asiapay"
}

func (p *AsiaPayProvider) Initialize(config *paymentpb.PaymentProviderConfig) error {
	asiaPayAuth := config.GetAsiapayAuth()
	if asiaPayAuth == nil {
		return fmt.Errorf("AsiaPay auth configuration is required")
	}

	if asiaPayAuth.MerchantId == "" {
		return fmt.Errorf("merchant_id is required for AsiaPay provider")
	}
	p.merchantID = asiaPayAuth.MerchantId

	if asiaPayAuth.SecureSecret == "" {
		return fmt.Errorf("secure_secret is required for AsiaPay provider")
	}
	p.secureSecret = asiaPayAuth.SecureSecret

	if asiaPayAuth.CurrencyCode != "" {
		p.currencyCode = asiaPayAuth.CurrencyCode
	}
	if asiaPayAuth.PaymentType != "" {
		p.paymentType = asiaPayAuth.PaymentType
	}
	if asiaPayAuth.Language != "" {
		p.language = asiaPayAuth.Language
	}
	if asiaPayAuth.PaymentMethod != "" {
		p.paymentMethod = asiaPayAuth.PaymentMethod
	}

	p.sandboxMode = config.SandboxMode
	if p.sandboxMode {
		p.apiEndpoint = asiaPaySandboxURL
	} else {
		p.apiEndpoint = asiaPayProductionURL
	}

	if config.ApiEndpoint != "" {
		p.apiEndpoint = config.ApiEndpoint
	} else if p.sandboxMode && config.SandboxEndpoint != "" {
		p.apiEndpoint = config.SandboxEndpoint
	}

	if redirects := config.RedirectUrls; redirects != nil {
		p.baseURL = redirects.BaseUrl
		p.successPath = redirects.SuccessPath
		p.failurePath = redirects.FailurePath
		p.cancelPath = redirects.CancelPath
		p.webhookPath = redirects.WebhookPath
	}
	if p.successPath == "" {
		p.successPath = "/payment/success"
	}
	if p.failurePath == "" {
		p.failurePath = "/payment/fail"
	}
	if p.cancelPath == "" {
		p.cancelPath = "/payment/cancel"
	}
	if p.webhookPath == "" {
		p.webhookPath = "/integration/payment/webhook"
	}

	if config.TimeoutSeconds > 0 {
		p.timeout = time.Duration(config.TimeoutSeconds) * time.Second
	}

	p.enabled = config.Enabled
	log.Printf("✅ AsiaPay payment provider initialized (Sandbox: %v)", p.sandboxMode)
	return nil
}

func (p *AsiaPayProvider) CreateCheckoutSession(ctx context.Context, req *paymentpb.CreateCheckoutSessionRequest) (*paymentpb.CreateCheckoutSessionResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("AsiaPay provider is not initialized")
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

	currencyCode := p.currencyCode
	if data.Currency != "" {
		currencyCode = p.getCurrencyNumericCode(data.Currency)
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

	// Convert amount from dollars to cents (AsiaPay expects cents)
	amountCents := int64(data.Amount * 100)
	amountStr := p.formatAmount(amountCents, currencyCode)
	secureHash := p.generateSecureHash(merchantRef, currencyCode, amountStr)

	// Build redirect URLs
	// If URL is provided but doesn't include the path, append it
	successURL := data.SuccessUrl
	if successURL == "" && p.baseURL != "" {
		successURL = p.baseURL + p.successPath
	} else if successURL != "" && !strings.Contains(successURL, p.successPath) && p.successPath != "" {
		successURL = strings.TrimSuffix(successURL, "/") + p.successPath
	}
	failureURL := data.FailureUrl
	if failureURL == "" && p.baseURL != "" {
		failureURL = p.baseURL + p.failurePath
	} else if failureURL != "" && !strings.Contains(failureURL, p.failurePath) && p.failurePath != "" {
		failureURL = strings.TrimSuffix(failureURL, "/") + p.failurePath
	}
	cancelURL := data.CancelUrl
	if cancelURL == "" && p.baseURL != "" {
		cancelURL = p.baseURL + p.cancelPath
	} else if cancelURL != "" && !strings.Contains(cancelURL, p.cancelPath) && p.cancelPath != "" {
		cancelURL = strings.TrimSuffix(cancelURL, "/") + p.cancelPath
	}
	datafeedURL := p.baseURL + p.webhookPath

	checkoutURL := p.buildCheckoutURL(merchantRef, currencyCode, amountStr, secureHash,
		successURL, failureURL, cancelURL, datafeedURL, data.Description)

	now := timestamppb.Now()
	expiresAt := timestamppb.New(time.Now().Add(30 * time.Minute))
	if data.ExpiresInMinutes > 0 {
		expiresAt = timestamppb.New(time.Now().Add(time.Duration(data.ExpiresInMinutes) * time.Minute))
	}

	session := &paymentpb.CheckoutSession{
		Id:                fmt.Sprintf("ap_%s", merchantRef),
		ProviderSessionId: merchantRef,
		ProviderId:        "asiapay",
		ProviderType:      paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_GATEWAY,
		Amount:            data.Amount,
		Currency:          data.Currency,
		Status:            paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING,
		CheckoutUrl:       checkoutURL,
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
			"merchant_id":   p.merchantID,
			"currency_code": currencyCode,
			"secure_hash":   secureHash,
			"sandbox_mode":  fmt.Sprintf("%v", p.sandboxMode),
		},
	}

	log.Printf("📦 AsiaPay checkout session created: %s", session.Id)
	return &paymentpb.CreateCheckoutSessionResponse{Success: true, Data: []*paymentpb.CheckoutSession{session}}, nil
}

func (p *AsiaPayProvider) ProcessWebhook(ctx context.Context, req *paymentpb.ProcessWebhookRequest) (*paymentpb.ProcessWebhookResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("AsiaPay provider is not initialized")
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

	contentType := data.ContentType
	if contentType == "" {
		if ct, ok := data.Headers["Content-Type"]; ok {
			contentType = ct
		}
	}

	datafeed, err := p.parseWebhookPayload(data.Payload, contentType)
	if err != nil {
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "WEBHOOK_PARSE_ERROR",
				Description: fmt.Sprintf("Failed to parse webhook: %v", err),
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	var status paymentpb.PaymentStatus
	var action string
	if datafeed.IsSuccess() {
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS
		action = "success"
	} else {
		status = paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED
		action = "failure"
	}

	transaction := &paymentpb.PaymentTransaction{
		Id:                 datafeed.Ord,
		ProviderRef:        datafeed.Ref,
		ProviderPaymentRef: datafeed.PayRef,
		ProviderId:         "asiapay",
		Status:             status,
		Amount:             p.parseAmount(datafeed.Amt),
		Currency:           p.getCurrencyISOCode(datafeed.Cur),
		PaymentMethod:      datafeed.PayMethod,
		MethodDetails: &paymentpb.PaymentMethodDetails{
			Type:           datafeed.PayMethod,
			HolderName:     datafeed.Holder,
			IssuingCountry: datafeed.CardIssuingCountry,
			AuthId:         datafeed.AuthId,
			Eci:            datafeed.ECI,
			PayerAuth:      datafeed.PayerAuth,
		},
		ResponseCode: datafeed.PRC,
		PaymentId:    datafeed.Ref,
		OrderRef:     datafeed.Ord,
		ProcessedAt:  timestamppb.Now(),
		RawData: map[string]string{
			"prc": datafeed.PRC, "src": datafeed.SRC, "successcode": datafeed.SuccessCode,
		},
	}

	return &paymentpb.ProcessWebhookResponse{
		Success: true,
		Data: []*paymentpb.WebhookResult{{
			Transaction: transaction,
			Status:      status,
			Action:      action,
			PaymentId:   datafeed.Ref,
		}},
	}, nil
}

func (p *AsiaPayProvider) GetPaymentStatus(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("AsiaPay provider is not initialized")
	}
	return &paymentpb.GetPaymentStatusResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:        "NOT_SUPPORTED",
			Description: "AsiaPay does not support direct status queries - use webhook callbacks",
			Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
		},
	}, nil
}

func (p *AsiaPayProvider) RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("AsiaPay provider is not initialized")
	}
	return &paymentpb.RefundPaymentResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:        "NOT_SUPPORTED",
			Description: "Refunds require AsiaPay merchant portal or additional API configuration",
			Category:    commonpb.ErrorCategory_ERROR_CATEGORY_EXTERNAL_SERVICE,
		},
	}, nil
}

func (p *AsiaPayProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("AsiaPay provider is not initialized")
	}
	if p.merchantID == "" || p.secureSecret == "" {
		return fmt.Errorf("AsiaPay provider is not properly configured")
	}
	return nil
}

func (p *AsiaPayProvider) Close() error {
	p.enabled = false
	log.Printf("🔌 AsiaPay provider closed")
	return nil
}

func (p *AsiaPayProvider) IsEnabled() bool {
	return p.enabled
}

func (p *AsiaPayProvider) GetCapabilities() []paymentpb.PaymentCapability {
	return []paymentpb.PaymentCapability{
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_ONE_TIME,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_WEBHOOKS,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_3DS,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_MULTI_CURRENCY,
	}
}

func (p *AsiaPayProvider) GetSupportedCurrencies() []string {
	return []string{"PHP", "HKD", "USD", "SGD", "CNY", "JPY", "TWD", "AUD", "EUR", "GBP", "CAD", "MYR", "THB"}
}

// Helper methods
func (p *AsiaPayProvider) generateSecureHash(orderRef, currencyCode, amount string) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s", p.merchantID, orderRef, currencyCode, amount, p.paymentType, p.secureSecret)
	hash := sha1.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (p *AsiaPayProvider) buildCheckoutURL(orderRef, currencyCode, amount, secureHash, successURL, failureURL, cancelURL, datafeedURL, description string) string {
	params := url.Values{}
	params.Set("merchantId", p.merchantID)
	params.Set("orderRef", orderRef)
	params.Set("currCode", currencyCode)
	params.Set("amount", amount)
	params.Set("payType", p.paymentType)
	params.Set("lang", p.language)
	params.Set("payMethod", p.paymentMethod)
	params.Set("secureHash", secureHash)
	if successURL != "" {
		params.Set("successUrl", successURL)
	}
	if failureURL != "" {
		params.Set("failUrl", failureURL)
	}
	if cancelURL != "" {
		params.Set("cancelUrl", cancelURL)
	}
	if datafeedURL != "" {
		params.Set("datafeedUrl", datafeedURL)
	}
	if description != "" {
		params.Set("remark", description)
	}
	return p.apiEndpoint + "?" + params.Encode()
}

func (p *AsiaPayProvider) parseWebhookPayload(payload []byte, contentType string) (*AsiaPayDatafeedRequest, error) {
	datafeed := &AsiaPayDatafeedRequest{}

	if strings.Contains(contentType, "application/json") {
		var jsonReq PayMayaWebhookRequest
		if err := json.Unmarshal(payload, &jsonReq); err != nil {
			return nil, fmt.Errorf("failed to parse JSON payload: %w", err)
		}
		datafeed.Ref = jsonReq.Body.RequestReferenceNumber
		switch jsonReq.Body.Status {
		case "PAYMENT_SUCCESS":
			datafeed.SuccessCode = "0"
		default:
			datafeed.SuccessCode = "1"
		}
		if jsonReq.Body.PaymentDetails != nil {
			if amt, ok := jsonReq.Body.PaymentDetails["amount"].(float64); ok {
				datafeed.Amt = fmt.Sprintf("%.2f", amt)
			}
		}
		datafeed.Remark = jsonReq.Body.Status
		return datafeed, nil
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") || contentType == "" {
		values, err := url.ParseQuery(string(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to parse form data: %w", err)
		}
		datafeed.PRC = values.Get("prc")
		datafeed.SRC = values.Get("src")
		datafeed.SuccessCode = values.Get("successcode")
		datafeed.Ord = values.Get("Ord")
		datafeed.Ref = values.Get("Ref")
		datafeed.PayRef = values.Get("PayRef")
		datafeed.Amt = values.Get("Amt")
		datafeed.Cur = values.Get("Cur")
		datafeed.Holder = values.Get("Holder")
		datafeed.AuthId = values.Get("AuthId")
		datafeed.PayMethod = values.Get("payMethod")
		datafeed.CardIssuingCountry = values.Get("cardIssuingCountry")
		return datafeed, nil
	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

func (p *AsiaPayProvider) getCurrencyNumericCode(isoCode string) string {
	codes := map[string]string{
		"PHP": "608", "HKD": "344", "USD": "840", "SGD": "702", "CNY": "156",
		"JPY": "392", "TWD": "901", "AUD": "036", "EUR": "978", "GBP": "826",
		"CAD": "124", "MYR": "458", "THB": "764",
	}
	if code, ok := codes[strings.ToUpper(isoCode)]; ok {
		return code
	}
	return p.currencyCode
}

func (p *AsiaPayProvider) getCurrencyISOCode(numericCode string) string {
	codes := map[string]string{
		"608": "PHP", "344": "HKD", "840": "USD", "702": "SGD", "156": "CNY",
		"392": "JPY", "901": "TWD", "036": "AUD", "978": "EUR", "826": "GBP",
		"124": "CAD", "458": "MYR", "764": "THB",
	}
	if code, ok := codes[numericCode]; ok {
		return code
	}
	return "PHP"
}

func (p *AsiaPayProvider) formatAmount(cents int64, currencyCode string) string {
	if currencyCode == "392" {
		return fmt.Sprintf("%d", cents)
	}
	return fmt.Sprintf("%.2f", float64(cents)/100.0)
}

func (p *AsiaPayProvider) parseAmount(amountStr string) int64 {
	if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
		return int64(amount * 100)
	}
	return 0
}

var _ ports.PaymentProvider = (*AsiaPayProvider)(nil)
