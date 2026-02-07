//go:build google && gmail

package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
	"leapfor.xyz/espyna/internal/application/ports"
	googleclient "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/common/google"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	emailpb "leapfor.xyz/esqyma/golang/v1/integration/email"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterEmailProvider(
		"gmail",
		func() ports.EmailProvider {
			return NewGoogleEmailProvider()
		},
		transformConfig,
	)
	registry.RegisterEmailBuildFromEnv("gmail", buildFromEnv)
}

// buildFromEnv creates and initializes a Gmail email provider from environment variables.
func buildFromEnv() (ports.EmailProvider, error) {
	delegateEmail := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_DELEGATE_EMAIL")
	fromEmail := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_FROM_EMAIL")
	fromName := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_FROM_NAME")
	replyToEmail := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_REPLY_TO_EMAIL")
	serviceAccountKeyPath := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_SERVICE_ACCOUNT_KEY_PATH")
	secretManagerPath := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_SECRET_MANAGER_PATH")
	useSecretManager := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_USE_SECRET_MANAGER") == "true"
	timeoutStr := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_GMAIL_TIMEOUT")

	if delegateEmail == "" {
		return nil, fmt.Errorf("gmail: LEAPFOR_INTEGRATION_EMAIL_GMAIL_DELEGATE_EMAIL is required")
	}
	if fromEmail == "" {
		fromEmail = delegateEmail
	}

	timeout := 30
	if timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	settings := make(map[string]string)
	if secretManagerPath != "" {
		settings["secret_manager_path"] = secretManagerPath
	}
	if useSecretManager {
		settings["use_secret_manager"] = "true"
	}

	protoConfig := &emailpb.EmailProviderConfig{
		ProviderId:         "gmail",
		ProviderType:       emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_OAUTH,
		Enabled:            true,
		DefaultFromAddress: fromEmail,
		DefaultFromName:    fromName,
		DefaultReplyTo:     replyToEmail,
		TimeoutSeconds:     int32(timeout),
		Settings:           settings,
		Auth: &emailpb.EmailProviderConfig_Oauth2Auth{
			Oauth2Auth: &emailpb.OAuth2Auth{
				DelegatedEmail:    delegateEmail,
				ServiceAccountKey: serviceAccountKeyPath,
			},
		},
	}

	p := NewGoogleEmailProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("gmail: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to Gmail proto config.
func transformConfig(rawConfig map[string]any) (*emailpb.EmailProviderConfig, error) {
	protoConfig := &emailpb.EmailProviderConfig{
		ProviderId:   "gmail",
		ProviderType: emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_OAUTH,
		Enabled:      true,
		Settings:     make(map[string]string),
	}

	if fromEmail, ok := rawConfig["from_email"].(string); ok && fromEmail != "" {
		protoConfig.DefaultFromAddress = fromEmail
	} else {
		return nil, fmt.Errorf("gmail: from_email is required")
	}
	if fromName, ok := rawConfig["from_name"].(string); ok {
		protoConfig.DefaultFromName = fromName
	}
	if replyTo, ok := rawConfig["reply_to_email"].(string); ok {
		protoConfig.DefaultReplyTo = replyTo
	}

	oauth2Auth := &emailpb.OAuth2Auth{}
	if delegateEmail, ok := rawConfig["delegate_email"].(string); ok && delegateEmail != "" {
		oauth2Auth.DelegatedEmail = delegateEmail
	} else {
		return nil, fmt.Errorf("gmail: delegate_email is required")
	}
	if keyPath, ok := rawConfig["service_account_key_path"].(string); ok {
		oauth2Auth.ServiceAccountKey = keyPath
	}
	protoConfig.Auth = &emailpb.EmailProviderConfig_Oauth2Auth{Oauth2Auth: oauth2Auth}

	if secretPath, ok := rawConfig["secret_manager_path"].(string); ok && secretPath != "" {
		protoConfig.Settings["secret_manager_path"] = secretPath
	}
	if useSecretManager, ok := rawConfig["use_secret_manager"].(bool); ok && useSecretManager {
		protoConfig.Settings["use_secret_manager"] = "true"
	}
	if timeout, ok := rawConfig["timeout"].(int); ok && timeout > 0 {
		protoConfig.TimeoutSeconds = int32(timeout)
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// GoogleEmailProvider implements Google Gmail API email provider using service account delegation
type GoogleEmailProvider struct {
	enabled       bool
	clientManager *googleclient.GmailClientManager
	timeout       time.Duration
}

// Gmail API response structures (for HTTP API fallback)
type gmailMessage struct {
	ID           string        `json:"id"`
	ThreadID     string        `json:"threadId"`
	LabelIDs     []string      `json:"labelIds"`
	Snippet      string        `json:"snippet"`
	HistoryID    string        `json:"historyId"`
	InternalDate string        `json:"internalDate"`
	Payload      *gmailPayload `json:"payload,omitempty"`
	SizeEstimate int           `json:"sizeEstimate"`
	Raw          string        `json:"raw,omitempty"`
}

type gmailPayload struct {
	PartID   string          `json:"partId"`
	MimeType string          `json:"mimeType"`
	Filename string          `json:"filename"`
	Headers  []gmailHeader   `json:"headers"`
	Body     *gmailBody      `json:"body,omitempty"`
	Parts    []*gmailPayload `json:"parts,omitempty"`
}

type gmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type gmailBody struct {
	AttachmentID string `json:"attachmentId"`
	Size         int    `json:"size"`
	Data         string `json:"data"`
}

type gmailListResponse struct {
	Messages           []gmailMessage `json:"messages"`
	NextPageToken      string         `json:"nextPageToken"`
	ResultSizeEstimate int            `json:"resultSizeEstimate"`
}

// GoogleAttachmentItem represents an email attachment
type GoogleAttachmentItem struct {
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	ContentBytes string `json:"contentBytes"`
}

// NewGoogleEmailProvider creates a new Google Email provider
func NewGoogleEmailProvider() ports.EmailProvider {
	return &GoogleEmailProvider{
		enabled: false,
		timeout: 30 * time.Second,
	}
}

// Name returns the name of this email provider
func (p *GoogleEmailProvider) Name() string {
	return "google_gmail"
}

// Initialize sets up the Google Email provider with configuration
func (p *GoogleEmailProvider) Initialize(config *emailpb.EmailProviderConfig) error {
	ctx := context.Background()

	// Build Gmail config from proto or use defaults from environment
	gmailConfig := googleclient.DefaultGmailConfig()

	// Override with provided proto config values
	if config != nil {
		if config.DefaultFromAddress != "" {
			gmailConfig.FromEmail = config.DefaultFromAddress
		}
		if config.DefaultFromName != "" {
			gmailConfig.FromName = config.DefaultFromName
		}
		if config.DefaultReplyTo != "" {
			gmailConfig.ReplyToEmail = config.DefaultReplyTo
		}

		// Handle OAuth2 auth config
		if oauth2 := config.GetOauth2Auth(); oauth2 != nil {
			if oauth2.DelegatedEmail != "" {
				gmailConfig.DelegateEmail = oauth2.DelegatedEmail
			}
			if oauth2.ServiceAccountKey != "" {
				gmailConfig.ServiceAccountKeyPath = oauth2.ServiceAccountKey
			}
		}

		// Handle settings map for additional config
		if config.Settings != nil {
			if secretPath, exists := config.Settings["secret_manager_path"]; exists && secretPath != "" {
				gmailConfig.SecretManagerPath = secretPath
				gmailConfig.UseSecretManager = true
			}
			if useSecret, exists := config.Settings["use_secret_manager"]; exists {
				gmailConfig.UseSecretManager = useSecret == "true"
			}
		}

		// Handle timeout
		if config.TimeoutSeconds > 0 {
			gmailConfig.Timeout = time.Duration(config.TimeoutSeconds) * time.Second
			p.timeout = gmailConfig.Timeout
		}
	}

	// Create the client manager
	clientManager, err := googleclient.NewGmailClientManager(ctx, gmailConfig)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client manager: %w", err)
	}

	p.clientManager = clientManager
	p.enabled = config.Enabled

	log.Printf("✅ Google Gmail provider initialized successfully")
	return nil
}

// sendEmailLegacy sends an email message using Gmail API with service account delegation (legacy method)
func (p *GoogleEmailProvider) sendEmailLegacy(ctx context.Context, message ports.EmailMessage) error {
	if !p.enabled {
		return fmt.Errorf("Google Email provider is not initialized")
	}

	gmailService := p.clientManager.GetService()
	delegateEmail := p.clientManager.GetDelegateEmail()

	// Use configured from address or message from address
	fromEmail := message.From
	if fromEmail == "" {
		fromEmail = p.clientManager.GetFromEmail()
	}

	fromName := p.clientManager.GetFromName()

	// Determine content type
	contentType := "text/plain"
	body := message.TextBody
	if message.HTMLBody != "" {
		contentType = "text/html"
		body = message.HTMLBody
	}

	// Convert attachments
	var attachments []GoogleAttachmentItem
	for _, att := range message.Attachments {
		attachments = append(attachments, GoogleAttachmentItem{
			Name:         att.Name,
			ContentType:  att.ContentType,
			ContentBytes: base64.StdEncoding.EncodeToString(att.Data),
		})
	}

	// Create Gmail message
	gmailMsg := createGmailMessage(
		fromName,
		fromEmail,
		p.clientManager.GetReplyToEmail(),
		message.To,
		message.CC,
		message.BCC,
		message.Subject,
		body,
		contentType,
		attachments,
	)

	log.Printf("📧 Sending email as user: %s, from address: %s", delegateEmail, fromEmail)

	// Send email using the delegated user
	_, err := gmailService.Users.Messages.Send(delegateEmail, gmailMsg).Do()
	if err != nil {
		log.Printf("❌ Gmail API error: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("✅ Email sent successfully to: %v", message.To)
	return nil
}

// createGmailMessage creates a properly formatted Gmail message with MIME
func createGmailMessage(displayName, from, replyTo string, to, cc, bcc []string, subject, body, contentType string, attachments []GoogleAttachmentItem) *gmail.Message {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add email headers with display name in the From field
	var headers string
	if displayName != "" {
		headers = fmt.Sprintf("From: %s <%s>\r\n", displayName, from)
	} else {
		headers = fmt.Sprintf("From: %s\r\n", from)
	}

	if len(to) > 0 {
		headers += fmt.Sprintf("To: %s\r\n", strings.Join(to, ", "))
	}

	if len(cc) > 0 {
		headers += fmt.Sprintf("Cc: %s\r\n", strings.Join(cc, ", "))
	}

	if replyTo != "" {
		headers += fmt.Sprintf("Reply-To: %s\r\n", replyTo)
	}

	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"

	boundary := writer.Boundary()
	headers += fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)

	buf.WriteString(headers)

	// Add message body part
	bodyPart, _ := writer.CreatePart(map[string][]string{
		"Content-Type": {contentType + "; charset=UTF-8"},
	})
	bodyPart.Write([]byte(body))

	// Add attachments
	for _, attachment := range attachments {
		filePart, _ := writer.CreatePart(map[string][]string{
			"Content-Type":              {attachment.ContentType},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, attachment.Name)},
		})

		decodedBytes, _ := base64.StdEncoding.DecodeString(attachment.ContentBytes)
		filePart.Write(decodedBytes)
	}

	writer.Close()

	// Encode the entire message as base64url
	raw := base64.URLEncoding.EncodeToString(buf.Bytes())
	return &gmail.Message{Raw: raw}
}

// getInboxMessagesLegacy retrieves messages from Gmail inbox (legacy method)
func (p *GoogleEmailProvider) getInboxMessagesLegacy(ctx context.Context, options ports.InboxOptions) ([]ports.EmailMessage, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Google Email provider is not initialized")
	}

	gmailService := p.clientManager.GetService()
	delegateEmail := p.clientManager.GetDelegateEmail()

	// Build list request
	listReq := gmailService.Users.Messages.List(delegateEmail)

	if options.MaxResults > 0 {
		listReq = listReq.MaxResults(int64(options.MaxResults))
	}
	if options.PageToken != "" {
		listReq = listReq.PageToken(options.PageToken)
	}
	if options.Query != "" {
		listReq = listReq.Q(options.Query)
	}
	if len(options.LabelIDs) > 0 {
		listReq = listReq.LabelIds(options.LabelIDs...)
	}

	// Execute list request
	listResp, err := listReq.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Convert to common format
	var messages []ports.EmailMessage
	for _, msg := range listResp.Messages {
		fullMessage, err := p.getMessageLegacy(ctx, msg.Id)
		if err != nil {
			continue // Skip messages that can't be retrieved
		}
		messages = append(messages, *fullMessage)
	}

	return messages, nil
}

// getMessageLegacy retrieves a specific email message by ID (legacy method)
func (p *GoogleEmailProvider) getMessageLegacy(ctx context.Context, messageID string) (*ports.EmailMessage, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Google Email provider is not initialized")
	}

	gmailService := p.clientManager.GetService()
	delegateEmail := p.clientManager.GetDelegateEmail()

	// Get the full message
	gmailMsg, err := gmailService.Users.Messages.Get(delegateEmail, messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Convert to common format
	message := p.convertGmailAPIMessage(gmailMsg)
	return &message, nil
}

// convertGmailAPIMessage converts Gmail API message to common format
func (p *GoogleEmailProvider) convertGmailAPIMessage(gmailMsg *gmail.Message) ports.EmailMessage {
	message := ports.EmailMessage{
		ID:      gmailMsg.Id,
		Headers: make(map[string]string),
	}

	if gmailMsg.Payload != nil {
		// Extract headers
		for _, header := range gmailMsg.Payload.Headers {
			message.Headers[header.Name] = header.Value

			switch strings.ToLower(header.Name) {
			case "from":
				message.From = header.Value
			case "to":
				message.To = strings.Split(header.Value, ",")
				for i, addr := range message.To {
					message.To[i] = strings.TrimSpace(addr)
				}
			case "cc":
				message.CC = strings.Split(header.Value, ",")
				for i, addr := range message.CC {
					message.CC[i] = strings.TrimSpace(addr)
				}
			case "subject":
				message.Subject = header.Value
			}
		}

		// Extract body
		p.extractAPIBody(gmailMsg.Payload, &message)
	}

	// Set timestamp from internal date
	if gmailMsg.InternalDate != 0 {
		message.Timestamp = gmailMsg.InternalDate / 1000 // Convert from milliseconds
	}

	return message
}

// extractAPIBody recursively extracts email body from Gmail API payload
func (p *GoogleEmailProvider) extractAPIBody(payload *gmail.MessagePart, message *ports.EmailMessage) {
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			switch payload.MimeType {
			case "text/plain":
				message.TextBody = string(data)
			case "text/html":
				message.HTMLBody = string(data)
			}
		}
	}

	// Process parts recursively
	for _, part := range payload.Parts {
		p.extractAPIBody(part, message)
	}
}

// IsHealthy checks if the Gmail API service is available
func (p *GoogleEmailProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("Google Email provider is not initialized")
	}

	gmailService := p.clientManager.GetService()
	delegateEmail := p.clientManager.GetDelegateEmail()

	// Test API access by getting profile
	_, err := gmailService.Users.GetProfile(delegateEmail).Do()
	if err != nil {
		return fmt.Errorf("Gmail API health check failed: %w", err)
	}

	return nil
}

// Close cleans up Google Email provider resources
func (p *GoogleEmailProvider) Close() error {
	p.enabled = false
	if p.clientManager != nil {
		return p.clientManager.Close()
	}
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *GoogleEmailProvider) IsEnabled() bool {
	return p.enabled
}

// GetCapabilities returns the capabilities supported by this provider
func (p *GoogleEmailProvider) GetCapabilities() []emailpb.EmailCapability {
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
func (p *GoogleEmailProvider) GetProviderType() emailpb.EmailProviderType {
	return emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_OAUTH
}

// SendEmail sends an email using protobuf request/response types (implements ports.EmailProvider interface)
func (p *GoogleEmailProvider) SendEmail(ctx context.Context, req *emailpb.SendEmailRequest) (*emailpb.SendEmailResponse, error) {
	if !p.enabled {
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Email provider is not initialized",
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
func (p *GoogleEmailProvider) SendBatchEmails(ctx context.Context, req *emailpb.SendBatchEmailsRequest) (*emailpb.SendBatchEmailsResponse, error) {
	if !p.enabled {
		return &emailpb.SendBatchEmailsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Email provider is not initialized",
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
func (p *GoogleEmailProvider) GetInboxMessages(ctx context.Context, req *emailpb.GetInboxMessagesRequest) (*emailpb.GetInboxMessagesResponse, error) {
	if !p.enabled {
		return &emailpb.GetInboxMessagesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Email provider is not initialized",
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
		protoMessages = append(protoMessages, convertToProtoMessage(msg))
	}

	return &emailpb.GetInboxMessagesResponse{
		Success: true,
		Data:    protoMessages,
	}, nil
}

// GetMessage retrieves a specific message by ID using protobuf types (implements ports.EmailProvider interface)
func (p *GoogleEmailProvider) GetMessage(ctx context.Context, req *emailpb.GetMessageRequest) (*emailpb.GetMessageResponse, error) {
	if !p.enabled {
		return &emailpb.GetMessageResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Email provider is not initialized",
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
		Data:    []*emailpb.EmailMessage{convertToProtoMessage(*message)},
	}, nil
}

// convertToProtoMessage converts internal EmailMessage to protobuf EmailMessage
func convertToProtoMessage(msg ports.EmailMessage) *emailpb.EmailMessage {
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

// Legacy HTTP-based methods for backward compatibility

// convertGmailMessage converts HTTP API Gmail message to common format (legacy)
func (p *GoogleEmailProvider) convertGmailMessage(gmailMsg gmailMessage) ports.EmailMessage {
	message := ports.EmailMessage{
		ID:      gmailMsg.ID,
		Headers: make(map[string]string),
	}

	if gmailMsg.Payload != nil {
		for _, header := range gmailMsg.Payload.Headers {
			message.Headers[header.Name] = header.Value

			switch strings.ToLower(header.Name) {
			case "from":
				message.From = header.Value
			case "to":
				message.To = strings.Split(header.Value, ",")
				for i, addr := range message.To {
					message.To[i] = strings.TrimSpace(addr)
				}
			case "cc":
				message.CC = strings.Split(header.Value, ",")
				for i, addr := range message.CC {
					message.CC[i] = strings.TrimSpace(addr)
				}
			case "subject":
				message.Subject = header.Value
			}
		}

		p.extractBody(gmailMsg.Payload, &message)
	}

	if gmailMsg.InternalDate != "" {
		if timestamp, err := strconv.ParseInt(gmailMsg.InternalDate, 10, 64); err == nil {
			message.Timestamp = timestamp / 1000
		} else {
			message.Timestamp = time.Now().Unix()
		}
	}

	return message
}

// extractBody recursively extracts email body from HTTP API Gmail payload (legacy)
func (p *GoogleEmailProvider) extractBody(payload *gmailPayload, message *ports.EmailMessage) {
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			switch payload.MimeType {
			case "text/plain":
				message.TextBody = string(data)
			case "text/html":
				message.HTMLBody = string(data)
			}
		}
	}

	for _, part := range payload.Parts {
		p.extractBody(part, message)
	}
}

// HTTP fallback methods (unused but kept for reference)
var _ = url.Values{}
var _ = http.NewRequest
var _ = io.ReadAll
var _ = json.NewDecoder
