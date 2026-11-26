package collector

import (
	"maps"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/pulumi-labs/pulumi-exporter/internal/client"
)

const (
	testAgentsActive  = "org=test-org,paused=false"
	testAgentsPaused  = "org=test-org,paused=true"
	testKindScheduled = "scheduled"
)

func TestCollectAgents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		agents      *client.ListAgentsResponse
		wantCount   map[string]int64
		wantStarted map[string]int64
		wantRunning map[string]int64
	}{
		{
			name: "agents with run counts",
			agents: &client.ListAgentsResponse{Agents: []client.Agent{
				{ID: "1", Name: "qa", Runs: &client.AgentRunCounts{Total: 98, Running: 0}},
				{ID: "2", Name: "dispatcher", Runs: &client.AgentRunCounts{Total: 288, Running: 2}},
				{ID: "3", Name: "reporter", Paused: true, Runs: &client.AgentRunCounts{Total: 91}},
			}},
			wantCount: map[string]int64{
				testAgentsActive: 2,
				testAgentsPaused: 1,
			},
			wantStarted: map[string]int64{
				"agent=qa,org=test-org":         98,
				"agent=dispatcher,org=test-org": 288,
				"agent=reporter,org=test-org":   91,
			},
			wantRunning: map[string]int64{
				"agent=qa,org=test-org":         0,
				"agent=dispatcher,org=test-org": 2,
				"agent=reporter,org=test-org":   0,
			},
		},
		{
			name: "agent without run counts",
			agents: &client.ListAgentsResponse{Agents: []client.Agent{
				{ID: "1", Name: "qa"},
			}},
			wantCount: map[string]int64{
				testAgentsActive: 1,
				testAgentsPaused: 0,
			},
		},
		{
			name:   "no agents",
			agents: &client.ListAgentsResponse{},
			wantCount: map[string]int64{
				testAgentsActive: 0,
				testAgentsPaused: 0,
			},
		},
		{
			name:   "pulumi agents not enabled",
			agents: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &mockAPI{agents: map[string]*client.ListAgentsResponse{testOrg: tt.agents}}
			c, reader := newTestCollector(t, api)
			ctx := t.Context()

			c.collectAgents(ctx, testOrg)

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(ctx, &rm); err != nil {
				t.Fatalf("failed to collect metrics: %v", err)
			}

			assertGaugePoints(t, rm, "pulumi_org_agent_count", tt.wantCount)
			assertGaugePoints(t, rm, "pulumi_org_agent_runs_started", tt.wantStarted)
			assertGaugePoints(t, rm, "pulumi_org_agent_runs_running", tt.wantRunning)
		})
	}
}

func TestCollectAgentAutomations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		automations *client.ListAgentAutomationsResponse
		wantCount   map[string]int64
		wantNextRun map[string]float64
	}{
		{
			name: "scheduled and triggered automations",
			automations: &client.ListAgentAutomationsResponse{Automations: []client.AgentAutomation{
				{ID: "1", Name: "dispatch", Kind: testKindScheduled, LastRunStatus: testResultOK, NextExecution: "2026-11-16T14:00:00Z"},
				{ID: "2", Name: "intake", Kind: "triggered"},
				{ID: "3", Name: "drift", Kind: "triggered"},
				{ID: "4", Name: "nightly", Kind: testKindScheduled, Paused: true, LastRunStatus: "failed"},
			}},
			wantCount: map[string]int64{
				"kind=scheduled,last_run_status=succeeded,org=test-org,paused=false": 1,
				"kind=triggered,last_run_status=none,org=test-org,paused=false":      2,
				"kind=scheduled,last_run_status=failed,org=test-org,paused=true":     1,
			},
			wantNextRun: map[string]float64{
				"automation=dispatch,org=test-org": 1794837600,
			},
		},
		{
			name: "invalid next execution time is skipped",
			automations: &client.ListAgentAutomationsResponse{Automations: []client.AgentAutomation{
				{ID: "1", Name: "dispatch", Kind: testKindScheduled, NextExecution: "not-a-time"},
			}},
			wantCount: map[string]int64{
				"kind=scheduled,last_run_status=none,org=test-org,paused=false": 1,
			},
		},
		{
			name:        "pulumi agents not enabled",
			automations: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &mockAPI{automations: map[string]*client.ListAgentAutomationsResponse{testOrg: tt.automations}}
			c, reader := newTestCollector(t, api)
			ctx := t.Context()

			c.collectAgentAutomations(ctx, testOrg)

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(ctx, &rm); err != nil {
				t.Fatalf("failed to collect metrics: %v", err)
			}

			assertGaugePoints(t, rm, "pulumi_org_agent_automation_count", tt.wantCount)
			assertGaugePoints(t, rm, "pulumi_org_agent_automation_next_run_timestamp", tt.wantNextRun)
		})
	}
}

// assertGaugePoints checks that the named gauge has exactly the wanted data
// points, keyed by their encoded attribute set. A nil want asserts that the
// metric was not recorded at all.
func assertGaugePoints[N int64 | float64](t *testing.T, rm metricdata.ResourceMetrics, name string, want map[string]N) {
	t.Helper()

	got := gaugePoints[N](t, rm, name)
	if want == nil {
		if got != nil {
			t.Errorf("%s: expected no data points, got %v", name, got)
		}
		return
	}
	if !maps.Equal(got, want) {
		t.Errorf("%s:\n got  %v\n want %v", name, got, want)
	}
}

// gaugePoints returns the data points of the named gauge keyed by their encoded
// attribute set, or nil when the metric was not recorded.
func gaugePoints[N int64 | float64](t *testing.T, rm metricdata.ResourceMetrics, name string) map[string]N {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			g, ok := m.Data.(metricdata.Gauge[N])
			if !ok {
				t.Fatalf("metric %s has unexpected data type %T", name, m.Data)
			}
			points := make(map[string]N, len(g.DataPoints))
			for _, dp := range g.DataPoints {
				points[dp.Attributes.Encoded(attribute.DefaultEncoder())] = dp.Value
			}
			return points
		}
	}
	return nil
}
