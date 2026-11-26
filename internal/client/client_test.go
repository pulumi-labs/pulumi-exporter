package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientAPIURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		wantErr error
	}{
		{name: "default", baseURL: "https://api.pulumi.com"},
		{name: "trailing slash", baseURL: "https://api.pulumi.com/"},
		{name: "with path", baseURL: "https://pulumi.example.com/api-proxy"},
		{name: "missing scheme", baseURL: "api.pulumi.com", wantErr: ErrInvalidAPIURL},
		{name: "missing host", baseURL: "https://", wantErr: ErrInvalidAPIURL},
		{name: "empty", baseURL: "", wantErr: ErrInvalidAPIURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewClient(tt.baseURL, "token")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewClient(%q) error = %v, want %v", tt.baseURL, err, tt.wantErr)
			}
		})
	}
}

func TestClientStatusErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		wantErr error
	}{
		{name: "not found means feature disabled", status: http.StatusNotFound, wantErr: ErrFeatureNotEnabled},
		{name: "server error", status: http.StatusInternalServerError, wantErr: ErrUnexpectedStatus},
		{name: "forbidden", status: http.StatusForbidden, wantErr: ErrUnexpectedStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "token secret" {
					t.Errorf("Authorization header = %q, want %q", got, "token secret")
				}
				if r.URL.Path != "/api/preview/agents/acme/definitions" {
					t.Errorf("request path = %q, want %q", r.URL.Path, "/api/preview/agents/acme/definitions")
				}
				w.WriteHeader(tt.status)
			}))
			t.Cleanup(srv.Close)

			c, err := NewClient(srv.URL, "secret")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			resp, err := c.ListAgents(t.Context(), "acme")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ListAgents() error = %v, want %v", err, tt.wantErr)
			}
			if resp != nil {
				t.Errorf("ListAgents() response = %v, want nil", resp)
			}
		})
	}
}

func TestListAgentAutomationsNotFoundOnLaterPage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("continuationToken") != "" {
			// An expired continuation token, not a disabled feature.
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"automations":[],"continuationToken":"page-2"}`)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = c.ListAgentAutomations(t.Context(), "acme")
	if !errors.Is(err, ErrUnexpectedStatus) {
		t.Fatalf("ListAgentAutomations() error = %v, want %v", err, ErrUnexpectedStatus)
	}
	if errors.Is(err, ErrFeatureNotEnabled) {
		t.Errorf("ListAgentAutomations() error = %v, must not report the feature as disabled", err)
	}
}
