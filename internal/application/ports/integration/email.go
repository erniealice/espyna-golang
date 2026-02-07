package integration

import (
	"context"

	emailpb "leapfor.xyz/esqyma/golang/v1/integration/email"
)

// EmailProvider defines the contract for email providers
// This interface abstracts email services like Gmail, SendGrid, SMTP, etc.
// following the hexagonal architecture pattern.
type EmailProvider interface {
	// Name returns the name of the email provider (e.g., "gmail", "sendgrid", "smtp")
	Name() string

	// Initialize sets up the email provider with the given configuration
	Initialize(config *emailpb.EmailProviderConfig) error

	// SendEmail sends an email message
	SendEmail(ctx context.Context, req *emailpb.SendEmailRequest) (*emailpb.SendEmailResponse, error)

	// SendBatchEmails sends multiple emails in a batch
	SendBatchEmails(ctx context.Context, req *emailpb.SendBatchEmailsRequest) (*emailpb.SendBatchEmailsResponse, error)

	// GetInboxMessages retrieves messages from inbox
	GetInboxMessages(ctx context.Context, req *emailpb.GetInboxMessagesRequest) (*emailpb.GetInboxMessagesResponse, error)

	// GetMessage retrieves a specific email message by ID
	GetMessage(ctx context.Context, req *emailpb.GetMessageRequest) (*emailpb.GetMessageResponse, error)

	// IsHealthy checks if the email service is available
	IsHealthy(ctx context.Context) error

	// Close cleans up email provider resources
	Close() error

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool

	// GetCapabilities returns the capabilities supported by this provider
	GetCapabilities() []emailpb.EmailCapability

	// GetProviderType returns the provider type
	GetProviderType() emailpb.EmailProviderType
}

// EmailMessage represents a simplified email message for convenience
// This is used by adapters that need a simpler interface
type EmailMessage struct {
	// Unique identifier for the message
	ID string

	// From address
	From string

	// To recipients
	To []string

	// CC recipients
	CC []string

	// BCC recipients
	BCC []string

	// Subject line
	Subject string

	// Plain text body
	TextBody string

	// HTML body
	HTMLBody string

	// Email attachments
	Attachments []EmailAttachment

	// Custom headers
	Headers map[string]string

	// Timestamp (Unix seconds)
	Timestamp int64
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	// Attachment ID
	ID string

	// Filename
	Name string

	// MIME content type
	ContentType string

	// File size in bytes
	Size int64

	// File content bytes
	Data []byte

	// Inline content ID (for embedded images)
	ContentID string

	// Whether this is an inline attachment
	IsInline bool
}

// InboxOptions provides options for retrieving inbox messages
type InboxOptions struct {
	// Maximum results to return
	MaxResults int

	// Page token for pagination
	PageToken string

	// Gmail search query
	Query string

	// Label/folder IDs to filter by
	LabelIDs []string

	// Whether to include spam/trash
	IncludeSpamTrash bool
}

// ToProtoRequest converts InboxOptions to protobuf GetInboxMessagesRequest
func (o *InboxOptions) ToProtoRequest() *emailpb.GetInboxMessagesRequest {
	return &emailpb.GetInboxMessagesRequest{
		Data: &emailpb.InboxQueryData{
			MaxResults: int32(o.MaxResults),
			PageToken:  o.PageToken,
			Query:      o.Query,
			LabelIds:   o.LabelIDs,
		},
	}
}

// ToProtoRequest converts EmailMessage to protobuf SendEmailRequest
func (m *EmailMessage) ToProtoRequest() *emailpb.SendEmailRequest {
	data := &emailpb.EmailData{
		Subject:  m.Subject,
		TextBody: m.TextBody,
		HtmlBody: m.HTMLBody,
		Headers:  m.Headers,
	}

	// Set From
	if m.From != "" {
		data.From = &emailpb.EmailAddress{Address: m.From}
	}

	// Set To recipients
	for _, addr := range m.To {
		data.To = append(data.To, &emailpb.EmailAddress{Address: addr})
	}

	// Set CC recipients
	for _, addr := range m.CC {
		data.Cc = append(data.Cc, &emailpb.EmailAddress{Address: addr})
	}

	// Set BCC recipients
	for _, addr := range m.BCC {
		data.Bcc = append(data.Bcc, &emailpb.EmailAddress{Address: addr})
	}

	// Set attachments
	for _, att := range m.Attachments {
		data.Attachments = append(data.Attachments, &emailpb.EmailAttachment{
			Id:          att.ID,
			Name:        att.Name,
			ContentType: att.ContentType,
			Size:        att.Size,
			Data:        att.Data,
			ContentId:   att.ContentID,
			IsInline:    att.IsInline,
		})
	}

	return &emailpb.SendEmailRequest{
		Data: data,
	}
}

// FromProtoMessage converts protobuf EmailMessage to EmailMessage
func FromProtoMessage(msg *emailpb.EmailMessage) EmailMessage {
	result := EmailMessage{
		ID:       msg.Id,
		Subject:  msg.Subject,
		TextBody: msg.TextBody,
		HTMLBody: msg.HtmlBody,
		Headers:  msg.Headers,
	}

	if msg.From != nil {
		result.From = msg.From.Address
	}

	for _, addr := range msg.To {
		result.To = append(result.To, addr.Address)
	}

	for _, addr := range msg.Cc {
		result.CC = append(result.CC, addr.Address)
	}

	for _, addr := range msg.Bcc {
		result.BCC = append(result.BCC, addr.Address)
	}

	for _, att := range msg.Attachments {
		result.Attachments = append(result.Attachments, EmailAttachment{
			ID:          att.Id,
			Name:        att.Name,
			ContentType: att.ContentType,
			Size:        att.Size,
			Data:        att.Data,
			ContentID:   att.ContentId,
			IsInline:    att.IsInline,
		})
	}

	if msg.Timestamp != nil {
		result.Timestamp = msg.Timestamp.Seconds
	}

	return result
}
