# RepoScope

RepoScope is a full-stack portfolio project for discovering and quickly assessing public GitHub repositories. It combines activity, popularity, license, languages, contributors, releases and non-PR issues into one responsive UI, computes an explainable 0–100 health score, and stores favorites in PostgreSQL.

## Highlights

- Go 1.23 REST API built on `net/http` + chi;
- typed GitHub REST client with context cancellation, timeouts, optional token and TTL cache;
- PostgreSQL favorites with parameterized SQL and duplicate protection;
- React + TypeScript + Vite UI with loading, empty and error states;
- unified errors, request IDs, structured logging, recovery, CORS and graceful shutdown;
- OpenAPI, tested Postman collection, Bash/PowerShell cURL demos;
- unit/integration tests, multi-stage Docker image, Compose and GitHub Actions.

## Quick start

```bash
cp .env.example .env
docker compose up --build -d
```

Open <http://localhost:3000>. API health is at <http://localhost:8080/api/v1/health>. A GitHub token is optional; if configured as `GITHUB_TOKEN`, it is used only by the backend and must never be committed.

```bash
go test ./...
cd web && npm ci && npm run build
```

See the full [Russian README](README.md), [architecture](docs/architecture.md), [OpenAPI contract](docs/openapi.yaml), and the three practical-work reports under `docs/`.

## License

MIT.
