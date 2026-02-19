package collector

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (c *Collector) collectOrgDeployments(ctx context.Context, org string) {
	resp, err := c.client.ListOrgDeployments(ctx, org)
	if err != nil {
		c.logger.Error("failed to list org deployments", "org", org, "error", err)
		return
	}

	// Count deployments by status.
	statusCounts := make(map[string]int64)
	for _, d := range resp.Deployments {
		statusCounts[d.Status]++
	}

	for status, count := range statusCounts {
		c.instruments.deploymentStatus.Record(ctx, count, metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("status", status),
		))
	}
}
