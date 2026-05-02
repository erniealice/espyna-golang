// Package dashboard implements the read-only Service (product_kind=service)
// Dashboard use case (Phase 5 — Pyeza dashboard block + per-app live
// dashboards plan). Products are filtered to kind="service" to match the
// centymo service-mount surface.
//
// Wiring deferred: the orchestrator must construct
// *GetServiceDashboardPageDataUseCase from the postgres product adapters
// and add it to ProductUseCases.Dashboard (see usecases/product/usecases.go).
package dashboard

import (
	"context"
	"sort"
	"time"

	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// ProductDashboardQueries is the slice of the postgres product adapter the
// dashboard use case needs. Implemented by *PostgresProductRepository.
type ProductDashboardQueries interface {
	// CountByStatusAndKind returns {active: n, inactive: m} for the given
	// product_kind. Workspace-scoped.
	CountByStatusAndKind(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	// CountByLine returns line_id → count for active products of the given
	// product_kind. Workspace-scoped.
	CountByLine(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	// RecentlyListed returns the most-recently-created active products of
	// the given product_kind, newest-first. Workspace-scoped.
	RecentlyListed(ctx context.Context, workspaceID string, kind string, limit int32) ([]*productpb.Product, error)
}

// TopRevenueRow is a placeholder row for the top-revenue widget. Revenue
// values are not available until a line-item revenue join is implemented;
// the use case returns top-by-recency as a stand-in.
type TopRevenueRow struct {
	ProductID   string
	ProductName string
	Total       int64 // centavos — 0 until revenue join lands
}

// Stats holds tile values for the service dashboard.
type Stats struct {
	TotalActive      int64
	TopRevenueName   string
	TopRevenueValue  int64 // centavos — placeholder until line-item join lands
	LineCount        int64
	RecentlyAddedCnt int64
}

// GetServiceDashboardPageDataRequest is the request shape.
type GetServiceDashboardPageDataRequest struct {
	WorkspaceID string
	Kind        string // defaults to "service"
	Now         time.Time
}

// GetServiceDashboardPageDataResponse is the projection the view layer reads.
type GetServiceDashboardPageDataResponse struct {
	Stats Stats

	// Services-by-line for chart-bar (parallel slices, sorted by count desc).
	LineLabels []string
	LineValues []float64

	// Top revenue services (placeholder — top-by-recency until revenue join).
	TopRevenue []TopRevenueRow

	// Recent service additions (newest-first).
	Recent []*productpb.Product
}

// GetServiceDashboardPageDataUseCase orchestrates the service dashboard
// projection.
type GetServiceDashboardPageDataUseCase struct {
	products ProductDashboardQueries
}

// NewGetServiceDashboardPageDataUseCase constructs the use case.
func NewGetServiceDashboardPageDataUseCase(
	products ProductDashboardQueries,
) *GetServiceDashboardPageDataUseCase {
	return &GetServiceDashboardPageDataUseCase{products: products}
}

// Execute assembles the service dashboard response. Failures degrade
// gracefully — nil products adapter returns zero-valued response.
func (uc *GetServiceDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetServiceDashboardPageDataRequest,
) (*GetServiceDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetServiceDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	kind := req.Kind
	if kind == "" {
		kind = "service"
	}
	resp := &GetServiceDashboardPageDataResponse{}

	if uc.products == nil {
		return resp, nil
	}

	// Active count.
	if byStatus, err := uc.products.CountByStatusAndKind(ctx, req.WorkspaceID, kind); err == nil {
		resp.Stats.TotalActive = byStatus["active"]
	}

	// By-line breakdown for the chart.
	if byLine, err := uc.products.CountByLine(ctx, req.WorkspaceID, kind); err == nil {
		type kv struct {
			key string
			val int64
		}
		rows := make([]kv, 0, len(byLine))
		for k, v := range byLine {
			rows = append(rows, kv{k, v})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].val > rows[j].val })

		resp.LineLabels = make([]string, 0, len(rows))
		resp.LineValues = make([]float64, 0, len(rows))
		for _, r := range rows {
			resp.LineLabels = append(resp.LineLabels, r.key)
			resp.LineValues = append(resp.LineValues, float64(r.val))
		}
		resp.Stats.LineCount = int64(len(byLine))
	}

	// Recent additions (limit 5). Also used as top-revenue stand-in.
	if recent, err := uc.products.RecentlyListed(ctx, req.WorkspaceID, kind, 5); err == nil {
		resp.Recent = recent
		resp.Stats.RecentlyAddedCnt = int64(len(recent))

		// Populate TopRevenue as top-by-recency placeholder (revenue join deferred).
		resp.TopRevenue = make([]TopRevenueRow, 0, len(recent))
		for _, p := range recent {
			resp.TopRevenue = append(resp.TopRevenue, TopRevenueRow{
				ProductID:   p.GetId(),
				ProductName: p.GetName(),
				Total:       0, // centavos — 0 until revenue join
			})
		}
		if len(recent) > 0 {
			resp.Stats.TopRevenueName = recent[0].GetName()
		}
	}

	return resp, nil
}
