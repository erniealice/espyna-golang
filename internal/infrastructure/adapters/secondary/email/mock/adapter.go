//go:build mock_email

package mock

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterEmailProvider(
		"mock_email",
		func() ports.EmailProvider {
			return NewMockEmailProvider()
		},
		transformConfig,
	)
	registry.RegisterEmailBuildFromEnv("mock_email", buildFromEnv)
}

// buildFromEnv creates and initializes a mock email provider.
func buildFromEnv() (ports.EmailProvider, error) {
	protoConfig := &emailpb.EmailProviderConfig{
		ProviderId: "mock",
		Enabled:    true,
	}
	p := NewMockEmailProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("mock_email: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to mock email proto config.
func transformConfig(rawConfig map[string]any) (*emailpb.EmailProviderConfig, error) {
	return &emailpb.EmailProviderConfig{
		ProviderId: "mock",
		Enabled:    true,
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// MockEmailProvider implements a mock email provider for testing
type MockEmailProvider struct {
	enabled          bool
	messages         []ports.EmailMessage
	sentMessages     []ports.EmailMessage
	messageIDCounter int64
	mu               sync.RWMutex
	shouldFail       bool
	failureMessage   string
}

// NewMockEmailProvider creates a new mock email provider
func NewMockEmailProvider() ports.EmailProvider {
	return &MockEmailProvider{
		enabled:          false,
		messages:         make([]ports.EmailMessage, 0),
		sentMessages:     make([]ports.EmailMessage, 0),
		messageIDCounter: 1,
	}
}

// Name returns the name of this email provider
func (p *MockEmailProvider) Name() string {
	return "mock"
}

// Initialize sets up the mock email provider with configuration
func (p *MockEmailProvider) Initialize(config *emailpb.EmailProviderConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Optional: configure failure mode for testing from settings map
	if config.Settings != nil {
		if failStr, exists := config.Settings["should_fail"]; exists {
			p.shouldFail = failStr == "true"
		}
		if failureMessage, exists := config.Settings["failure_message"]; exists {
			p.failureMessage = failureMessage
		}
		if initWithData, exists := config.Settings["init_with_mock_data"]; exists && initWithData == "true" {
			p.initializeMockData()
		}
	}

	p.enabled = config.Enabled
	log.Println("✅ Mock email provider initialized successfully")
	return nil
}

// initializeMockData creates some sample email messages for testing
func (p *MockEmailProvider) initializeMockData() {
	now := time.Now().Unix()

	mockMessages := []ports.EmailMessage{
		{
			ID:        "mock-1",
			From:      "sender1@example.com",
			To:        []string{"test@example.com"},
			Subject:   "Welcome to our service",
			TextBody:  "Welcome! Thank you for signing up for our service.",
			HTMLBody:  "<h1>Welcome!</h1><p>Thank you for signing up for our service.</p>",
			Headers:   map[string]string{"Content-Type": "text/html"},
			Timestamp: now - 3600, // 1 hour ago
		},
		{
			ID:        "mock-2",
			From:      "support@example.com",
			To:        []string{"test@example.com"},
			CC:        []string{"manager@example.com"},
			Subject:   "Important notification",
			TextBody:  "This is an important notification about your account.",
			Timestamp: now - 7200, // 2 hours ago
		},
		{
			ID:       "mock-3",
			From:     "newsletter@example.com",
			To:       []string{"test@example.com"},
			Subject:  "Weekly Newsletter - Issue #123",
			HTMLBody: "<h2>Weekly Newsletter</h2><p>Here are this week's highlights...</p>",
			Attachments: []ports.EmailAttachment{
				{
					Name:        "newsletter.pdf",
					ContentType: "application/pdf",
					Data:        []byte("mock pdf content"),
					Size:        1024,
				},
			},
			Timestamp: now - 86400, // 1 day ago
		},
	}

	p.messages = append(p.messages, mockMessages...)
	p.messageIDCounter = int64(len(mockMessages) + 1)
}

// sendEmailLegacy simulates sending an email message (legacy method)
func (p *MockEmailProvider) sendEmailLegacy(ctx context.Context, message ports.EmailMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled {
		return fmt.Errorf("mock email provider is not initialized")
	}

	if p.shouldFail {
		failMsg := p.failureMessage
		if failMsg == "" {
			failMsg = "mock email provider configured to fail"
		}
		return fmt.Errorf("%s", failMsg)
	}

	// Simulate message processing
	sentMessage := message
	sentMessage.ID = fmt.Sprintf("mock-sent-%d", p.messageIDCounter)
	sentMessage.Timestamp = time.Now().Unix()

	// Add some mock headers
	if sentMessage.Headers == nil {
		sentMessage.Headers = make(map[string]string)
	}
	sentMessage.Headers["Message-ID"] = fmt.Sprintf("<%s@mock.example.com>", sentMessage.ID)
	sentMessage.Headers["X-Mock-Provider"] = "true"

	p.sentMessages = append(p.sentMessages, sentMessage)
	p.messageIDCounter++

	log.Printf("📧 Mock email sent: ID=%s, To=%v, Subject=%s",
		sentMessage.ID, sentMessage.To, sentMessage.Subject)

	return nil
}

// getInboxMessagesLegacy returns mock inbox messages (legacy method)
func (p *MockEmailProvider) getInboxMessagesLegacy(ctx context.Context, options ports.InboxOptions) ([]ports.EmailMessage, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return nil, fmt.Errorf("mock email provider is not initialized")
	}

	if p.shouldFail {
		failMsg := p.failureMessage
		if failMsg == "" {
			failMsg = "mock email provider configured to fail"
		}
		return nil, fmt.Errorf(failMsg)
	}

	// Apply filtering and pagination
	filteredMessages := p.messages

	// Simple query filtering (case-insensitive substring search)
	if options.Query != "" {
		var filtered []ports.EmailMessage
		for _, msg := range filteredMessages {
			if p.messageMatchesQuery(msg, options.Query) {
				filtered = append(filtered, msg)
			}
		}
		filteredMessages = filtered
	}

	// Apply result limit
	maxResults := options.MaxResults
	if maxResults <= 0 || maxResults > len(filteredMessages) {
		maxResults = len(filteredMessages)
	}

	result := make([]ports.EmailMessage, maxResults)
	copy(result, filteredMessages[:maxResults])

	log.Printf("📥 Mock inbox messages retrieved: count=%d, total=%d", len(result), len(p.messages))

	return result, nil
}

// messageMatchesQuery performs simple query matching on message fields
func (p *MockEmailProvider) messageMatchesQuery(message ports.EmailMessage, query string) bool {
	query = fmt.Sprintf(query)

	// Check subject, from, to, and body
	if fmt.Sprintf(message.Subject) == query ||
		fmt.Sprintf(message.From) == query ||
		fmt.Sprintf(message.TextBody) == query ||
		fmt.Sprintf(message.HTMLBody) == query {
		return true
	}

	// Check to recipients
	for _, to := range message.To {
		if fmt.Sprintf(to) == query {
			return true
		}
	}

	// Check CC recipients
	for _, cc := range message.CC {
		if fmt.Sprintf(cc) == query {
			return true
		}
	}

	return false
}

// getMessageLegacy retrieves a specific email message by ID (legacy method)
func (p *MockEmailProvider) getMessageLegacy(ctx context.Context, messageID string) (*ports.EmailMessage, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return nil, fmt.Errorf("mock email provider is not initialized")
	}

	if p.shouldFail {
		failMsg := p.failureMessage
		if failMsg == "" {
			failMsg = "mock email provider configured to fail"
		}
		return nil, fmt.Errorf(failMsg)
	}

	// Search in inbox messages
	for _, message := range p.messages {
		if message.ID == messageID {
			log.Printf("📬 Mock message retrieved: ID=%s, Subject=%s", message.ID, message.Subject)
			return &message, nil
		}
	}

	// Search in sent messages
	for _, message := range p.sentMessages {
		if message.ID == messageID {
			log.Printf("📤 Mock sent message retrieved: ID=%s, Subject=%s", message.ID, message.Subject)
			return &message, nil
		}
	}

	return nil, fmt.Errorf("message not found: %s", messageID)
}

// IsHealthy always returns healthy for mock provider
func (p *MockEmailProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("mock email provider is not initialized")
	}

	if p.shouldFail {
		failMsg := p.failureMessage
		if failMsg == "" {
			failMsg = "mock email provider configured to fail health check"
		}
		return fmt.Errorf(failMsg)
	}

	return nil
}

// Close cleans up mock provider resources
func (p *MockEmailProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false
	p.messages = nil
	p.sentMessages = nil

	log.Println("✅ Mock email provider closed successfully")
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *MockEmailProvider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// GetCapabilities returns the capabilities supported by this provider
func (p *MockEmailProvider) GetCapabilities() []emailpb.EmailCapability {
	return []emailpb.EmailCapability{
		emailpb.EmailCapability_EMAIL_CAPABILITY_SEND,
		emailpb.EmailCapability_EMAIL_CAPABILITY_ATTACHMENTS,
		emailpb.EmailCapability_EMAIL_CAPABILITY_READ_INBOX,
	}
}

// GetProviderType returns the provider type
func (p *MockEmailProvider) GetProviderType() emailpb.EmailProviderType {
	return emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_UNSPECIFIED
}

// SendEmail sends an email using protobuf request/response types (implements ports.EmailProvider interface)
// Uses the Data wrapper pattern for consistent API structure
func (p *MockEmailProvider) SendEmail(ctx context.Context, req *emailpb.SendEmailRequest) (*emailpb.SendEmailResponse, error) {
	// Extract data from request (Data wrapper pattern)
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
// Uses the Data wrapper pattern for consistent API structure
func (p *MockEmailProvider) SendBatchEmails(ctx context.Context, req *emailpb.SendBatchEmailsRequest) (*emailpb.SendBatchEmailsResponse, error) {
	// Extract data from request (Data wrapper pattern)
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
		emailReq := &emailpb.SendEmailRequest{Data: emailData}
		resp, _ := p.SendEmail(ctx, emailReq)
		if resp.Success {
			successCount++
			results = append(results, &emailpb.EmailResult{
				MessageId: fmt.Sprintf("mock-batch-%d", successCount),
			})
		} else {
			failCount++
			if data.FailFast {
				break
			}
		}
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
// Uses the Data wrapper pattern for consistent API structure
func (p *MockEmailProvider) GetInboxMessages(ctx context.Context, req *emailpb.GetInboxMessagesRequest) (*emailpb.GetInboxMessagesResponse, error) {
	// Extract data from request (Data wrapper pattern)
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
				Code:    "INBOX_READ_FAILED",
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
// Uses the Data wrapper pattern for consistent API structure
func (p *MockEmailProvider) GetMessage(ctx context.Context, req *emailpb.GetMessageRequest) (*emailpb.GetMessageResponse, error) {
	// Extract data from request (Data wrapper pattern)
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
				Code:    "MESSAGE_NOT_FOUND",
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
func (p *MockEmailProvider) convertToProtoMessage(msg ports.EmailMessage) *emailpb.EmailMessage {
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

// Mock-specific methods for testing

// GetSentMessages returns messages sent through this mock provider
func (p *MockEmailProvider) GetSentMessages() []ports.EmailMessage {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]ports.EmailMessage, len(p.sentMessages))
	copy(result, p.sentMessages)
	return result
}

// AddMockMessage adds a mock message to the inbox for testing
func (p *MockEmailProvider) AddMockMessage(message ports.EmailMessage) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if message.ID == "" {
		message.ID = fmt.Sprintf("mock-added-%d", p.messageIDCounter)
		p.messageIDCounter++
	}

	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}

	p.messages = append(p.messages, message)
	log.Printf("📬 Mock message added: ID=%s, Subject=%s", message.ID, message.Subject)
}

// ClearMessages clears all mock messages for testing
func (p *MockEmailProvider) ClearMessages() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.messages = make([]ports.EmailMessage, 0)
	p.sentMessages = make([]ports.EmailMessage, 0)
	p.messageIDCounter = 1

	log.Println("🧹 Mock messages cleared")
}

// SetFailureMode configures the mock provider to simulate failures
func (p *MockEmailProvider) SetFailureMode(shouldFail bool, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.shouldFail = shouldFail
	p.failureMessage = message

	log.Printf("⚙️ Mock failure mode: enabled=%v, message=%s", shouldFail, message)
}

// GetMessageCount returns the count of inbox and sent messages
func (p *MockEmailProvider) GetMessageCount() (inbox int, sent int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.messages), len(p.sentMessages)
}
