package client

import (
	"context"
	"fmt"
)

// ListDeployments returns the deployments for a specific stack.
func (c *PulumiClient) ListDeployments(ctx context.Context, org, project, stack string) (*ListDeploymentsResponse, error) {
	path := fmt.Sprintf("/api/stacks/%s/%s/%s/deployments", org, project, stack)

	body, err := c.doRequest(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}

	resp, err := decodeJSON[ListDeploymentsResponse](body)
	if err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}

	return resp, nil
}

// ListOrgDeployments returns the deployments for an organization.
func (c *PulumiClient) ListOrgDeployments(ctx context.Context, org string) (*ListDeploymentsResponse, error) {
	path := fmt.Sprintf("/api/orgs/%s/deployments", org)

	body, err := c.doRequest(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing org deployments: %w", err)
	}

	resp, err := decodeJSON[ListDeploymentsResponse](body)
	if err != nil {
		return nil, fmt.Errorf("listing org deployments: %w", err)
	}

	return resp, nil
}
