//go:build postgresql

package payroll

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	leavebalancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_balance"
	leaverequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_request"
	paycyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/pay_cycle"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"
	"github.com/lib/pq"
)

// This read-only probe calls the actual page-data methods. It walks offset
// pages as an oracle, then follows Next to the end and Prev to the start.
func TestPayrollPageDataKeysetLiveParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is unset")
	}
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	var readOnly string
	if err := db.QueryRow("SHOW default_transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("read-only local database required: setting=%q err=%v", readOnly, err)
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "default-workspace"})
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	cases := []struct {
		name, table string
		fetch       func(*commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error)
	}{
		{"LeaveBalance", "leave_balance", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresLeaveBalanceRepository(ops, entityid.LeaveBalance)
			resp, err := repo.GetLeaveBalanceListPageData(ctx, &leavebalancepb.GetLeaveBalanceListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.LeaveBalanceList))
			for _, row := range resp.LeaveBalanceList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"LeaveRequest", "leave_request", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresLeaveRequestRepository(ops, entityid.LeaveRequest)
			resp, err := repo.GetLeaveRequestListPageData(ctx, &leaverequestpb.GetLeaveRequestListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.LeaveRequestList))
			for _, row := range resp.LeaveRequestList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"PayCycle", "pay_cycle", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresPayCycleRepository(ops, entityid.PayCycle)
			resp, err := repo.GetPayCycleListPageData(ctx, &paycyclepb.GetPayCycleListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.PayCycleList))
			for _, row := range resp.PayCycleList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"PayrollRemittance", "payroll_remittance", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresPayrollRemittanceRepository(ops, entityid.PayrollRemittance)
			resp, err := repo.GetPayrollRemittanceListPageData(ctx, &payrollremittancepb.GetPayrollRemittanceListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.PayrollRemittanceList))
			for _, row := range resp.PayrollRemittanceList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"PayrollRun", "payroll_run", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresPayrollRunRepository(ops, entityid.PayrollRun)
			resp, err := repo.GetPayrollRunListPageData(ctx, &payrollrunpb.GetPayrollRunListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.PayrollRunList))
			for _, row := range resp.PayrollRunList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"RateBand", "rate_band", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresRateBandRepository(ops, entityid.RateBand)
			resp, err := repo.GetRateBandListPageData(ctx, &ratebandpb.GetRateBandListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.RateBandList))
			for _, row := range resp.RateBandList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"RateTable", "rate_table", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresRateTableRepository(ops, entityid.RateTable)
			resp, err := repo.GetRateTableListPageData(ctx, &ratetablepb.GetRateTableListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.RateTableList))
			for _, row := range resp.RateTableList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
	}
	offset := func(page int32) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 1, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}}}
	}
	cursor := func(token string) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 1, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var present sql.NullString
			if err := db.QueryRow("SELECT to_regclass($1)", tc.table).Scan(&present); err != nil {
				t.Fatal(err)
			}
			if !present.Valid {
				t.Skipf("%s absent on lane DB", tc.table)
			}
			first, meta, err := tc.fetch(offset(1))
			if err != nil {
				t.Fatal(err)
			}
			if meta == nil {
				t.Fatal("missing pagination metadata")
			}
			if meta.GetTotalPages() < 2 {
				t.Logf("LIVE_UNPROVEN: %s has %d rows / %d pages", tc.table, meta.GetTotalItems(), meta.GetTotalPages())
				return
			}
			if meta.GetTotalPages() > 50 {
				t.Fatalf("unexpected walk size %d", meta.GetTotalPages())
			}
			oracle := [][]string{first}
			for n := int32(2); n <= meta.GetTotalPages(); n++ {
				ids, _, err := tc.fetch(offset(n))
				if err != nil {
					t.Fatal(err)
				}
				oracle = append(oracle, ids)
			}
			got := [][]string{first}
			k1Hops, offsetHops := 0, 0
			for n := int32(2); n <= meta.GetTotalPages(); n++ {
				if meta.GetNextCursor() == "" {
					t.Fatalf("page %d missing next cursor", n-1)
				}
				if strings.HasPrefix(meta.GetNextCursor(), "k1:") {
					k1Hops++
				} else if strings.HasPrefix(meta.GetNextCursor(), "offset:") {
					offsetHops++
				}
				ids, next, err := tc.fetch(cursor(meta.GetNextCursor()))
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, ids)
				meta = next
			}
			if !slices.EqualFunc(got, oracle, func(a, b []string) bool { return slices.Equal(a, b) }) {
				t.Fatalf("next walk mismatch: got=%v offset=%v", got, oracle)
			}
			for n := len(oracle) - 2; n >= 0; n-- {
				if meta.GetPrevCursor() == "" {
					t.Fatalf("page %d missing prev cursor", n+2)
				}
				if strings.HasPrefix(meta.GetPrevCursor(), "k1:") {
					k1Hops++
				} else if strings.HasPrefix(meta.GetPrevCursor(), "offset:") {
					offsetHops++
				}
				ids, prev, err := tc.fetch(cursor(meta.GetPrevCursor()))
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(ids, oracle[n]) {
					t.Fatalf("prev page %d mismatch: got=%v offset=%v", n+1, ids, oracle[n])
				}
				meta = prev
			}
			t.Logf("LIVE parity: %s %d rows %d pages next/prev exact; k1 hops=%d offset hops=%d", tc.table, meta.GetTotalItems(), len(oracle), k1Hops, offsetHops)
		})
	}
}
