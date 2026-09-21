package service

import (
	"testing"
	"time"

	"github.com/example/reposcope/internal/model"
)

func TestHealthScoreRewardsMaintainedLicensedRepository(t *testing.T) {
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	pushed := now.Add(-7 * 24 * time.Hour)
	repo := model.Repository{Stars: 12000, Forks: 900, Description: "useful", Topics: []string{"go"}, License: &model.License{SPDXID: "MIT"}, PushedAt: &pushed}
	score, summary := health(repo, now)
	if score < 75 || summary != "активный" {
		t.Fatalf("score=%d summary=%s", score, summary)
	}
}

func TestValidateName(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{{"go-chi", true}, {"repo.js", true}, {"bad/name", false}, {"", false}} {
		if got := ValidateName(tc.value) == nil; got != tc.valid {
			t.Errorf("ValidateName(%q) valid=%v", tc.value, got)
		}
	}
}
