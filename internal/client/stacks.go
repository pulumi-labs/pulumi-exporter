package client

import (
	"context"
	"fmt"
)

// ListStacks returns all stacks accessible to the authenticated user, handling pagination.
func (c *PulumiClient) ListStacks(ctx context.Context) (*ListStacksResponse, error) {
	var allStacks []StackSummary
	continuationToken := ""

	for {
		path := "/api/user/stacks"
		if continuationToken != "" {
			path += "?continuationToken=" + continuationToken
		}

		body, err := c.doRequest(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("listing stacks: %w", err)
		}

		resp, err := decodeJSON[ListStacksResponse](body)
		if err != nil {
			return nil, fmt.Errorf("listing stacks: %w", err)
		}

		allStacks = append(allStacks, resp.Stacks...)

		if resp.ContinuationToken == "" {
			break
		}
		continuationToken = resp.ContinuationToken
	}

	return &ListStacksResponse{Stacks: allStacks}, nil
}

// GetResourceCount returns the resource count for a specific stack.
func (c *PulumiClient) GetResourceCount(ctx context.Context, org, project, stack string) (*ResourceCountResponse, error) {
	path := fmt.Sprintf("/api/stacks/%s/%s/%s/resources/count", org, project, stack)

	body, err := c.doRequest(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting resource count: %w", err)
	}

	resp, err := decodeJSON[ResourceCountResponse](body)
	if err != nil {
		return nil, fmt.Errorf("getting resource count: %w", err)
	}

	return resp, nil
}
