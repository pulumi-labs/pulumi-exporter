package collector

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// noLastRunStatus labels automations that have not run yet.
const noLastRunStatus = "none"

func (c *Collector) collectAgents(ctx context.Context, org string) {
	resp, err := c.client.ListAgents(ctx, org)
	if err != nil {
		c.logger.Error("failed to list agents", "org", org, "error", err)
		return
	}
	if resp == nil {
		// Pulumi Agents are not enabled for the organization; nothing to record.
		return
	}

	// Record both paused states so dashboards show 0 rather than no data.
	pausedCounts := map[bool]int64{false: 0, true: 0}
	for _, a := range resp.Agents {
		pausedCounts[a.Paused]++
		if a.Runs == nil {
			continue
		}
		agentAttrs := metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("agent", a.Name),
		)
		c.instruments.orgAgentRunsStarted.Record(ctx, a.Runs.Total, agentAttrs)
		c.instruments.orgAgentRunsRunning.Record(ctx, a.Runs.Running, agentAttrs)
	}

	for paused, count := range pausedCounts {
		c.instruments.orgAgentCount.Record(ctx, count, metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("paused", strconv.FormatBool(paused)),
		))
	}
}

func (c *Collector) collectAgentAutomations(ctx context.Context, org string) {
	resp, err := c.client.ListAgentAutomations(ctx, org)
	if err != nil {
		c.logger.Error("failed to list agent automations", "org", org, "error", err)
		return
	}
	if resp == nil {
		// Pulumi Agents are not enabled for the organization; nothing to record.
		return
	}

	counts := make(map[[3]string]int64) // [kind, paused, last_run_status] -> count
	for _, a := range resp.Automations {
		status := a.LastRunStatus
		if status == "" {
			status = noLastRunStatus
		}
		counts[[3]string{a.Kind, strconv.FormatBool(a.Paused), status}]++

		if a.NextExecution == "" {
			continue
		}
		next, err := time.Parse(time.RFC3339, a.NextExecution)
		if err != nil {
			c.logger.Warn("invalid agent automation next execution time",
				"org", org, "automation", a.Name, "value", a.NextExecution, "error", err)
			continue
		}
		c.instruments.orgAgentAutomationNextRun.Record(ctx, float64(next.Unix()), metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("automation", a.Name),
		))
	}

	for key, count := range counts {
		c.instruments.orgAgentAutomationCount.Record(ctx, count, metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("kind", key[0]),
			attribute.String("paused", key[1]),
			attribute.String("last_run_status", key[2]),
		))
	}
}
