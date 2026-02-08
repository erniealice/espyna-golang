//go:build microsoft && microsoftgraph

package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	microsoftclient "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/common/microsoft"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	fmt.Println("[MicrosoftEmailAdapter] init() called - registering microsoft email provider")
	registry.RegisterEmailProvider(
		"microsoft",
		func() ports.EmailProvider {
			return NewMicrosoftGraphProvider()
		},
		transformConfig,
	)
	registry.RegisterEmailBuildFromEnv("microsoft", buildFromEnv)
	fmt.Println("[MicrosoftEmailAdapter] init() complete - microsoft email provider registered")
}

// buildFromEnv creates a Microsoft Graph email provider from environment variables
func buildFromEnv() (ports.EmailProvider, error) {
	provider := NewMicrosoftGraphProvider()

	// Read required environment variables
	tenantID := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_TENANT_ID")
	if tenantID == "" {
		return nil, fmt.Errorf("microsoft: LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_TENANT_ID is required")
	}

	clientID := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("microsoft: LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_CLIENT_ID is required")
	}

	clientSecret := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("microsoft: LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_CLIENT_SECRET is required")
	}

	delegateEmail := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_DELEGATE_EMAIL")
	if delegateEmail == "" {
		return nil, fmt.Errorf("microsoft: LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_DELEGATE_EMAIL is required")
	}

	// Optional environment variables
	fromEmail := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = delegateEmail
	}

	fromName := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_FROM_NAME")

	// Build protobuf config
	config := &emailpb.EmailProviderConfig{
		ProviderId:         "microsoft",
		ProviderType:       emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_OAUTH,
		Enabled:            true,
		DefaultFromAddress: fromEmail,
		Auth: &emailpb.EmailProviderConfig_Oauth2Auth{
			Oauth2Auth: &emailpb.OAuth2Auth{
				TenantId:       tenantID,
				ClientId:       clientID,
				ClientSecret:   clientSecret,
				DelegatedEmail: delegateEmail,
			},
		},
		Settings: make(map[string]string),
	}

	if fromName != "" {
		config.DefaultFromName = fromName
	}

	// Optional settings
	if redirectURL := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_REDIRECT_URL"); redirectURL != "" {
		config.GetOauth2Auth().RedirectUrl = redirectURL
	}
	if accessToken := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_ACCESS_TOKEN"); accessToken != "" {
		config.GetOauth2Auth().AccessToken = accessToken
	}
	if refreshToken := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_REFRESH_TOKEN"); refreshToken != "" {
		config.GetOauth2Auth().RefreshToken = refreshToken
	}
	if tokenType := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_TOKEN_TYPE"); tokenType != "" {
		config.Settings["token_type"] = tokenType
	}
	if tokenExpiry := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_TOKEN_EXPIRY"); tokenExpiry != "" {
		config.Settings["token_expiry"] = tokenExpiry
	}
	if timeout := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.TimeoutSeconds = int32(duration.Seconds())
		}
	}

	// Initialize the provider
	if err := provider.Initialize(config); err != nil {
		return nil, fmt.Errorf("microsoft: failed to initialize: %w", err)
	}

	return provider, nil
}

// transformConfig converts raw config map to Microsoft Graph proto config.
func transformConfig(rawConfig map[string]any) (*emailpb.EmailProviderConfig, error) {
	protoConfig := &emailpb.EmailProviderConfig{
		ProviderId:   "microsoft",
		ProviderType: emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_OAUTH,
		Enabled:      true,
		Settings:     make(map[string]string),
	}

	var fromEmail string
	if fe, ok := rawConfig["from_email"].(string); ok && fe != "" {
		fromEmail = fe
		protoConfig.DefaultFromAddress = fe
	} else {
		return nil, fmt.Errorf("microsoft: from_email is required")
	}
	if fromName, ok := rawConfig["from_name"].(string); ok {
		protoConfig.DefaultFromName = fromName
	}

	oauth2Auth := &emailpb.OAuth2Auth{DelegatedEmail: fromEmail}
	if clientID, ok := rawConfig["client_id"].(string); ok && clientID != "" {
		oauth2Auth.ClientId = clientID
	} else {
		return nil, fmt.Errorf("microsoft: client_id is required")
	}
	if clientSecret, ok := rawConfig["client_secret"].(string); ok && clientSecret != "" {
		oauth2Auth.ClientSecret = clientSecret
	} else {
		return nil, fmt.Errorf("microsoft: client_secret is required")
	}
	if tenantID, ok := rawConfig["tenant_id"].(string); ok && tenantID != "" {
		oauth2Auth.TenantId = tenantID
	} else {
		return nil, fmt.Errorf("microsoft: tenant_id is required")
	}
	protoConfig.Auth = &emailpb.EmailProviderConfig_Oauth2Auth{Oauth2Auth: oauth2Auth}

	if timeout, ok := rawConfig["timeout"].(int); ok && timeout > 0 {
		protoConfig.TimeoutSeconds = int32(timeout)
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// MicrosoftGraphProvider implements Microsoft Graph API email provider
type MicrosoftGraphProvider struct {
	enabled   bool
	userEmail string
	timeout   time.Duration
}

// Microsoft Graph API response structures
type graphMessage struct {
	ID                     string            `json:"id,omitempty"`
	Subject                string            `json:"subject,omitempty"`
	Body                   *graphItemBody    `json:"body,omitempty"`
	From                   *graphRecipient   `json:"from,omitempty"`
	ToRecipients           []graphRecipient  `json:"toRecipients"`
	CcRecipients           []graphRecipient  `json:"ccRecipients"`
	BccRecipients          []graphRecipient  `json:"bccRecipients"`
	ReceivedDateTime       string            `json:"receivedDateTime,omitempty"`
	SentDateTime           string            `json:"sentDateTime,omitempty"`
	HasAttachments         bool              `json:"hasAttachments,omitempty"`
	Attachments            []graphAttachment `json:"attachments,omitempty"`
	InternetMessageHeaders []graphHeader     `json:"internetMessageHeaders,omitempty"`
}

type graphItemBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type graphRecipient struct {
	EmailAddress *graphEmailAddress `json:"emailAddress"`
}

type graphEmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type graphAttachment struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	Size         int64  `json:"size"`
	ContentBytes string `json:"contentBytes"`
}

type graphHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type graphMessageListResponse struct {
	Value    []graphMessage `json:"value"`
	NextLink string         `json:"@odata.nextLink"`
}

type graphSendMessageRequest struct {
	Message         *graphMessage `json:"message"`
	SaveToSentItems bool          `json:"saveToSentItems"`
}

// NewMicrosoftGraphProvider creates a new Microsoft Graph email provider
func NewMicrosoftGraphProvider() ports.EmailProvider {
	return &MicrosoftGraphProvider{
		enabled: false,
		timeout: 30 * time.Second,
	}
}

// Name returns the name of this email provider
func (p *MicrosoftGraphProvider) Name() string {
	return "microsoft_graph"
}

// Initialize sets up the Microsoft Graph provider with configuration
func (p *MicrosoftGraphProvider) Initialize(config *emailpb.EmailProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required for Microsoft Graph provider")
	}

	// Get OAuth2 auth config
	oauth2 := config.GetOauth2Auth()
	if oauth2 == nil {
		return fmt.Errorf("OAuth2 configuration is required for Microsoft Graph provider")
	}

	// Extract user email (delegated email in OAuth2 context)
	p.userEmail = oauth2.DelegatedEmail
	if p.userEmail == "" && config.Settings != nil {
		p.userEmail = config.Settings["user_email"]
	}

	// Optional: timeout configuration
	if config.TimeoutSeconds > 0 {
		p.timeout = time.Duration(config.TimeoutSeconds) * time.Second
	}

	// Validate required fields
	if p.userEmail == "" {
		return fmt.Errorf("user_email (or delegated_email) cannot be empty")
	}

	// Initialize Microsoft common client with simple Client Credentials flow
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	microsoftConfig := &microsoftclient.MicrosoftConfig{
		TenantID:     oauth2.TenantId,
		ClientID:     oauth2.ClientId,
		ClientSecret: oauth2.ClientSecret,
		Timeout:      p.timeout,
	}

	if err := microsoftclient.InitializeMicrosoftClient(ctx, microsoftConfig); err != nil {
		return fmt.Errorf("failed to initialize Microsoft Graph client: %w", err)
	}

	p.enabled = config.Enabled

	// Test Microsoft Graph connectivity
	if err := p.testGraphAccess(ctx); err != nil {
		p.enabled = false
		return fmt.Errorf("Microsoft Graph access test failed: %w", err)
	}

	return nil
}

// testGraphAccess tests if we can access Microsoft Graph API
func (p *MicrosoftGraphProvider) testGraphAccess(ctx context.Context) error {
	return microsoftclient.TestConnection(ctx)
}

// sendEmailLegacy sends an email message using Microsoft Graph API (legacy method)
func (p *MicrosoftGraphProvider) sendEmailLegacy(ctx context.Context, message ports.EmailMessage) error {
	if !p.enabled {
		return fmt.Errorf("Microsoft Graph provider is not initialized")
	}

	// Convert to Graph message format
	graphMessage := p.convertToGraphMessage(message)

	// Create send request
	sendRequest := graphSendMessageRequest{
		Message:         &graphMessage,
		SaveToSentItems: true,
	}

	jsonBody, err := json.Marshal(sendRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal send request: %w", err)
	}

	// Get authenticated client
	client, err := microsoftclient.GetAuthenticatedClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated client: %w", err)
	}

	// Create HTTP request
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", p.userEmail)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create send request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Microsoft Graph send failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// convertToGraphMessage converts common EmailMessage to Graph message format
func (p *MicrosoftGraphProvider) convertToGraphMessage(message ports.EmailMessage) graphMessage {
	graphMsg := graphMessage{
		Subject:       message.Subject,
		ToRecipients:  []graphRecipient{},  // Initialize empty to avoid null in JSON
		CcRecipients:  []graphRecipient{},  // Initialize empty to avoid null in JSON
		BccRecipients: []graphRecipient{},  // Initialize empty to avoid null in JSON
	}

	// Set body
	if message.HTMLBody != "" {
		graphMsg.Body = &graphItemBody{
			ContentType: "HTML",
			Content:     message.HTMLBody,
		}
	} else {
		graphMsg.Body = &graphItemBody{
			ContentType: "Text",
			Content:     message.TextBody,
		}
	}

	// Set from (if provided)
	if message.From != "" {
		graphMsg.From = &graphRecipient{
			EmailAddress: &graphEmailAddress{
				Address: message.From,
			},
		}
	}

	// Set to recipients
	for _, to := range message.To {
		graphMsg.ToRecipients = append(graphMsg.ToRecipients, graphRecipient{
			EmailAddress: &graphEmailAddress{
				Address: to,
			},
		})
	}

	// Set CC recipients
	for _, cc := range message.CC {
		graphMsg.CcRecipients = append(graphMsg.CcRecipients, graphRecipient{
			EmailAddress: &graphEmailAddress{
				Address: cc,
			},
		})
	}

	// Set BCC recipients
	for _, bcc := range message.BCC {
		graphMsg.BccRecipients = append(graphMsg.BccRecipients, graphRecipient{
			EmailAddress: &graphEmailAddress{
				Address: bcc,
			},
		})
	}

	return graphMsg
}

// getInboxMessagesLegacy retrieves messages from Microsoft Graph inbox (legacy method)
func (p *MicrosoftGraphProvider) getInboxMessagesLegacy(ctx context.Context, options ports.InboxOptions) ([]ports.EmailMessage, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Microsoft Graph provider is not initialized")
	}

	// Build query parameters
	params := url.Values{}
	if options.MaxResults > 0 {
		params.Set("$top", fmt.Sprintf("%d", options.MaxResults))
	}
	if options.Query != "" {
		params.Set("$filter", options.Query)
	}

	// Get authenticated client
	client, err := microsoftclient.GetAuthenticatedClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	// Create HTTP request
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/messages", p.userEmail)
	if params.Encode() != "" {
		apiURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Microsoft Graph list failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var listResp graphMessageListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	// Convert to common format
	var messages []ports.EmailMessage
	for _, graphMsg := range listResp.Value {
		message := p.convertFromGraphMessage(graphMsg)
		messages = append(messages, message)
	}

	return messages, nil
}

// getMessageLegacy retrieves a specific email message by ID (legacy method)
func (p *MicrosoftGraphProvider) getMessageLegacy(ctx context.Context, messageID string) (*ports.EmailMessage, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Microsoft Graph provider is not initialized")
	}

	// Get authenticated client
	client, err := microsoftclient.GetAuthenticatedClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	// Create HTTP request
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/messages/%s", p.userEmail, messageID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get request: %w", err)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Microsoft Graph get failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var graphMsg graphMessage
	if err := json.NewDecoder(resp.Body).Decode(&graphMsg); err != nil {
		return nil, fmt.Errorf("failed to decode message response: %w", err)
	}

	// Convert to common format
	message := p.convertFromGraphMessage(graphMsg)
	return &message, nil
}

// convertFromGraphMessage converts Graph message to common EmailMessage format
func (p *MicrosoftGraphProvider) convertFromGraphMessage(graphMsg graphMessage) ports.EmailMessage {
	message := ports.EmailMessage{
		ID:      graphMsg.ID,
		Subject: graphMsg.Subject,
		Headers: make(map[string]string),
	}

	// Set body
	if graphMsg.Body != nil {
		if strings.ToLower(graphMsg.Body.ContentType) == "html" {
			message.HTMLBody = graphMsg.Body.Content
		} else {
			message.TextBody = graphMsg.Body.Content
		}
	}

	// Set from
	if graphMsg.From != nil && graphMsg.From.EmailAddress != nil {
		message.From = graphMsg.From.EmailAddress.Address
	}

	// Set to recipients
	for _, recipient := range graphMsg.ToRecipients {
		if recipient.EmailAddress != nil {
			message.To = append(message.To, recipient.EmailAddress.Address)
		}
	}

	// Set CC recipients
	for _, recipient := range graphMsg.CcRecipients {
		if recipient.EmailAddress != nil {
			message.CC = append(message.CC, recipient.EmailAddress.Address)
		}
	}

	// Set BCC recipients
	for _, recipient := range graphMsg.BccRecipients {
		if recipient.EmailAddress != nil {
			message.BCC = append(message.BCC, recipient.EmailAddress.Address)
		}
	}

	// Set headers
	for _, header := range graphMsg.InternetMessageHeaders {
		message.Headers[header.Name] = header.Value
	}

	// Set timestamp
	if graphMsg.ReceivedDateTime != "" {
		if timestamp, err := time.Parse(time.RFC3339, graphMsg.ReceivedDateTime); err == nil {
			message.Timestamp = timestamp.Unix()
		}
	} else if graphMsg.SentDateTime != "" {
		if timestamp, err := time.Parse(time.RFC3339, graphMsg.SentDateTime); err == nil {
			message.Timestamp = timestamp.Unix()
		}
	}

	// Set attachments
	for _, attachment := range graphMsg.Attachments {
		emailAttachment := ports.EmailAttachment{
			Name:        attachment.Name,
			ContentType: attachment.ContentType,
			Size:        attachment.Size,
		}

		// Decode base64 content if available
		if attachment.ContentBytes != "" {
			// In a real implementation, you would decode the base64 content
			// For now, we'll just store the encoded string
			emailAttachment.Data = []byte(attachment.ContentBytes)
		}

		message.Attachments = append(message.Attachments, emailAttachment)
	}

	return message
}

// IsHealthy checks if the Microsoft Graph service is available
func (p *MicrosoftGraphProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("Microsoft Graph provider is not initialized")
	}

	return p.testGraphAccess(ctx)
}

// Close cleans up Microsoft Graph provider resources
func (p *MicrosoftGraphProvider) Close() error {
	p.enabled = false
	return microsoftclient.Close()
}

// IsEnabled returns whether this provider is currently enabled
func (p *MicrosoftGraphProvider) IsEnabled() bool {
	return p.enabled
}

// GetCapabilities returns the capabilities supported by this provider
func (p *MicrosoftGraphProvider) GetCapabilities() []emailpb.EmailCapability {
	return []emailpb.EmailCapability{
		emailpb.EmailCapability_EMAIL_CAPABILITY_SEND,
		emailpb.EmailCapability_EMAIL_CAPABILITY_ATTACHMENTS,
		emailpb.EmailCapability_EMAIL_CAPABILITY_READ_INBOX,
		emailpb.EmailCapability_EMAIL_CAPABILITY_THREADING,
		emailpb.EmailCapability_EMAIL_CAPABILITY_LABELS,
		emailpb.EmailCapability_EMAIL_CAPABILITY_SEARCH,
	}
}

// GetProviderType returns the provider type
func (p *MicrosoftGraphProvider) GetProviderType() emailpb.EmailProviderType {
	return emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_OAUTH
}

// SendEmail sends an email using protobuf request/response types (implements ports.EmailProvider interface)
func (p *MicrosoftGraphProvider) SendEmail(ctx context.Context, req *emailpb.SendEmailRequest) (*emailpb.SendEmailResponse, error) {
	if !p.enabled {
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Microsoft Graph provider is not initialized",
			},
		}, nil
	}

	// Extract data from request
	data := req.Data
	if data == nil {
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Convert protobuf request to internal message format
	message := ports.EmailMessage{
		Subject:  data.Subject,
		TextBody: data.TextBody,
		HTMLBody: data.HtmlBody,
		Headers:  data.Headers,
	}

	if data.From != nil {
		message.From = data.From.Address
	}

	for _, addr := range data.To {
		message.To = append(message.To, addr.Address)
	}

	for _, addr := range data.Cc {
		message.CC = append(message.CC, addr.Address)
	}

	for _, addr := range data.Bcc {
		message.BCC = append(message.BCC, addr.Address)
	}

	for _, att := range data.Attachments {
		message.Attachments = append(message.Attachments, ports.EmailAttachment{
			ID:          att.Id,
			Name:        att.Name,
			ContentType: att.ContentType,
			Size:        att.Size,
			Data:        att.Data,
			ContentID:   att.ContentId,
			IsInline:    att.IsInline,
		})
	}

	// Use the legacy send method
	err := p.sendEmailLegacy(ctx, message)
	if err != nil {
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SEND_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &emailpb.SendEmailResponse{
		Success: true,
	}, nil
}

// SendBatchEmails sends multiple emails in a batch
func (p *MicrosoftGraphProvider) SendBatchEmails(ctx context.Context, req *emailpb.SendBatchEmailsRequest) (*emailpb.SendBatchEmailsResponse, error) {
	if !p.enabled {
		return &emailpb.SendBatchEmailsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Microsoft Graph provider is not initialized",
			},
		}, nil
	}

	data := req.Data
	if data == nil {
		return &emailpb.SendBatchEmailsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	var results []*emailpb.EmailResult
	var successCount, failCount int

	for _, emailData := range data.Emails {
		// Wrap each email data in a SendEmailRequest
		emailReq := &emailpb.SendEmailRequest{
			Data: emailData,
		}
		resp, _ := p.SendEmail(ctx, emailReq)

		// Convert SendEmailResponse to EmailResult
		result := &emailpb.EmailResult{}
		if resp != nil && resp.Success {
			successCount++
			// We could populate result.Message here if we tracked the sent message
		} else {
			failCount++
			if data.FailFast {
				break
			}
		}
		results = append(results, result)
	}

	return &emailpb.SendBatchEmailsResponse{
		Success: failCount == 0,
		Data: []*emailpb.BatchEmailResult{
			{
				Results:      results,
				SuccessCount: int32(successCount),
				FailureCount: int32(failCount),
				BatchId:      data.BatchId,
			},
		},
	}, nil
}

// GetInboxMessages retrieves messages from inbox using protobuf types (implements ports.EmailProvider interface)
func (p *MicrosoftGraphProvider) GetInboxMessages(ctx context.Context, req *emailpb.GetInboxMessagesRequest) (*emailpb.GetInboxMessagesResponse, error) {
	if !p.enabled {
		return &emailpb.GetInboxMessagesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Microsoft Graph provider is not initialized",
			},
		}, nil
	}

	data := req.Data
	if data == nil {
		return &emailpb.GetInboxMessagesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	options := ports.InboxOptions{
		MaxResults: int(data.MaxResults),
		PageToken:  data.PageToken,
		Query:      data.Query,
		LabelIDs:   data.LabelIds,
	}

	messages, err := p.getInboxMessagesLegacy(ctx, options)
	if err != nil {
		return &emailpb.GetInboxMessagesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "RETRIEVAL_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	var protoMessages []*emailpb.EmailMessage
	for _, msg := range messages {
		protoMessages = append(protoMessages, p.convertToProtoMessage(msg))
	}

	return &emailpb.GetInboxMessagesResponse{
		Success: true,
		Data:    protoMessages,
	}, nil
}

// GetMessage retrieves a specific message by ID using protobuf types (implements ports.EmailProvider interface)
func (p *MicrosoftGraphProvider) GetMessage(ctx context.Context, req *emailpb.GetMessageRequest) (*emailpb.GetMessageResponse, error) {
	if !p.enabled {
		return &emailpb.GetMessageResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Microsoft Graph provider is not initialized",
			},
		}, nil
	}

	data := req.Data
	if data == nil {
		return &emailpb.GetMessageResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	message, err := p.getMessageLegacy(ctx, data.MessageId)
	if err != nil {
		return &emailpb.GetMessageResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "RETRIEVAL_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &emailpb.GetMessageResponse{
		Success: true,
		Data:    []*emailpb.EmailMessage{p.convertToProtoMessage(*message)},
	}, nil
}

// convertToProtoMessage converts internal EmailMessage to protobuf EmailMessage
func (p *MicrosoftGraphProvider) convertToProtoMessage(msg ports.EmailMessage) *emailpb.EmailMessage {
	protoMsg := &emailpb.EmailMessage{
		Id:       msg.ID,
		Subject:  msg.Subject,
		TextBody: msg.TextBody,
		HtmlBody: msg.HTMLBody,
		Headers:  msg.Headers,
	}

	if msg.From != "" {
		protoMsg.From = &emailpb.EmailAddress{Address: msg.From}
	}

	for _, addr := range msg.To {
		protoMsg.To = append(protoMsg.To, &emailpb.EmailAddress{Address: addr})
	}

	for _, addr := range msg.CC {
		protoMsg.Cc = append(protoMsg.Cc, &emailpb.EmailAddress{Address: addr})
	}

	for _, addr := range msg.BCC {
		protoMsg.Bcc = append(protoMsg.Bcc, &emailpb.EmailAddress{Address: addr})
	}

	for _, att := range msg.Attachments {
		protoMsg.Attachments = append(protoMsg.Attachments, &emailpb.EmailAttachment{
			Id:          att.ID,
			Name:        att.Name,
			ContentType: att.ContentType,
			Size:        att.Size,
			Data:        att.Data,
			ContentId:   att.ContentID,
			IsInline:    att.IsInline,
		})
	}

	return protoMsg
}
