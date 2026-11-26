package client

import "time"

// ListStacksResponse represents the response from GET /api/user/stacks.
type ListStacksResponse struct {
	Stacks            []StackSummary `json:"stacks"`
	ContinuationToken string         `json:"continuationToken,omitempty"`
}

// StackSummary represents a stack summary from the Pulumi Cloud API.
type StackSummary struct {
	OrgName       string `json:"orgName"`
	ProjectName   string `json:"projectName"`
	StackName     string `json:"stackName"`
	LastUpdate    int64  `json:"lastUpdate,omitempty"`
	ResourceCount int    `json:"resourceCount,omitempty"`
}

// ListUpdatesResponse represents the response from GET /api/stacks/{org}/{project}/{stack}/updates.
type ListUpdatesResponse struct {
	Updates []UpdateInfo `json:"updates"`
}

// UpdateInfo represents a single stack update.
type UpdateInfo struct {
	Kind            string         `json:"kind"`
	Result          string         `json:"result"`
	StartTime       int64          `json:"startTime"`
	EndTime         int64          `json:"endTime"`
	ResourceChanges map[string]int `json:"resourceChanges,omitempty"`
	Version         int            `json:"version"`
}

// ResourceCountResponse represents the response from GET /api/stacks/{org}/{project}/{stack}/resources/count.
type ResourceCountResponse struct {
	Count   int `json:"count"`
	Version int `json:"version"`
}

// ListDeploymentsResponse represents the response from deployment list endpoints.
type ListDeploymentsResponse struct {
	Deployments []DeploymentInfo `json:"deployments"`
}

// DeploymentInfo represents a single deployment.
type DeploymentInfo struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Created string `json:"created"`
}

// ListMembersResponse represents the response from GET /api/orgs/{org}/members.
type ListMembersResponse struct {
	Members           []MemberInfo `json:"members"`
	ContinuationToken string       `json:"continuationToken,omitempty"`
}

// MemberInfo represents an organization member.
type MemberInfo struct {
	Role string   `json:"role"`
	User UserInfo `json:"user"`
}

// UserInfo represents basic user information.
type UserInfo struct {
	Name        string `json:"name"`
	GitHubLogin string `json:"githubLogin"`
}

// ListTeamsResponse represents the response from GET /api/orgs/{org}/teams.
type ListTeamsResponse struct {
	Teams []TeamInfo `json:"teams"`
}

// TeamInfo represents a team.
type TeamInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Kind        string `json:"kind"`
	Members     []any  `json:"members"`
}

// ListEnvironmentsResponse represents the response from GET /api/esc/environments/{org}.
type ListEnvironmentsResponse struct {
	Environments []EnvironmentInfo `json:"environments"`
	NextToken    string            `json:"nextToken,omitempty"`
}

// EnvironmentInfo represents an ESC environment.
type EnvironmentInfo struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Project      string `json:"project"`
}

// ListPolicyGroupsResponse represents the response from GET /api/orgs/{org}/policygroups.
type ListPolicyGroupsResponse struct {
	PolicyGroups []PolicyGroupInfo `json:"policyGroups"`
}

// PolicyGroupInfo represents a policy group summary.
type PolicyGroupInfo struct {
	Name                  string `json:"name"`
	NumStacks             int    `json:"numStacks"`
	NumEnabledPolicyPacks int    `json:"numEnabledPolicyPacks"`
	IsOrgDefault          bool   `json:"isOrgDefault"`
}

// ListPolicyPacksResponse represents the response from GET /api/orgs/{org}/policypacks.
type ListPolicyPacksResponse struct {
	PolicyPacks []PolicyPackInfo `json:"policyPacks"`
}

// PolicyPackInfo represents a policy pack with versions.
type PolicyPackInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// ListPolicyViolationsResponse represents the response from GET /api/orgs/{org}/policyresults/violationsv2.
type ListPolicyViolationsResponse struct {
	PolicyViolations  []PolicyViolation `json:"policyViolations"`
	ContinuationToken string            `json:"continuationToken,omitempty"`
}

// PolicyViolation represents a policy violation.
type PolicyViolation struct {
	ID          string `json:"id"`
	ProjectName string `json:"projectName"`
	StackName   string `json:"stackName"`
	PolicyPack  string `json:"policyPack"`
	PolicyName  string `json:"policyName"`
	Level       string `json:"level"`
	Kind        string `json:"kind"`
}

// PolicyResultsMetadataResponse represents the response from GET /api/orgs/{org}/policyresults/metadata.
type PolicyResultsMetadataResponse struct {
	PolicyTotalCount         int64
	PolicyWithIssuesCount    int64
	ResourcesTotalCount      int64
	ResourcesWithIssuesCount int64
}

// ListNeoTasksResponse represents the response from GET /api/preview/agents/{org}/tasks.
type ListNeoTasksResponse struct {
	Tasks             []NeoTask `json:"tasks"`
	ContinuationToken string    `json:"continuationToken,omitempty"`
}

// NeoTask represents a Neo AI task.
type NeoTask struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	TokensUsed int64     `json:"tokensUsed"`
	CreatedAt  time.Time `json:"createdAt"`
}

// NeoTokenBudgetResponse represents the response from GET /api/orgs/{org}/neo/token-budget.
type NeoTokenBudgetResponse struct {
	BaseAllowanceTokens      int64
	EffectiveAllowanceTokens int64
	ConsumedTokens           int64
	WindowEnd                int64
	WindowKind               string
	Exhausted                bool
}

// ListAgentsResponse represents the response from GET /api/preview/agents/{org}/definitions.
type ListAgentsResponse struct {
	Agents []Agent
}

// Agent represents a Pulumi Agent definition.
type Agent struct {
	ID     string
	Name   string
	Paused bool
	// Runs is nil when the API does not report run counts for the agent.
	Runs *AgentRunCounts
}

// AgentRunCounts holds the organization-wide run counts of a Pulumi Agent.
type AgentRunCounts struct {
	Total   int64
	Running int64
}

// ListAgentAutomationsResponse represents the response from GET /api/v2/agents/{org}/automations.
type ListAgentAutomationsResponse struct {
	Automations []AgentAutomation
}

// AgentAutomation represents a scheduled or triggered Pulumi Agent automation.
type AgentAutomation struct {
	ID            string
	Name          string
	Kind          string
	Paused        bool
	LastRunStatus string
	// NextExecution is the RFC 3339 time of the next scheduled run, empty when none is scheduled.
	NextExecution string
}

// ListDeliveryProgramsResponse represents the response from GET /api/preview/delivery/{org}/programs.
type ListDeliveryProgramsResponse struct {
	Programs []DeliveryProgram
}

// DeliveryProgram represents the Pulumi Delivery program of a stack.
type DeliveryProgram struct {
	Project          string
	Stack            string
	Registered       bool
	PendingApprovals int64
	// LastRun is nil when the program has not run yet.
	LastRun *DeliveryRun
	Stages  []DeliveryStage
}

// DeliveryRun summarizes a Pulumi Delivery run.
type DeliveryRun struct {
	Number    int64
	Status    string
	WaitKind  string
	CreatedAt time.Time
	// FinishedAt is zero while the run has not ended.
	FinishedAt time.Time
}

// DeliveryStage represents a stage of a Delivery program as of its latest run.
type DeliveryStage struct {
	Name     string
	Status   string
	WaitKind string
	// DeployedRelease is 0 when no release is deployed at the stage.
	DeployedRelease int64
}

// ListDeliveryApprovalsResponse represents the response from GET /api/preview/delivery/{org}/approvals.
type ListDeliveryApprovalsResponse struct {
	Approvals []DeliveryApproval
}

// DeliveryApproval represents an approval requested by a Delivery gate.
type DeliveryApproval struct {
	Project     string
	Stack       string
	Stage       string
	RequestedAt time.Time
}
