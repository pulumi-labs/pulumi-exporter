package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListStacks(t *testing.T) {
	t.Parallel()

	resp := ListStacksResponse{
		Stacks: []StackSummary{
			{
				OrgName:       "myorg",
				ProjectName:   "myproject",
				StackName:     "dev",
				LastUpdate:    1700000000,
				ResourceCount: 10,
			},
			{
				OrgName:       "myorg",
				ProjectName:   "myproject",
				StackName:     "prod",
				LastUpdate:    1700000001,
				ResourceCount: 25,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/stacks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "test-token")
	result, err := c.ListStacks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(result.Stacks))
	}
	if result.Stacks[0].OrgName != "myorg" {
		t.Errorf("expected orgName 'myorg', got %q", result.Stacks[0].OrgName)
	}
	if result.Stacks[0].ProjectName != "myproject" {
		t.Errorf("expected projectName 'myproject', got %q", result.Stacks[0].ProjectName)
	}
	if result.Stacks[0].StackName != "dev" {
		t.Errorf("expected stackName 'dev', got %q", result.Stacks[0].StackName)
	}
	if result.Stacks[0].LastUpdate != 1700000000 {
		t.Errorf("expected lastUpdate 1700000000, got %d", result.Stacks[0].LastUpdate)
	}
	if result.Stacks[0].ResourceCount != 10 {
		t.Errorf("expected resourceCount 10, got %d", result.Stacks[0].ResourceCount)
	}
	if result.Stacks[1].StackName != "prod" {
		t.Errorf("expected second stack name 'prod', got %q", result.Stacks[1].StackName)
	}
}

func TestListStacksPagination(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			if r.URL.Query().Get("continuationToken") != "" {
				t.Error("first request should not have continuationToken")
			}
			resp := ListStacksResponse{
				Stacks: []StackSummary{
					{OrgName: "org1", ProjectName: "proj1", StackName: "dev"},
				},
				ContinuationToken: "page2token",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if callCount == 2 {
			token := r.URL.Query().Get("continuationToken")
			if token != "page2token" {
				t.Errorf("expected continuationToken 'page2token', got %q", token)
			}
			resp := ListStacksResponse{
				Stacks: []StackSummary{
					{OrgName: "org2", ProjectName: "proj2", StackName: "prod"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Error("unexpected third request")
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "test-token")
	result, err := c.ListStacks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
	if len(result.Stacks) != 2 {
		t.Fatalf("expected 2 stacks total, got %d", len(result.Stacks))
	}
	if result.Stacks[0].StackName != "dev" {
		t.Errorf("expected first stack 'dev', got %q", result.Stacks[0].StackName)
	}
	if result.Stacks[1].StackName != "prod" {
		t.Errorf("expected second stack 'prod', got %q", result.Stacks[1].StackName)
	}
}

func TestGetResourceCount(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/stacks/myorg/myproject/dev/resources/count"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResourceCountResponse{Count: 42, Version: 7})
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "test-token")
	result, err := c.GetResourceCount(context.Background(), "myorg", "myproject", "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 42 {
		t.Errorf("expected count 42, got %d", result.Count)
	}
	if result.Version != 7 {
		t.Errorf("expected version 7, got %d", result.Version)
	}
}

func TestListUpdates(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/stacks/myorg/myproject/dev/updates"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("expected page=1, got %q", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("pageSize") != "10" {
			t.Errorf("expected pageSize=10, got %q", r.URL.Query().Get("pageSize"))
		}

		resp := ListUpdatesResponse{
			Updates: []UpdateInfo{
				{
					Kind:      "update",
					Result:    "succeeded",
					StartTime: 1700000000,
					EndTime:   1700000060,
					ResourceChanges: map[string]int{
						"create": 3,
						"same":   5,
					},
					Version: 1,
				},
				{
					Kind:      "destroy",
					Result:    "succeeded",
					StartTime: 1700001000,
					EndTime:   1700001030,
					ResourceChanges: map[string]int{
						"delete": 8,
					},
					Version: 2,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "test-token")
	result, err := c.ListUpdates(context.Background(), "myorg", "myproject", "dev", 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Updates) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(result.Updates))
	}
	if result.Updates[0].Kind != "update" {
		t.Errorf("expected kind 'update', got %q", result.Updates[0].Kind)
	}
	if result.Updates[0].Result != "succeeded" {
		t.Errorf("expected result 'succeeded', got %q", result.Updates[0].Result)
	}
	if result.Updates[0].ResourceChanges["create"] != 3 {
		t.Errorf("expected 3 creates, got %d", result.Updates[0].ResourceChanges["create"])
	}
	if result.Updates[0].ResourceChanges["same"] != 5 {
		t.Errorf("expected 5 same, got %d", result.Updates[0].ResourceChanges["same"])
	}
	if result.Updates[1].Kind != "destroy" {
		t.Errorf("expected kind 'destroy', got %q", result.Updates[1].Kind)
	}
	if result.Updates[1].ResourceChanges["delete"] != 8 {
		t.Errorf("expected 8 deletes, got %d", result.Updates[1].ResourceChanges["delete"])
	}
}

func TestListOrgDeployments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/orgs/myorg/deployments"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		resp := ListDeploymentsResponse{
			Deployments: []DeploymentInfo{
				{
					ID:      "deploy-1",
					Status:  "running",
					Created: "2024-01-15T10:00:00Z",
				},
				{
					ID:      "deploy-2",
					Status:  "succeeded",
					Created: "2024-01-14T09:00:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "test-token")
	result, err := c.ListOrgDeployments(context.Background(), "myorg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Deployments) != 2 {
		t.Fatalf("expected 2 deployments, got %d", len(result.Deployments))
	}
	if result.Deployments[0].ID != "deploy-1" {
		t.Errorf("expected ID 'deploy-1', got %q", result.Deployments[0].ID)
	}
	if result.Deployments[0].Status != "running" {
		t.Errorf("expected status 'running', got %q", result.Deployments[0].Status)
	}
	if result.Deployments[0].Created != "2024-01-15T10:00:00Z" {
		t.Errorf("expected created '2024-01-15T10:00:00Z', got %q", result.Deployments[0].Created)
	}
	if result.Deployments[1].ID != "deploy-2" {
		t.Errorf("expected ID 'deploy-2', got %q", result.Deployments[1].ID)
	}
	if result.Deployments[1].Status != "succeeded" {
		t.Errorf("expected status 'succeeded', got %q", result.Deployments[1].Status)
	}
}

func TestAuthHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "token my-secret-token"
		if authHeader != expectedAuth {
			t.Errorf("expected Authorization header %q, got %q", expectedAuth, authHeader)
		}

		acceptHeader := r.Header.Get("Accept")
		if acceptHeader != "application/json" {
			t.Errorf("expected Accept header 'application/json', got %q", acceptHeader)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListStacksResponse{Stacks: []StackSummary{}})
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "my-secret-token")
	_, err := c.ListStacks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNon200StatusCode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "test-token")
	_, err := c.ListStacks(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 status code, got nil")
	}

	expectedSubstring := "unexpected status code 500"
	if !containsSubstring(err.Error(), expectedSubstring) {
		t.Errorf("expected error containing %q, got %q", expectedSubstring, err.Error())
	}
}

func TestMalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	c := NewPulumiClient(server.URL, "test-token")
	_, err := c.ListStacks(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}

	expectedSubstring := "decoding response"
	if !containsSubstring(err.Error(), expectedSubstring) {
		t.Errorf("expected error containing %q, got %q", expectedSubstring, err.Error())
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
