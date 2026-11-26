package collector

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// Instruments holds all OTel metric instruments used by the collector.
type Instruments struct {
	stackResourceCount    metric.Int64Gauge
	updateDuration        metric.Float64Histogram
	updateTotal           metric.Int64Counter
	updateResourceChanges metric.Int64Counter
	deploymentStatus      metric.Int64Gauge
	stackLastUpdate       metric.Float64Gauge

	orgMemberCount        metric.Int64Gauge
	orgTeamCount          metric.Int64Gauge
	orgEnvironmentCount   metric.Int64Gauge
	orgPolicyGroupCount   metric.Int64Gauge
	orgPolicyPackCount    metric.Int64Gauge
	orgPolicyViolations   metric.Int64Gauge
	orgNeoTaskCount       metric.Int64Gauge
	orgNeoTokensUsedMonth metric.Int64Gauge
	orgNeoTokensUsedTotal metric.Int64Gauge

	orgNeoTokenBudgetConsumed  metric.Int64Gauge
	orgNeoTokenBudgetAllowance metric.Int64Gauge
	orgNeoTokenBudgetExhausted metric.Int64Gauge

	orgPolicyTotal      metric.Int64Gauge
	orgPolicyWithIssues metric.Int64Gauge
	orgResourcesTotal   metric.Int64Gauge
	orgResourcesIssues  metric.Int64Gauge

	orgAgentCount             metric.Int64Gauge
	orgAgentRunsStarted       metric.Int64Gauge
	orgAgentRunsRunning       metric.Int64Gauge
	orgAgentAutomationCount   metric.Int64Gauge
	orgAgentAutomationNextRun metric.Float64Gauge

	orgDeliveryProgramCount         metric.Int64Gauge
	orgDeliveryProgramsByLastRun    metric.Int64Gauge
	orgDeliveryStagesByStatus       metric.Int64Gauge
	orgDeliveryStagesWaiting        metric.Int64Gauge
	orgDeliveryOldestApprovalAge    metric.Float64Gauge
	deliveryProgramPendingApprovals metric.Int64Gauge
	deliveryProgramLastRunTimestamp metric.Float64Gauge
	deliveryProgramLastRunDuration  metric.Float64Gauge
	deliveryStageDeployedRelease    metric.Int64Gauge
}

// updateDurationBuckets are the histogram bucket boundaries, in seconds, for
// pulumi_update_duration_seconds.
var updateDurationBuckets = []float64{5, 10, 30, 60, 120, 300, 600, 1800}

// instrumentBuilder creates instruments on a meter and collects every creation
// error, so callers can register instruments without an error check per call.
type instrumentBuilder struct {
	meter metric.Meter
	errs  []error
}

func (b *instrumentBuilder) record(name string, err error) {
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("creating instrument %s: %w", name, err))
	}
}

func (b *instrumentBuilder) int64Gauge(name, description string) metric.Int64Gauge {
	g, err := b.meter.Int64Gauge(name, metric.WithDescription(description))
	b.record(name, err)
	return g
}

func (b *instrumentBuilder) float64Gauge(name, description string) metric.Float64Gauge {
	g, err := b.meter.Float64Gauge(name, metric.WithDescription(description))
	b.record(name, err)
	return g
}

func (b *instrumentBuilder) int64Counter(name, description string) metric.Int64Counter {
	c, err := b.meter.Int64Counter(name, metric.WithDescription(description))
	b.record(name, err)
	return c
}

func (b *instrumentBuilder) float64Histogram(name, description string, buckets []float64) metric.Float64Histogram {
	h, err := b.meter.Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithExplicitBucketBoundaries(buckets...),
	)
	b.record(name, err)
	return h
}

func (b *instrumentBuilder) err() error {
	return errors.Join(b.errs...)
}

// NewInstruments creates all OTel metric instruments.
func NewInstruments(meter metric.Meter) (*Instruments, error) {
	var ins Instruments
	b := &instrumentBuilder{meter: meter}

	ins.stackResourceCount = b.int64Gauge("pulumi_stack_resource_count",
		"Number of resources in a Pulumi stack")
	ins.updateDuration = b.float64Histogram("pulumi_update_duration_seconds",
		"Duration of Pulumi stack updates in seconds", updateDurationBuckets)
	ins.updateTotal = b.int64Counter("pulumi_update_total",
		"Total number of Pulumi stack updates")
	ins.updateResourceChanges = b.int64Counter("pulumi_update_resource_changes",
		"Number of resource changes per Pulumi update")
	ins.deploymentStatus = b.int64Gauge("pulumi_deployment_status",
		"Number of Pulumi deployments by status")
	ins.stackLastUpdate = b.float64Gauge("pulumi_stack_last_update_timestamp",
		"Unix timestamp of the last update to a Pulumi stack")

	newOrgInstruments(b, &ins)
	newOrgNeoInstruments(b, &ins)
	newOrgAgentInstruments(b, &ins)
	newDeliveryInstruments(b, &ins)

	if err := b.err(); err != nil {
		return nil, err
	}

	return &ins, nil
}

// newOrgInstruments registers the organization membership and policy instruments.
func newOrgInstruments(b *instrumentBuilder, ins *Instruments) {
	ins.orgMemberCount = b.int64Gauge("pulumi_org_member_count",
		"Number of members in a Pulumi organization")
	ins.orgTeamCount = b.int64Gauge("pulumi_org_team_count",
		"Number of teams in a Pulumi organization")
	ins.orgEnvironmentCount = b.int64Gauge("pulumi_org_environment_count",
		"Number of ESC environments in a Pulumi organization")
	ins.orgPolicyGroupCount = b.int64Gauge("pulumi_org_policy_group_count",
		"Number of policy groups in a Pulumi organization")
	ins.orgPolicyPackCount = b.int64Gauge("pulumi_org_policy_pack_count",
		"Number of policy packs in a Pulumi organization")
	ins.orgPolicyViolations = b.int64Gauge("pulumi_org_policy_violations",
		"Number of policy violations by level and kind")
	ins.orgPolicyTotal = b.int64Gauge("pulumi_org_policy_total",
		"Total number of policies in a Pulumi organization")
	ins.orgPolicyWithIssues = b.int64Gauge("pulumi_org_policy_with_issues",
		"Number of policies with issues in a Pulumi organization")
	ins.orgResourcesTotal = b.int64Gauge("pulumi_org_governed_resources_total",
		"Total number of resources governed by policies in a Pulumi organization")
	ins.orgResourcesIssues = b.int64Gauge("pulumi_org_governed_resources_with_issues",
		"Number of governed resources with issues in a Pulumi organization")
}

// newOrgNeoInstruments registers the Pulumi Neo (AI agent) instruments.
func newOrgNeoInstruments(b *instrumentBuilder, ins *Instruments) {
	ins.orgNeoTaskCount = b.int64Gauge("pulumi_org_neo_task_count",
		"Number of Pulumi Neo AI tasks by status")
	ins.orgNeoTokensUsedMonth = b.int64Gauge("pulumi_org_neo_tokens_used_current_month",
		"Neo tokens consumed by AI tasks created in the current calendar month "+
			"(matches the Pulumi Cloud billing-period usage)")
	ins.orgNeoTokensUsedTotal = b.int64Gauge("pulumi_org_neo_tokens_used_total",
		"Total Neo tokens consumed across all Pulumi Neo AI tasks (lifetime)")
	ins.orgNeoTokenBudgetConsumed = b.int64Gauge("pulumi_org_neo_token_budget_consumed",
		"Neo tokens consumed in the current budget window for a Pulumi organization")
	ins.orgNeoTokenBudgetAllowance = b.int64Gauge("pulumi_org_neo_token_budget_allowance",
		"Effective Neo token allowance for the current budget window (base plus any active bonus)")
	ins.orgNeoTokenBudgetExhausted = b.int64Gauge("pulumi_org_neo_token_budget_exhausted",
		"Whether the Neo token budget for the current window is exhausted (1) or not (0)")
}

// newOrgAgentInstruments registers the Pulumi Agent and agent automation instruments.
func newOrgAgentInstruments(b *instrumentBuilder, ins *Instruments) {
	ins.orgAgentCount = b.int64Gauge("pulumi_org_agent_count",
		"Number of Pulumi Agents in an organization by paused state")
	ins.orgAgentRunsStarted = b.int64Gauge("pulumi_org_agent_runs_started",
		"Total number of runs a Pulumi Agent has started (lifetime, organization-wide)")
	ins.orgAgentRunsRunning = b.int64Gauge("pulumi_org_agent_runs_running",
		"Number of runs of a Pulumi Agent that are currently running")
	ins.orgAgentAutomationCount = b.int64Gauge("pulumi_org_agent_automation_count",
		"Number of Pulumi Agent automations by kind, paused state, and last run status")
	ins.orgAgentAutomationNextRun = b.float64Gauge("pulumi_org_agent_automation_next_run_timestamp",
		"Unix timestamp of the next scheduled run of a Pulumi Agent automation")
}

// newDeliveryInstruments registers the Pulumi Delivery instruments.
func newDeliveryInstruments(b *instrumentBuilder, ins *Instruments) {
	ins.orgDeliveryProgramCount = b.int64Gauge("pulumi_org_delivery_program_count",
		"Number of Pulumi Delivery programs in an organization by registration state")
	ins.orgDeliveryProgramsByLastRun = b.int64Gauge("pulumi_org_delivery_programs_by_last_run_status",
		"Number of Pulumi Delivery programs by the status of their latest run")
	ins.orgDeliveryStagesByStatus = b.int64Gauge("pulumi_org_delivery_stages_by_status",
		"Number of Pulumi Delivery stages by their status in the latest run")
	ins.orgDeliveryStagesWaiting = b.int64Gauge("pulumi_org_delivery_stages_waiting",
		"Number of waiting Pulumi Delivery stages by what they wait for")
	ins.orgDeliveryOldestApprovalAge = b.float64Gauge("pulumi_org_delivery_oldest_pending_approval_age_seconds",
		"Age in seconds of the oldest pending Pulumi Delivery approval (0 when none is pending)")
	ins.deliveryProgramPendingApprovals = b.int64Gauge("pulumi_delivery_program_pending_approvals",
		"Number of approvals a Pulumi Delivery program is waiting for")
	ins.deliveryProgramLastRunTimestamp = b.float64Gauge("pulumi_delivery_program_last_run_timestamp",
		"Unix timestamp of when the latest run of a Pulumi Delivery program was created")
	ins.deliveryProgramLastRunDuration = b.float64Gauge("pulumi_delivery_program_last_run_duration_seconds",
		"Duration in seconds of the latest finished run of a Pulumi Delivery program")
	ins.deliveryStageDeployedRelease = b.int64Gauge("pulumi_delivery_stage_deployed_release",
		"Number of the release deployed at a Pulumi Delivery stage")
}
