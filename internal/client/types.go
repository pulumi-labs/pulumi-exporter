package client

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
