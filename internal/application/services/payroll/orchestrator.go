package payroll

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	leavebalancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_balance"
	paycyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/pay_cycle"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"

	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// OrchestratorRepositories bundles every repo the payroll orchestrator touches.
type OrchestratorRepositories struct {
	Workspace            workspacepb.WorkspaceDomainServiceServer
	Supplier             supplierpb.SupplierDomainServiceServer
	SupplierContract     suppliercontractpb.SupplierContractDomainServiceServer
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
	PayrollRun           payrollrunpb.PayrollRunDomainServiceServer
	PayCycle             paycyclepb.PayCycleDomainServiceServer
	RateTable            ratetablepb.RateTableDomainServiceServer
	RateBand             ratebandpb.RateBandDomainServiceServer
	LeaveBalance         leavebalancepb.LeaveBalanceDomainServiceServer
	Expenditure          expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem  expenditurelinepb.ExpenditureLineItemDomainServiceServer
}

// Orchestrator is the payroll lifecycle entry point: cycle generation + calculate.
// It wraps the per-region PayrollCalculator and ties it to repository writes.
type Orchestrator struct {
	repos     OrchestratorRepositories
	idService ports.IDService
}

// NewOrchestrator wires the orchestrator with all required repos.
func NewOrchestrator(repos OrchestratorRepositories, idService ports.IDService) *Orchestrator {
	return &Orchestrator{repos: repos, idService: idService}
}

// CalculateResult summarizes a calculate run.
type CalculateResult struct {
	PayrollRunID    string
	CyclesProcessed int
	EmployeesPaid   int
	TotalGross      int64
	TotalDeductions int64
	TotalNet        int64
}

// GeneratePayCycles materializes 1+ PayCycle rows for a freshly-created PayrollRun
// based on the run's pay_period_start..pay_period_end window. The pay_frequency is
// derived from active employment contracts of the workspace (fallback "semi_monthly").
//
// Semi-monthly produces 2 cycles: 1st-15th (first half) and 16th-EOM (second half).
// Monthly produces 1 (full).
// Weekly produces ⌈days/7⌉ (basic chunking).
func (o *Orchestrator) GeneratePayCycles(ctx context.Context, runID string) error {
	if runID == "" {
		return fmt.Errorf("runID is required")
	}

	runResp, err := o.repos.PayrollRun.ReadPayrollRun(ctx, &payrollrunpb.ReadPayrollRunRequest{
		Data: &payrollrunpb.PayrollRun{Id: runID},
	})
	if err != nil {
		return fmt.Errorf("read payroll_run: %w", err)
	}
	if runResp == nil || len(runResp.Data) == 0 {
		return fmt.Errorf("payroll_run %s not found", runID)
	}
	run := runResp.Data[0]

	startStr := run.PayPeriodStart
	endStr := run.PayPeriodEnd
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return fmt.Errorf("invalid pay_period_start %q: %w", startStr, err)
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return fmt.Errorf("invalid pay_period_end %q: %w", endStr, err)
	}
	if !end.After(start) && !end.Equal(start) {
		return fmt.Errorf("pay_period_end must be on/after pay_period_start")
	}

	freq := detectPayFrequency(run, "semi_monthly")
	cycles := buildCycles(start, end, freq)

	wsID := ""
	if run.WorkspaceId != nil {
		wsID = *run.WorkspaceId
	}

	for i, c := range cycles {
		id := o.idService.GenerateID()
		cycle := &paycyclepb.PayCycle{
			Id:           id,
			WorkspaceId:  wsID,
			PayrollRunId: &runID,
			CutoffStart:  c.CutoffStart,
			CutoffEnd:    c.CutoffEnd,
			PayDate:      c.PayDate,
			HalfIndex:    c.HalfIndex,
			Status:       paycyclepb.PayCycleStatus_PAY_CYCLE_STATUS_OPEN,
			SequenceNo:   int32(i + 1),
			Active:       true,
		}
		if _, err := o.repos.PayCycle.CreatePayCycle(ctx, &paycyclepb.CreatePayCycleRequest{Data: cycle}); err != nil {
			return fmt.Errorf("create pay_cycle %d: %w", i+1, err)
		}
	}
	return nil
}

// CalculatePayrollRun is the centerpiece. For every PayCycle in the run that is
// in {OPEN, CALCULATED, REOPENED}, fan out by employee with active employment contract.
// Each employee × cycle produces:
//   - one Expenditure(expenditure_type='payroll', vendor=employee)
//   - one ExpenditureLineItem per LineResolution from the calculator
//
// Aggregates roll up to the cycle and run.
func (o *Orchestrator) CalculatePayrollRun(ctx context.Context, runID string) (*CalculateResult, error) {
	runResp, err := o.repos.PayrollRun.ReadPayrollRun(ctx, &payrollrunpb.ReadPayrollRunRequest{
		Data: &payrollrunpb.PayrollRun{Id: runID},
	})
	if err != nil || runResp == nil || len(runResp.Data) == 0 {
		return nil, fmt.Errorf("payroll_run %s not found: %w", runID, err)
	}
	run := runResp.Data[0]
	if run.Status == payrollrunpb.PayrollRunStatus_PAYROLL_RUN_STATUS_POSTED {
		return nil, fmt.Errorf("payroll_run is POSTED; reopen first")
	}

	region := derefStr(run.ComplianceRegion)
	if region == "" && run.WorkspaceId != nil {
		// fallback: read workspace.compliance_region
		wsResp, _ := o.repos.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{
			Data: &workspacepb.Workspace{Id: *run.WorkspaceId},
		})
		if wsResp != nil && len(wsResp.Data) > 0 && wsResp.Data[0].ComplianceRegion != nil {
			region = *wsResp.Data[0].ComplianceRegion
		}
	}
	if region == "" {
		region = "PH" // default
	}
	calc := Get(region)

	// List cycles for this run.
	cyclesResp, err := o.repos.PayCycle.ListPayCycles(ctx, &paycyclepb.ListPayCyclesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list pay_cycles: %w", err)
	}
	var cycles []*paycyclepb.PayCycle
	for _, c := range cyclesResp.Data {
		if c.PayrollRunId != nil && *c.PayrollRunId == runID && c.Status != paycyclepb.PayCycleStatus_PAY_CYCLE_STATUS_POSTED {
			cycles = append(cycles, c)
		}
	}
	if len(cycles) == 0 {
		return nil, fmt.Errorf("no eligible pay cycles for run %s — generate cycles first", runID)
	}

	// List employees: Supplier.kind == 'employee'.
	suppliersResp, err := o.repos.Supplier.ListSuppliers(ctx, &supplierpb.ListSuppliersRequest{})
	if err != nil {
		return nil, fmt.Errorf("list suppliers: %w", err)
	}
	var employees []*supplierpb.Supplier
	for _, s := range suppliersResp.Data {
		if s.Kind != nil && *s.Kind == "employee" && s.Active {
			employees = append(employees, s)
		}
	}

	// List active employment contracts (filter by kind=EMPLOYMENT in caller code).
	contractsResp, err := o.repos.SupplierContract.ListSupplierContracts(ctx, &suppliercontractpb.ListSupplierContractsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list supplier_contracts: %w", err)
	}
	contractsBySupplier := make(map[string][]*suppliercontractpb.SupplierContract)
	for _, sc := range contractsResp.Data {
		if sc.Kind != suppliercontractpb.SupplierContractKind_SUPPLIER_CONTRACT_KIND_EMPLOYMENT {
			continue
		}
		if sc.Status == suppliercontractpb.SupplierContractStatus_SUPPLIER_CONTRACT_STATUS_TERMINATED ||
			sc.Status == suppliercontractpb.SupplierContractStatus_SUPPLIER_CONTRACT_STATUS_REJECTED ||
			sc.Status == suppliercontractpb.SupplierContractStatus_SUPPLIER_CONTRACT_STATUS_EXPIRED {
			continue
		}
		contractsBySupplier[sc.SupplierId] = append(contractsBySupplier[sc.SupplierId], sc)
	}

	// List all contract lines once; bucket by contract.
	linesResp, err := o.repos.SupplierContractLine.ListSupplierContractLines(ctx, &suppliercontractlinepb.ListSupplierContractLinesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list supplier_contract_lines: %w", err)
	}
	linesByContract := make(map[string][]*suppliercontractlinepb.SupplierContractLine)
	for _, ln := range linesResp.Data {
		if !ln.Active {
			continue
		}
		linesByContract[ln.SupplierContractId] = append(linesByContract[ln.SupplierContractId], ln)
	}

	rateResolver := o.makeRateResolver(ctx)

	result := &CalculateResult{PayrollRunID: runID}

	for _, cycle := range cycles {
		cycleGross, cycleDed, cycleNet := int64(0), int64(0), int64(0)
		empCount := 0

		for _, emp := range employees {
			contracts := contractsBySupplier[emp.Id]
			if len(contracts) == 0 {
				continue
			}
			// Pick the contract whose window covers cycle.cutoff_start (single-contract MVP).
			contract := pickActiveContractAt(contracts, cycle.CutoffStart)
			if contract == nil {
				continue
			}
			lines := linesByContract[contract.Id]

			payFreq := derefStr(contract.PayFrequency)
			if payFreq == "" {
				payFreq = "semi_monthly"
			}

			pctx := &PayslipContext{
				PayCycle:           cycle,
				EmployeeID:         emp.Id,
				EmploymentContract: contract,
				ContractLines:      lines,
				PayFrequency:       payFreq,
				HalfIndex:          cycle.HalfIndex,
				RateResolver:       rateResolver,
			}
			resolutions, err := calc.Calculate(ctx, pctx)
			if err != nil {
				return nil, fmt.Errorf("calculator failed for employee %s cycle %s: %w", emp.Id, cycle.Id, err)
			}

			gross, ded := splitGrossDeductions(resolutions)
			net := gross - ded

			// Materialize as Expenditure(type=payroll) + lines.
			expID := o.idService.GenerateID()
			expName := fmt.Sprintf("Payslip %s — %s", emp.Name, cycle.PayDate)
			payDateMillis := parseDateMillis(cycle.PayDate)
			payDateStr := cycle.PayDate

			workspaceID := emp.ClientId
			_ = workspaceID
			exp := &expenditurepb.Expenditure{
				Id:                    expID,
				Active:                true,
				Name:                  expName,
				ExpenditureType:       "payroll",
				VendorId:              emp.Id,
				ExpenditureDate:       &payDateMillis,
				ExpenditureDateString: &payDateStr,
				TotalAmount:           net,
				Currency:              "PHP",
				Status:                "draft",
				LocationId:            "",
				SupplierId:            &emp.Id,
			}
			if _, err := o.repos.Expenditure.CreateExpenditure(ctx, &expenditurepb.CreateExpenditureRequest{Data: exp}); err != nil {
				return nil, fmt.Errorf("create payslip expenditure: %w", err)
			}

			for _, lr := range resolutions {
				lineID := o.idService.GenerateID()
				meta := lr.CalcMetadata
				rateID := lr.RateTableID
				cycleID := cycle.Id
				appliedBasis := lr.AppliedBasis
				proration := lr.ProrationFactor
				lineKind := lr.LineKind
				eli := &expenditurelinepb.ExpenditureLineItem{
					Id:                  lineID,
					Active:              true,
					ExpenditureId:       expID,
					Description:         lr.Description,
					Quantity:            lr.Quantity,
					UnitPrice:           lr.UnitPrice,
					TotalPrice:          lr.Amount,
					LineItemType:        "item",
					RateTableId:         &rateID,
					PayCycleId:          &cycleID,
					AppliedBasisAmount:  &appliedBasis,
					ProrationFactor:     &proration,
					CalcMetadata:        &meta,
					LineKind:            &lineKind,
				}
				if _, err := o.repos.ExpenditureLineItem.CreateExpenditureLineItem(ctx, &expenditurelinepb.CreateExpenditureLineItemRequest{Data: eli}); err != nil {
					return nil, fmt.Errorf("create payslip line: %w", err)
				}
			}

			cycleGross += gross
			cycleDed += ded
			cycleNet += net
			empCount++
		}

		// Update PayCycle aggregates and status → CALCULATED.
		cycle.TotalGross = cycleGross
		cycle.TotalDeductions = cycleDed
		cycle.TotalNet = cycleNet
		cycle.EmployeeCount = int32(empCount)
		cycle.Status = paycyclepb.PayCycleStatus_PAY_CYCLE_STATUS_CALCULATED
		if _, err := o.repos.PayCycle.UpdatePayCycle(ctx, &paycyclepb.UpdatePayCycleRequest{Data: cycle}); err != nil {
			return nil, fmt.Errorf("update pay_cycle %s: %w", cycle.Id, err)
		}

		result.CyclesProcessed++
		result.EmployeesPaid += empCount
		result.TotalGross += cycleGross
		result.TotalDeductions += cycleDed
		result.TotalNet += cycleNet
	}

	// Update run aggregates.
	run.TotalGross = result.TotalGross
	run.TotalDeductions = result.TotalDeductions
	run.TotalNet = result.TotalNet
	run.EmployeeCount = int32(result.EmployeesPaid)
	run.Status = payrollrunpb.PayrollRunStatus_PAYROLL_RUN_STATUS_CALCULATED
	calcVer := calc.Version()
	run.CalculatorVersion = &calcVer
	regionRef := region
	run.ComplianceRegion = &regionRef
	if _, err := o.repos.PayrollRun.UpdatePayrollRun(ctx, &payrollrunpb.UpdatePayrollRunRequest{Data: run}); err != nil {
		return nil, fmt.Errorf("update payroll_run: %w", err)
	}

	return result, nil
}

// makeRateResolver returns a RateResolver bound to the orchestrator's repos.
// It looks up the active RateTable for (kind, region, asOf) and returns its bands.
func (o *Orchestrator) makeRateResolver(ctx context.Context) func(context.Context, string, string, time.Time) (*ratetablepb.RateTable, []*ratebandpb.RateBand, error) {
	return func(rctx context.Context, kind, region string, asOf time.Time) (*ratetablepb.RateTable, []*ratebandpb.RateBand, error) {
		tablesResp, err := o.repos.RateTable.ListRateTables(rctx, &ratetablepb.ListRateTablesRequest{})
		if err != nil {
			return nil, nil, fmt.Errorf("list rate_tables: %w", err)
		}
		var picked *ratetablepb.RateTable
		for _, t := range tablesResp.Data {
			if t.ComplianceRegion != region || t.Kind != kind {
				continue
			}
			if t.Status != ratetablepb.RateTableStatus_RATE_TABLE_STATUS_ACTIVE {
				continue
			}
			eff, err := time.Parse("2006-01-02", t.EffectiveFrom)
			if err != nil {
				continue
			}
			if eff.After(asOf) {
				continue
			}
			if t.EffectiveTo != nil && *t.EffectiveTo != "" {
				to, err := time.Parse("2006-01-02", *t.EffectiveTo)
				if err == nil && to.Before(asOf) {
					continue
				}
			}
			if picked == nil {
				picked = t
				continue
			}
			pickedEff, _ := time.Parse("2006-01-02", picked.EffectiveFrom)
			if eff.After(pickedEff) {
				picked = t
			}
		}
		if picked == nil {
			return nil, nil, fmt.Errorf("no active rate_table for region=%s kind=%s asOf=%s", region, kind, asOf.Format("2006-01-02"))
		}
		bandsResp, err := o.repos.RateBand.ListRateBands(rctx, &ratebandpb.ListRateBandsRequest{})
		if err != nil {
			return nil, nil, fmt.Errorf("list rate_bands: %w", err)
		}
		var bands []*ratebandpb.RateBand
		for _, b := range bandsResp.Data {
			if b.RateTableId == picked.Id && b.Active {
				bands = append(bands, b)
			}
		}
		return picked, bands, nil
	}
}

// ---------- internal helpers ----------

type cycleSpec struct {
	CutoffStart, CutoffEnd, PayDate, HalfIndex string
}

func detectPayFrequency(run *payrollrunpb.PayrollRun, fallback string) string {
	// Fallback only — full freq detection happens per-employee from contract.
	_ = run
	return fallback
}

func buildCycles(start, end time.Time, freq string) []cycleSpec {
	switch freq {
	case "monthly":
		return []cycleSpec{{
			CutoffStart: start.Format("2006-01-02"),
			CutoffEnd:   end.Format("2006-01-02"),
			PayDate:     end.Format("2006-01-02"),
			HalfIndex:   "full",
		}}
	case "semi_monthly":
		// Find the 15th within the run window. If start..end spans the 15th, split into
		// first half (start..15th) and second half (16th..end). Otherwise single cycle.
		mid := time.Date(start.Year(), start.Month(), 15, 0, 0, 0, 0, start.Location())
		if mid.Before(start) || mid.After(end) {
			return []cycleSpec{{
				CutoffStart: start.Format("2006-01-02"),
				CutoffEnd:   end.Format("2006-01-02"),
				PayDate:     end.Format("2006-01-02"),
				HalfIndex:   "full",
			}}
		}
		secondStart := mid.AddDate(0, 0, 1)
		return []cycleSpec{
			{
				CutoffStart: start.Format("2006-01-02"),
				CutoffEnd:   mid.Format("2006-01-02"),
				PayDate:     mid.Format("2006-01-02"),
				HalfIndex:   "first",
			},
			{
				CutoffStart: secondStart.Format("2006-01-02"),
				CutoffEnd:   end.Format("2006-01-02"),
				PayDate:     end.Format("2006-01-02"),
				HalfIndex:   "second",
			},
		}
	case "weekly":
		var out []cycleSpec
		s := start
		i := 0
		for !s.After(end) {
			e := s.AddDate(0, 0, 6)
			if e.After(end) {
				e = end
			}
			out = append(out, cycleSpec{
				CutoffStart: s.Format("2006-01-02"),
				CutoffEnd:   e.Format("2006-01-02"),
				PayDate:     e.Format("2006-01-02"),
				HalfIndex:   "full",
			})
			s = e.AddDate(0, 0, 1)
			i++
			if i > 6 {
				break
			}
		}
		return out
	default:
		return []cycleSpec{{
			CutoffStart: start.Format("2006-01-02"),
			CutoffEnd:   end.Format("2006-01-02"),
			PayDate:     end.Format("2006-01-02"),
			HalfIndex:   "full",
		}}
	}
}

func pickActiveContractAt(contracts []*suppliercontractpb.SupplierContract, asOfStr string) *suppliercontractpb.SupplierContract {
	asOf, err := time.Parse("2006-01-02", asOfStr)
	if err != nil {
		return nil
	}
	for _, sc := range contracts {
		if !sc.Active {
			continue
		}
		startStr := sc.DateTimeStart
		if startStr == "" {
			continue
		}
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil || start.After(asOf) {
			continue
		}
		if sc.DateTimeEnd != nil && *sc.DateTimeEnd != "" {
			end, err := time.Parse("2006-01-02", *sc.DateTimeEnd)
			if err == nil && end.Before(asOf) {
				continue
			}
		}
		return sc
	}
	return nil
}

func splitGrossDeductions(rs []LineResolution) (gross, ded int64) {
	for _, r := range rs {
		if strings.HasPrefix(r.LineKind, "earning_") {
			gross += r.Amount
		} else if strings.HasPrefix(r.LineKind, "deduction_") {
			ded += r.Amount
		}
	}
	return
}

func parseDateMillis(s string) int64 {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

