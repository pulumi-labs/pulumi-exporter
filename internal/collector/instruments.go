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
}

// NewInstruments creates all OTel metric instruments.
func NewInstruments(meter metric.Meter) (*Instruments, error) {
	stackResourceCount, err := meter.Int64Gauge("pulumi_stack_resource_count",
		metric.WithDescription("Number of resources in a Pulumi stack"),
	)
	if err != nil {
		return nil, err
	}

	updateDuration, err := meter.Float64Histogram("pulumi_update_duration_seconds",
		metric.WithDescription("Duration of Pulumi stack updates in seconds"),
		metric.WithExplicitBucketBoundaries(5, 10, 30, 60, 120, 300, 600, 1800),
	)
	if err != nil {
		return nil, err
	}

	updateTotal, err := meter.Int64Counter("pulumi_update_total",
		metric.WithDescription("Total number of Pulumi stack updates"),
	)
	if err != nil {
		return nil, err
	}

	updateResourceChanges, err := meter.Int64Counter("pulumi_update_resource_changes",
		metric.WithDescription("Number of resource changes per Pulumi update"),
	)
	if err != nil {
		return nil, err
	}

	deploymentStatus, err := meter.Int64Gauge("pulumi_deployment_status",
		metric.WithDescription("Number of Pulumi deployments by status"),
	)
	if err != nil {
		return nil, err
	}

	stackLastUpdate, err := meter.Float64Gauge("pulumi_stack_last_update_timestamp",
		metric.WithDescription("Unix timestamp of the last update to a Pulumi stack"),
	)
	if err != nil {
		return nil, err
	}

	orgMemberCount, err := meter.Int64Gauge("pulumi_org_member_count",
		metric.WithDescription("Number of members in a Pulumi organization"),
	)
	if err != nil {
		return nil, err
	}

	orgTeamCount, err := meter.Int64Gauge("pulumi_org_team_count",
		metric.WithDescription("Number of teams in a Pulumi organization"),
	)
	if err != nil {
		return nil, err
	}

	orgEnvironmentCount, err := meter.Int64Gauge("pulumi_org_environment_count",
		metric.WithDescription("Number of ESC environments in a Pulumi organization"),
	)
	if err != nil {
		return nil, err
	}

	orgPolicyGroupCount, err := meter.Int64Gauge("pulumi_org_policy_group_count",
		metric.WithDescription("Number of policy groups in a Pulumi organization"),
	)
	if err != nil {
		return nil, err
	}

	orgPolicyPackCount, err := meter.Int64Gauge("pulumi_org_policy_pack_count",
		metric.WithDescription("Number of policy packs in a Pulumi organization"),
	)
	if err != nil {
		return nil, err
	}

	orgPolicyViolations, err := meter.Int64Gauge("pulumi_org_policy_violations",
		metric.WithDescription("Number of policy violations by level and kind"),
	)
	if err != nil {
		return nil, err
	}

	orgNeoTaskCount, err := meter.Int64Gauge("pulumi_org_neo_task_count",
		metric.WithDescription("Number of Pulumi Neo AI tasks by status"),
	)
	if err != nil {
		return nil, err
	}

	return &Instruments{
		stackResourceCount:    stackResourceCount,
		updateDuration:        updateDuration,
		updateTotal:           updateTotal,
		updateResourceChanges: updateResourceChanges,
		deploymentStatus:      deploymentStatus,
		stackLastUpdate:       stackLastUpdate,
		orgMemberCount:        orgMemberCount,
		orgTeamCount:          orgTeamCount,
		orgEnvironmentCount:   orgEnvironmentCount,
		orgPolicyGroupCount:   orgPolicyGroupCount,
		orgPolicyPackCount:    orgPolicyPackCount,
		orgPolicyViolations:   orgPolicyViolations,
		orgNeoTaskCount:       orgNeoTaskCount,
	}, nil
}
