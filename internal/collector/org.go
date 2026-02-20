package collector

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"
)

func (c *Collector) collectOrgMetrics(ctx context.Context, org string) {
	orgAttr := metric.WithAttributes(attribute.String("org", org))

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error { c.collectMembers(gCtx, org, orgAttr); return nil })
	g.Go(func() error { c.collectTeams(gCtx, org, orgAttr); return nil })
	g.Go(func() error { c.collectEnvironments(gCtx, org, orgAttr); return nil })
	g.Go(func() error { c.collectPolicyGroups(gCtx, org, orgAttr); return nil })
	g.Go(func() error { c.collectPolicyPacks(gCtx, org, orgAttr); return nil })
	g.Go(func() error { c.collectPolicyViolations(gCtx, org); return nil })
	g.Go(func() error { c.collectNeoTasks(gCtx, org); return nil })
	g.Go(func() error { c.collectPolicyResultsMetadata(gCtx, org, orgAttr); return nil })
	_ = g.Wait()
}

func (c *Collector) collectMembers(ctx context.Context, org string, attrs metric.MeasurementOption) {
	resp, err := c.client.ListMembers(ctx, org)
	if err != nil {
		c.logger.Error("failed to list members", "org", org, "error", err)
		return
	}
	c.instruments.orgMemberCount.Record(ctx, int64(len(resp.Members)), attrs)
}

func (c *Collector) collectTeams(ctx context.Context, org string, attrs metric.MeasurementOption) {
	resp, err := c.client.ListTeams(ctx, org)
	if err != nil {
		c.logger.Error("failed to list teams", "org", org, "error", err)
		return
	}
	c.instruments.orgTeamCount.Record(ctx, int64(len(resp.Teams)), attrs)
}

func (c *Collector) collectEnvironments(ctx context.Context, org string, attrs metric.MeasurementOption) {
	resp, err := c.client.ListEnvironments(ctx, org)
	if err != nil {
		c.logger.Error("failed to list environments", "org", org, "error", err)
		return
	}
	c.instruments.orgEnvironmentCount.Record(ctx, int64(len(resp.Environments)), attrs)
}

func (c *Collector) collectPolicyGroups(ctx context.Context, org string, attrs metric.MeasurementOption) {
	resp, err := c.client.ListPolicyGroups(ctx, org)
	if err != nil {
		c.logger.Error("failed to list policy groups", "org", org, "error", err)
		return
	}
	c.instruments.orgPolicyGroupCount.Record(ctx, int64(len(resp.PolicyGroups)), attrs)
}

func (c *Collector) collectPolicyPacks(ctx context.Context, org string, attrs metric.MeasurementOption) {
	resp, err := c.client.ListPolicyPacks(ctx, org)
	if err != nil {
		c.logger.Error("failed to list policy packs", "org", org, "error", err)
		return
	}
	c.instruments.orgPolicyPackCount.Record(ctx, int64(len(resp.PolicyPacks)), attrs)
}

func (c *Collector) collectPolicyViolations(ctx context.Context, org string) {
	resp, err := c.client.ListPolicyViolations(ctx, org)
	if err != nil {
		c.logger.Error("failed to list policy violations", "org", org, "error", err)
		return
	}

	counts := make(map[[2]string]int64) // [level, kind] -> count
	for _, v := range resp.PolicyViolations {
		level := v.Level
		if level == "" {
			level = "unknown"
		}
		kind := v.Kind
		if kind == "" {
			kind = "unknown"
		}
		counts[[2]string{level, kind}]++
	}

	for key, count := range counts {
		c.instruments.orgPolicyViolations.Record(ctx, count, metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("level", key[0]),
			attribute.String("kind", key[1]),
		))
	}
}

func (c *Collector) collectPolicyResultsMetadata(ctx context.Context, org string, attrs metric.MeasurementOption) {
	resp, err := c.client.GetPolicyResultsMetadata(ctx, org)
	if err != nil {
		c.logger.Error("failed to get policy results metadata", "org", org, "error", err)
		return
	}
	c.instruments.orgPolicyTotal.Record(ctx, resp.PolicyTotalCount, attrs)
	c.instruments.orgPolicyWithIssues.Record(ctx, resp.PolicyWithIssuesCount, attrs)
	c.instruments.orgResourcesTotal.Record(ctx, resp.ResourcesTotalCount, attrs)
	c.instruments.orgResourcesIssues.Record(ctx, resp.ResourcesWithIssuesCount, attrs)
}

func (c *Collector) collectNeoTasks(ctx context.Context, org string) {
	resp, err := c.client.ListNeoTasks(ctx, org)
	if err != nil {
		c.logger.Error("failed to list neo tasks", "org", org, "error", err)
		return
	}

	statusCounts := make(map[string]int64)
	for _, t := range resp.Tasks {
		statusCounts[t.Status]++
	}

	for status, count := range statusCounts {
		c.instruments.orgNeoTaskCount.Record(ctx, count, metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("status", status),
		))
	}
}
