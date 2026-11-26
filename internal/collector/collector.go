// Package collector implements periodic metrics collection from Pulumi Cloud.
package collector

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/pulumi-labs/pulumi-exporter/internal/client"
	"github.com/pulumi-labs/pulumi-exporter/internal/config"
)

// PulumiAPI defines the interface for interacting with the Pulumi Cloud API.
type PulumiAPI interface {
	ListStacks(ctx context.Context) (*client.ListStacksResponse, error)
	GetResourceCount(ctx context.Context, org, project, stack string) (*client.ResourceCountResponse, error)
	ListUpdates(ctx context.Context, org, project, stack string, page, pageSize int) (*client.ListUpdatesResponse, error)
	ListOrgDeployments(ctx context.Context, org string) (*client.ListDeploymentsResponse, error)
	ListMembers(ctx context.Context, org string) (*client.ListMembersResponse, error)
	ListTeams(ctx context.Context, org string) (*client.ListTeamsResponse, error)
	ListEnvironments(ctx context.Context, org string) (*client.ListEnvironmentsResponse, error)
	ListPolicyGroups(ctx context.Context, org string) (*client.ListPolicyGroupsResponse, error)
	ListPolicyPacks(ctx context.Context, org string) (*client.ListPolicyPacksResponse, error)
	ListPolicyViolations(ctx context.Context, org string) (*client.ListPolicyViolationsResponse, error)
	ListNeoTasks(ctx context.Context, org string) (*client.ListNeoTasksResponse, error)
	GetOrgNeoTokenBudget(ctx context.Context, org string) (*client.NeoTokenBudgetResponse, error)
	GetPolicyResultsMetadata(ctx context.Context, org string) (*client.PolicyResultsMetadataResponse, error)
	ListAgents(ctx context.Context, org string) (*client.ListAgentsResponse, error)
	ListAgentAutomations(ctx context.Context, org string) (*client.ListAgentAutomationsResponse, error)
	ListDeliveryPrograms(ctx context.Context, org string) (*client.ListDeliveryProgramsResponse, error)
	ListPendingDeliveryApprovals(ctx context.Context, org string) (*client.ListDeliveryApprovalsResponse, error)
}

// Collector periodically collects metrics from the Pulumi Cloud API.
type Collector struct {
	client          PulumiAPI
	cfg             *config.Config
	logger          *slog.Logger
	mu              sync.Mutex
	lastSeenVersion map[stackKey]int
	instruments     *Instruments
}

// stackKey identifies a stack across organizations and projects.
type stackKey struct {
	org, project, stack string
}

const (
	// collectTimeoutRatio is the share of the collect interval a single
	// collection run may take.
	collectTimeoutRatio = 0.9
	// minCollectTimeout is the lower bound for a single collection run.
	minCollectTimeout = 10 * time.Second
	// orgConcurrency bounds how many organizations are collected in parallel.
	orgConcurrency = 3
)

// NewCollector creates a new Collector.
func NewCollector(apiClient PulumiAPI, cfg *config.Config, meter metric.Meter, logger *slog.Logger) (*Collector, error) {
	instruments, err := NewInstruments(meter)
	if err != nil {
		return nil, fmt.Errorf("creating instruments: %w", err)
	}

	return &Collector{
		client:          apiClient,
		cfg:             cfg,
		logger:          logger,
		lastSeenVersion: make(map[stackKey]int),
		instruments:     instruments,
	}, nil
}

// Run starts the collection loop, polling at the configured interval.
func (c *Collector) Run(ctx context.Context) error {
	c.logger.InfoContext(ctx, "starting collector", "interval", c.cfg.Pulumi.CollectInterval)

	// Collect immediately on start.
	c.collect(ctx)

	ticker := time.NewTicker(c.cfg.Pulumi.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.InfoContext(ctx, "collector stopped")
			return fmt.Errorf("collector stopped: %w", ctx.Err())
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

func (c *Collector) collect(ctx context.Context) {
	c.logger.InfoContext(ctx, "collecting metrics")

	// Apply a collection timeout: 90% of the collect interval, clamped to a 10s minimum.
	timeout := max(time.Duration(float64(c.cfg.Pulumi.CollectInterval)*collectTimeoutRatio), minCollectTimeout)
	collectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stacks, err := c.client.ListStacks(collectCtx)
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to list stacks", "error", err)
		return
	}

	// Keep only the stacks of configured organizations.
	orgSet := make(map[string]struct{}, len(c.cfg.Pulumi.Organizations))
	for _, org := range c.cfg.Pulumi.Organizations {
		orgSet[org] = struct{}{}
	}
	monitored := make([]client.StackSummary, 0, len(stacks.Stacks))
	for _, stack := range stacks.Stacks {
		if _, ok := orgSet[stack.OrgName]; ok {
			monitored = append(monitored, stack)
		}
	}

	runBounded(collectCtx, c.cfg.Pulumi.MaxConcurrency, monitored, func(stack client.StackSummary) {
		c.collectStack(collectCtx, stack)
	})

	runBounded(collectCtx, orgConcurrency, c.cfg.Pulumi.Organizations, func(org string) {
		c.collectOrgDeployments(collectCtx, org)
		c.collectOrgMetrics(collectCtx, org)
	})

	if ctxErr := collectCtx.Err(); ctxErr != nil {
		c.logger.WarnContext(ctx, "collection stopped before completion", "error", ctxErr)
		return
	}

	c.logger.InfoContext(ctx, "collection complete")
}

// runBounded calls fn for each item with at most limit calls in flight and
// waits for them to finish. Once ctx is done it stops starting new calls.
func runBounded[T any](ctx context.Context, limit int, items []T, fn func(T)) {
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	defer wg.Wait()

	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}
		// select picks at random when both cases are ready, so check again
		// after acquiring a slot.
		if ctx.Err() != nil {
			<-sem
			return
		}

		wg.Go(func() {
			defer func() { <-sem }()
			fn(item)
		})
	}
}
