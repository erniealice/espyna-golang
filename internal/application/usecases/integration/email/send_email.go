package email

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// SendEmailRepositories groups all repository dependencies
type SendEmailRepositories struct {
	// No repositories needed for external email provider integration
}

// SendEmailServices groups all service dependencies
type SendEmailServices struct {
	Provider ports.EmailProvider
}

// SendEmailUseCase handles sending emails
type SendEmailUseCase struct {
	repositories SendEmailRepositories
	services     SendEmailServices
}

// NewSendEmailUseCase creates a new SendEmailUseCase
func NewSendEmailUseCase(
	repositories SendEmailRepositories,
	services SendEmailServices,
) *SendEmailUseCase {
	return &SendEmailUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute sends an email using the configured provider
func (uc *SendEmailUseCase) Execute(ctx context.Context, req *emailpb.SendEmailRequest) (*emailpb.SendEmailResponse, error) {
	log.Printf("📧 [SendEmail] Execute called")

	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		log.Printf("❌ [SendEmail] Provider unavailable (nil=%v, enabled=%v)",
			uc.services.Provider == nil, uc.services.Provider != nil && uc.services.Provider.IsEnabled())
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Email provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		log.Printf("❌ [SendEmail] req.Data is nil")
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📧 [SendEmail] Data: to=%d, subject=%q, template_html=%d chars, template_values=%d",
		len(req.Data.To), req.Data.Subject, len(req.Data.TemplateHtml), len(req.Data.TemplateValues))

	// Validate request
	if len(req.Data.To) == 0 {
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "At least one recipient is required",
			},
		}, nil
	}

	if req.Data.Subject == "" {
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Subject is required",
			},
		}, nil
	}

	// Apply template replacement BEFORE body validation
	// This converts template_html → html_body
	uc.applyTemplateValues(req.Data)

	if req.Data.TextBody == "" && req.Data.HtmlBody == "" {
		log.Printf("❌ Email validation failed: no body content (text_body=%q, html_body=%q, template_html=%q)",
			req.Data.TextBody, req.Data.HtmlBody, req.Data.TemplateHtml)
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Either text_body, html_body, or template_html is required",
			},
		}, nil
	}

	// Build recipient list for logging
	var toAddrs []string
	for _, addr := range req.Data.To {
		toAddrs = append(toAddrs, addr.Address)
	}

	log.Printf("📧 Sending email: subject=%s, to=%v", req.Data.Subject, toAddrs)

	// Send via provider
	response, err := uc.services.Provider.SendEmail(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to send email: %v", err)
		return &emailpb.SendEmailResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SEND_FAILED",
				Message: fmt.Sprintf("Failed to send email: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("✅ Email sent successfully: message_id=%s", response.Data[0].MessageId)
	} else if !response.Success && response.Error != nil {
		log.Printf("❌ Email send failed: %s", response.Error.Message)
	}

	return response, nil
}

// applyTemplateValues applies template_values to template_html, subject, and text_body
// using {{key}} → value replacement pattern.
// If template_html is provided, it takes precedence over html_body.
func (uc *SendEmailUseCase) applyTemplateValues(data *emailpb.EmailData) {
	if data == nil {
		return
	}

	// If template_html is provided, use it as the base for html_body
	// This MUST happen before checking template_values
	if data.TemplateHtml != "" {
		log.Printf("📝 Using template_html as html_body base (%d chars)", len(data.TemplateHtml))
		data.HtmlBody = data.TemplateHtml
	}

	// Early return if no template values to apply
	if len(data.TemplateValues) == 0 {
		return
	}

	// Apply replacements to all templated fields
	for key, value := range data.TemplateValues {
		placeholder := "{{" + key + "}}"

		// Replace in subject
		if strings.Contains(data.Subject, placeholder) {
			data.Subject = strings.ReplaceAll(data.Subject, placeholder, value)
		}

		// Replace in text_body
		if strings.Contains(data.TextBody, placeholder) {
			data.TextBody = strings.ReplaceAll(data.TextBody, placeholder, value)
		}

		// Replace in html_body (which may come from template_html)
		if strings.Contains(data.HtmlBody, placeholder) {
			data.HtmlBody = strings.ReplaceAll(data.HtmlBody, placeholder, value)
		}
	}

	log.Printf("📝 Applied %d template values", len(data.TemplateValues))
}
