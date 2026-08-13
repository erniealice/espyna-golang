package subscription_group_document_template

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

type PublishUseCase struct {
	repo pb.SubscriptionGroupDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *PublishUseCase) Execute(ctx context.Context, req *pb.PublishSubscriptionGroupDocumentTemplateRequest) (*pb.PublishSubscriptionGroupDocumentTemplateResponse, error) {
	// Publish is a controlled state transition, gated on the binding's update
	// permission (there is no separate publish verb in the catalog).
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupDocumentTemplate, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Id != ""); err != nil {
		return nil, err
	}
	if uc.svc.Transactor == nil || !uc.svc.Transactor.SupportsTransactions() {
		return nil, fmt.Errorf("subscription group document template publish requires transaction support")
	}
	var response *pb.PublishSubscriptionGroupDocumentTemplateResponse
	err := uc.svc.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		var publishErr error
		response, publishErr = uc.repo.PublishSubscriptionGroupDocumentTemplate(txCtx, req)
		if publishErr != nil {
			return publishErr
		}
		if response == nil || !response.Success || response.Data == nil {
			return fmt.Errorf("subscription group document template publish returned an invalid response")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}
