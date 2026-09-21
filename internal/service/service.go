package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/example/reposcope/internal/model"
)

type GitHub interface {
	Search(context.Context, string, string, string, string, int, int) (model.SearchResult, error)
	Repository(context.Context, string, string) (model.Repository, error)
	Languages(context.Context, string, string) (map[string]int64, error)
	Contributors(context.Context, string, string, int, int) (model.Page[model.Contributor], error)
	Issues(context.Context, string, string, int, int) (model.Page[model.Issue], error)
	Releases(context.Context, string, string, int, int) (model.Page[model.Release], error)
	RateLimit(context.Context) (model.RateLimit, error)
}
type FavoriteStore interface {
	Ping(context.Context) error
	List(context.Context) ([]model.Favorite, error)
	Create(context.Context, model.Favorite) (model.Favorite, error)
	Delete(context.Context, string, string) error
}
type Service struct {
	GitHub    GitHub
	Favorites FavoriteStore
}

func (s *Service) Search(ctx context.Context, q, lang, sort, order string, page, per int) (model.SearchResult, error) {
	return s.GitHub.Search(ctx, q, lang, sort, order, page, per)
}
func (s *Service) Repository(ctx context.Context, owner, repo string) (model.Repository, error) {
	r, err := s.GitHub.Repository(ctx, owner, repo)
	if err != nil {
		return r, err
	}
	r.HealthScore, r.HealthSummary = health(r, time.Now())
	return r, nil
}
func (s *Service) CreateFavorite(ctx context.Context, in model.CreateFavorite) (model.Favorite, error) {
	r, err := s.GitHub.Repository(ctx, in.Owner, in.Repo)
	if err != nil {
		return model.Favorite{}, err
	}
	return s.Favorites.Create(ctx, model.Favorite{Owner: r.Owner.Login, Repo: r.Name, FullName: r.FullName, Description: r.Description, HTMLURL: r.HTMLURL, Stars: r.Stars, Language: r.Language})
}

func health(r model.Repository, now time.Time) (int, string) {
	score := 0.0
	score += math.Min(25, math.Log10(float64(r.Stars)+1)*7)
	score += math.Min(15, math.Log10(float64(r.Forks)+1)*5)
	if r.License != nil && r.License.SPDXID != "NOASSERTION" {
		score += 15
	}
	if !r.Archived {
		score += 10
	}
	if r.Description != "" {
		score += 5
	}
	if len(r.Topics) > 0 {
		score += 5
	}
	if r.PushedAt != nil {
		days := now.Sub(*r.PushedAt).Hours() / 24
		switch {
		case days <= 30:
			score += 25
		case days <= 180:
			score += 18
		case days <= 365:
			score += 10
		}
	}
	n := int(math.Round(math.Min(100, score)))
	summary := "требует внимания"
	if n >= 75 {
		summary = "активный"
	} else if n >= 50 {
		summary = "стабильный"
	}
	return n, summary
}

func ValidateName(v string) error {
	if len(v) < 1 || len(v) > 100 {
		return fmt.Errorf("длина должна быть от 1 до 100 символов")
	}
	for _, r := range v {
		switch {
		case r == '-', r == '_', r == '.', r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			continue
		default:
			return fmt.Errorf("недопустимый символ")
		}
	}
	return nil
}
func Clean(v string) string { return strings.TrimSpace(v) }
