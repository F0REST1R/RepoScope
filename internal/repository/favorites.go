package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/reposcope/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicate = errors.New("favorite already exists")
var ErrNotFound = errors.New("favorite not found")

type Favorites struct{ db *pgxpool.Pool }

func NewFavorites(db *pgxpool.Pool) *Favorites      { return &Favorites{db: db} }
func (r *Favorites) Ping(ctx context.Context) error { return r.db.Ping(ctx) }
func (r *Favorites) List(ctx context.Context) ([]model.Favorite, error) {
	rows, err := r.db.Query(ctx, `SELECT id, owner, repo, full_name, description, html_url, stars, language, created_at FROM favorites ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()
	items := make([]model.Favorite, 0)
	for rows.Next() {
		var f model.Favorite
		if err := rows.Scan(&f.ID, &f.Owner, &f.Repo, &f.FullName, &f.Description, &f.HTMLURL, &f.Stars, &f.Language, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan favorite: %w", err)
		}
		items = append(items, f)
	}
	return items, rows.Err()
}
func (r *Favorites) Create(ctx context.Context, f model.Favorite) (model.Favorite, error) {
	err := r.db.QueryRow(ctx, `INSERT INTO favorites(owner,repo,full_name,description,html_url,stars,language) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,created_at`, f.Owner, f.Repo, f.FullName, f.Description, f.HTMLURL, f.Stars, f.Language).Scan(&f.ID, &f.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return model.Favorite{}, ErrDuplicate
	}
	if err != nil {
		return model.Favorite{}, fmt.Errorf("create favorite: %w", err)
	}
	return f, nil
}
func (r *Favorites) Delete(ctx context.Context, owner, repo string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM favorites WHERE lower(owner)=lower($1) AND lower(repo)=lower($2)`, owner, repo)
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	statements := []string{`CREATE TABLE IF NOT EXISTS favorites (
id BIGSERIAL PRIMARY KEY, owner TEXT NOT NULL, repo TEXT NOT NULL, full_name TEXT NOT NULL,
description TEXT NOT NULL DEFAULT '', html_url TEXT NOT NULL, stars INTEGER NOT NULL DEFAULT 0 CHECK (stars >= 0),
language TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
CONSTRAINT favorites_owner_repo_unique UNIQUE (owner, repo))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS favorites_owner_repo_ci_idx ON favorites(lower(owner),lower(repo))`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
	}
	return nil
}

var _ = pgx.ErrNoRows
