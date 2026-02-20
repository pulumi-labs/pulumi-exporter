package collector

import (
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

	orgMemberCount      metric.Int64Gauge
	orgTeamCount        metric.Int64Gauge
	orgEnvironmentCount metric.Int64Gauge
	orgPolicyGroupCount metric.Int64Gauge
	orgPolicyPackCount  metric.Int64Gauge
	orgPolicyViolations metric.Int64Gauge
	orgNeoTaskCount     metric.Int64Gauge
	orgPolicyTotal      metric.Int64Gauge
	orgPolicyWithIssues metric.Int64Gauge
	orgResourcesTotal   metric.Int64Gauge
	orgResourcesIssues  metric.Int64Gauge
}

// NewInstruments creates all OTel metric instruments.
func NewInstruments(meter metric.Meter) (*Instruments, error) {
	var ins Instruments
	var err error

	if ins.stackResourceCount, err = meter.Int64Gauge("pulumi_stack_resource_count",
		metric.WithDescription("Number of resources in a Pulumi stack"),
	); err != nil {
		return nil, err
	}

	if ins.updateDuration, err = meter.Float64Histogram("pulumi_update_duration_seconds",
		metric.WithDescription("Duration of Pulumi stack updates in seconds"),
		metric.WithExplicitBucketBoundaries(5, 10, 30, 60, 120, 300, 600, 1800),
	); err != nil {
		return nil, err
	}

	if ins.updateTotal, err = meter.Int64Counter("pulumi_update_total",
		metric.WithDescription("Total number of Pulumi stack updates"),
	); err != nil {
		return nil, err
	}

	if ins.updateResourceChanges, err = meter.Int64Counter("pulumi_update_resource_changes",
		metric.WithDescription("Number of resource changes per Pulumi update"),
	); err != nil {
		return nil, err
	}

	if ins.deploymentStatus, err = meter.Int64Gauge("pulumi_deployment_status",
		metric.WithDescription("Number of Pulumi deployments by status"),
	); err != nil {
		return nil, err
	}

	if ins.stackLastUpdate, err = meter.Float64Gauge("pulumi_stack_last_update_timestamp",
		metric.WithDescription("Unix timestamp of the last update to a Pulumi stack"),
	); err != nil {
		return nil, err
	}

	if err = newOrgInstruments(meter, &ins); err != nil {
		return nil, err
	}

	return &ins, nil
}

func newOrgInstruments(meter metric.Meter, ins *Instruments) error {
	var err error

	if ins.orgMemberCount, err = meter.Int64Gauge("pulumi_org_member_count",
		metric.WithDescription("Number of members in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgTeamCount, err = meter.Int64Gauge("pulumi_org_team_count",
		metric.WithDescription("Number of teams in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgEnvironmentCount, err = meter.Int64Gauge("pulumi_org_environment_count",
		metric.WithDescription("Number of ESC environments in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgPolicyGroupCount, err = meter.Int64Gauge("pulumi_org_policy_group_count",
		metric.WithDescription("Number of policy groups in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgPolicyPackCount, err = meter.Int64Gauge("pulumi_org_policy_pack_count",
		metric.WithDescription("Number of policy packs in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgPolicyViolations, err = meter.Int64Gauge("pulumi_org_policy_violations",
		metric.WithDescription("Number of policy violations by level and kind"),
	); err != nil {
		return err
	}

	if ins.orgNeoTaskCount, err = meter.Int64Gauge("pulumi_org_neo_task_count",
		metric.WithDescription("Number of Pulumi Neo AI tasks by status"),
	); err != nil {
		return err
	}

	if ins.orgPolicyTotal, err = meter.Int64Gauge("pulumi_org_policy_total",
		metric.WithDescription("Total number of policies in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgPolicyWithIssues, err = meter.Int64Gauge("pulumi_org_policy_with_issues",
		metric.WithDescription("Number of policies with issues in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgResourcesTotal, err = meter.Int64Gauge("pulumi_org_governed_resources_total",
		metric.WithDescription("Total number of resources governed by policies in a Pulumi organization"),
	); err != nil {
		return err
	}

	if ins.orgResourcesIssues, err = meter.Int64Gauge("pulumi_org_governed_resources_with_issues",
		metric.WithDescription("Number of governed resources with issues in a Pulumi organization"),
	); err != nil {
		return err
	}

	return nil
}
