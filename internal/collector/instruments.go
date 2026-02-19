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

	return &Instruments{
		stackResourceCount:    stackResourceCount,
		updateDuration:        updateDuration,
		updateTotal:           updateTotal,
		updateResourceChanges: updateResourceChanges,
		deploymentStatus:      deploymentStatus,
		stackLastUpdate:       stackLastUpdate,
	}, nil
}
