package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSearchBuildsRequestAndDecodes(t *testing.T) {
	t.Helper()
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.Header.Get("X-GitHub-Api-Version"); got != "2026-03-10" {
			t.Errorf("version header = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.URL.Query().Get("q"); got != "http server language:Go" {
			t.Errorf("query = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total_count":1,"items":[{"id":1,"name":"chi","full_name":"go-chi/chi","owner":{"login":"go-chi"},"stargazers_count":10}]}`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "secret", "2026-03-10", time.Second, time.Minute)
	got, err := c.Search(context.Background(), "http server", "Go", "stars", "desc", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalCount != 1 || len(got.Items) != 1 || got.Items[0].FullName != "go-chi/chi" {
		t.Fatalf("unexpected result: %#v", got)
	}
	_, _ = c.Search(context.Background(), "http server", "Go", "stars", "desc", 1, 20)
	if calls != 1 {
		t.Fatalf("cache did not prevent second request: calls=%d", calls)
	}
}

func TestIssuesExcludePullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"number":1,"title":"issue"},{"number":2,"title":"pr","pull_request":{"url":"x"}}]`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "", "2026-03-10", time.Second, 0)
	got, err := c.Issues(context.Background(), "o", "r", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Number != 1 {
		t.Fatalf("unexpected issues: %#v", got.Items)
	}
}

func TestRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"rate limit exceeded"}`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "", "2026-03-10", time.Second, 0)
	_, err := c.Repository(context.Background(), "o", "r")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 403 || apiErr.RetryAfter != "60" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
