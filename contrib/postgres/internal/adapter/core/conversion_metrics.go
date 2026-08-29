//go:build postgresql

package core

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// ConversionMetricSample is the aggregate-only measurement emitted for one
// PostgreSQL row-hydration stage. It deliberately has no fields for SQL,
// arguments, row values, identifiers, workspace/principal data, or errors.
type ConversionMetricSample struct {
	Table        string
	Path         string
	Operation    string
	Outcome      string
	FailureStage string
	Rows         int
	Columns      int
	Failures     int
	Total        time.Duration
	Phases       map[string]time.Duration
}

// ConversionMetricSink receives a completed aggregate sample. Production uses
// the fixed structured log formatter; tests can install a request-local sink.
type ConversionMetricSink func(ConversionMetricSample)

type conversionMetricConfig struct {
	all    bool
	tables map[string]struct{}
}

var conversionMetricsConfig atomic.Pointer[conversionMetricConfig]

type conversionMetricSinkKey struct{}

// WithConversionMetricSink installs an explicitly scoped sink and enables
// metrics for that context. This avoids global configuration mutations in
// deterministic adapter tests.
func WithConversionMetricSink(ctx context.Context, sink ConversionMetricSink) context.Context {
	return context.WithValue(ctx, conversionMetricSinkKey{}, sink)
}

// ValidateConversionMetricsSpec validates the disabled-by-default metric
// selector without mutating process state. Accepted enabled forms are "all"
// and a comma-separated list of bare PostgreSQL table identifiers.
func ValidateConversionMetricsSpec(spec string) error {
	_, err := parseConversionMetricsSpec(spec)
	return err
}

// ConfigureConversionMetrics replaces the process-wide metric selector. It is
// called only after the PostgreSQL adapter has validated and connected.
func ConfigureConversionMetrics(spec string) error {
	cfg, err := parseConversionMetricsSpec(spec)
	if err != nil {
		return err
	}
	conversionMetricsConfig.Store(cfg)
	return nil
}

func parseConversionMetricsSpec(spec string) (*conversionMetricConfig, error) {
	trimmed := strings.TrimSpace(spec)
	switch strings.ToLower(trimmed) {
	case "", "off", "false", "0":
		return &conversionMetricConfig{}, nil
	case "all":
		return &conversionMetricConfig{all: true}, nil
	}

	cfg := &conversionMetricConfig{tables: make(map[string]struct{})}
	for _, raw := range strings.Split(trimmed, ",") {
		table := strings.TrimSpace(raw)
		if table == "" {
			return nil, fmt.Errorf("postgresql: DATABASE_POSTGRES_CONVERSION_METRICS contains an empty table name")
		}
		if strings.Contains(table, ".") || ValidateSQLIdent(table) != nil {
			return nil, fmt.Errorf("postgresql: DATABASE_POSTGRES_CONVERSION_METRICS contains invalid table name %q", table)
		}
		cfg.tables[table] = struct{}{}
	}
	return cfg, nil
}

// ConversionMetricSpan accumulates phase timings for one operation. A span is
// created only when the table is enabled, so disabled callers can gate all
// time.Now calls with a single nil check.
type ConversionMetricSpan struct {
	sample  ConversionMetricSample
	started time.Time
	sink    ConversionMetricSink
}

// BeginConversionMetric returns nil when the table is not selected. New spans
// default to an error outcome; callers explicitly mark the successful end so
// an early return cannot accidentally be logged as success.
func BeginConversionMetric(ctx context.Context, table, operation, path string) *ConversionMetricSpan {
	if sink, ok := ctx.Value(conversionMetricSinkKey{}).(ConversionMetricSink); ok && sink != nil {
		return newConversionMetricSpan(table, operation, path, sink)
	}

	cfg := conversionMetricsConfig.Load()
	if cfg == nil || (!cfg.all && !cfg.hasTable(table)) {
		return nil
	}
	return newConversionMetricSpan(table, operation, path, logConversionMetric)
}

func (c *conversionMetricConfig) hasTable(table string) bool {
	if c == nil || c.tables == nil {
		return false
	}
	_, ok := c.tables[table]
	return ok
}

func newConversionMetricSpan(table, operation, path string, sink ConversionMetricSink) *ConversionMetricSpan {
	return &ConversionMetricSpan{
		sample: ConversionMetricSample{
			Table:        table,
			Operation:    operation,
			Path:         path,
			Outcome:      "error",
			FailureStage: "operation",
			Phases:       make(map[string]time.Duration),
		},
		started: time.Now(),
		sink:    sink,
	}
}

func (s *ConversionMetricSpan) StartPhase() time.Time { return time.Now() }

// EndPhase accumulates repeated phases, such as one JSON marshal per row.
func (s *ConversionMetricSpan) EndPhase(name string, started time.Time) {
	s.sample.Phases[name] += time.Since(started)
}

func (s *ConversionMetricSpan) SetRows(rows int)       { s.sample.Rows = rows }
func (s *ConversionMetricSpan) SetColumns(columns int) { s.sample.Columns = columns }
func (s *ConversionMetricSpan) AddFailure()            { s.sample.Failures++ }

func (s *ConversionMetricSpan) Fail(stage string) {
	s.sample.Outcome = "error"
	s.sample.FailureStage = stage
}

// Success marks a completed operation. Existing adapter loops may deliberately
// skip malformed rows; those retain their success return while reporting a
// distinct partial metric outcome.
func (s *ConversionMetricSpan) Success() {
	if s.sample.Failures > 0 {
		s.sample.Outcome = "partial"
	} else {
		s.sample.Outcome = "ok"
	}
	s.sample.FailureStage = ""
}

func (s *ConversionMetricSpan) Finish() {
	s.sample.Total = time.Since(s.started)
	s.sink(s.sample)
}

func logConversionMetric(sample ConversionMetricSample) {
	log.Print(formatConversionMetric(sample))
}

// formatConversionMetric uses a fixed field set and sorted phase keys so the
// line is stable for log queries and contains aggregate measurements only.
func formatConversionMetric(sample ConversionMetricSample) string {
	var b strings.Builder
	fmt.Fprintf(&b,
		"POSTGRES_CONVERSION_METRIC table=%s path=%s operation=%s outcome=%s rows=%d columns=%d failures=%d total_us=%d",
		sample.Table, sample.Path, sample.Operation, sample.Outcome,
		sample.Rows, sample.Columns, sample.Failures, sample.Total.Microseconds(),
	)
	if sample.FailureStage != "" {
		fmt.Fprintf(&b, " failure_stage=%s", sample.FailureStage)
	}
	phaseNames := make([]string, 0, len(sample.Phases))
	for name := range sample.Phases {
		phaseNames = append(phaseNames, name)
	}
	sort.Strings(phaseNames)
	for _, name := range phaseNames {
		fmt.Fprintf(&b, " phase_%s_us=%d", name, sample.Phases[name].Microseconds())
	}
	return b.String()
}
