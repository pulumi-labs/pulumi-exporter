// Package client provides a typed wrapper around the generated Pulumi Cloud OpenAPI client.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pulumi-labs/pulumi-exporter/internal/pulumiapi"
)

// Client wraps the generated OpenAPI client with typed convenience methods.
type Client struct {
	gen *pulumiapi.ClientWithResponses
}

// NewClient creates a new Pulumi Cloud API client.
func NewClient(baseURL, token string) (*Client, error) {
	authProvider := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "token "+token)
		return nil
	}

	// The generated client appends the trailing slash it needs for relative
	// URL resolution; reject anything that is not an absolute URL up front.
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing API URL: %w", err)
	}
	if !u.IsAbs() || u.Host == "" {
		return nil, fmt.Errorf("parsing API URL %q: %w", baseURL, ErrInvalidAPIURL)
	}

	httpClient := &http.Client{Timeout: requestTimeout}
	gen, err := pulumiapi.NewClientWithResponses(u.String(),
		pulumiapi.WithHTTPClient(httpClient),
		pulumiapi.WithRequestEditorFn(authProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	return &Client{gen: gen}, nil
}

// ListStacks returns all stacks accessible to the authenticated user, handling pagination.
func (c *Client) ListStacks(ctx context.Context) (*ListStacksResponse, error) {
	var allStacks []StackSummary
	var contToken *string

	for {
		resp, err := c.gen.ListUserStacksWithResponse(ctx, &pulumiapi.ListUserStacksParams{
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing stacks: %w", err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, unexpectedStatus("listing stacks", resp.StatusCode())
		}

		for i := range resp.JSON200.Stacks {
			s := &resp.JSON200.Stacks[i]
			allStacks = append(allStacks, StackSummary{
				OrgName:       s.OrgName,
				ProjectName:   s.ProjectName,
				StackName:     s.StackName,
				LastUpdate:    derefInt64(s.LastUpdate),
				ResourceCount: int(derefInt64(s.ResourceCount)),
			})
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListStacksResponse{Stacks: allStacks}, nil
}

// GetResourceCount returns the resource count for a specific stack.
func (c *Client) GetResourceCount(ctx context.Context, org, project, stack string) (*ResourceCountResponse, error) {
	resp, err := c.gen.GetStackResourceCountWithResponse(ctx, org, project, stack)
	if err != nil {
		return nil, fmt.Errorf("getting resource count: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("getting resource count", resp.StatusCode())
	}

	return &ResourceCountResponse{
		Count:   int(resp.JSON200.ResourceCount),
		Version: int(resp.JSON200.Version),
	}, nil
}

// ListUpdates returns the updates for a specific stack.
// The Pulumi OpenAPI spec returns an untyped response for this endpoint,
// so we parse the raw JSON body from the generated client's response.
func (c *Client) ListUpdates(ctx context.Context, org, project, stack string, page, pageSize int) (*ListUpdatesResponse, error) {
	p := int64(page)
	ps := int64(pageSize)
	resp, err := c.gen.GetStackUpdatesWithResponse(ctx, org, project, stack, &pulumiapi.GetStackUpdatesParams{
		Page:     &p,
		PageSize: &ps,
	})
	if err != nil {
		return nil, fmt.Errorf("listing updates: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, unexpectedStatus("listing updates", resp.StatusCode())
	}

	var result ListUpdatesResponse
	if err = json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("listing updates: decoding response: %w", err)
	}

	return &result, nil
}

// ListOrgDeployments returns the deployments for an organization.
func (c *Client) ListOrgDeployments(ctx context.Context, org string) (*ListDeploymentsResponse, error) {
	resp, err := c.gen.ListOrgDeploymentsWithResponse(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("listing org deployments: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("listing org deployments", resp.StatusCode())
	}

	deployments := make([]DeploymentInfo, 0, len(resp.JSON200.Deployments))
	for i := range resp.JSON200.Deployments {
		d := &resp.JSON200.Deployments[i]
		deployments = append(deployments, DeploymentInfo{
			ID:      d.Id,
			Status:  string(d.Status),
			Created: d.Created,
		})
	}

	return &ListDeploymentsResponse{Deployments: deployments}, nil
}

// ListMembers returns the members of an organization, handling pagination.
func (c *Client) ListMembers(ctx context.Context, org string) (*ListMembersResponse, error) {
	var allMembers []MemberInfo
	var contToken *string

	for {
		resp, err := c.gen.ListOrganizationMembersWithResponse(ctx, org,
			&pulumiapi.ListOrganizationMembersParams{ContinuationToken: contToken})
		if err != nil {
			return nil, fmt.Errorf("listing members: %w", err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, unexpectedStatus("listing members", resp.StatusCode())
		}

		for i := range resp.JSON200.Members {
			m := &resp.JSON200.Members[i]
			allMembers = append(allMembers, MemberInfo{
				Role: string(m.Role),
				User: UserInfo{Name: m.User.Name, GitHubLogin: m.User.GithubLogin},
			})
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListMembersResponse{Members: allMembers}, nil
}

// ListTeams returns the teams of an organization.
func (c *Client) ListTeams(ctx context.Context, org string) (*ListTeamsResponse, error) {
	resp, err := c.gen.ListTeamsWithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("listing teams", resp.StatusCode())
	}

	teams := make([]TeamInfo, 0, len(resp.JSON200.Teams))
	for i := range resp.JSON200.Teams {
		t := &resp.JSON200.Teams[i]
		teams = append(teams, TeamInfo{
			Name:        t.Name,
			DisplayName: t.DisplayName,
			Kind:        string(t.Kind),
		})
	}

	return &ListTeamsResponse{Teams: teams}, nil
}

// ListEnvironments returns the ESC environments of an organization, handling pagination.
func (c *Client) ListEnvironments(ctx context.Context, org string) (*ListEnvironmentsResponse, error) {
	var allEnvs []EnvironmentInfo
	var contToken *string

	for {
		resp, err := c.gen.ListOrgEnvironmentsEscWithResponse(ctx, org,
			&pulumiapi.ListOrgEnvironmentsEscParams{ContinuationToken: contToken})
		if err != nil {
			return nil, fmt.Errorf("listing environments: %w", err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, unexpectedStatus("listing environments", resp.StatusCode())
		}

		for i := range resp.JSON200.Environments {
			e := &resp.JSON200.Environments[i]
			allEnvs = append(allEnvs, EnvironmentInfo{
				Name:         e.Name,
				Organization: e.Organization,
				Project:      derefStr(e.Project),
			})
		}

		if resp.JSON200.NextToken == nil || *resp.JSON200.NextToken == "" {
			break
		}
		contToken = resp.JSON200.NextToken
	}

	return &ListEnvironmentsResponse{Environments: allEnvs}, nil
}

// ListPolicyGroups returns the policy groups of an organization.
func (c *Client) ListPolicyGroups(ctx context.Context, org string) (*ListPolicyGroupsResponse, error) {
	resp, err := c.gen.ListPolicyGroupsWithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing policy groups: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("listing policy groups", resp.StatusCode())
	}

	groups := make([]PolicyGroupInfo, 0, len(resp.JSON200.PolicyGroups))
	for i := range resp.JSON200.PolicyGroups {
		g := &resp.JSON200.PolicyGroups[i]
		groups = append(groups, PolicyGroupInfo{
			Name:                  g.Name,
			NumStacks:             int(g.NumStacks),
			NumEnabledPolicyPacks: int(g.NumEnabledPolicyPacks),
			IsOrgDefault:          g.IsOrgDefault,
		})
	}

	return &ListPolicyGroupsResponse{PolicyGroups: groups}, nil
}

// ListPolicyPacks returns the policy packs of an organization.
func (c *Client) ListPolicyPacks(ctx context.Context, org string) (*ListPolicyPacksResponse, error) {
	resp, err := c.gen.ListPolicyPacksOrgsWithResponse(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("listing policy packs: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("listing policy packs", resp.StatusCode())
	}

	packs := make([]PolicyPackInfo, 0, len(resp.JSON200.PolicyPacks))
	for i := range resp.JSON200.PolicyPacks {
		p := &resp.JSON200.PolicyPacks[i]
		packs = append(packs, PolicyPackInfo{
			Name:        p.Name,
			DisplayName: p.DisplayName,
		})
	}

	return &ListPolicyPacksResponse{PolicyPacks: packs}, nil
}

// ListPolicyViolations returns the policy violations for an organization.
func (c *Client) ListPolicyViolations(ctx context.Context, org string) (*ListPolicyViolationsResponse, error) {
	resp, err := c.gen.ListPolicyViolationsV2WithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing policy violations: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("listing policy violations", resp.StatusCode())
	}

	violations := make([]PolicyViolation, 0, len(resp.JSON200.PolicyViolations))
	for i := range resp.JSON200.PolicyViolations {
		v := &resp.JSON200.PolicyViolations[i]
		violations = append(violations, PolicyViolation{
			ID:          v.Id,
			ProjectName: v.ProjectName,
			StackName:   derefStr(v.StackName),
			PolicyPack:  v.PolicyPack,
			PolicyName:  v.PolicyName,
			Level:       v.Level,
			Kind:        string(v.Kind),
		})
	}

	return &ListPolicyViolationsResponse{PolicyViolations: violations}, nil
}

// GetPolicyResultsMetadata returns policy compliance metadata for an organization.
func (c *Client) GetPolicyResultsMetadata(ctx context.Context, org string) (*PolicyResultsMetadataResponse, error) {
	resp, err := c.gen.GetPolicyResultsMetadataWithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("getting policy results metadata: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("getting policy results metadata", resp.StatusCode())
	}

	return &PolicyResultsMetadataResponse{
		PolicyTotalCount:         resp.JSON200.PolicyTotalCount,
		PolicyWithIssuesCount:    resp.JSON200.PolicyWithIssuesCount,
		ResourcesTotalCount:      resp.JSON200.ResourcesTotalCount,
		ResourcesWithIssuesCount: resp.JSON200.ResourcesWithIssuesCount,
	}, nil
}

// ListNeoTasks returns all Neo AI tasks for an organization, handling pagination.
func (c *Client) ListNeoTasks(ctx context.Context, org string) (*ListNeoTasksResponse, error) {
	var allTasks []NeoTask
	var contToken *string
	pageSize := int64(defaultPageSize)

	for {
		resp, err := c.gen.ListTasksWithResponse(ctx, org, &pulumiapi.ListTasksParams{
			PageSize:          &pageSize,
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing neo tasks: %w", err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, unexpectedStatus("listing neo tasks", resp.StatusCode())
		}

		for i := range resp.JSON200.Tasks {
			t := &resp.JSON200.Tasks[i]
			allTasks = append(allTasks, NeoTask{
				ID:         t.Id,
				Name:       t.Name,
				Status:     string(t.Status),
				TokensUsed: t.TokensUsed,
				CreatedAt:  t.CreatedAt,
			})
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListNeoTasksResponse{Tasks: allTasks}, nil
}

// GetOrgNeoTokenBudget returns the Pulumi Neo token budget for an organization.
// It returns an error wrapping ErrFeatureNotEnabled when the organization has
// no Neo token budget.
func (c *Client) GetOrgNeoTokenBudget(ctx context.Context, org string) (*NeoTokenBudgetResponse, error) {
	// No taskId: report the effective budget for the token's own identity.
	resp, err := c.gen.GetOrgNeoTokenBudgetWithResponse(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("getting neo token budget: %w", err)
	}
	// A 404 means the organization has no Neo token budget (Neo not enabled or
	// no plan-derived allowance).
	if resp.StatusCode() == http.StatusNotFound {
		return nil, featureNotEnabled("getting neo token budget")
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("getting neo token budget", resp.StatusCode())
	}

	return &NeoTokenBudgetResponse{
		BaseAllowanceTokens:      resp.JSON200.BaseAllowanceTokens,
		EffectiveAllowanceTokens: resp.JSON200.EffectiveAllowanceTokens,
		ConsumedTokens:           resp.JSON200.ConsumedTokens,
		WindowEnd:                resp.JSON200.WindowEnd,
		WindowKind:               string(resp.JSON200.WindowKind),
		Exhausted:                resp.JSON200.Exhausted,
	}, nil
}

// ListAgents returns the Pulumi Agents defined in an organization.
// It returns an error wrapping ErrFeatureNotEnabled when the organization does
// not have Pulumi Agents enabled.
func (c *Client) ListAgents(ctx context.Context, org string) (*ListAgentsResponse, error) {
	resp, err := c.gen.ListAgentDefinitionsWithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, featureNotEnabled("listing agents")
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, unexpectedStatus("listing agents", resp.StatusCode())
	}

	agents := make([]Agent, 0, len(resp.JSON200.Definitions))
	for i := range resp.JSON200.Definitions {
		d := &resp.JSON200.Definitions[i]
		a := Agent{
			ID:     d.Id,
			Name:   d.Name,
			Paused: derefStr(d.PausedAt) != "",
		}
		if d.TaskCounts != nil {
			a.Runs = &AgentRunCounts{
				Total:   int64(d.TaskCounts.Total),
				Running: int64(d.TaskCounts.Running),
			}
		}
		agents = append(agents, a)
	}

	return &ListAgentsResponse{Agents: agents}, nil
}

// ListAgentAutomations returns all Pulumi Agent automations for an organization,
// handling pagination. It returns an error wrapping ErrFeatureNotEnabled
// when the organization does not have Pulumi Agents enabled.
func (c *Client) ListAgentAutomations(ctx context.Context, org string) (*ListAgentAutomationsResponse, error) {
	var all []AgentAutomation
	var contToken *string
	pageSize := int64(defaultPageSize)

	for {
		resp, err := c.gen.ListAgentAutomationsV2WithResponse(ctx, org, &pulumiapi.ListAgentAutomationsV2Params{
			PageSize:          &pageSize,
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing agent automations: %w", err)
		}
		// A 404 on the first page means the feature is off; on a later page it is a failure.
		if resp.StatusCode() == http.StatusNotFound && contToken == nil {
			return nil, featureNotEnabled("listing agent automations")
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, unexpectedStatus("listing agent automations", resp.StatusCode())
		}

		for i := range resp.JSON200.Automations {
			a := &resp.JSON200.Automations[i]
			all = append(all, AgentAutomation{
				ID:            a.Id,
				Name:          a.Name,
				Kind:          string(a.Kind),
				Paused:        a.Paused,
				LastRunStatus: derefStr(a.LastRunStatus),
				NextExecution: derefStr(a.NextExecution),
			})
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListAgentAutomationsResponse{Automations: all}, nil
}

// ListDeliveryPrograms returns all Pulumi Delivery programs of an organization,
// handling pagination. It returns an error wrapping ErrFeatureNotEnabled
// when the organization does not have Delivery enabled.
func (c *Client) ListDeliveryPrograms(ctx context.Context, org string) (*ListDeliveryProgramsResponse, error) {
	var programs []DeliveryProgram
	var contToken *string
	maxResults := int64(defaultPageSize)

	for {
		resp, err := c.gen.ListDeliveryProgramsWithResponse(ctx, org, &pulumiapi.ListDeliveryProgramsParams{
			MaxResults:        &maxResults,
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing delivery programs: %w", err)
		}
		// A 404 on the first page means the feature is off; on a later page it is a failure.
		if resp.StatusCode() == http.StatusNotFound && contToken == nil {
			return nil, featureNotEnabled("listing delivery programs")
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, unexpectedStatus("listing delivery programs", resp.StatusCode())
		}

		if resp.JSON200.Programs != nil {
			for i := range *resp.JSON200.Programs {
				p := &(*resp.JSON200.Programs)[i]
				programs = append(programs, toDeliveryProgram(p))
			}
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListDeliveryProgramsResponse{Programs: programs}, nil
}

func toDeliveryProgram(p *pulumiapi.DeliveryProgramRow) DeliveryProgram {
	program := DeliveryProgram{
		Project:          p.Project,
		Stack:            p.Stack,
		Registered:       p.Registered,
		PendingApprovals: p.PendingApprovals,
	}
	if p.LastRun != nil {
		run := &DeliveryRun{
			Number:    p.LastRun.Number,
			Status:    string(p.LastRun.Status.Status),
			WaitKind:  deliveryWaitKind(p.LastRun.Status),
			CreatedAt: p.LastRun.CreatedAt,
		}
		if p.LastRun.FinishedAt != nil {
			run.FinishedAt = *p.LastRun.FinishedAt
		}
		program.LastRun = run
	}
	if p.Stages != nil {
		for i := range *p.Stages {
			s := &(*p.Stages)[i]
			program.Stages = append(program.Stages, DeliveryStage{
				Name:            s.Name,
				Status:          string(s.Status.Status),
				WaitKind:        deliveryWaitKind(s.Status),
				DeployedRelease: derefInt64(s.DeployedReleaseNumber),
			})
		}
	}
	return program
}

func deliveryWaitKind(s pulumiapi.DeliveryStatus) string {
	if s.WaitKind == nil {
		return ""
	}
	return string(*s.WaitKind)
}

// ListPendingDeliveryApprovals returns the pending approvals of Delivery gates in
// an organization, handling pagination. It returns an error wrapping
// ErrFeatureNotEnabled when the organization does not have Delivery enabled.
func (c *Client) ListPendingDeliveryApprovals(ctx context.Context, org string) (*ListDeliveryApprovalsResponse, error) {
	var approvals []DeliveryApproval
	var contToken *string
	maxResults := int64(defaultPageSize)
	status := pulumiapi.Pending

	for {
		resp, err := c.gen.ListDeliveryApprovalsWithResponse(ctx, org, &pulumiapi.ListDeliveryApprovalsParams{
			MaxResults:        &maxResults,
			ContinuationToken: contToken,
			Status:            &status,
		})
		if err != nil {
			return nil, fmt.Errorf("listing delivery approvals: %w", err)
		}
		// A 404 on the first page means the feature is off; on a later page it is a failure.
		if resp.StatusCode() == http.StatusNotFound && contToken == nil {
			return nil, featureNotEnabled("listing delivery approvals")
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, unexpectedStatus("listing delivery approvals", resp.StatusCode())
		}

		if resp.JSON200.Approvals != nil {
			for i := range *resp.JSON200.Approvals {
				a := &(*resp.JSON200.Approvals)[i]
				approvals = append(approvals, DeliveryApproval{
					Project:     a.Project,
					Stack:       a.Stack,
					Stage:       derefStr(a.StageName),
					RequestedAt: a.RequestedAt,
				})
			}
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListDeliveryApprovalsResponse{Approvals: approvals}, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
