package consumer

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// captureEmailProvider is a minimal EmailProvider test double that records the
// last SendEmailRequest it received so the test can assert what actually
// reached the provider boundary.
type captureEmailProvider struct {
	last *emailpb.SendEmailRequest
}

func (p *captureEmailProvider) Name() string { return "capture" }
func (p *captureEmailProvider) Initialize(*emailpb.EmailProviderConfig) error {
	return nil
}
func (p *captureEmailProvider) SendEmail(_ context.Context, req *emailpb.SendEmailRequest) (*emailpb.SendEmailResponse, error) {
	p.last = req
	return &emailpb.SendEmailResponse{}, nil
}
func (p *captureEmailProvider) SendBatchEmails(context.Context, *emailpb.SendBatchEmailsRequest) (*emailpb.SendBatchEmailsResponse, error) {
	return &emailpb.SendBatchEmailsResponse{}, nil
}
func (p *captureEmailProvider) GetInboxMessages(context.Context, *emailpb.GetInboxMessagesRequest) (*emailpb.GetInboxMessagesResponse, error) {
	return &emailpb.GetInboxMessagesResponse{}, nil
}
func (p *captureEmailProvider) GetMessage(context.Context, *emailpb.GetMessageRequest) (*emailpb.GetMessageResponse, error) {
	return &emailpb.GetMessageResponse{}, nil
}
func (p *captureEmailProvider) IsHealthy(context.Context) error { return nil }
func (p *captureEmailProvider) Close() error                    { return nil }
func (p *captureEmailProvider) IsEnabled() bool                 { return true }
func (p *captureEmailProvider) GetCapabilities() []emailpb.EmailCapability {
	return nil
}
func (p *captureEmailProvider) GetProviderType() emailpb.EmailProviderType {
	return emailpb.EmailProviderType_EMAIL_PROVIDER_TYPE_UNSPECIFIED
}

var _ ports.EmailProvider = (*captureEmailProvider)(nil)

// TestSendHTMLEmailWithAttachment_DeliversAttachment is the regression guard for
// the composition-layer bug where the email closure accepted attachment bytes
// but called SendHTMLEmail (no attachment slot), silently dropping them. The
// typed adapter method must route the bytes and name through to the provider.
func TestSendHTMLEmailWithAttachment_DeliversAttachment(t *testing.T) {
	cap := &captureEmailProvider{}
	adapter := &EmailAdapter{provider: cap}

	wantName := "report-card.pdf"
	wantData := []byte("%PDF-1.4 fake attachment bytes")

	if _, err := adapter.SendHTMLEmailWithAttachment(
		context.Background(),
		[]string{"parent@example.com"},
		"Your report card",
		"<p>See attached</p>",
		"See attached",
		wantName,
		wantData,
	); err != nil {
		t.Fatalf("SendHTMLEmailWithAttachment returned error: %v", err)
	}

	if cap.last == nil || cap.last.Data == nil {
		t.Fatal("provider received no request/data")
	}
	atts := cap.last.Data.Attachments
	if len(atts) != 1 {
		t.Fatalf("expected exactly 1 attachment to reach the provider, got %d", len(atts))
	}
	if atts[0].Name != wantName {
		t.Errorf("attachment name = %q, want %q", atts[0].Name, wantName)
	}
	if string(atts[0].Data) != string(wantData) {
		t.Errorf("attachment data = %q, want %q", string(atts[0].Data), string(wantData))
	}
}

// TestSendHTMLEmailWithAttachment_NoAttachmentWhenEmpty confirms the method is a
// clean superset of SendHTMLEmail: empty name+data adds no attachment.
func TestSendHTMLEmailWithAttachment_NoAttachmentWhenEmpty(t *testing.T) {
	cap := &captureEmailProvider{}
	adapter := &EmailAdapter{provider: cap}

	if _, err := adapter.SendHTMLEmailWithAttachment(
		context.Background(),
		[]string{"parent@example.com"},
		"Plain",
		"<p>body</p>",
		"body",
		"",
		nil,
	); err != nil {
		t.Fatalf("SendHTMLEmailWithAttachment returned error: %v", err)
	}

	if cap.last == nil || cap.last.Data == nil {
		t.Fatal("provider received no request/data")
	}
	if got := len(cap.last.Data.Attachments); got != 0 {
		t.Errorf("expected 0 attachments, got %d", got)
	}
}
