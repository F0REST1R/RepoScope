CREATE TABLE IF NOT EXISTS favorites (
    id BIGSERIAL PRIMARY KEY,
    owner TEXT NOT NULL,
    repo TEXT NOT NULL,
    full_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    html_url TEXT NOT NULL,
    stars INTEGER NOT NULL DEFAULT 0 CHECK (stars >= 0),
    language TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT favorites_owner_repo_unique UNIQUE (owner, repo)
);

CREATE UNIQUE INDEX IF NOT EXISTS favorites_owner_repo_ci_idx
    ON favorites (lower(owner), lower(repo));

