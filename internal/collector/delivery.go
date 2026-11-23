package collector

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/pulumi-labs/pulumi-exporter/internal/client"
)

// noLastRun labels Delivery programs that have not run yet.
const noLastRun = "none"

// deliveryStatuses are the statuses the Pulumi Cloud API documents for Delivery
// runs and stages. They are always recorded, so dashboards show 0 instead of no
// data; statuses the API adds later are still counted.
var deliveryStatuses = []string{
	"in_progress", "waiting", "running", "pending", "blocked", "succeeded",
	"failed", "canceled", "superseded", "skipped", "closed", "unchanged",
}

// deliveryWaitKinds are the documented reasons a Delivery stage waits.
var deliveryWaitKinds = []string{
	"approval", "decision", "manual", "queue", "stage", "dependency",
	"target", "inputs", "scope", "pass_retry", "paused",
}

func (c *Collector) collectDeliveryPrograms(ctx context.Context, org string) {
	resp, err := c.client.ListDeliveryPrograms(ctx, org)
	if err != nil {
		c.logger.Error("failed to list delivery programs", "org", org, "error", err)
		return
	}
	if resp == nil {
		// Delivery is not enabled for the organization; nothing to record.
		return
	}

	registered := map[bool]int64{false: 0, true: 0}
	lastRunCounts := zeroCounts(append([]string{noLastRun}, deliveryStatuses...))
	stageCounts := zeroCounts(deliveryStatuses)
	waitCounts := zeroCounts(deliveryWaitKinds)

	for _, p := range resp.Programs {
		registered[p.Registered]++
		if p.LastRun == nil {
			lastRunCounts[noLastRun]++
		} else {
			lastRunCounts[p.LastRun.Status]++
		}
		for _, s := range p.Stages {
			stageCounts[s.Status]++
			if s.WaitKind != "" {
				waitCounts[s.WaitKind]++
			}
		}
		c.recordDeliveryProgram(ctx, org, p)
	}

	for reg, count := range registered {
		c.instruments.orgDeliveryProgramCount.Record(ctx, count, metric.WithAttributes(
			attribute.String("org", org),
			attribute.String("registered", strconv.FormatBool(reg)),
		))
	}
	c.recordCounts(ctx, c.instruments.orgDeliveryProgramsByLastRun, org, "status", lastRunCounts)
	c.recordCounts(ctx, c.instruments.orgDeliveryStagesByStatus, org, "status", stageCounts)
	c.recordCounts(ctx, c.instruments.orgDeliveryStagesWaiting, org, "wait_kind", waitCounts)
}

// recordDeliveryProgram records the per-program and per-stage Delivery metrics.
func (c *Collector) recordDeliveryProgram(ctx context.Context, org string, p client.DeliveryProgram) {
	programAttrs := []attribute.KeyValue{
		attribute.String("org", org),
		attribute.String("project", p.Project),
		attribute.String("stack", p.Stack),
	}
	programOpt := metric.WithAttributes(programAttrs...)

	c.instruments.deliveryProgramPendingApprovals.Record(ctx, p.PendingApprovals, programOpt)

	if p.LastRun != nil {
		c.instruments.deliveryProgramLastRunTimestamp.Record(ctx, float64(p.LastRun.CreatedAt.Unix()), programOpt)
		if !p.LastRun.FinishedAt.IsZero() {
			duration := p.LastRun.FinishedAt.Sub(p.LastRun.CreatedAt).Seconds()
			c.instruments.deliveryProgramLastRunDuration.Record(ctx, duration, programOpt)
		}
	}

	for _, s := range p.Stages {
		if s.DeployedRelease == 0 {
			continue
		}
		stageAttrs := append(append([]attribute.KeyValue{}, programAttrs...), attribute.String("stage", s.Name))
		c.instruments.deliveryStageDeployedRelease.Record(ctx, s.DeployedRelease, metric.WithAttributes(stageAttrs...))
	}
}

func (c *Collector) collectDeliveryApprovals(ctx context.Context, org string) {
	resp, err := c.client.ListPendingDeliveryApprovals(ctx, org)
	if err != nil {
		c.logger.Error("failed to list delivery approvals", "org", org, "error", err)
		return
	}
	if resp == nil {
		// Delivery is not enabled for the organization; nothing to record.
		return
	}

	var oldest time.Time
	for _, a := range resp.Approvals {
		if oldest.IsZero() || a.RequestedAt.Before(oldest) {
			oldest = a.RequestedAt
		}
	}

	var age float64
	if !oldest.IsZero() {
		age = time.Since(oldest).Seconds()
	}
	c.instruments.orgDeliveryOldestApprovalAge.Record(ctx, age, metric.WithAttributes(attribute.String("org", org)))
}

// zeroCounts returns a count map with an entry of 0 for each key.
func zeroCounts(keys []string) map[string]int64 {
	counts := make(map[string]int64, len(keys))
	for _, k := range keys {
		counts[k] = 0
	}
	return counts
}

// recordCounts records one data point per key, labeled with the org and the key.
func (c *Collector) recordCounts(ctx context.Context, gauge metric.Int64Gauge, org, label string, counts map[string]int64) {
	for key, count := range counts {
		gauge.Record(ctx, count, metric.WithAttributes(
			attribute.String("org", org),
			attribute.String(label, key),
		))
	}
}
