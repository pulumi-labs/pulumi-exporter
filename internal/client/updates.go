package client

import (
	"context"
	"fmt"
)

// ListUpdates returns the updates for a specific stack with pagination.
func (c *PulumiClient) ListUpdates(ctx context.Context, org, project, stack string, page, pageSize int) (*ListUpdatesResponse, error) {
	path := fmt.Sprintf("/api/stacks/%s/%s/%s/updates?page=%d&pageSize=%d", org, project, stack, page, pageSize)

	body, err := c.doRequest(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing updates: %w", err)
	}

	resp, err := decodeJSON[ListUpdatesResponse](body)
	if err != nil {
		return nil, fmt.Errorf("listing updates: %w", err)
	}

	return resp, nil
}
