//go:build postgresql

// Package postgres is the PostgreSQL adapter's self-registration entry point.
//
// init() registers three things with the espyna registry:
//   - Provider factory (NewPostgresAdapter)
//   - BuildFromEnv builder (reads DATABASE_POSTGRES_* env vars, returns initialized adapter)
//   - TableConfigBuilder (buildPgTableConfig — Q-TABLE-NAMES: table names are entityid
//     constants, the single source; the DATABASE_POSTGRES_TABLE_* per-entity override axis
//     is RETIRED. buildPgTableConfig returns the default config unconditionally and only
//     warns, at boot, if any such env var is still set — it is never read or applied.)
//
// The 145+ entity adapters in subdirectories (entity/, product/, revenue/, etc.)
// each have their own init() that calls registry.RegisterRepositoryFactory to
// register a "postgresql:<entityid>" factory.
//
// Adding a new PostgreSQL entity adapter:
//  1. Create an adapter file in the appropriate subdomain directory.
//  2. Add an init() that calls registry.RegisterRepositoryFactory("postgresql", entityid.X, factory).
//  3. Blank-import the adapter package in the consumer binary so init() fires.
//
// Table name resolution: table names come from registry/entityid constants ONLY
// (Q-TABLE-NAMES, 20260703 table-name-single-source). There is no runtime/env override
// axis; buildPgTableConfig always returns registry.NewDefaultTableConfig().
//
// Import order matters: adapter packages must be blank-imported in the consumer
// binary for their init() registrations to execute.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
	_ "github.com/lib/pq"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterDatabaseProvider(
		"postgresql",
		func() ports.DatabaseProvider {
			return NewPostgresAdapter()
		},
		transformConfig,
	)
	registry.RegisterDatabaseBuildFromEnv("postgresql", buildFromEnv)
	registry.RegisterDatabaseTableConfigBuilder("postgresql", buildPgTableConfig)
	warnDeprecatedTableNameEnvVars()
	// Plan 2 (reflectionless CRUD): register the boot-shot schema validator so the
	// dialect-neutral container can resolve and run it for the postgresql provider
	// without importing this postgresql-tagged package directly. Mirrors the
	// RegisterDatabaseTableConfigBuilder hook above.
	registry.RegisterSchemaValidator("postgresql", core.ValidateSchema)
}

// buildPgTableConfig returns the default table config: table names are the
// registry/entityid constants, unconditionally (Q-TABLE-NAMES). The former
// DATABASE_POSTGRES_TABLE_* per-entity/prefix override axis is retired — this
// function no longer reads any environment variable. See warnDeprecatedTableNameEnvVars.
func buildPgTableConfig() *registry.TableConfig {
	return registry.NewDefaultTableConfig()
}

// warnDeprecatedTableNameEnvVars logs a boot-time warning naming any
// DATABASE_POSTGRES_TABLE_* env var still set in the environment. The retired
// override axis never reads these values — this is an operator signal only
// (fail-closed to the entityid default, never fail-open to an env-supplied name).
func warnDeprecatedTableNameEnvVars() {
	var found []string
	for _, e := range os.Environ() {
		key, _, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if strings.Contains(key, "DATABASE_POSTGRES_TABLE_") {
			found = append(found, key)
		}
	}
	if len(found) == 0 {
		return
	}
	sort.Strings(found)
	log.Printf(
		"WARN postgresql: %d DATABASE_POSTGRES_TABLE_* env var(s) set but IGNORED — the table-name override axis is retired (Q-TABLE-NAMES); table names come from registry/entityid only: %s",
		len(found), strings.Join(found, ", "),
	)
}

// =============================================================================
// Configuration knobs, bounds and the ONE strict parser
// =============================================================================

// Environment keys and bounds for the numeric knobs this adapter resolves.
//
// The three timeout knobs are expressed in SECONDS at every config surface and
// converted to PostgreSQL's millisecond units only when the DSN is assembled.
// Semantics are uniform across the environment path and the raw-map path (D-2):
//
//	absent            → the documented default below
//	0                 → disabled (the setting is omitted from the DSN options)
//	negative/malformed → configuration error naming the key, unit and range
//	> maxTimeoutSeconds → configuration error (unit-mistake guard: someone who
//	                      supplied milliseconds to a _SECONDS key is told so
//	                      instead of silently getting a multi-hour timeout)
const (
	maxTimeoutSeconds = 3600

	envStatementTimeout = "DATABASE_POSTGRES_STATEMENT_TIMEOUT_SECONDS"
	envLockTimeout      = "DATABASE_POSTGRES_LOCK_TIMEOUT_SECONDS"
	envIdleTxTimeout    = "DATABASE_POSTGRES_IDLE_TX_TIMEOUT_SECONDS"

	defaultStatementTimeoutSeconds = 30
	defaultLockTimeoutSeconds      = 10
	defaultIdleTxTimeoutSeconds    = 60

	envMaxConnections     = "DATABASE_POSTGRES_MAX_CONNECTIONS"
	defaultMaxConnections = 25
	maxMaxConnections     = 10_000
)

// timeoutDisabledSentinel encodes an operator's explicit "0 = disabled" choice
// through a proto3 int32 field, where a literal 0 is indistinguishable from
// "never set". Only this file writes it (buildFromEnv / transformConfig) and
// only this file reads it (resolveProtoTimeout) — it is never *interpreted*
// outside the adapter, though the proto transformConfig returns does carry it
// to the registry's transformer contract, so a config dump can legitimately
// show statement_timeout_seconds: -1 meaning "disabled".
//
// Consequence worth knowing (in-band sentinel, unavoidable): a typed-config
// writer outside this file that sets the proto field to -1 for some other
// reason is read as "disabled" rather than rejected, whereas every other
// negative value is a hard error. All in-repo writers are sentinel-aware.
const timeoutDisabledSentinel int32 = -1

// ErrAlreadyInitialized is returned by Initialize when the adapter already
// holds a live pool (D-4). The lifecycle is single-shot per live instance:
// callers that genuinely want to reconfigure must Close() first, which
// releases the pool and joins the monitor goroutine before returning.
var ErrAlreadyInitialized = errors.New("postgresql: adapter is already initialized")

// parseIntValue is THE strict parser behind every numeric knob on every config
// path — environment strings and raw config maps alike. It accepts exactly the
// shapes the central DatabaseConfigAdapter accepts (int, int32, int64, float64
// and numeric string; see internal/application/ports/infrastructure/helpers.go
// getInt32) and returns an actionable error naming the key, the unit and the
// allowed range for everything else. Validation happens before any narrowing
// or unit multiplication, so nothing can wrap or overflow downstream.
//
// It never substitutes a default for a value that was supplied but invalid.
func parseIntValue(key, unit string, raw any, min, max int) (int, error) {
	rangeHint := fmt.Sprintf("allowed range %d-%d %s", min, max, unit)
	if min == 0 {
		rangeHint += " (0 disables)"
	}

	var v int64
	switch t := raw.(type) {
	case int:
		v = int64(t)
	case int32:
		v = int64(t)
	case int64:
		v = t
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || t != math.Trunc(t) {
			return 0, fmt.Errorf("postgresql: %s must be a whole number of %s, got %v; %s", key, unit, t, rangeHint)
		}
		if t < float64(min) || t > float64(max) {
			return 0, fmt.Errorf("postgresql: %s is out of range: %v %s; %s", key, t, unit, rangeHint)
		}
		v = int64(t)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("postgresql: %s must be an integer number of %s, got %q; %s", key, unit, t, rangeHint)
		}
		v = parsed
	default:
		return 0, fmt.Errorf("postgresql: %s must be a number of %s (int, float or numeric string), got %T; %s", key, unit, raw, rangeHint)
	}

	if v < int64(min) || v > int64(max) {
		return 0, fmt.Errorf("postgresql: %s is out of range: %d %s; %s", key, v, unit, rangeHint)
	}
	return int(v), nil
}

// parseSecondsValue is the timeout-flavoured wrapper over parseIntValue: 0 is
// legal and means "disabled", the ceiling is maxTimeoutSeconds.
func parseSecondsValue(key string, raw any) (int, error) {
	return parseIntValue(key, "seconds", raw, 0, maxTimeoutSeconds)
}

// lookupSecondsEnv resolves a seconds-valued environment knob. An unset or
// blank variable yields defaultValue; anything else goes through the strict
// parser, so a malformed or out-of-range value is an error — never a silently
// substituted default.
func lookupSecondsEnv(key string, defaultValue int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	return parseSecondsValue(key, raw)
}

// lookupIntEnv is lookupSecondsEnv's general sibling for non-timeout knobs
// (connection counts), which have their own unit, floor and ceiling.
func lookupIntEnv(key, unit string, defaultValue, min, max int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	return parseIntValue(key, unit, raw, min, max)
}

// rawMapInt resolves a numeric key out of a raw config map through the same
// strict parser. An absent (or explicitly nil) key reports present=false so the
// caller can apply its own default; a present-but-invalid value is an error.
func rawMapInt(m map[string]any, key, unit string, min, max int) (value int, present bool, err error) {
	raw, ok := m[key]
	if !ok || raw == nil {
		return 0, false, nil
	}
	v, perr := parseIntValue(key, unit, raw, min, max)
	if perr != nil {
		return 0, false, perr
	}
	return v, true, nil
}

// rawMapSeconds is rawMapInt with the timeout flavour (0 disables, 3600 cap).
func rawMapSeconds(m map[string]any, key string) (value int, present bool, err error) {
	return rawMapInt(m, key, "seconds", 0, maxTimeoutSeconds)
}

// encodeTimeoutSeconds narrows an already-validated seconds value (0..3600)
// into the proto's int32 field, mapping an explicit 0 to the disabled sentinel.
func encodeTimeoutSeconds(seconds int) int32 {
	if seconds == 0 {
		return timeoutDisabledSentinel
	}
	return int32(seconds)
}

// resolveProtoTimeout resolves one timeout from the typed proto config, falling
// back to its environment knob when the proto carries no value. Precedence:
// typed config wins when it says anything at all (including "disabled"); the
// environment knob is consulted only for the proto's zero value.
func resolveProtoTimeout(protoKey, envKey string, protoVal int32, defaultValue int) (int, error) {
	switch {
	case protoVal == timeoutDisabledSentinel:
		return 0, nil
	case protoVal == 0:
		return lookupSecondsEnv(envKey, defaultValue)
	case protoVal < 0 || int(protoVal) > maxTimeoutSeconds:
		return 0, fmt.Errorf("postgresql: %s is out of range: %d seconds; allowed range 0-%d seconds (0 disables)",
			protoKey, protoVal, maxTimeoutSeconds)
	default:
		return int(protoVal), nil
	}
}

// buildFromEnv creates and initializes a PostgreSQL adapter from environment variables.
func buildFromEnv() (ports.DatabaseProvider, error) {
	host := getEnv("DATABASE_POSTGRES_HOST", "localhost")
	port := getEnv("DATABASE_POSTGRES_PORT", "5432")
	name := getEnv("DATABASE_POSTGRES_DBNAME", "espyna")
	user := getEnv("DATABASE_POSTGRES_USER", "postgres")
	password := getEnv("DATABASE_POSTGRES_PASSWORD", "")
	sslMode := getEnv("DATABASE_POSTGRES_SSLMODE", "disable")

	maxConns, err := lookupIntEnv(envMaxConnections, "connections", defaultMaxConnections, 1, maxMaxConnections)
	if err != nil {
		return nil, err
	}
	statementTimeoutSecs, err := lookupSecondsEnv(envStatementTimeout, defaultStatementTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	if host == "" {
		return nil, fmt.Errorf("postgresql: DATABASE_POSTGRES_HOST is required")
	}
	if user == "" {
		return nil, fmt.Errorf("postgresql: DATABASE_POSTGRES_USER is required")
	}

	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_POSTGRESQL,
		Enabled:  true,
		Config: &dbpb.DatabaseProviderConfig_Postgresql{
			Postgresql: &dbpb.PostgreSQLConfig{
				Host:                    host,
				Port:                    port,
				Database:                name,
				Username:                user,
				Password:                password,
				SslMode:                 sslMode,
				MaxConnections:          int32(maxConns),
				StatementTimeoutSeconds: encodeTimeoutSeconds(statementTimeoutSecs),
			},
		},
	}

	adapter := NewPostgresAdapter()
	if err := adapter.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("postgresql: failed to initialize: %w", err)
	}
	return adapter, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// transformConfig converts raw config map to PostgreSQL proto config.
func transformConfig(rawConfig map[string]any) (*dbpb.DatabaseProviderConfig, error) {
	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_POSTGRESQL,
		Enabled:  true,
	}

	pgConfig := &dbpb.PostgreSQLConfig{}

	if host, ok := rawConfig["host"].(string); ok && host != "" {
		pgConfig.Host = host
	} else {
		return nil, fmt.Errorf("postgresql: host is required")
	}

	switch p := rawConfig["port"].(type) {
	case int:
		pgConfig.Port = fmt.Sprintf("%d", p)
	case string:
		pgConfig.Port = p
	default:
		pgConfig.Port = "5432"
	}

	if name, ok := rawConfig["name"].(string); ok && name != "" {
		pgConfig.Database = name
	} else if name, ok := rawConfig["database"].(string); ok && name != "" {
		pgConfig.Database = name
	} else {
		return nil, fmt.Errorf("postgresql: name/database is required")
	}

	if user, ok := rawConfig["user"].(string); ok && user != "" {
		pgConfig.Username = user
	} else if user, ok := rawConfig["username"].(string); ok && user != "" {
		pgConfig.Username = user
	} else {
		return nil, fmt.Errorf("postgresql: user/username is required")
	}

	if password, ok := rawConfig["password"].(string); ok {
		pgConfig.Password = password
	} else {
		return nil, fmt.Errorf("postgresql: password is required")
	}

	if sslMode, ok := rawConfig["ssl_mode"].(string); ok && sslMode != "" {
		pgConfig.SslMode = sslMode
	} else {
		pgConfig.SslMode = "disable"
	}

	// Numeric knobs go through the ONE strict parser: the same int/int32/int64/
	// float64/numeric-string shapes DatabaseConfigAdapter accepts, with present-
	// but-invalid values rejected instead of silently dropped to a default.
	if v, present, err := rawMapInt(rawConfig, "max_connections", "connections", 1, maxMaxConnections); err != nil {
		return nil, err
	} else if present {
		pgConfig.MaxConnections = int32(v)
	}

	if v, present, err := rawMapInt(rawConfig, "max_idle_connections", "connections", 1, maxMaxConnections); err != nil {
		return nil, err
	} else if present {
		pgConfig.MaxIdleConnections = int32(v)
	}

	if v, present, err := rawMapSeconds(rawConfig, "statement_timeout_seconds"); err != nil {
		return nil, err
	} else if present {
		pgConfig.StatementTimeoutSeconds = encodeTimeoutSeconds(v)
	}

	protoConfig.Config = &dbpb.DatabaseProviderConfig_Postgresql{
		Postgresql: pgConfig,
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// PostgresAdapter implements DatabaseProvider and RepositoryProvider for PostgreSQL.
// This adapter follows the same self-registration pattern as Firestore/Mock.
//
// Lifecycle state (db, config, enabled, connected, initialized, stopMonitor) is
// guarded by mu; the monitor goroutine is joined through monitorWG so Close
// returns only once it has exited. Initialize is single-shot per live instance
// (see ErrAlreadyInitialized); Close is idempotent and safe under concurrent
// callers.
//
// Locking rule (load-bearing): Close holds mu across monitorWG.Wait(), and
// Initialize holds it across the dial+Ping. That is deadlock-free only because
// every monitorWG participant is a *free function* (monitorPool) that never
// touches the adapter and so can never need mu. If the monitor is ever turned
// into a method that reads adapter state, it must take a snapshot at launch
// instead of locking, or Close will self-deadlock.
type PostgresAdapter struct {
	mu          sync.Mutex
	db          *sql.DB
	config      *PostgresConfig
	enabled     bool
	connected   bool
	initialized bool
	stopMonitor chan struct{}
	monitorWG   sync.WaitGroup

	// monitorsRunning counts pool-monitor goroutines that have started and not
	// yet returned. It is the machine-checkable form of Close's contract ("no
	// post-close tick is possible"): attach increments it under mu, the
	// goroutine decrements it before its monitorWG.Done, so it is guaranteed
	// to read 0 once Close has completed its join. Observability only — no
	// production control flow reads it.
	monitorsRunning atomic.Int64
}

// PostgresConfig holds the resolved, already-validated PostgreSQL settings.
// Every numeric field here has passed the strict parser: timeouts are 0
// (disabled) to maxTimeoutSeconds, MaxIdleConns is 1..MaxConns.
type PostgresConfig struct {
	Host                    string
	Port                    string
	Name                    string
	User                    string
	Password                string
	SSLMode                 string
	MaxConns                int
	MaxIdleConns            int
	StatementTimeoutSeconds int
	LockTimeoutSeconds      int
	IdleTxTimeoutSeconds    int
	MigrationsPath          string
}

// NewPostgresAdapter creates a new PostgreSQL database adapter.
func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{
		enabled: true,
	}
}

// Name returns the provider name.
func (a *PostgresAdapter) Name() string {
	return "postgresql"
}

// resolvePostgresConfig turns the typed proto config plus the environment-only
// knobs into a fully validated PostgresConfig, or an error. It touches no
// adapter state and opens no connection, so Initialize can reject a bad config
// before mutating anything.
//
// The lock and idle-in-transaction timeouts have no proto field; the
// environment is their only channel (DATABASE_POSTGRES_LOCK_TIMEOUT_SECONDS,
// DATABASE_POSTGRES_IDLE_TX_TIMEOUT_SECONDS).
func resolvePostgresConfig(pgProto *dbpb.PostgreSQLConfig) (*PostgresConfig, error) {
	cfg := &PostgresConfig{
		Host:           pgProto.GetHost(),
		Port:           pgProto.GetPort(),
		Name:           pgProto.GetDatabase(),
		User:           pgProto.GetUsername(),
		Password:       pgProto.GetPassword(),
		SSLMode:        pgProto.GetSslMode(),
		MigrationsPath: "./migrations",
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	switch mc := pgProto.GetMaxConnections(); {
	case mc == 0:
		cfg.MaxConns = defaultMaxConnections
	case mc < 0 || int(mc) > maxMaxConnections:
		return nil, fmt.Errorf("postgresql: max_connections is out of range: %d connections; allowed range 1-%d connections",
			mc, maxMaxConnections)
	default:
		cfg.MaxConns = int(mc)
	}

	// D-3: idle capacity defaults to the open cap (time-based pruning via
	// ConnMaxIdleTime keeps a quiet pool from holding server slots), but the
	// existing typed knob is honored whenever an operator sets it.
	switch mi := pgProto.GetMaxIdleConnections(); {
	case mi == 0:
		cfg.MaxIdleConns = cfg.MaxConns
	case mi < 0 || int(mi) > cfg.MaxConns:
		return nil, fmt.Errorf("postgresql: max_idle_connections is out of range: %d connections; allowed range 1-%d (must not exceed max_connections=%d)",
			mi, cfg.MaxConns, cfg.MaxConns)
	default:
		cfg.MaxIdleConns = int(mi)
	}

	statementTimeout, err := resolveProtoTimeout("statement_timeout_seconds", envStatementTimeout,
		pgProto.GetStatementTimeoutSeconds(), defaultStatementTimeoutSeconds)
	if err != nil {
		return nil, err
	}
	cfg.StatementTimeoutSeconds = statementTimeout

	lockTimeout, err := lookupSecondsEnv(envLockTimeout, defaultLockTimeoutSeconds)
	if err != nil {
		return nil, err
	}
	cfg.LockTimeoutSeconds = lockTimeout

	idleTxTimeout, err := lookupSecondsEnv(envIdleTxTimeout, defaultIdleTxTimeoutSeconds)
	if err != nil {
		return nil, err
	}
	cfg.IdleTxTimeoutSeconds = idleTxTimeout

	return cfg, nil
}

// pgKV renders one libpq keyword/value pair for a connection string, quoting
// and escaping the value per the libpq rules: backslashes and single quotes are
// escaped with a backslash, and the value is wrapped in single quotes when it
// is empty, contains a quote, or contains any whitespace rune.
//
// The whitespace predicate is deliberately unicode.IsSpace, not the ASCII set.
// An unquoted value ends at the first character the *parser* considers
// whitespace, and the parser in use is the driver's, not libpq's: lib/pq's
// parseOpts terminates keywords and unquoted values on unicode.IsSpace
// (conn.go, v1.10.9), so U+00A0, U+2028, U+3000 and friends split a value there
// even though C libpq's isspace() would not. unicode.IsSpace is a superset of
// the ASCII set, so quoting on it is correct for both parsers.
//
// Every interpolated DSN field goes through this helper — including the
// password and the assembled options payload — so a value carrying a space (of
// any kind), quote or backslash cannot terminate its own field or introduce a
// further connection keyword. See adapter_dsn_test.go, which asserts this
// against the real lib/pq parser (pq.NewConnector), not only against the test's
// reference parser.
func pgKV(key, value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value)
	if value == "" || strings.ContainsRune(value, '\'') || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return key + "='" + escaped + "'"
	}
	return key + "=" + escaped
}

// sessionOptions renders the startup `options` payload carrying the three
// per-session timeouts, converting the validated seconds values to
// PostgreSQL's millisecond units. A timeout configured as 0 is disabled and is
// simply omitted; when all three are disabled the payload is empty and the
// caller drops the `options` keyword entirely.
//
// These are connection-session DEFAULTS, not enforced boundaries: they are the
// initial value of each GUC on every physical pooled connection, and any
// consumer holding the raw *sql.DB (see GetConnection) can SET past them.
func sessionOptions(cfg *PostgresConfig) string {
	var opts []string
	if cfg.StatementTimeoutSeconds > 0 {
		opts = append(opts, fmt.Sprintf("-c statement_timeout=%d", cfg.StatementTimeoutSeconds*1000))
	}
	if cfg.IdleTxTimeoutSeconds > 0 {
		opts = append(opts, fmt.Sprintf("-c idle_in_transaction_session_timeout=%d", cfg.IdleTxTimeoutSeconds*1000))
	}
	if cfg.LockTimeoutSeconds > 0 {
		opts = append(opts, fmt.Sprintf("-c lock_timeout=%d", cfg.LockTimeoutSeconds*1000))
	}
	return strings.Join(opts, " ")
}

// buildDSN assembles the libpq keyword/value connection string. Every value is
// rendered through pgKV.
func buildDSN(cfg *PostgresConfig) string {
	connParts := []string{
		pgKV("host", cfg.Host),
		pgKV("port", cfg.Port),
		pgKV("dbname", cfg.Name),
		pgKV("user", cfg.User),
		pgKV("sslmode", cfg.SSLMode),
		pgKV("connect_timeout", "5"),
	}
	if opts := sessionOptions(cfg); opts != "" {
		connParts = append(connParts, pgKV("options", opts))
	}
	if cfg.Password != "" {
		connParts = append(connParts, pgKV("password", cfg.Password))
	}
	return strings.Join(connParts, " ")
}

// secondsLabel renders a timeout for the boot log: "disabled" or "<n>s".
func secondsLabel(seconds int) string {
	if seconds <= 0 {
		return "disabled"
	}
	return strconv.Itoa(seconds) + "s"
}

// Initialize sets up the PostgreSQL connection.
//
// It is single-shot per live adapter: calling it on an adapter that already
// holds a pool returns ErrAlreadyInitialized rather than orphaning the previous
// pool and its monitor goroutine (D-4). Close() releases the adapter and makes
// it initializable again.
//
// Adapter state is mutated only after a successful Ping; a configuration,
// dial or ping failure leaves db/config/enabled/connected exactly as they were.
func (a *PostgresAdapter) Initialize(config *dbpb.DatabaseProviderConfig) error {
	pgProto := config.GetPostgresql()
	if pgProto == nil {
		return fmt.Errorf("postgresql adapter requires postgresql configuration")
	}

	pgConfig, err := resolvePostgresConfig(pgProto)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return ErrAlreadyInitialized
	}

	db, err := sql.Open("postgres", buildDSN(pgConfig))
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Idle capacity defaults to the open cap: a lower idle ceiling forces
	// constant close/re-dial churn under sustained concurrent load, and every
	// re-dial pays a TCP + auth handshake (the historical source of auth
	// flaking). Idle cleanup is time-based instead — ConnMaxIdleTime prunes a
	// pool that has actually gone quiet, so nothing stays warm on an idle
	// server. Operators who need a smaller footprint set max_idle_connections.
	db.SetMaxOpenConns(pgConfig.MaxConns)
	db.SetMaxIdleConns(pgConfig.MaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	a.attach(db, pgConfig, config.Enabled)

	log.Printf("✅ PostgreSQL adapter connected to %s:%s/%s (pool max=%d idle=%d, statement_timeout=%s lock_timeout=%s idle_in_transaction_session_timeout=%s)",
		pgConfig.Host, pgConfig.Port, pgConfig.Name, pgConfig.MaxConns, pgConfig.MaxIdleConns,
		secondsLabel(pgConfig.StatementTimeoutSeconds), secondsLabel(pgConfig.LockTimeoutSeconds),
		secondsLabel(pgConfig.IdleTxTimeoutSeconds))
	return nil
}

// attach makes db the adapter's live pool: it publishes the resolved state and
// starts the pool monitor as a monitorWG participant so Close can join it.
//
// The caller must hold mu and must already have rejected a live adapter
// (a.initialized) and completed the Ping — attach is the point of no return,
// after which PoolStats/GetConnection/IsHealthy see the new pool. Registering
// the monitor on monitorWG *before* the goroutine starts, and doing so under
// the same mu the detach in Close takes, is what makes the join race-free.
//
// It exists as a single block so that the ONE go-live sequence is shared by
// Initialize and by the lifecycle tests' fixture, rather than duplicated in a
// test where an ordering regression here would go unnoticed.
func (a *PostgresAdapter) attach(db *sql.DB, cfg *PostgresConfig, enabled bool) {
	stop := make(chan struct{})
	a.db = db
	a.config = cfg
	a.enabled = enabled
	a.connected = true
	a.initialized = true
	a.stopMonitor = stop
	a.monitorWG.Add(1)
	a.monitorsRunning.Add(1)
	go func() {
		defer a.monitorWG.Done()
		defer a.monitorsRunning.Add(-1)
		monitorPool(db, stop)
	}()
}

// MaxConns returns the effective max-open-connections cap configured on the
// underlying *sql.DB pool. Implements the optional ports.PoolSizer capability
// so concurrency-sensitive callers can clamp their fanout to the pool budget.
//
// Returns 0 when Initialize has not been called or the adapter is in a zero
// state; callers should treat 0 as "unknown" and fall back to a conservative
// default rather than dividing by it.
func (a *PostgresAdapter) MaxConns() int {
	if a == nil {
		return 0
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.config == nil {
		return 0
	}
	return a.config.MaxConns
}

// PoolStats returns a driver-neutral snapshot of the pool. Implements the
// optional ports.PoolStatser capability so observability surfaces (debug
// endpoints, load investigations) can read pool health without importing
// database/sql. Returns the zero value before Initialize / after Close.
//
// Safe to call concurrently with Initialize/Close: the *sql.DB pointer is
// snapshotted under the adapter mutex and sql.DB.Stats is itself
// concurrency-safe, so a pool being closed underneath simply reports its
// final counters.
func (a *PostgresAdapter) PoolStats() ports.PoolStats {
	if a == nil {
		return ports.PoolStats{}
	}
	a.mu.Lock()
	db := a.db
	a.mu.Unlock()
	if db == nil {
		return ports.PoolStats{}
	}
	s := db.Stats()
	return ports.PoolStats{
		MaxOpen:           s.MaxOpenConnections,
		Open:              s.OpenConnections,
		InUse:             s.InUse,
		Idle:              s.Idle,
		WaitCount:         s.WaitCount,
		WaitDuration:      s.WaitDuration,
		MaxIdleClosed:     s.MaxIdleClosed,
		MaxIdleTimeClosed: s.MaxIdleTimeClosed,
		MaxLifetimeClosed: s.MaxLifetimeClosed,
	}
}

// monitorPool logs a pool line whenever callers had to wait for a connection
// since the previous sample — i.e. when pool saturation was observed. A wait
// means callers reached the configured cap and blocked; the cause may be an
// undersized cap, but equally slow queries, lock contention, a slow server or
// leaked transactions, so this signal must be correlated with query/lock
// latency before any cap is raised.
//
// Both the interval delta and the cumulative totals come from a single
// db.Stats() snapshot, and the baseline advances from that same snapshot, so
// the cumulative figures are always exact even though the interval split of a
// wait spanning two samples is approximate (database/sql increments WaitCount
// when a wait begins but adds WaitDuration only when it ends).
//
// Runs until stop is closed (adapter Close, which joins this goroutine).
func monitorPool(db *sql.DB, stop <-chan struct{}) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	var lastWaitCount int64
	var lastWaitDuration time.Duration
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s := db.Stats()
			deltaCount := s.WaitCount - lastWaitCount
			deltaDuration := s.WaitDuration - lastWaitDuration
			lastWaitCount = s.WaitCount
			lastWaitDuration = s.WaitDuration
			if deltaCount <= 0 {
				continue
			}
			log.Printf("WARN postgresql pool saturation observed: %d caller(s) waited %s for a connection in the last interval; cumulative %d wait(s) totalling %s (open=%d inUse=%d idle=%d cap=%d)",
				deltaCount, deltaDuration, s.WaitCount, s.WaitDuration,
				s.OpenConnections, s.InUse, s.Idle, s.MaxOpenConnections)
		}
	}
}

// GetConnection returns the PostgreSQL database connection.
//
// This hands out the raw *sql.DB: holders can run anything on it, including
// SET statement_timeout / RESET ALL, which is why the DSN timeouts are session
// defaults rather than enforced limits.
func (a *PostgresAdapter) GetConnection() any {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.db
}

// Close closes the PostgreSQL connection.
//
// Idempotent and safe under concurrent callers: the stop channel and the pool
// are detached under the adapter mutex exactly once, so no second caller can
// close either twice. Close returns only after the monitor goroutine has
// actually exited (monitorWG join), so no post-close tick or log is possible.
// A closed adapter is back to its uninitialized state and may be Initialized
// again.
//
// "Uninitialized" here means exactly what NewPostgresAdapter produces, enabled
// flag included: enabled is a provider-configuration bit (config.Enabled), not
// a connection bit, so IsEnabled keeps reporting true after Close just as it
// does on a fresh adapter. Liveness is IsHealthy's answer, not IsEnabled's.
func (a *PostgresAdapter) Close() error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	stop := a.stopMonitor
	db := a.db
	a.stopMonitor = nil
	a.db = nil
	a.config = nil
	a.connected = false
	a.initialized = false

	if stop != nil {
		close(stop)
	}
	// Join the monitor before releasing the pool it reads.
	a.monitorWG.Wait()

	if db == nil {
		return nil
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
	}
	log.Println("✅ PostgreSQL adapter closed")
	return nil
}

// IsHealthy checks if the PostgreSQL connection is healthy.
func (a *PostgresAdapter) IsHealthy(ctx context.Context) error {
	if a == nil {
		return fmt.Errorf("postgresql adapter is nil")
	}

	a.mu.Lock()
	enabled := a.enabled
	db := a.db
	a.mu.Unlock()

	if !enabled {
		return fmt.Errorf("postgresql adapter is disabled")
	}
	if db == nil {
		return fmt.Errorf("postgresql connection is nil")
	}

	err := db.PingContext(ctx)

	// Only report on the pool we actually pinged: a concurrent Close or a later
	// Initialize must not have its state stomped by this result.
	a.mu.Lock()
	if a.db == db {
		a.connected = err == nil
	}
	a.mu.Unlock()

	if err != nil {
		return fmt.Errorf("postgresql health check failed: %w", err)
	}
	return nil
}

// IsEnabled returns whether this adapter is currently enabled.
func (a *PostgresAdapter) IsEnabled() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.enabled
}

// =============================================================================
// RepositoryProvider Implementation - Delegates to Registry
// =============================================================================

// CreateRepository creates a repository by looking up the registered factory.
// This replaces the giant switch statement by delegating to self-registered factories.
func (a *PostgresAdapter) CreateRepository(entityName string, conn any, tableName string) (any, error) {
	return registry.CreateRepository("postgresql", entityName, conn, tableName)
}

// GetTransactionManager returns the PostgreSQL transaction manager.
func (a *PostgresAdapter) GetTransactionManager() interfaces.TransactionManager {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	db := a.db
	connected := a.connected
	a.mu.Unlock()

	if db == nil || !connected {
		return nil
	}
	return core.NewPostgreSQLTransactionManager(db)
}

// HealthCheck checks if the PostgreSQL adapter is healthy.
func (a *PostgresAdapter) HealthCheck(ctx context.Context) error {
	return a.IsHealthy(ctx)
}

// Compile-time interface checks
var _ ports.DatabaseProvider = (*PostgresAdapter)(nil)
var _ ports.PoolSizer = (*PostgresAdapter)(nil)
var _ ports.PoolStatser = (*PostgresAdapter)(nil)
var _ ports.RepositoryProvider = (*PostgresAdapter)(nil)
