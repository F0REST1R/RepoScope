package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/example/reposcope/internal/github"
	"github.com/example/reposcope/internal/model"
	"github.com/example/reposcope/internal/repository"
	"github.com/example/reposcope/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func New(svc *service.Service, log *slog.Logger, origins []string) http.Handler {
	h := &Handler{svc: svc, log: log}
	r := chi.NewRouter()
	r.Use(requestIDMiddleware, recoverer(log), accessLog(log), cors(origins), securityHeaders)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", h.health)
		r.Get("/ready", h.ready)
		r.Get("/rate-limit", h.rateLimit)
		r.Get("/repositories/search", h.search)
		r.Get("/repositories/{owner}/{repo}", h.repository)
		r.Route("/repositories/{owner}/{repo}", func(r chi.Router) {
			r.Get("/languages", h.languages)
			r.Get("/contributors", h.contributors)
			r.Get("/issues", h.issues)
			r.Get("/releases", h.releases)
		})
		r.Get("/favorites", h.favorites)
		r.Post("/favorites", h.createFavorite)
		r.Delete("/favorites/{owner}/{repo}", h.deleteFavorite)
	})
	return r
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "reposcope-api"})
}
func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Favorites.Ping(r.Context()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "not_ready", "База данных недоступна", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 || len(q) > 256 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "Параметр q должен содержать от 2 до 256 символов", map[string]string{"q": "invalid"})
		return
	}
	sort := defaultValue(r.URL.Query().Get("sort"), "stars")
	if !oneOf(sort, "stars", "forks", "help-wanted-issues", "updated") {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "Недопустимое значение sort", nil)
		return
	}
	order := defaultValue(r.URL.Query().Get("order"), "desc")
	if !oneOf(order, "asc", "desc") {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "Недопустимое значение order", nil)
		return
	}
	page, per, ok := pagination(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Search(r.Context(), q, strings.TrimSpace(r.URL.Query().Get("language")), sort, order, page, per)
	h.respond(w, r, out, err)
}
func (h *Handler) repository(w http.ResponseWriter, r *http.Request) {
	owner, repo, ok := names(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Repository(r.Context(), owner, repo)
	h.respond(w, r, out, err)
}
func (h *Handler) languages(w http.ResponseWriter, r *http.Request) {
	owner, repo, ok := names(w, r)
	if !ok {
		return
	}
	out, err := h.svc.GitHub.Languages(r.Context(), owner, repo)
	h.respond(w, r, map[string]any{"languages": out}, err)
}
func (h *Handler) contributors(w http.ResponseWriter, r *http.Request) {
	owner, repo, ok := names(w, r)
	if !ok {
		return
	}
	page, per, ok := pagination(w, r)
	if !ok {
		return
	}
	out, err := h.svc.GitHub.Contributors(r.Context(), owner, repo, page, per)
	h.respond(w, r, out, err)
}
func (h *Handler) issues(w http.ResponseWriter, r *http.Request) {
	owner, repo, ok := names(w, r)
	if !ok {
		return
	}
	page, per, ok := pagination(w, r)
	if !ok {
		return
	}
	out, err := h.svc.GitHub.Issues(r.Context(), owner, repo, page, per)
	h.respond(w, r, out, err)
}
func (h *Handler) releases(w http.ResponseWriter, r *http.Request) {
	owner, repo, ok := names(w, r)
	if !ok {
		return
	}
	page, per, ok := pagination(w, r)
	if !ok {
		return
	}
	out, err := h.svc.GitHub.Releases(r.Context(), owner, repo, page, per)
	h.respond(w, r, out, err)
}
func (h *Handler) rateLimit(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.GitHub.RateLimit(r.Context())
	h.respond(w, r, out, err)
}
func (h *Handler) favorites(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Favorites.List(r.Context())
	h.respond(w, r, map[string]any{"items": out, "count": len(out)}, err)
}

func (h *Handler) createFavorite(w http.ResponseWriter, r *http.Request) {
	var in model.CreateFavorite
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	in.Owner = service.Clean(in.Owner)
	in.Repo = service.Clean(in.Repo)
	details := map[string]string{}
	if err := service.ValidateName(in.Owner); err != nil {
		details["owner"] = err.Error()
	}
	if err := service.ValidateName(in.Repo); err != nil {
		details["repo"] = err.Error()
	}
	if len(details) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "Поля запроса не прошли валидацию", details)
		return
	}
	out, err := h.svc.CreateFavorite(r.Context(), in)
	if errors.Is(err, repository.ErrDuplicate) {
		writeError(w, r, http.StatusConflict, "favorite_exists", "Репозиторий уже добавлен в избранное", nil)
		return
	}
	if err != nil {
		h.respond(w, r, nil, err)
		return
	}
	w.Header().Set("Location", "/api/v1/favorites/"+url.PathEscape(out.Owner)+"/"+url.PathEscape(out.Repo))
	writeJSON(w, http.StatusCreated, out)
}
func (h *Handler) deleteFavorite(w http.ResponseWriter, r *http.Request) {
	owner, repo, ok := names(w, r)
	if !ok {
		return
	}
	err := h.svc.Favorites.Delete(r.Context(), owner, repo)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "favorite_not_found", "Запись избранного не найдена", nil)
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "storage_error", "Не удалось удалить запись", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, data)
		return
	}
	switch {
	case errors.Is(err, github.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "github_not_found", "Репозиторий или ресурс не найден", nil)
	case errors.Is(err, r.Context().Err()):
		return
	default:
		var apiErr *github.APIError
		if errors.As(err, &apiErr) {
			if apiErr.Status == http.StatusForbidden || apiErr.Status == http.StatusTooManyRequests {
				details := map[string]string{}
				if apiErr.RetryAfter != "" {
					details["retry_after"] = apiErr.RetryAfter
				}
				writeError(w, r, http.StatusTooManyRequests, "github_rate_limited", "Лимит запросов GitHub API исчерпан", details)
				return
			}
			writeError(w, r, http.StatusBadGateway, "github_error", "GitHub API вернул ошибку", map[string]any{"upstream_status": apiErr.Status})
			return
		}
		h.log.ErrorContext(r.Context(), "request failed", "request_id", requestID(r.Context()), "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
	}
}

func names(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	owner, repo := chi.URLParam(r, "owner"), chi.URLParam(r, "repo")
	if service.ValidateName(owner) != nil || service.ValidateName(repo) != nil {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "Некорректные owner или repo", nil)
		return "", "", false
	}
	return owner, repo, true
}
func pagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	page, err := positive(r.URL.Query().Get("page"), 1, 100)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "page должен быть числом от 1 до 100", nil)
		return 0, 0, false
	}
	per, err := positive(r.URL.Query().Get("per_page"), 20, 100)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "per_page должен быть числом от 1 до 100", nil)
		return 0, 0, false
	}
	return page, per, true
}
func positive(raw string, fallback, maxValue int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > maxValue {
		return 0, fmt.Errorf("out of range")
	}
	return n, nil
}
func defaultValue(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
func oneOf(v string, values ...string) bool {
	for _, x := range values {
		if v == x {
			return true
		}
	}
	return false
}
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("некорректное JSON-тело: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("JSON-тело должно содержать один объект")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message, "request_id": requestID(r.Context()), "details": details}})
}
