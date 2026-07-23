package job_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// job_template carries its own workspace_id, but its optional FKs
// (job_category_id, output_product_id) are posted directly from the drawer and
// must be proven in-workspace so a crafted POST cannot link a foreign
// workspace's category or product (red-team HIGH #2). Both referenced entities
// carry their own workspace_id:
//   - job_category.workspace_id (job_category.proto field 8)
//   - product.workspace_id      (product.proto field 30)
//
// The guard resolves each non-empty FK and requires its workspace_id to equal
// the caller's request workspace. An empty FK is skipped (both are optional). A
// nil repo, empty request workspace, read error, or empty result set all
// fail-closed.

// templateFKScopeRepos bundles the FK-resolution repos.
type templateFKScopeRepos struct {
	JobCategory jobcategorypb.JobCategoryDomainServiceServer
	Product     productpb.ProductDomainServiceServer
}

// requireJobCategoryInWorkspace fail-closes unless the referenced job_category
// exists AND its workspace_id equals wsID. An empty categoryID is a no-op (the
// FK is optional).
func requireJobCategoryInWorkspace(ctx context.Context, repo jobcategorypb.JobCategoryDomainServiceServer, tr ports.Translator, wsID, categoryID string) error {
	if categoryID == "" {
		return nil
	}
	if repo == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template.validation.category_not_found",
			"referenced job category not found [DEFAULT]"))
	}
	resp, err := repo.ReadJobCategory(ctx, &jobcategorypb.ReadJobCategoryRequest{
		Data: &jobcategorypb.JobCategory{Id: categoryID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template.validation.category_not_found",
			"referenced job category not found [DEFAULT]"))
	}
	ws := resp.GetData()[0].GetWorkspaceId()
	if wsID == "" || ws == "" || ws != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template.validation.cross_workspace_fk",
			"job template references must stay within your workspace [DEFAULT]"))
	}
	return nil
}

// requireProductInWorkspace fail-closes unless the referenced product exists AND
// its workspace_id equals wsID. An empty productID is a no-op (the FK is
// optional).
func requireProductInWorkspace(ctx context.Context, repo productpb.ProductDomainServiceServer, tr ports.Translator, wsID, productID string) error {
	if productID == "" {
		return nil
	}
	if repo == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template.validation.product_not_found",
			"referenced output product not found [DEFAULT]"))
	}
	resp, err := repo.ReadProduct(ctx, &productpb.ReadProductRequest{
		Data: &productpb.Product{Id: productID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template.validation.product_not_found",
			"referenced output product not found [DEFAULT]"))
	}
	ws := resp.GetData()[0].GetWorkspaceId()
	if wsID == "" || ws == "" || ws != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template.validation.cross_workspace_fk",
			"job template references must stay within your workspace [DEFAULT]"))
	}
	return nil
}

// requireTemplateFKsInWorkspace validates every non-empty FK on the effective
// template data against the caller's request workspace.
func requireTemplateFKsInWorkspace(ctx context.Context, r templateFKScopeRepos, tr ports.Translator, wsID, categoryID, productID string) error {
	if err := requireJobCategoryInWorkspace(ctx, r.JobCategory, tr, wsID, categoryID); err != nil {
		return err
	}
	if err := requireProductInWorkspace(ctx, r.Product, tr, wsID, productID); err != nil {
		return err
	}
	return nil
}
