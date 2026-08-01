//go:build postgresql

package postgres

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
)

// =============================================================================
// Env fixtures
// =============================================================================

// cfgEnvKeys is every environment knob this adapter reads for numeric config.
var cfgEnvKeys = []string{envStatementTimeout, envLockTimeout, envIdleTxTimeout, envMaxConnections}

// cfgSetEnv sets one key for the duration of the test, restoring whatever was
// there before (including "not set at all").
func cfgSetEnv(t *testing.T, key, value string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("setenv %s: %v", key, err)
	}
}

// cfgUnsetEnv removes one key for the duration of the test.
func cfgUnsetEnv(t *testing.T, key string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetenv %s: %v", key, err)
	}
}

// cfgCleanEnv removes every numeric knob so a test starts from "operator set
// nothing" regardless of the ambient shell.
func cfgCleanEnv(t *testing.T) {
	t.Helper()
	for _, k := range cfgEnvKeys {
		cfgUnsetEnv(t, k)
	}
}

// =============================================================================
// The shared value-shape table (D-2)
// =============================================================================

type shapeCase struct {
	name string
	raw  any
	// str is the same value as an environment string, or "" when the shape has
	// no string spelling (float64/int32/bool/nil are map-only shapes).
	str     string
	want    int
	wantErr bool
	// errNeeds are substrings the rejection message must contain.
	errNeeds []string
}

// secondsShapes exercises every accepted and rejected shape for a seconds knob:
// 0 disables, the ceiling is maxTimeoutSeconds, and nothing supplied-but-invalid
// is ever replaced by a default.
var secondsShapes = []shapeCase{
	{name: "int", raw: 45, str: "45", want: 45},
	{name: "int32", raw: int32(45), want: 45},
	{name: "int64", raw: int64(45), want: 45},
	{name: "float64 integral", raw: float64(45), want: 45},
	{name: "numeric string", raw: "45", str: "45", want: 45},
	{name: "numeric string padded", raw: "  45\t", str: "  45\t", want: 45},
	{name: "explicit zero disables", raw: 0, str: "0", want: 0},
	{name: "explicit zero string disables", raw: "0", str: "0", want: 0},
	{name: "lower boundary", raw: 1, str: "1", want: 1},
	{name: "upper boundary", raw: maxTimeoutSeconds, str: "3600", want: maxTimeoutSeconds},

	{name: "negative int", raw: -1, str: "-1", wantErr: true, errNeeds: []string{"out of range", "0-3600 seconds", "0 disables"}},
	{name: "negative string", raw: "-30", str: "-30", wantErr: true, errNeeds: []string{"out of range"}},
	{name: "one past ceiling", raw: maxTimeoutSeconds + 1, str: "3601", wantErr: true, errNeeds: []string{"out of range", "0-3600 seconds"}},
	{name: "milliseconds unit mistake", raw: 30000, str: "30000", wantErr: true, errNeeds: []string{"out of range", "0-3600 seconds"}},
	{name: "malformed word", raw: "thirty", str: "thirty", wantErr: true, errNeeds: []string{"must be an integer number of seconds"}},
	{name: "malformed trailing unit", raw: "30s", str: "30s", wantErr: true, errNeeds: []string{"must be an integer number of seconds"}},
	{name: "malformed hex", raw: "0x1e", str: "0x1e", wantErr: true, errNeeds: []string{"must be an integer number of seconds"}},
	{name: "overflow int64 max", raw: int64(math.MaxInt64), str: "9223372036854775807", wantErr: true, errNeeds: []string{"out of range"}},
	{name: "overflow int max", raw: math.MaxInt, wantErr: true, errNeeds: []string{"out of range"}},
	{name: "overflow past int64", raw: "99999999999999999999", str: "99999999999999999999", wantErr: true, errNeeds: []string{"must be an integer number of seconds"}},
	{name: "overflow int32 max", raw: int32(math.MaxInt32), str: "2147483647", wantErr: true, errNeeds: []string{"out of range"}},
	{name: "float64 max", raw: math.MaxFloat64, wantErr: true, errNeeds: []string{"out of range"}},
	{name: "float64 fractional", raw: 45.5, wantErr: true, errNeeds: []string{"whole number of seconds"}},
	{name: "float64 NaN", raw: math.NaN(), wantErr: true, errNeeds: []string{"whole number of seconds"}},
	{name: "float64 +Inf", raw: math.Inf(1), wantErr: true, errNeeds: []string{"whole number of seconds"}},
	{name: "float64 -Inf", raw: math.Inf(-1), wantErr: true, errNeeds: []string{"whole number of seconds"}},
	{name: "bool", raw: true, wantErr: true, errNeeds: []string{"int, float or numeric string"}},
	{name: "uint", raw: uint(45), wantErr: true, errNeeds: []string{"int, float or numeric string"}},
	{name: "slice", raw: []int{45}, wantErr: true, errNeeds: []string{"int, float or numeric string"}},
}

// connectionShapes exercises a connection-count knob, where 0 is NOT a
// sanctioned disable and the floor is 1.
var connectionShapes = []shapeCase{
	{name: "int", raw: 10, str: "10", want: 10},
	{name: "int32", raw: int32(10), want: 10},
	{name: "int64", raw: int64(10), want: 10},
	{name: "float64 integral", raw: float64(10), want: 10},
	{name: "numeric string", raw: "10", str: "10", want: 10},
	{name: "lower boundary", raw: 1, str: "1", want: 1},
	{name: "upper boundary", raw: maxMaxConnections, str: "10000", want: maxMaxConnections},

	{name: "zero is not a disable", raw: 0, str: "0", wantErr: true, errNeeds: []string{"out of range", "1-10000 connections"}},
	{name: "negative", raw: -5, str: "-5", wantErr: true, errNeeds: []string{"out of range"}},
	{name: "one past ceiling", raw: maxMaxConnections + 1, str: "10001", wantErr: true, errNeeds: []string{"out of range"}},
	{name: "malformed", raw: "many", str: "many", wantErr: true, errNeeds: []string{"must be an integer number of connections"}},
	{name: "overflow int64 max", raw: int64(math.MaxInt64), str: "9223372036854775807", wantErr: true, errNeeds: []string{"out of range"}},
	{name: "float64 fractional", raw: 10.5, wantErr: true, errNeeds: []string{"whole number of connections"}},
	{name: "bool", raw: false, wantErr: true, errNeeds: []string{"int, float or numeric string"}},
}

// checkShape asserts one parse result against a table row, including that the
// rejection message names the key.
func checkShape(t *testing.T, key string, tc shapeCase, got int, err error) {
	t.Helper()
	if tc.wantErr {
		if err == nil {
			t.Fatalf("%s = %d, want an error", key, got)
		}
		msg := err.Error()
		if !strings.Contains(msg, key) {
			t.Errorf("error does not name the key %q: %s", key, msg)
		}
		for _, need := range tc.errNeeds {
			if !strings.Contains(msg, need) {
				t.Errorf("error missing %q: %s", need, msg)
			}
		}
		return
	}
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", key, err)
	}
	if got != tc.want {
		t.Errorf("%s = %d, want %d", key, got, tc.want)
	}
}

// =============================================================================
// The strict parser itself
// =============================================================================

func TestParseSecondsValueShapes(t *testing.T) {
	for _, tc := range secondsShapes {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSecondsValue(envStatementTimeout, tc.raw)
			checkShape(t, envStatementTimeout, tc, got, err)
		})
	}
}

func TestParseIntValueConnectionShapes(t *testing.T) {
	for _, tc := range connectionShapes {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIntValue(envMaxConnections, "connections", tc.raw, 1, maxMaxConnections)
			checkShape(t, envMaxConnections, tc, got, err)
		})
	}
}

func TestRangeHintOnlyAdvertisesDisableWhenZeroIsLegal(t *testing.T) {
	_, secErr := parseSecondsValue(envStatementTimeout, "bogus")
	if secErr == nil || !strings.Contains(secErr.Error(), "(0 disables)") {
		t.Errorf("seconds knob must advertise the disable value: %v", secErr)
	}
	_, connErr := parseIntValue(envMaxConnections, "connections", "bogus", 1, maxMaxConnections)
	if connErr == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(connErr.Error(), "0 disables") {
		t.Errorf("connection knob must not advertise 0 as a disable: %v", connErr)
	}
}

// =============================================================================
// Environment path
// =============================================================================

func TestLookupSecondsEnvShapes(t *testing.T) {
	for _, key := range []string{envStatementTimeout, envLockTimeout, envIdleTxTimeout} {
		t.Run(key, func(t *testing.T) {
			for _, tc := range secondsShapes {
				if tc.str == "" {
					continue // shape has no environment-string spelling
				}
				t.Run(tc.name, func(t *testing.T) {
					cfgSetEnv(t, key, tc.str)
					got, err := lookupSecondsEnv(key, 30)
					checkShape(t, key, tc, got, err)
				})
			}
		})
	}
}

func TestLookupSecondsEnvAbsentUsesDefault(t *testing.T) {
	const key = envStatementTimeout
	cases := []struct {
		name  string
		unset bool
		value string
	}{
		{name: "unset", unset: true},
		{name: "empty", value: ""},
		{name: "blank", value: "   "},
		{name: "tab", value: "\t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unset {
				cfgUnsetEnv(t, key)
			} else {
				cfgSetEnv(t, key, tc.value)
			}
			got, err := lookupSecondsEnv(key, defaultStatementTimeoutSeconds)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != defaultStatementTimeoutSeconds {
				t.Errorf("= %d, want the default %d", got, defaultStatementTimeoutSeconds)
			}
		})
	}
}

func TestLookupIntEnvShapes(t *testing.T) {
	for _, tc := range connectionShapes {
		if tc.str == "" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			cfgSetEnv(t, envMaxConnections, tc.str)
			got, err := lookupIntEnv(envMaxConnections, "connections", defaultMaxConnections, 1, maxMaxConnections)
			checkShape(t, envMaxConnections, tc, got, err)
		})
	}
}

// =============================================================================
// Raw-map path
// =============================================================================

func TestRawMapSecondsShapes(t *testing.T) {
	const key = "statement_timeout_seconds"
	for _, tc := range secondsShapes {
		t.Run(tc.name, func(t *testing.T) {
			got, present, err := rawMapSeconds(map[string]any{key: tc.raw}, key)
			if !tc.wantErr && !present {
				t.Fatalf("a supplied key reported present=false")
			}
			checkShape(t, key, tc, got, err)
		})
	}
}

func TestRawMapAbsentAndNilReportNotPresent(t *testing.T) {
	const key = "statement_timeout_seconds"
	for _, tc := range []struct {
		name string
		m    map[string]any
	}{
		{"absent", map[string]any{}},
		{"explicit nil", map[string]any{key: nil}},
		{"nil map", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, present, err := rawMapSeconds(tc.m, key)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if present {
				t.Errorf("present = true, want false")
			}
			if got != 0 {
				t.Errorf("value = %d, want 0", got)
			}
		})
	}
}

func cfgRawBase() map[string]any {
	return map[string]any{"host": "h", "database": "d", "user": "u", "password": "p"}
}

func cfgRawWith(key string, value any) map[string]any {
	m := cfgRawBase()
	m[key] = value
	return m
}

func TestTransformConfigStatementTimeoutShapes(t *testing.T) {
	const key = "statement_timeout_seconds"
	for _, tc := range secondsShapes {
		t.Run(tc.name, func(t *testing.T) {
			proto, err := transformConfig(cfgRawWith(key, tc.raw))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("transformConfig = %v, want an error", proto.GetPostgresql().GetStatementTimeoutSeconds())
				}
				if !strings.Contains(err.Error(), key) {
					t.Errorf("error does not name the key: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			pg := proto.GetPostgresql()
			if tc.want == 0 {
				// An explicit 0 has no in-band proto encoding: the field stays
				// unset and "disabled" travels as the typed mark.
				if got := pg.GetStatementTimeoutSeconds(); got != 0 {
					t.Fatalf("explicit 0 must leave the proto field unset, got %d", got)
				}
				if !statementTimeoutDisabledMark(pg) {
					t.Fatal("explicit 0 did not set the typed disabled mark")
				}
				// The typed channel must win at resolution even when the
				// environment says otherwise.
				cfgCleanEnv(t)
				cfgSetEnv(t, envStatementTimeout, "77")
				cfg, rerr := resolvePostgresConfig(pg)
				if rerr != nil {
					t.Fatalf("resolve of a disabled-marked proto: %v", rerr)
				}
				if !cfg.StatementTimeout.disabled || cfg.StatementTimeout.enabled() {
					t.Errorf("raw-map 0 resolved to %+v, want the typed disabled state", cfg.StatementTimeout)
				}
				return
			}
			if got := pg.GetStatementTimeoutSeconds(); got != int32(tc.want) {
				t.Errorf("StatementTimeoutSeconds = %d, want %d", got, tc.want)
			}
			if statementTimeoutDisabledMark(pg) {
				t.Error("a non-zero timeout must not carry the disabled mark")
			}
		})
	}
}

func TestTransformConfigConnectionShapes(t *testing.T) {
	for _, key := range []string{"max_connections", "max_idle_connections"} {
		t.Run(key, func(t *testing.T) {
			for _, tc := range connectionShapes {
				t.Run(tc.name, func(t *testing.T) {
					proto, err := transformConfig(cfgRawWith(key, tc.raw))
					if tc.wantErr {
						if err == nil {
							t.Fatal("transformConfig = nil error, want a rejection")
						}
						if !strings.Contains(err.Error(), key) {
							t.Errorf("error does not name the key: %v", err)
						}
						return
					}
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					pg := proto.GetPostgresql()
					got := pg.GetMaxConnections()
					if key == "max_idle_connections" {
						got = pg.GetMaxIdleConnections()
					}
					if got != int32(tc.want) {
						t.Errorf("%s = %d, want %d", key, got, tc.want)
					}
				})
			}
		})
	}
}

func TestTransformConfigLeavesUnsetNumericKnobsAtProtoZero(t *testing.T) {
	proto, err := transformConfig(cfgRawBase())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pg := proto.GetPostgresql()
	if pg.GetMaxConnections() != 0 || pg.GetMaxIdleConnections() != 0 || pg.GetStatementTimeoutSeconds() != 0 {
		t.Errorf("unset knobs should stay at proto zero so resolve applies defaults: %+v", pg)
	}
}

// =============================================================================
// resolvePostgresConfig — the single place every knob is finalised
// =============================================================================

func TestResolveStatementTimeoutPrecedence(t *testing.T) {
	cases := []struct {
		name     string
		protoVal int32
		env      *string
		want     int
		wantErr  bool
	}{
		{name: "proto unset, env unset, default applies", protoVal: 0, want: defaultStatementTimeoutSeconds},
		{name: "proto unset, env wins", protoVal: 0, env: strptr("77"), want: 77},
		{name: "proto unset, env disables", protoVal: 0, env: strptr("0"), want: 0},
		{name: "proto unset, env malformed errors", protoVal: 0, env: strptr("nope"), wantErr: true},
		{name: "proto unset, env out of range errors", protoVal: 0, env: strptr("3601"), wantErr: true},
		{name: "proto value wins over env", protoVal: 45, env: strptr("77"), want: 45},
		{name: "proto -1 (former sentinel, bridge-overflow shape) errors", protoVal: -1, env: strptr("77"), wantErr: true},
		{name: "proto upper boundary", protoVal: maxTimeoutSeconds, want: maxTimeoutSeconds},
		{name: "proto past ceiling errors", protoVal: maxTimeoutSeconds + 1, wantErr: true},
		{name: "proto max int32 (large-positive overflow shape) errors", protoVal: math.MaxInt32, wantErr: true},
		{name: "proto negative errors", protoVal: -2, wantErr: true},
		{name: "proto min int32 (truncation shape) errors", protoVal: math.MinInt32, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgCleanEnv(t)
			if tc.env != nil {
				cfgSetEnv(t, envStatementTimeout, *tc.env)
			}
			cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{
				Host: "h", StatementTimeoutSeconds: tc.protoVal,
			})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolvePostgresConfig = %+v, want an error", cfg)
				}
				// The rejection must name the knob whichever surface supplied
				// it: the proto field (lower case) or its env key (upper case).
				if !strings.Contains(strings.ToLower(err.Error()), "statement_timeout") {
					t.Errorf("error does not name the knob: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := rtSeconds(cfg.StatementTimeout); got != tc.want {
				t.Errorf("StatementTimeout = %+v (%ds), want %d", cfg.StatementTimeout, got, tc.want)
			}
			if tc.want == 0 && !cfg.StatementTimeout.disabled {
				t.Errorf("want the explicit typed disabled state, got %+v", cfg.StatementTimeout)
			}
		})
	}
}

func strptr(s string) *string { return &s }

// rtSeconds flattens a resolvedTimeout for table assertions: disabled (or
// unresolved) is 0, enabled is its seconds value.
func rtSeconds(t resolvedTimeout) int {
	if !t.enabled() {
		return 0
	}
	return t.seconds
}

func TestResolveEnvOnlyTimeouts(t *testing.T) {
	knobs := []struct {
		env     string
		def     int
		extract func(*PostgresConfig) int
	}{
		{envLockTimeout, defaultLockTimeoutSeconds, func(c *PostgresConfig) int { return rtSeconds(c.LockTimeout) }},
		{envIdleTxTimeout, defaultIdleTxTimeoutSeconds, func(c *PostgresConfig) int { return rtSeconds(c.IdleTxTimeout) }},
	}
	for _, knob := range knobs {
		t.Run(knob.env, func(t *testing.T) {
			t.Run("absent uses default", func(t *testing.T) {
				cfgCleanEnv(t)
				cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h"})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got := knob.extract(cfg); got != knob.def {
					t.Errorf("= %d, want the default %d", got, knob.def)
				}
			})
			for _, tc := range secondsShapes {
				if tc.str == "" {
					continue
				}
				t.Run(tc.name, func(t *testing.T) {
					cfgCleanEnv(t)
					cfgSetEnv(t, knob.env, tc.str)
					cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h"})
					got := 0
					if cfg != nil {
						got = knob.extract(cfg)
					}
					checkShape(t, knob.env, tc, got, err)
				})
			}
		})
	}
}

func TestResolveMaxConnections(t *testing.T) {
	cases := []struct {
		name    string
		val     int32
		want    int
		wantErr bool
	}{
		{name: "unset uses default", val: 0, want: defaultMaxConnections},
		{name: "lower boundary", val: 1, want: 1},
		{name: "explicit value", val: 40, want: 40},
		{name: "upper boundary", val: maxMaxConnections, want: maxMaxConnections},
		{name: "past ceiling errors", val: maxMaxConnections + 1, wantErr: true},
		{name: "negative errors", val: -1, wantErr: true},
		{name: "max int32 errors", val: math.MaxInt32, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgCleanEnv(t)
			cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h", MaxConnections: tc.val})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolvePostgresConfig = %+v, want an error", cfg)
				}
				if !strings.Contains(err.Error(), "max_connections") {
					t.Errorf("error does not name the knob: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.MaxConns != tc.want {
				t.Errorf("MaxConns = %d, want %d", cfg.MaxConns, tc.want)
			}
		})
	}
}

// TestResolveMaxIdleConnections pins D-3: idle defaults to the open cap and is
// honored when explicitly set, constrained to 0 < idle <= open.
func TestResolveMaxIdleConnections(t *testing.T) {
	cases := []struct {
		name    string
		open    int32
		idle    int32
		want    int
		wantErr bool
	}{
		{name: "unset defaults to the open cap", open: 0, idle: 0, want: defaultMaxConnections},
		{name: "unset defaults to an explicit open cap", open: 8, idle: 0, want: 8},
		{name: "honored below the open cap", open: 20, idle: 5, want: 5},
		{name: "honored at the floor", open: 20, idle: 1, want: 1},
		{name: "honored at the open cap", open: 20, idle: 20, want: 20},
		{name: "one past the open cap errors", open: 20, idle: 21, wantErr: true},
		{name: "greater than a small open cap errors", open: 2, idle: 25, wantErr: true},
		{name: "greater than the default open cap errors", open: 0, idle: defaultMaxConnections + 1, wantErr: true},
		{name: "negative errors", open: 20, idle: -1, wantErr: true},
		{name: "max int32 errors", open: 20, idle: math.MaxInt32, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgCleanEnv(t)
			cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{
				Host: "h", MaxConnections: tc.open, MaxIdleConnections: tc.idle,
			})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolvePostgresConfig = %+v, want an error", cfg)
				}
				if !strings.Contains(err.Error(), "max_idle_connections") {
					t.Errorf("error does not name the knob: %v", err)
				}
				if !strings.Contains(err.Error(), "max_connections=") {
					t.Errorf("error should state the open cap it is bounded by: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.MaxIdleConns != tc.want {
				t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, tc.want)
			}
			if cfg.MaxIdleConns > cfg.MaxConns {
				t.Errorf("MaxIdleConns %d exceeds MaxConns %d", cfg.MaxIdleConns, cfg.MaxConns)
			}
		})
	}
}

func TestResolveDefaultsSSLModeButNotOtherFields(t *testing.T) {
	cfgCleanEnv(t)
	cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h", Port: "6543", Database: "d", Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want %q", cfg.SSLMode, "disable")
	}
	if cfg.Host != "h" || cfg.Port != "6543" || cfg.Name != "d" || cfg.User != "u" || cfg.Password != "p" {
		t.Errorf("string fields not carried through verbatim: %+v", cfg)
	}
	explicit, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h", SslMode: "require"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if explicit.SSLMode != "require" {
		t.Errorf("SSLMode = %q, want %q", explicit.SSLMode, "require")
	}
}

// =============================================================================
// Boot path: buildFromEnv rejects bad config before any dialing
// =============================================================================

func TestBuildFromEnvRejectsBadConfigBeforeDialing(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
		names string
	}{
		{name: "malformed statement timeout", key: envStatementTimeout, value: "abc", names: envStatementTimeout},
		{name: "negative statement timeout", key: envStatementTimeout, value: "-1", names: envStatementTimeout},
		{name: "statement timeout past ceiling", key: envStatementTimeout, value: "86400", names: envStatementTimeout},
		{name: "malformed lock timeout", key: envLockTimeout, value: "abc", names: envLockTimeout},
		{name: "negative lock timeout", key: envLockTimeout, value: "-1", names: envLockTimeout},
		{name: "malformed idle-tx timeout", key: envIdleTxTimeout, value: "10m", names: envIdleTxTimeout},
		{name: "idle-tx timeout past ceiling", key: envIdleTxTimeout, value: "999999", names: envIdleTxTimeout},
		{name: "zero max connections", key: envMaxConnections, value: "0", names: envMaxConnections},
		{name: "malformed max connections", key: envMaxConnections, value: "lots", names: envMaxConnections},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgCleanEnv(t)
			cfgSetEnv(t, tc.key, tc.value)
			// Point the boot path at an unroutable address: if the rejection
			// were to happen after dialing, this would surface as a connection
			// error instead of a configuration error.
			cfgSetEnv(t, "DATABASE_POSTGRES_HOST", "127.0.0.1")
			cfgSetEnv(t, "DATABASE_POSTGRES_PORT", "1")

			provider, err := buildFromEnv()
			if err == nil {
				if provider != nil {
					_ = provider.Close()
				}
				t.Fatal("buildFromEnv = nil error, want a configuration rejection")
			}
			if provider != nil {
				t.Errorf("buildFromEnv returned a provider alongside an error: %v", provider)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.names) {
				t.Errorf("error does not name %s: %s", tc.names, msg)
			}
			for _, dialWord := range []string{"connection refused", "failed to connect", "dial "} {
				if strings.Contains(msg, dialWord) {
					t.Errorf("configuration was rejected only after dialing (%q): %s", dialWord, msg)
				}
			}
		})
	}
}

// TestProtoPathNegativeIsHardError pins the MED-1 fix: the proto int32 carries
// no disabled encoding, so every negative value arriving on the typed-config
// path — including the exact -1 that a truncating int64-to-int32 bridge (such
// as DatabaseConfigAdapter's getInt32) produces from an overflowed
// int64(math.MaxInt64) — is a hard error naming the field. Nothing on the
// proto path can silently disable the statement timeout.
func TestProtoPathNegativeIsHardError(t *testing.T) {
	overflowedMax := int64(math.MaxInt64)           // narrows to -1
	overflowedPastInt32 := int64(math.MaxInt32) + 1 // narrows to math.MinInt32

	cases := []struct {
		name string
		val  int32
	}{
		{"minus one (the former in-band sentinel)", -1},
		{"int64 max truncated by a narrowing bridge", int32(overflowedMax)},
		{"one past int32 max truncated", int32(overflowedPastInt32)},
		{"min int32", math.MinInt32},
		{"minus two", -2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgCleanEnv(t)
			cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h", StatementTimeoutSeconds: tc.val})
			if err == nil {
				t.Fatalf("resolvePostgresConfig(%d) = %+v, want a hard error — a negative proto value silently disabled the timeout", tc.val, cfg.StatementTimeout)
			}
			if !strings.Contains(err.Error(), "statement_timeout_seconds") {
				t.Errorf("error does not name the field: %v", err)
			}
		})
	}

	// Large-positive overflow shape: a truncation that lands positive but past
	// the ceiling must also be a hard error, never a huge silent timeout.
	t.Run("large positive overflow shape", func(t *testing.T) {
		cfgCleanEnv(t)
		if cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h", StatementTimeoutSeconds: math.MaxInt32}); err == nil {
			t.Fatalf("= %+v, want a hard error", cfg.StatementTimeout)
		} else if !strings.Contains(err.Error(), "statement_timeout_seconds") {
			t.Errorf("error does not name the field: %v", err)
		}
	})

	// The whole path through Initialize: the hard error surfaces there and is
	// a validation rejection, not a dial failure.
	t.Run("Initialize rejects the bridge-collision shape", func(t *testing.T) {
		cfgCleanEnv(t)
		a := NewPostgresAdapter()
		err := a.Initialize(&dbpb.DatabaseProviderConfig{
			Enabled: true,
			Config: &dbpb.DatabaseProviderConfig_Postgresql{
				Postgresql: &dbpb.PostgreSQLConfig{
					Host: "127.0.0.1", Port: "1", Database: "d", Username: "u",
					StatementTimeoutSeconds: -1,
				},
			},
		})
		if err == nil {
			t.Fatal("Initialize accepted a -1 statement timeout")
		}
		if !strings.Contains(err.Error(), "statement_timeout_seconds") {
			t.Errorf("error does not name the field: %v", err)
		}
	})
}

// TestTypedDisabledSurvivesTheProtoHandOff replaces the retired -1 sentinel
// round trip: an explicit operator "0" (raw map or environment) reaches the
// resolved config as the TYPED disabled state, while the proto's int32 field
// never encodes it.
func TestTypedDisabledSurvivesTheProtoHandOff(t *testing.T) {
	t.Run("raw-map 0 via transformConfig", func(t *testing.T) {
		cfgCleanEnv(t)
		proto, err := transformConfig(cfgRawWith("statement_timeout_seconds", 0))
		if err != nil {
			t.Fatalf("transformConfig: %v", err)
		}
		pg := proto.GetPostgresql()
		if pg.GetStatementTimeoutSeconds() != 0 {
			t.Fatalf("proto field = %d, want 0 (no in-band disabled encoding)", pg.GetStatementTimeoutSeconds())
		}
		cfgSetEnv(t, envStatementTimeout, "77") // env must NOT win over the typed mark
		cfg, err := resolvePostgresConfig(pg)
		if err != nil {
			t.Fatalf("resolvePostgresConfig: %v", err)
		}
		if !cfg.StatementTimeout.disabled {
			t.Errorf("raw-map 0 resolved to %+v, want typed disabled", cfg.StatementTimeout)
		}
	})

	t.Run("env 0 still disables", func(t *testing.T) {
		cfgCleanEnv(t)
		cfgSetEnv(t, envStatementTimeout, "0")
		cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h"})
		if err != nil {
			t.Fatalf("resolvePostgresConfig: %v", err)
		}
		if !cfg.StatementTimeout.disabled {
			t.Errorf("env 0 resolved to %+v, want typed disabled", cfg.StatementTimeout)
		}
		want := fmt.Sprintf("-c idle_in_transaction_session_timeout=%d -c lock_timeout=%d",
			defaultIdleTxTimeoutSeconds*1000, defaultLockTimeoutSeconds*1000)
		if got := sessionOptions(cfg); got != want {
			t.Errorf("disabled statement timeout still reached the session options: %q", got)
		}
	})

	t.Run("positive values still travel in-band", func(t *testing.T) {
		cfgCleanEnv(t)
		proto, err := transformConfig(cfgRawWith("statement_timeout_seconds", 45))
		if err != nil {
			t.Fatalf("transformConfig: %v", err)
		}
		cfg, err := resolvePostgresConfig(proto.GetPostgresql())
		if err != nil {
			t.Fatalf("resolvePostgresConfig: %v", err)
		}
		if rtSeconds(cfg.StatementTimeout) != 45 || cfg.StatementTimeout.disabled {
			t.Errorf("= %+v, want enabled 45s", cfg.StatementTimeout)
		}
	})
}

// TestNoSilentSubstitution is the D-2 invariant in one place: for every
// supplied-but-invalid value, on every config surface, the result is an error —
// never the documented default.
func TestNoSilentSubstitution(t *testing.T) {
	invalid := []string{"-1", "3601", "abc", "30s", "9223372036854775807"}
	for _, value := range invalid {
		t.Run(fmt.Sprintf("value=%s", value), func(t *testing.T) {
			for _, key := range []string{envStatementTimeout, envLockTimeout, envIdleTxTimeout} {
				cfgCleanEnv(t)
				cfgSetEnv(t, key, value)
				if cfg, err := resolvePostgresConfig(&dbpb.PostgreSQLConfig{Host: "h"}); err == nil {
					t.Errorf("%s=%s silently resolved to %+v", key, value, cfg)
				}
			}
			cfgCleanEnv(t)
			if proto, err := transformConfig(cfgRawWith("statement_timeout_seconds", value)); err == nil {
				t.Errorf("raw-map statement_timeout_seconds=%s silently resolved to %d",
					value, proto.GetPostgresql().GetStatementTimeoutSeconds())
			}
		})
	}
}
