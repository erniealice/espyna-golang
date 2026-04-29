// Package main is a CLI helper that drives the payroll orchestrator end-to-end
// against the live PostgreSQL database. It exists so the phase20-payroll E2E
// test can exercise the calculator without depending on the (currently-unmounted)
// /api/payroll/* HTTP routes.
//
// Usage:
//
//	payroll-orchestrate --run-id <payroll_run_id>
//
// Phases:
//  1. Generate pay cycles for the run.
//  2. Calculate the run (per-employee fan-out → Expenditure(type='payroll') rows).
//
// Both are idempotent at the orchestrator level; re-running on a cycle in
// CALCULATED state is a no-op (status guard).

//go:build postgresql

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/google/uuid"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	payrollservice "github.com/erniealice/espyna-golang/internal/application/services/payroll"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	pgentity "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/entity"
	pgexpenditure "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/expenditure"
	pgpayroll "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/payroll"
)

type uuidv7 struct{}

func (uuidv7) GenerateID() string                       { return uuid.New().String() }
func (uuidv7) GenerateIDWithPrefix(prefix string) string { return prefix + "_" + uuid.New().String() }
func (uuidv7) IsEnabled() bool                          { return true }
func (uuidv7) GetProviderInfo() string                  { return "google_uuidv7" }

var _ ports.IDService = uuidv7{}

func main() {
	runID := flag.String("run-id", "", "PayrollRun.id to orchestrate")
	stage := flag.String("stage", "all", "all | generate | calculate")
	flag.Parse()

	if *runID == "" {
		log.Fatal("--run-id is required")
	}

	dsn := buildDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	dbOps := postgresCore.NewWorkspaceAwareOperations(db)

	// Wire all adapters for the orchestrator's needs.
	repos := payrollservice.OrchestratorRepositories{
		Workspace:            pgentity.NewPostgresWorkspaceRepository(dbOps, "workspace"),
		Supplier:             pgentity.NewPostgresSupplierRepository(dbOps, "supplier"),
		SupplierContract:     pgexpenditure.NewPostgresSupplierContractRepository(dbOps, "supplier_contract"),
		SupplierContractLine: pgexpenditure.NewPostgresSupplierContractLineRepository(dbOps, "supplier_contract_line"),
		PayrollRun:           pgpayroll.NewPostgresPayrollRunRepository(dbOps, "payroll_run"),
		PayCycle:             pgpayroll.NewPostgresPayCycleRepository(dbOps, "pay_cycle"),
		RateTable:            pgpayroll.NewPostgresRateTableRepository(dbOps, "rate_table"),
		RateBand:             pgpayroll.NewPostgresRateBandRepository(dbOps, "rate_band"),
		LeaveBalance:         pgpayroll.NewPostgresLeaveBalanceRepository(dbOps, "leave_balance"),
		Expenditure:          pgexpenditure.NewPostgresExpenditureRepository(dbOps, "expenditure"),
		ExpenditureLineItem:  pgexpenditure.NewPostgresExpenditureLineItemRepository(dbOps, "expenditure_line_item"),
	}

	orch := payrollservice.NewOrchestrator(repos, uuidv7{})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if *stage == "all" || *stage == "generate" {
		if err := orch.GeneratePayCycles(ctx, *runID); err != nil {
			log.Fatalf("generate-cycles: %v", err)
		}
		fmt.Printf("✓ generated cycles for run=%s\n", *runID)
	}

	if *stage == "all" || *stage == "calculate" {
		result, err := orch.CalculatePayrollRun(ctx, *runID)
		if err != nil {
			log.Fatalf("calculate: %v", err)
		}
		fmt.Printf("✓ calculated run=%s cycles=%d employees=%d gross=%d deductions=%d net=%d\n",
			result.PayrollRunID,
			result.CyclesProcessed,
			result.EmployeesPaid,
			result.TotalGross,
			result.TotalDeductions,
			result.TotalNet,
		)
	}
}

func buildDSN() string {
	host := getenv("POSTGRES_HOST", "127.0.0.1")
	port := getenv("POSTGRES_PORT", "5432")
	user := getenv("POSTGRES_USER", "")
	pass := getenv("POSTGRES_PASSWORD", "")
	dbname := getenv("POSTGRES_NAME", "professional1")
	sslmode := getenv("POSTGRES_SSL_MODE", "disable")
	if user == "" {
		log.Fatal("POSTGRES_USER must be set")
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, dbname, sslmode)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
