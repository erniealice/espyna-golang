package consumer

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	emailpb "leapfor.xyz/esqyma/golang/v1/integration/email"
)

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Email Adapter

Provides direct access to email operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY email provider (Gmail, Microsoft, SendGrid, Mock)
based on your CONFIG_EMAIL_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewEmailAdapterFromContainer(container)

	// Send email
	resp, err := adapter.SendEmail(ctx, consumer.EmailMessage{
	    To:      []string{"recipient@example.com"},
	    Subject: "Hello",
	    TextBody: "World",
	})

	// Get inbox messages
	messages, err := adapter.GetInboxMessages(ctx, &consumer.InboxOptions{
	    MaxResults: 10,
	})
*/

// EmailAdapter provides technology-agnostic access to email services.
// It wraps the EmailProvider interface and works with Gmail, Microsoft, SendGrid, etc.
type EmailAdapter struct {
	provider  ports.EmailProvider
	container *Container
}

// NewEmailAdapterFromContainer creates an EmailAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's provider.
func NewEmailAdapterFromContainer(container *Container) *EmailAdapter {
	if container == nil {
		return nil
	}

	provider := container.GetEmailProvider()
	if provider == nil {
		return nil
	}

	return &EmailAdapter{
		provider:  provider,
		container: container,
	}
}

// Close closes the email adapter.
// Note: If created from container, this does NOT close the container.
func (a *EmailAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetProvider returns the underlying EmailProvider for advanced operations.
func (a *EmailAdapter) GetProvider() ports.EmailProvider {
	return a.provider
}

// Name returns the name of the underlying email provider (e.g., "gmail", "microsoft", "mock")
func (a *EmailAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the email provider is enabled
func (a *EmailAdapter) IsEnabled() bool {
	return a.provider != nil && a.provider.IsEnabled()
}

// --- Email Operations ---

// SendEmail sends an email message using the simplified EmailMessage type.
// Returns the message ID on success.
func (a *EmailAdapter) SendEmail(ctx context.Context, msg ports.EmailMessage) (*emailpb.SendEmailResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("email provider not initialized")
	}

	req := msg.ToProtoRequest()
	return a.provider.SendEmail(ctx, req)
}

// SendEmailProto sends an email using the protobuf request type directly.
// Use this for full control over all email parameters.
func (a *EmailAdapter) SendEmailProto(ctx context.Context, req *emailpb.SendEmailRequest) (*emailpb.SendEmailResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("email provider not initialized")
	}
	return a.provider.SendEmail(ctx, req)
}

// SendBatchEmails sends multiple emails in a batch.
func (a *EmailAdapter) SendBatchEmails(ctx context.Context, req *emailpb.SendBatchEmailsRequest) (*emailpb.SendBatchEmailsResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("email provider not initialized")
	}
	return a.provider.SendBatchEmails(ctx, req)
}

// GetInboxMessages retrieves messages from inbox with optional filtering.
func (a *EmailAdapter) GetInboxMessages(ctx context.Context, opts *ports.InboxOptions) (*emailpb.GetInboxMessagesResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("email provider not initialized")
	}

	var req *emailpb.GetInboxMessagesRequest
	if opts != nil {
		req = opts.ToProtoRequest()
	} else {
		req = &emailpb.GetInboxMessagesRequest{
			Data: &emailpb.InboxQueryData{
				MaxResults: 20,
			},
		}
	}

	return a.provider.GetInboxMessages(ctx, req)
}

// GetMessage retrieves a specific email message by ID.
func (a *EmailAdapter) GetMessage(ctx context.Context, messageID string) (*emailpb.GetMessageResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("email provider not initialized")
	}

	req := &emailpb.GetMessageRequest{
		Data: &emailpb.MessageLookupData{
			MessageId: messageID,
		},
	}

	return a.provider.GetMessage(ctx, req)
}

// IsHealthy checks if the email provider is healthy and available.
func (a *EmailAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("email provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// GetCapabilities returns the capabilities supported by the email provider.
func (a *EmailAdapter) GetCapabilities() []emailpb.EmailCapability {
	if a.provider == nil {
		return nil
	}
	return a.provider.GetCapabilities()
}

// GetProviderType returns the provider type.
func (a *EmailAdapter) GetProviderType() emailpb.EmailProviderType {
	if a.provider == nil {
		return emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_UNSPECIFIED
	}
	return a.provider.GetProviderType()
}

// --- Convenience Methods ---

// SendSimpleEmail sends a plain text email with minimal parameters.
func (a *EmailAdapter) SendSimpleEmail(ctx context.Context, to []string, subject, body string) (*emailpb.SendEmailResponse, error) {
	return a.SendEmail(ctx, ports.EmailMessage{
		To:       to,
		Subject:  subject,
		TextBody: body,
	})
}

// SendHTMLEmail sends an HTML email with optional plain text fallback.
func (a *EmailAdapter) SendHTMLEmail(ctx context.Context, to []string, subject, htmlBody, textBody string) (*emailpb.SendEmailResponse, error) {
	return a.SendEmail(ctx, ports.EmailMessage{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
}
