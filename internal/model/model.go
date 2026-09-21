package model

import "time"

type Owner struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type License struct {
	Name   string `json:"name"`
	SPDXID string `json:"spdx_id"`
}

type Repository struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	FullName      string     `json:"full_name"`
	Description   string     `json:"description"`
	HTMLURL       string     `json:"html_url"`
	Homepage      string     `json:"homepage,omitempty"`
	Language      string     `json:"language,omitempty"`
	Stars         int        `json:"stargazers_count"`
	Forks         int        `json:"forks_count"`
	OpenIssues    int        `json:"open_issues_count"`
	DefaultBranch string     `json:"default_branch"`
	Archived      bool       `json:"archived"`
	Topics        []string   `json:"topics"`
	Owner         Owner      `json:"owner"`
	License       *License   `json:"license,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	PushedAt      *time.Time `json:"pushed_at,omitempty"`
	HealthScore   int        `json:"health_score,omitempty"`
	HealthSummary string     `json:"health_summary,omitempty"`
}

type SearchResult struct {
	Items      []Repository `json:"items"`
	TotalCount int          `json:"total_count"`
	Page       int          `json:"page"`
	PerPage    int          `json:"per_page"`
	TotalPages int          `json:"total_pages"`
}

type Contributor struct {
	Login         string `json:"login"`
	AvatarURL     string `json:"avatar_url"`
	HTMLURL       string `json:"html_url"`
	Contributions int    `json:"contributions"`
}
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}
type Issue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	HTMLURL     string    `json:"html_url"`
	User        Owner     `json:"user"`
	Labels      []Label   `json:"labels"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PullRequest any       `json:"pull_request,omitempty"`
}
type Release struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	TagName     string     `json:"tag_name"`
	HTMLURL     string     `json:"html_url"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	PublishedAt *time.Time `json:"published_at"`
}
type Page[T any] struct {
	Items   []T  `json:"items"`
	Page    int  `json:"page"`
	PerPage int  `json:"per_page"`
	HasNext bool `json:"has_next"`
}

type RateBucket struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Used      int   `json:"used"`
	Reset     int64 `json:"reset"`
}
type RateLimit struct {
	Core   RateBucket `json:"core"`
	Search RateBucket `json:"search"`
}

type Favorite struct {
	ID          int64     `json:"id"`
	Owner       string    `json:"owner"`
	Repo        string    `json:"repo"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	HTMLURL     string    `json:"html_url"`
	Stars       int       `json:"stars"`
	Language    string    `json:"language,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
type CreateFavorite struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}
