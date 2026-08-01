//go:build postgresql

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/ports"
	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
)

// =============================================================================
// Fixtures
// =============================================================================

// infrastructurePkgPath is the package that declares the port types. Both alias
// layers (internal/application/ports and the public ports package) must resolve
// to types declared here.
const infrastructurePkgPath = "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"

// lcPoolMaxConns is the open cap stamped onto every fixture pool, chosen so a
// live pool's PoolStats().MaxOpen is distinguishable from the zero value.
const lcPoolMaxConns = 17

// lcPool opens a pool against an unroutable address. database/sql defers all
// I/O until first use, so this is a real, closable *sql.DB that never contacts
// a server; Stats() and Close() are exercised for real.
func lcPool(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", "host=127.0.0.1 port=1 dbname=lifecycle user=lifecycle sslmode=disable connect_timeout=1")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(lcPoolMaxConns)
	db.SetMaxIdleConns(lcPoolMaxConns)
	return db
}

func lcConfig() *PostgresConfig {
	return &PostgresConfig{
		Host: "127.0.0.1", Port: "1", Name: "lifecycle", User: "lifecycle",
		SSLMode: "disable", MaxConns: lcPoolMaxConns, MaxIdleConns: lcPoolMaxConns,
		StatementTimeoutSeconds: 30, LockTimeoutSeconds: 10, IdleTxTimeoutSeconds: 60,
	}
}

// lcProtoConfig is a minimal valid proto config: enough for
// resolvePostgresConfig to succeed so Initialize reaches its lifecycle guard.
func lcProtoConfig() *dbpb.DatabaseProviderConfig {
	return &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_POSTGRESQL,
		Enabled:  true,
		Config: &dbpb.DatabaseProviderConfig_Postgresql{
			Postgresql: &dbpb.PostgreSQLConfig{
				Host: "127.0.0.1", Port: "1", Database: "lifecycle", Username: "lifecycle",
			},
		},
	}
}

// lcSeedLive drives the adapter into the state a successful Initialize leaves
// behind, without reaching a server. It does so by calling the PRODUCTION
// go-live block (attach) under mu — exactly as Initialize does after its Ping —
// so an ordering regression inside attach (monitorWG.Add moved after the
// goroutine starts, stopMonitor left unassigned, monitor launched before the
// state is published) breaks this suite instead of hiding behind a
// hand-rolled duplicate.
//
// Monitor liveness is observed through a.monitorsRunning: attach increments it
// under mu and the goroutine decrements it before its monitorWG.Done, so it
// reads 0 only once the monitor has genuinely returned.
func lcSeedLive(t *testing.T, a *PostgresAdapter) *sql.DB {
	t.Helper()
	db := lcPool(t)

	a.mu.Lock()
	a.attach(db, lcConfig(), true)
	a.mu.Unlock()

	if a.monitorsRunning.Load() != 1 {
		t.Fatalf("attach registered %d monitors, want 1", a.monitorsRunning.Load())
	}
	return db
}

// lcRace runs fn in n goroutines released simultaneously and waits for all.
func lcRace(n int, fn func(i int)) {
	var ready, done sync.WaitGroup
	start := make(chan struct{})
	ready.Add(n)
	done.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer done.Done()
			ready.Done()
			<-start
			fn(i)
		}(i)
	}
	ready.Wait()
	close(start)
	done.Wait()
}

// =============================================================================
// Close: idempotence and concurrency
// =============================================================================

func TestCloseIsIdempotent(t *testing.T) {
	a := NewPostgresAdapter()
	lcSeedLive(t, a)

	for i := 1; i <= 3; i++ {
		if err := a.Close(); err != nil {
			t.Fatalf("Close #%d: %v", i, err)
		}
	}
	if got := a.monitorsRunning.Load(); got != 0 {
		t.Errorf("%d monitor goroutine(s) still running when Close returned, want 0", got)
	}

	a.mu.Lock()
	db, cfg, stop, init, conn := a.db, a.config, a.stopMonitor, a.initialized, a.connected
	a.mu.Unlock()
	if db != nil || cfg != nil || stop != nil {
		t.Errorf("Close left state attached: db=%v config=%v stop=%v", db, cfg, stop)
	}
	if init || conn {
		t.Errorf("Close left initialized=%v connected=%v, want false/false", init, conn)
	}
}

func TestCloseIsSafeUnderConcurrentCallers(t *testing.T) {
	for trial := 0; trial < 20; trial++ {
		a := NewPostgresAdapter()
		lcSeedLive(t, a)

		errs := make([]error, 32)
		lcRace(len(errs), func(i int) { errs[i] = a.Close() })

		for i, err := range errs {
			if err != nil {
				t.Fatalf("trial %d: concurrent Close #%d: %v", trial, i, err)
			}
		}
		if got := a.monitorsRunning.Load(); got != 0 {
			t.Fatalf("trial %d: %d monitor goroutine(s) outlived the last Close", trial, got)
		}
	}
}

func TestCloseOnFreshAdapterIsANoOp(t *testing.T) {
	a := NewPostgresAdapter()
	lcRace(16, func(int) {
		if err := a.Close(); err != nil {
			t.Errorf("Close on never-initialized adapter: %v", err)
		}
	})
	if got := a.PoolStats(); got != (ports.PoolStats{}) {
		t.Errorf("PoolStats on never-initialized adapter = %+v, want zero value", got)
	}
}

func TestNilReceiverAccessorsAreSafe(t *testing.T) {
	var a *PostgresAdapter
	if err := a.Close(); err != nil {
		t.Errorf("nil.Close() = %v, want nil", err)
	}
	if got := a.PoolStats(); got != (ports.PoolStats{}) {
		t.Errorf("nil.PoolStats() = %+v, want zero value", got)
	}
	if got := a.MaxConns(); got != 0 {
		t.Errorf("nil.MaxConns() = %d, want 0", got)
	}
	if got := a.GetConnection(); got != nil {
		t.Errorf("nil.GetConnection() = %v, want nil", got)
	}
	if a.IsEnabled() {
		t.Error("nil.IsEnabled() = true, want false")
	}
	if got := a.GetTransactionManager(); got != nil {
		t.Errorf("nil.GetTransactionManager() = %v, want nil", got)
	}
	if err := a.IsHealthy(context.Background()); err == nil {
		t.Error("nil.IsHealthy() = nil, want an error")
	}
}

// =============================================================================
// Telemetry polling concurrent with shutdown
// =============================================================================

func TestPoolStatsPollingConcurrentWithClose(t *testing.T) {
	for trial := 0; trial < 10; trial++ {
		a := NewPostgresAdapter()
		lcSeedLive(t, a)

		stopPolling := make(chan struct{})
		var pollers sync.WaitGroup
		for i := 0; i < 8; i++ {
			pollers.Add(1)
			go func() {
				defer pollers.Done()
				for {
					select {
					case <-stopPolling:
						return
					default:
					}
					_ = a.PoolStats()
					_ = a.MaxConns()
					_ = a.GetConnection()
					_ = a.IsEnabled()
					_ = a.GetTransactionManager()
				}
			}()
		}

		// Let the pollers observe the live pool before shutdown starts.
		for {
			if a.PoolStats().MaxOpen == lcPoolMaxConns {
				break
			}
			time.Sleep(time.Millisecond)
		}

		lcRace(4, func(int) { _ = a.Close() })
		close(stopPolling)
		pollers.Wait()

		if got := a.PoolStats(); got != (ports.PoolStats{}) {
			t.Fatalf("trial %d: PoolStats after Close = %+v, want zero value", trial, got)
		}
		if got := a.MaxConns(); got != 0 {
			t.Fatalf("trial %d: MaxConns after Close = %d, want 0", trial, got)
		}
		if got := a.GetTransactionManager(); got != nil {
			t.Fatalf("trial %d: GetTransactionManager after Close = %v, want nil", trial, got)
		}
	}
}

func TestPoolStatsReportsTheLivePool(t *testing.T) {
	a := NewPostgresAdapter()
	lcSeedLive(t, a)
	t.Cleanup(func() { _ = a.Close() })

	got := a.PoolStats()
	if got.MaxOpen != lcPoolMaxConns {
		t.Errorf("PoolStats().MaxOpen = %d, want %d", got.MaxOpen, lcPoolMaxConns)
	}
	if a.MaxConns() != lcPoolMaxConns {
		t.Errorf("MaxConns() = %d, want %d", a.MaxConns(), lcPoolMaxConns)
	}
}

// =============================================================================
// Monitor goroutine shutdown
// =============================================================================

func TestMonitorPoolReturnsWhenStopIsClosed(t *testing.T) {
	db := lcPool(t)
	t.Cleanup(func() { _ = db.Close() })

	stop := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		monitorPool(db, stop)
	}()

	close(stop)
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("monitorPool did not return after stop was closed")
	}
}

func TestCloseJoinsTheMonitorBeforeReturning(t *testing.T) {
	const linger = 250 * time.Millisecond

	a := NewPostgresAdapter()
	lcSeedLive(t, a)

	// A second, deliberately slow monitorWG participant: Close must not return
	// until this one has finished as well.
	lingered := &atomic.Bool{}
	a.monitorWG.Add(1)
	go func() {
		defer a.monitorWG.Done()
		time.Sleep(linger)
		lingered.Store(true)
	}()

	started := time.Now()
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	elapsed := time.Since(started)

	if !lingered.Load() {
		t.Error("Close returned while a monitorWG participant was still running")
	}
	if got := a.monitorsRunning.Load(); got != 0 {
		t.Errorf("Close returned while %d monitor goroutine(s) were still running", got)
	}
	if elapsed < linger {
		t.Errorf("Close returned after %s, want at least %s (join did not block)", elapsed, linger)
	}
}

// =============================================================================
// Initialize guard
// =============================================================================

func TestSecondInitializeIsRejected(t *testing.T) {
	a := NewPostgresAdapter()
	db := lcSeedLive(t, a)
	t.Cleanup(func() { _ = a.Close() })

	if err := a.Initialize(lcProtoConfig()); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("second Initialize = %v, want ErrAlreadyInitialized", err)
	}

	a.mu.Lock()
	sameDB, stop, cfg, init := a.db == db, a.stopMonitor, a.config, a.initialized
	a.mu.Unlock()
	if !sameDB || stop == nil || cfg == nil || !init {
		t.Errorf("rejected Initialize disturbed live state: sameDB=%v stop=%v config=%v initialized=%v",
			sameDB, stop, cfg, init)
	}
	if got := a.PoolStats().MaxOpen; got != lcPoolMaxConns {
		t.Errorf("PoolStats().MaxOpen after rejected Initialize = %d, want %d", got, lcPoolMaxConns)
	}
}

func TestConcurrentInitializeYieldsExactlyOneWinner(t *testing.T) {
	a := NewPostgresAdapter()
	lcSeedLive(t, a)
	t.Cleanup(func() { _ = a.Close() })

	errs := make([]error, 24)
	lcRace(len(errs), func(i int) { errs[i] = a.Initialize(lcProtoConfig()) })
	for i, err := range errs {
		if !errors.Is(err, ErrAlreadyInitialized) {
			t.Fatalf("concurrent Initialize #%d = %v, want ErrAlreadyInitialized", i, err)
		}
	}
}

func TestCloseRearmsInitialize(t *testing.T) {
	a := NewPostgresAdapter()

	for round := 1; round <= 3; round++ {
		lcSeedLive(t, a)
		if err := a.Initialize(lcProtoConfig()); !errors.Is(err, ErrAlreadyInitialized) {
			t.Fatalf("round %d: Initialize on live adapter = %v, want ErrAlreadyInitialized", round, err)
		}
		if err := a.Close(); err != nil {
			t.Fatalf("round %d: Close: %v", round, err)
		}
		a.mu.Lock()
		init := a.initialized
		a.mu.Unlock()
		if init {
			t.Fatalf("round %d: initialized still true after Close", round)
		}
	}
}

func TestInitializeRejectsBadConfigBeforeTouchingState(t *testing.T) {
	badTimeout := lcProtoConfig()
	badTimeout.GetPostgresql().StatementTimeoutSeconds = 99999

	cases := []struct {
		name   string
		config *dbpb.DatabaseProviderConfig
	}{
		{"missing postgresql config", &dbpb.DatabaseProviderConfig{Enabled: true}},
		{"statement timeout out of range", badTimeout},
	}

	for _, tc := range cases {
		t.Run("fresh adapter/"+tc.name, func(t *testing.T) {
			a := NewPostgresAdapter()
			if err := a.Initialize(tc.config); err == nil {
				t.Fatal("Initialize = nil, want an error")
			}
			a.mu.Lock()
			db, cfg, init, stop := a.db, a.config, a.initialized, a.stopMonitor
			a.mu.Unlock()
			if db != nil || cfg != nil || init || stop != nil {
				t.Errorf("failed Initialize mutated state: db=%v config=%v initialized=%v stop=%v", db, cfg, init, stop)
			}
		})

		t.Run("live adapter/"+tc.name, func(t *testing.T) {
			a := NewPostgresAdapter()
			db := lcSeedLive(t, a)
			t.Cleanup(func() { _ = a.Close() })

			err := a.Initialize(tc.config)
			if err == nil {
				t.Fatal("Initialize = nil, want an error")
			}
			if errors.Is(err, ErrAlreadyInitialized) {
				t.Fatalf("config rejection reported as %v; config must be validated before the lifecycle guard", err)
			}
			a.mu.Lock()
			sameDB, cfg, init := a.db == db, a.config, a.initialized
			a.mu.Unlock()
			if !sameDB || cfg == nil || !init {
				t.Errorf("failed Initialize disturbed the live pool: sameDB=%v config=%v initialized=%v", sameDB, cfg, init)
			}
		})
	}
}

// =============================================================================
// Mixed-traffic shakeout
// =============================================================================

func TestLifecycleUnderMixedConcurrentTraffic(t *testing.T) {
	for trial := 0; trial < 10; trial++ {
		a := NewPostgresAdapter()
		lcSeedLive(t, a)

		lcRace(40, func(i int) {
			switch i % 8 {
			case 0, 1:
				_ = a.Close()
			case 2:
				if err := a.Initialize(lcProtoConfig()); err != nil && !errors.Is(err, ErrAlreadyInitialized) {
					// A post-Close Initialize would have to dial; the guard is
					// only expected to report ErrAlreadyInitialized or a dial
					// error, never to panic.
					t.Logf("Initialize during shutdown: %v", err)
				}
			case 3:
				_ = a.PoolStats()
			case 4:
				_ = a.MaxConns()
			case 5:
				_ = a.GetConnection()
			case 6:
				_ = a.IsEnabled()
			case 7:
				_ = a.GetTransactionManager()
			}
		})

		if err := a.Close(); err != nil {
			t.Fatalf("trial %d: final Close: %v", trial, err)
		}
	}
}

func TestIsHealthyAfterCloseReportsNoConnection(t *testing.T) {
	a := NewPostgresAdapter()
	lcSeedLive(t, a)
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	err := a.IsHealthy(context.Background())
	if err == nil {
		t.Fatal("IsHealthy after Close = nil, want an error")
	}
	if got := err.Error(); got != "postgresql connection is nil" {
		t.Errorf("IsHealthy after Close = %q, want %q", got, "postgresql connection is nil")
	}
}

// =============================================================================
// Port compatibility: no new required method, aliases are type-identical
// =============================================================================

// legacyDatabaseProvider implements only the six DatabaseProvider methods that
// predate this workstream. It compiles as a ports.DatabaseProvider only while
// PoolSizer/PoolStatser remain separate optional capabilities.
type legacyDatabaseProvider struct{}

func (legacyDatabaseProvider) Name() string                                      { return "legacy" }
func (legacyDatabaseProvider) Initialize(*dbpb.DatabaseProviderConfig) error     { return nil }
func (legacyDatabaseProvider) GetConnection() any                                { return nil }
func (legacyDatabaseProvider) IsHealthy(context.Context) error                   { return nil }
func (legacyDatabaseProvider) Close() error                                      { return nil }
func (legacyDatabaseProvider) IsEnabled() bool                                   { return false }
func (legacyDatabaseProvider) CreateRepository(string, any, string) (any, error) { return nil, nil }
func (legacyDatabaseProvider) HealthCheck(context.Context) error                 { return nil }

var (
	_ ports.DatabaseProvider   = legacyDatabaseProvider{}
	_ ports.RepositoryProvider = legacyDatabaseProvider{}
	_ ports.DatabaseProvider   = (*PostgresAdapter)(nil)
	_ ports.PoolSizer          = (*PostgresAdapter)(nil)
	_ ports.PoolStatser        = (*PostgresAdapter)(nil)
	_ ports.RepositoryProvider = (*PostgresAdapter)(nil)
)

func TestPortAliasesAreTypeIdentical(t *testing.T) {
	statsType := reflect.TypeOf(ports.PoolStats{})
	if got := statsType.PkgPath(); got != infrastructurePkgPath {
		t.Errorf("ports.PoolStats is declared in %q, want %q — the public alias must not wrap the port type",
			got, infrastructurePkgPath)
	}
	if got := statsType.Name(); got != "PoolStats" {
		t.Errorf("ports.PoolStats resolves to type %q, want %q", got, "PoolStats")
	}

	for name, iface := range map[string]reflect.Type{
		"PoolStatser":      reflect.TypeOf((*ports.PoolStatser)(nil)).Elem(),
		"PoolSizer":        reflect.TypeOf((*ports.PoolSizer)(nil)).Elem(),
		"DatabaseProvider": reflect.TypeOf((*ports.DatabaseProvider)(nil)).Elem(),
	} {
		if got := iface.PkgPath(); got != infrastructurePkgPath {
			t.Errorf("ports.%s is declared in %q, want %q", name, got, infrastructurePkgPath)
		}
		if got := iface.Name(); got != name {
			t.Errorf("ports.%s resolves to type %q, want %q", name, got, name)
		}
		if !reflect.TypeOf((*PostgresAdapter)(nil)).Implements(iface) {
			t.Errorf("*PostgresAdapter does not implement ports.%s", name)
		}
	}

	// The capability's declared return type and the adapter's concrete return
	// type must be the same named struct, not two structurally equal ones.
	portMethod, ok := reflect.TypeOf((*ports.PoolStatser)(nil)).Elem().MethodByName("PoolStats")
	if !ok {
		t.Fatal("ports.PoolStatser has no PoolStats method")
	}
	if got := portMethod.Type.Out(0); got != statsType {
		t.Errorf("ports.PoolStatser.PoolStats returns %v, want %v", got, statsType)
	}
	adapterMethod, ok := reflect.TypeOf((*PostgresAdapter)(nil)).MethodByName("PoolStats")
	if !ok {
		t.Fatal("*PostgresAdapter has no PoolStats method")
	}
	if got := adapterMethod.Type.Out(0); got != statsType {
		t.Errorf("(*PostgresAdapter).PoolStats returns %v, want %v", got, statsType)
	}

	// DatabaseProvider itself gained no method, so the legacy shape still
	// satisfies it at exactly the historical method count.
	if got := reflect.TypeOf((*ports.DatabaseProvider)(nil)).Elem().NumMethod(); got != 6 {
		t.Errorf("ports.DatabaseProvider has %d methods, want 6 — the port is no longer additive", got)
	}
}

func TestPoolStatsMirrorsDBStats(t *testing.T) {
	db := lcPool(t)
	t.Cleanup(func() { _ = db.Close() })

	a := NewPostgresAdapter()
	a.mu.Lock()
	a.db = db
	a.config = lcConfig()
	a.initialized = true
	a.mu.Unlock()

	raw := db.Stats()
	got := a.PoolStats()
	want := ports.PoolStats{
		MaxOpen:           raw.MaxOpenConnections,
		Open:              raw.OpenConnections,
		InUse:             raw.InUse,
		Idle:              raw.Idle,
		WaitCount:         raw.WaitCount,
		WaitDuration:      raw.WaitDuration,
		MaxIdleClosed:     raw.MaxIdleClosed,
		MaxIdleTimeClosed: raw.MaxIdleTimeClosed,
		MaxLifetimeClosed: raw.MaxLifetimeClosed,
	}
	if got != want {
		t.Errorf("PoolStats() = %+v, want %+v", got, want)
	}
}
