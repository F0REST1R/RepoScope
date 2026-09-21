package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/example/reposcope/internal/model"
)

var ErrNotFound = errors.New("github resource not found")

type APIError struct {
	Status     int
	Message    string
	RetryAfter string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github API: status %d: %s", e.Status, e.Message)
}

type Client struct {
	baseURL, token, version string
	http                    *http.Client
	cache                   *responseCache
}

func NewClient(baseURL, token, version string, timeout, cacheTTL time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, version: version, http: &http.Client{Timeout: timeout}, cache: newResponseCache(cacheTTL)}
}

func (c *Client) Search(ctx context.Context, query, language, sort, order string, page, perPage int) (model.SearchResult, error) {
	q := strings.TrimSpace(query)
	if language != "" {
		q += " language:" + language
	}
	v := url.Values{"q": {q}, "sort": {sort}, "order": {order}, "page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
	var raw struct {
		Items []model.Repository `json:"items"`
		Total int                `json:"total_count"`
	}
	if err := c.get(ctx, "/search/repositories?"+v.Encode(), &raw); err != nil {
		return model.SearchResult{}, err
	}
	pages := 0
	if raw.Total > 0 {
		pages = (raw.Total + perPage - 1) / perPage
		if pages > 100 {
			pages = 100
		}
	}
	return model.SearchResult{Items: raw.Items, TotalCount: raw.Total, Page: page, PerPage: perPage, TotalPages: pages}, nil
}

func (c *Client) Repository(ctx context.Context, owner, repo string) (model.Repository, error) {
	var out model.Repository
	err := c.get(ctx, repoPath(owner, repo), &out)
	return out, err
}
func (c *Client) Languages(ctx context.Context, owner, repo string) (map[string]int64, error) {
	out := map[string]int64{}
	err := c.get(ctx, repoPath(owner, repo)+"/languages", &out)
	return out, err
}
func (c *Client) Contributors(ctx context.Context, owner, repo string, page, perPage int) (model.Page[model.Contributor], error) {
	var items []model.Contributor
	err := c.get(ctx, paged(repoPath(owner, repo)+"/contributors", page, perPage), &items)
	return model.Page[model.Contributor]{Items: items, Page: page, PerPage: perPage, HasNext: len(items) == perPage}, err
}
func (c *Client) Releases(ctx context.Context, owner, repo string, page, perPage int) (model.Page[model.Release], error) {
	var items []model.Release
	err := c.get(ctx, paged(repoPath(owner, repo)+"/releases", page, perPage), &items)
	return model.Page[model.Release]{Items: items, Page: page, PerPage: perPage, HasNext: len(items) == perPage}, err
}
func (c *Client) Issues(ctx context.Context, owner, repo string, page, perPage int) (model.Page[model.Issue], error) {
	v := url.Values{"state": {"all"}, "sort": {"updated"}, "direction": {"desc"}, "page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
	var raw []model.Issue
	if err := c.get(ctx, repoPath(owner, repo)+"/issues?"+v.Encode(), &raw); err != nil {
		return model.Page[model.Issue]{}, err
	}
	items := make([]model.Issue, 0, len(raw))
	for _, item := range raw {
		if item.PullRequest == nil {
			items = append(items, item)
		}
	}
	return model.Page[model.Issue]{Items: items, Page: page, PerPage: perPage, HasNext: len(raw) == perPage}, nil
}
func (c *Client) RateLimit(ctx context.Context) (model.RateLimit, error) {
	var raw struct {
		Resources model.RateLimit `json:"resources"`
	}
	err := c.getUncached(ctx, "/rate_limit", &raw)
	return raw.Resources, err
}

func repoPath(owner, repo string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}
func paged(path string, page, perPage int) string {
	v := url.Values{"page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
	return path + "?" + v.Encode()
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	if body, ok := c.cache.get(path); ok {
		return json.Unmarshal(body, target)
	}
	body, err := c.fetch(ctx, path)
	if err != nil {
		return err
	}
	c.cache.set(path, body)
	return json.Unmarshal(body, target)
}
func (c *Client) getUncached(ctx context.Context, path string, target any) error {
	body, err := c.fetch(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
func (c *Client) fetch(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "RepoScope/1.0")
	req.Header.Set("X-GitHub-Api-Version", c.version)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read GitHub response: %w", err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}
	var detail struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &detail)
	if detail.Message == "" {
		detail.Message = http.StatusText(resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	return nil, &APIError{Status: resp.StatusCode, Message: detail.Message, RetryAfter: resp.Header.Get("Retry-After")}
}
