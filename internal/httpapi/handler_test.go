package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/reposcope/internal/model"
	"github.com/example/reposcope/internal/repository"
	"github.com/example/reposcope/internal/service"
)

type fakeGitHub struct{}

func (fakeGitHub) Search(_ context.Context, _, _, _, _ string, page, per int) (model.SearchResult, error) {
	return model.SearchResult{Items: []model.Repository{}, TotalCount: 0, Page: page, PerPage: per}, nil
}
func (fakeGitHub) Repository(_ context.Context, o, r string) (model.Repository, error) {
	if r == "missing" {
		return model.Repository{}, errors.New("upstream")
	}
	return model.Repository{Name: r, FullName: o + "/" + r, Owner: model.Owner{Login: o}, HTMLURL: "https://github.com/" + o + "/" + r, Stars: 42}, nil
}
func (fakeGitHub) Languages(context.Context, string, string) (map[string]int64, error) {
	return map[string]int64{"Go": 100}, nil
}
func (fakeGitHub) Contributors(context.Context, string, string, int, int) (model.Page[model.Contributor], error) {
	return model.Page[model.Contributor]{Items: []model.Contributor{}}, nil
}
func (fakeGitHub) Issues(context.Context, string, string, int, int) (model.Page[model.Issue], error) {
	return model.Page[model.Issue]{Items: []model.Issue{}}, nil
}
func (fakeGitHub) Releases(context.Context, string, string, int, int) (model.Page[model.Release], error) {
	return model.Page[model.Release]{Items: []model.Release{}}, nil
}
func (fakeGitHub) RateLimit(context.Context) (model.RateLimit, error) { return model.RateLimit{}, nil }

type fakeStore struct{ items []model.Favorite }

func (s *fakeStore) Ping(context.Context) error                     { return nil }
func (s *fakeStore) List(context.Context) ([]model.Favorite, error) { return s.items, nil }
func (s *fakeStore) Create(_ context.Context, f model.Favorite) (model.Favorite, error) {
	for _, x := range s.items {
		if x.FullName == f.FullName {
			return model.Favorite{}, repository.ErrDuplicate
		}
	}
	f.ID = int64(len(s.items) + 1)
	f.CreatedAt = time.Now()
	s.items = append(s.items, f)
	return f, nil
}
func (s *fakeStore) Delete(_ context.Context, o, r string) error {
	for i, x := range s.items {
		if x.Owner == o && x.Repo == r {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func testHandler() http.Handler {
	store := &fakeStore{}
	return New(&service.Service{GitHub: fakeGitHub{}, Favorites: store}, slog.New(slog.NewTextHandler(io.Discard, nil)), []string{"http://localhost:5173"})
}
func TestSearchValidation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories/search?q=x", nil)
	res := httptest.NewRecorder()
	testHandler().ServeHTTP(res, req)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if res.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content-type=%q", res.Header().Get("Content-Type"))
	}
}
func TestFavoriteLifecycle(t *testing.T) {
	h := testHandler()
	body := []byte(`{"owner":"go-chi","repo":"chi"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/favorites", bytes.NewReader(body))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", res.Code, res.Body.String())
	}
	if res.Header().Get("Location") != "/api/v1/favorites/go-chi/chi" {
		t.Fatalf("location=%q", res.Header().Get("Location"))
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/favorites", bytes.NewReader(body))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d", res.Code)
	}
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/favorites/go-chi/chi", nil)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d", res.Code)
	}
}
