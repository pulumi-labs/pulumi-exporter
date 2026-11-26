package collector

import (
	"math"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/pulumi-labs/pulumi-exporter/internal/client"
)

const (
	testDeliveryWaiting  = "waiting"
	testDeliveryApproval = "approval"
	testDeliveryMain     = "main"
	testStackDev         = "dev"
	testDeliveryShop     = "org=test-org,project=shop,stack=main"
)

// deliveryProgramsCase is a test case for TestCollectDeliveryPrograms.
type deliveryProgramsCase struct {
	name          string
	programs      *client.ListDeliveryProgramsResponse
	wantPrograms  map[string]int64
	wantLastRun   map[string]int64
	wantStages    map[string]int64
	wantWaiting   map[string]int64
	wantPending   map[string]int64
	wantTimestamp map[string]float64
	wantDuration  map[string]float64
	wantRelease   map[string]int64
}

// deliveryProgramsCases returns the TestCollectDeliveryPrograms cases for runs created at created.
func deliveryProgramsCases(created time.Time) []deliveryProgramsCase {
	return []deliveryProgramsCase{
		{
			name: "programs with runs and stages",
			programs: &client.ListDeliveryProgramsResponse{Programs: []client.DeliveryProgram{
				{
					Project:    "shop",
					Stack:      testDeliveryMain,
					Registered: true,
					LastRun: &client.DeliveryRun{
						Number:     10,
						Status:     testResultOK,
						CreatedAt:  created,
						FinishedAt: created.Add(9 * time.Minute),
					},
					Stages: []client.DeliveryStage{{Name: testStackDev, Status: testResultOK, DeployedRelease: 10}},
				},
				{
					Project:          "gated",
					Stack:            testDeliveryMain,
					Registered:       true,
					PendingApprovals: 1,
					LastRun: &client.DeliveryRun{
						Number:    1,
						Status:    testDeliveryWaiting,
						WaitKind:  testDeliveryApproval,
						CreatedAt: created,
					},
					Stages: []client.DeliveryStage{
						{Name: testStackDev, Status: testResultOK, DeployedRelease: 1},
						{Name: "prod", Status: testDeliveryWaiting, WaitKind: testDeliveryApproval},
					},
				},
				{Project: "local", Stack: testStackDev},
			}},
			wantPrograms: map[string]int64{
				"org=test-org,registered=true":  2,
				"org=test-org,registered=false": 1,
			},
			wantLastRun: withZeroes("status", append([]string{noLastRun}, deliveryStatuses...), map[string]int64{
				testResultOK: 1, testDeliveryWaiting: 1, noLastRun: 1,
			}),
			wantStages: withZeroes("status", deliveryStatuses, map[string]int64{
				testResultOK: 2, testDeliveryWaiting: 1,
			}),
			wantWaiting: withZeroes("wait_kind", deliveryWaitKinds, map[string]int64{
				testDeliveryApproval: 1,
			}),
			wantPending: map[string]int64{
				testDeliveryShop:                        0,
				"org=test-org,project=gated,stack=main": 1,
				"org=test-org,project=local,stack=dev":  0,
			},
			wantTimestamp: map[string]float64{
				testDeliveryShop:                        float64(created.Unix()),
				"org=test-org,project=gated,stack=main": float64(created.Unix()),
			},
			wantDuration: map[string]float64{
				testDeliveryShop: 540,
			},
			wantRelease: map[string]int64{
				"org=test-org,project=shop,stack=main,stage=dev":  10,
				"org=test-org,project=gated,stack=main,stage=dev": 1,
			},
		},
		{
			name:     "no programs",
			programs: &client.ListDeliveryProgramsResponse{},
			wantPrograms: map[string]int64{
				"org=test-org,registered=true":  0,
				"org=test-org,registered=false": 0,
			},
			wantLastRun: withZeroes("status", append([]string{noLastRun}, deliveryStatuses...), nil),
			wantStages:  withZeroes("status", deliveryStatuses, nil),
			wantWaiting: withZeroes("wait_kind", deliveryWaitKinds, nil),
		},
		{
			name:     "delivery not enabled",
			programs: nil,
		},
	}
}

func TestCollectDeliveryPrograms(t *testing.T) {
	t.Parallel()

	tests := deliveryProgramsCases(time.Date(2026, time.November, 23, 14, 5, 15, 0, time.UTC))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &mockAPI{delivery: map[string]*client.ListDeliveryProgramsResponse{testOrg: tt.programs}}
			c, reader := newTestCollector(t, api)
			ctx := t.Context()

			c.collectDeliveryPrograms(ctx, testOrg)

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(ctx, &rm); err != nil {
				t.Fatalf("failed to collect metrics: %v", err)
			}

			assertGaugePoints(t, rm, "pulumi_org_delivery_program_count", tt.wantPrograms)
			assertGaugePoints(t, rm, "pulumi_org_delivery_programs_by_last_run_status", tt.wantLastRun)
			assertGaugePoints(t, rm, "pulumi_org_delivery_stages_by_status", tt.wantStages)
			assertGaugePoints(t, rm, "pulumi_org_delivery_stages_waiting", tt.wantWaiting)
			assertGaugePoints(t, rm, "pulumi_delivery_program_pending_approvals", tt.wantPending)
			assertGaugePoints(t, rm, "pulumi_delivery_program_last_run_timestamp", tt.wantTimestamp)
			assertGaugePoints(t, rm, "pulumi_delivery_program_last_run_duration_seconds", tt.wantDuration)
			assertGaugePoints(t, rm, "pulumi_delivery_stage_deployed_release", tt.wantRelease)
		})
	}
}

func TestCollectDeliveryApprovals(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		approvals *client.ListDeliveryApprovalsResponse
		wantAge   float64
		wantNone  bool
	}{
		{
			name: "oldest pending approval",
			approvals: &client.ListDeliveryApprovalsResponse{Approvals: []client.DeliveryApproval{
				{Project: "gated", Stack: testDeliveryMain, Stage: "prod", RequestedAt: now.Add(-30 * time.Minute)},
				{Project: "fleet", Stack: testStackDev, Stage: "canary", RequestedAt: now.Add(-2 * time.Hour)},
			}},
			wantAge: (2 * time.Hour).Seconds(),
		},
		{
			name:      "no pending approvals",
			approvals: &client.ListDeliveryApprovalsResponse{},
			wantAge:   0,
		},
		{
			name:      "delivery not enabled",
			approvals: nil,
			wantNone:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &mockAPI{approvals: map[string]*client.ListDeliveryApprovalsResponse{testOrg: tt.approvals}}
			c, reader := newTestCollector(t, api)
			ctx := t.Context()

			c.collectDeliveryApprovals(ctx, testOrg)

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(ctx, &rm); err != nil {
				t.Fatalf("failed to collect metrics: %v", err)
			}

			got := gaugePoints[float64](t, rm, "pulumi_org_delivery_oldest_pending_approval_age_seconds")
			if tt.wantNone {
				if got != nil {
					t.Fatalf("expected no data points, got %v", got)
				}
				return
			}
			age, ok := got["org=test-org"]
			if !ok {
				t.Fatalf("expected a data point for org=test-org, got %v", got)
			}
			// Allow for the time that passes between building the input and collecting.
			if math.Abs(age-tt.wantAge) > 60 {
				t.Errorf("oldest pending approval age: got %.0f, want %.0f", age, tt.wantAge)
			}
		})
	}
}

// withZeroes returns the expected data points of a zero-filled count metric:
// one entry per key, labeled with the test org, set to the matching value in
// counts or 0.
func withZeroes(label string, keys []string, counts map[string]int64) map[string]int64 {
	want := make(map[string]int64, len(keys))
	for _, k := range keys {
		want["org=test-org,"+label+"="+k] = counts[k]
	}
	return want
}
