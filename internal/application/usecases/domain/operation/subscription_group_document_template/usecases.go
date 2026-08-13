// Package subscription_group_document_template holds the section-group template-binding
// use cases: standard CRUD/List, the applicability resolver (FindApplicable), and
// the controlled publish transaction. It is the sibling binding family for
// subscription group + plan-scoped document rendering.
//
// One file per operation (create.go / read.go / update.go / delete.go /
// list.go / find_applicable.go / publish.go). This file carries only the shared
// wiring (Repositories / Services / UseCases / NewUseCases) + the requireRequest
// helper (S-5).
package subscription_group_document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

// Repositories groups the primary repository dependency.
type Repositories struct {
	DocumentTemplate                  documenttemplatepb.DocumentTemplateDomainServiceServer
	SubscriptionGroupDocumentTemplate pb.SubscriptionGroupDocumentTemplateDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases aggregates the binding CRUD + resolver + publish use cases.
type UseCases struct {
	CreateUploadPair                                *CreateUploadPairUseCase
	DeleteDraftPair                                 *DeleteDraftPairUseCase
	CreateSubscriptionGroupDocumentTemplate         *CreateUseCase
	ReadSubscriptionGroupDocumentTemplate           *ReadUseCase
	UpdateSubscriptionGroupDocumentTemplate         *UpdateUseCase
	DeleteSubscriptionGroupDocumentTemplate         *DeleteUseCase
	ListSubscriptionGroupDocumentTemplates          *ListUseCase
	FindApplicableSubscriptionGroupDocumentTemplate *FindApplicableUseCase
	PublishSubscriptionGroupDocumentTemplate        *PublishUseCase
}

// NewUseCases wires the binding use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		CreateUploadPair:                                NewCreateUploadPairUseCase(CreateUploadPairRepositories{DocumentTemplate: r.DocumentTemplate, SubscriptionGroupDocumentTemplate: r.SubscriptionGroupDocumentTemplate}, s),
		DeleteDraftPair:                                 NewDeleteDraftPairUseCase(r.SubscriptionGroupDocumentTemplate, s),
		CreateSubscriptionGroupDocumentTemplate:         &CreateUseCase{repo: r.SubscriptionGroupDocumentTemplate, svc: s},
		ReadSubscriptionGroupDocumentTemplate:           &ReadUseCase{repo: r.SubscriptionGroupDocumentTemplate, svc: s},
		UpdateSubscriptionGroupDocumentTemplate:         &UpdateUseCase{repo: r.SubscriptionGroupDocumentTemplate, svc: s},
		DeleteSubscriptionGroupDocumentTemplate:         &DeleteUseCase{repo: r.SubscriptionGroupDocumentTemplate, svc: s},
		ListSubscriptionGroupDocumentTemplates:          &ListUseCase{repo: r.SubscriptionGroupDocumentTemplate, svc: s},
		FindApplicableSubscriptionGroupDocumentTemplate: &FindApplicableUseCase{repo: r.SubscriptionGroupDocumentTemplate, svc: s},
		PublishSubscriptionGroupDocumentTemplate:        &PublishUseCase{repo: r.SubscriptionGroupDocumentTemplate, svc: s},
	}
}

func (s Services) requireRequest(ctx context.Context, ok bool) error {
	if ok {
		return nil
	}
	return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, s.Translator,
		"subscription_group_document_template.validation.request_required", "Request is required [DEFAULT]"))
}
