package github

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegrationPublicGitHub is opt-in because it consumes the real GitHub
// rate limit. CI and ordinary unit tests remain deterministic and offline.
func TestIntegrationPublicGitHub(t *testing.T) {
	if os.Getenv("RUN_GITHUB_INTEGRATION") != "1" {
		t.Skip("set RUN_GITHUB_INTEGRATION=1 to call the public GitHub API")
	}
	c := NewClient("https://api.github.com", os.Getenv("GITHUB_TOKEN"), "2026-03-10", 15*time.Second, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	repo, err := c.Repository(ctx, "golang", "go")
	if err != nil {
		t.Fatal(err)
	}
	if repo.FullName != "golang/go" || repo.Owner.Login != "golang" {
		t.Fatalf("unexpected repository: %#v", repo)
	}

	result, err := c.Search(ctx, "http server", "Go", "stars", "desc", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount < 1 || len(result.Items) < 1 {
		t.Fatalf("unexpected empty search: %#v", result)
	}
}
