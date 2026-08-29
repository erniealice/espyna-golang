//go:build postgresql

package core

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestConversionMetricsSpec(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantAll   bool
		wantTable string
		wantErr   bool
	}{
		{name: "absent is disabled"},
		{name: "off is disabled", spec: "off"},
		{name: "all", spec: "all", wantAll: true},
		{name: "table list", spec: "client, location_area", wantTable: "location_area"},
		{name: "qualified rejected", spec: "public.client", wantErr: true},
		{name: "empty member rejected", spec: "client,", wantErr: true},
		{name: "injection rejected", spec: "client;drop table client", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseConversionMetricsSpec(tc.spec)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseConversionMetricsSpec(%q) error = %v, wantErr %v", tc.spec, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if cfg.all != tc.wantAll {
				t.Fatalf("all = %v, want %v", cfg.all, tc.wantAll)
			}
			if tc.wantTable != "" && !cfg.hasTable(tc.wantTable) {
				t.Fatalf("table %q is not enabled", tc.wantTable)
			}
		})
	}
}

func TestConversionMetricSpanAggregatesSafeFields(t *testing.T) {
	var samples []ConversionMetricSample
	ctx := WithConversionMetricSink(context.Background(), func(sample ConversionMetricSample) {
		samples = append(samples, sample)
	})

	span := BeginConversionMetric(ctx, "client", "list", "generic-proto")
	started := span.StartPhase()
	time.Sleep(time.Microsecond)
	span.EndPhase("json_marshal", started)
	span.SetRows(3)
	span.SetColumns(8)
	span.AddFailure()
	span.Success()
	span.Finish()

	if len(samples) != 1 {
		t.Fatalf("samples = %d, want 1", len(samples))
	}
	sample := samples[0]
	if sample.Outcome != "partial" || sample.Rows != 3 || sample.Columns != 8 || sample.Failures != 1 {
		t.Fatalf("sample = %#v", sample)
	}
	if sample.Total <= 0 || sample.Phases["json_marshal"] <= 0 {
		t.Fatalf("durations = total %v phases %#v, want positive", sample.Total, sample.Phases)
	}

	line := formatConversionMetric(sample)
	for _, want := range []string{
		"POSTGRES_CONVERSION_METRIC table=client",
		"path=generic-proto",
		"operation=list",
		"outcome=partial",
		"rows=3 columns=8 failures=1",
		"phase_json_marshal_us=",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("metric line %q does not contain %q", line, want)
		}
	}
	for _, forbidden := range []string{"SELECT ", "workspace_id=", "principal_id=", "args=", "error="} {
		if strings.Contains(line, forbidden) {
			t.Errorf("metric line %q contains forbidden field %q", line, forbidden)
		}
	}
}

func TestConversionMetricContextSinkIsIsolated(t *testing.T) {
	if got := BeginConversionMetric(context.Background(), "definitely_disabled_table", "read", "generic-map"); got != nil {
		t.Fatal("unconfigured context created a metric span")
	}

	called := false
	ctx := WithConversionMetricSink(context.Background(), func(ConversionMetricSample) { called = true })
	span := BeginConversionMetric(ctx, "definitely_disabled_table", "read", "generic-map")
	if span == nil {
		t.Fatal("context sink did not create a metric span")
	}
	span.Success()
	span.Finish()
	if !called {
		t.Fatal("context sink was not called")
	}
}
