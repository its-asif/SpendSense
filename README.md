# SpendSense

Personal expense tracker for a single user — no teams, sharing, or multi-tenant features.

This document describes what is **implemented and runnable today**, not a future roadmap.

## What is built

### Backend (`backend/`)

- HTTP API on **Go 1.26** using the standard library (`net/http`, `ServeMux`) — handlers live in `internal/httpapi/`.
- **PostgreSQL 16** persistence (pgx), with SQL migrations in `backend/migrations/` (seven versions: core schema through 2FA).
- **Redis** (optional): refresh-token storage and currency caching; the API starts without Redis but auth refresh behavior is degraded.
- **Auth:** register, login, refresh, logout (single session / all sessions / other sessions), profile and preferences, password change, session list and revoke, TOTP 2FA setup/confirm/disable.
- **Resources (REST, `/api/v1/…`):** expenses, incomes, categories, wallets (including transfers), dashboard summary and widgets, currency list and conversion.
- **Ops:** health check, OpenAPI spec, Swagger UI at `/api/docs`, refresh-token cleanup endpoint.
- **Middleware:** CORS, rate limiting, request logging, panic recovery, JWT auth on protected routes.
- **Tests:** unit tests for auth, currency, expense, and wallet packages; integration tests in `backend/tests/` (require `DATABASE_URL`).

### Frontend (`frontend/`)

- **React 18**, **TypeScript**, **Vite**, **Tailwind CSS**, **axios**, **Recharts**.
- Authenticated app with routes: dashboard, expenses, incomes, wallets, categories, reports (client-side aggregates), and settings (account, general, profile, security, sessions).
- Token refresh via axios interceptors; theme toggle; dashboard KPIs, charts, and budget usage widgets (when budget rows exist in the database).

### CLI (`cli/`)

- **Cobra + Viper** binary `spendsense`: `auth`, `expense`, `category`, `wallet`, `income`, and `config` command groups.
- Config file: `~/.expenserc`; optional `SPENDSENSE_API_URL` / `--api-url`.
- GitHub Actions workflow publishes release binaries (see [cli/README.md](cli/README.md)).

## Tech stack

| Layer | Technology |
|-------|------------|
| API | Go 1.26, `net/http`, domain packages under `internal/` |
| Database | PostgreSQL 16 (Docker image in `backend/docker-compose.yml`) |
| Cache / sessions | Redis 7 (optional) |
| Auth | JWT access tokens (~15 min), refresh tokens (~7 days, stored hashed in Postgres) |
| API contract | OpenAPI YAML + `oapi-codegen` types (`make openapi`) |
| Web UI | React 18, TypeScript, Vite 5, Tailwind 3, Recharts |
| CLI | Go, Cobra, Viper |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) SQL files |

## Quick start

### Prerequisites

- Go 1.26+
- Node.js 18+
- Docker and Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) CLI (for applying SQL migrations)

### 1. Start Postgres and Redis

```bash
cd backend
docker compose up -d
```

Postgres: `localhost:5432` (user/password/db: `spendsense`). Adminer: `http://localhost:8081`.

### 2. Configure and migrate the database

Create `backend/.env`:

```env
PORT=8080
DATABASE_URL=postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=change-me-in-production
CORS_ALLOWED_ORIGINS=http://localhost:5173
```

Apply migrations:

```bash
migrate -path backend/migrations -database "$DATABASE_URL" up
```

### 3. Run the API

```bash
cd backend
go run ./cmd/api
```

- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/api/docs`
- OpenAPI spec: `http://localhost:8080/openapi.yaml`

### 4. Run the web app

```bash
cd frontend
npm install
npm run dev
```

Optional: `frontend/.env` with `VITE_API_URL=http://localhost:8080` (defaults to that URL).

### 5. Build the CLI (optional)

From the repository root:

```bash
make cli-build
./bin/spendsense --help
```

## Makefile targets

The root [Makefile](Makefile) currently provides:

| Target | Purpose |
|--------|---------|
| `make test` | Run `go test ./...` in `backend/` |
| `make openapi` | Regenerate Go types from `backend/internal/httpapi/openapi.yaml` |
| `make cli-build` | Build `bin/spendsense` from `cli/` |
| `make cli-run` | Build CLI and print help |

## API documentation

- Interactive docs: `http://localhost:8080/api/docs` (with the API running).
- Endpoint reference with example bodies: [backend/README.md](backend/README.md).
- Source of truth for request/response shapes: [backend/internal/httpapi/openapi.yaml](backend/internal/httpapi/openapi.yaml).

## CLI usage

```bash
# Authentication
./bin/spendsense auth register
./bin/spendsense auth login
./bin/spendsense auth me
./bin/spendsense auth refresh
./bin/spendsense auth logout
./bin/spendsense auth logout-all

# Expenses
./bin/spendsense expense add --amount 50 --category Food --date today --merchant "Cafe"
./bin/spendsense expense list
./bin/spendsense expense list --from 2024-01-01 --to 2024-01-31

# Other resources
./bin/spendsense category list
./bin/spendsense wallet list
./bin/spendsense income list

# Config (~/.expenserc)
./bin/spendsense config view
```

Override the API base URL with `SPENDSENSE_API_URL`, `cli/.env`, or `--api-url`.

## Project structure

```
.
├── backend/
│   ├── cmd/api/              # API entrypoint
│   ├── internal/
│   │   ├── httpapi/          # Routes, handlers, OpenAPI
│   │   ├── auth/             # JWT, passwords, sessions, 2FA
│   │   ├── expense/ income/ wallet/ category/ report/ currency/
│   │   ├── middleware/ infra/ domain/
│   │   └── …                 # budget, tag, receipt, etc. (no HTTP routes yet)
│   ├── migrations/           # Sequential SQL up/down files
│   ├── tests/                # Integration tests
│   └── docker-compose.yml    # Postgres, Redis, Adminer
├── frontend/
│   └── src/
│       ├── api/              # axios API clients
│       ├── pages/            # Route-level views
│       ├── components/       # UI building blocks
│       └── hooks/ lib/ types/
├── cli/
│   └── cmd/                  # Cobra commands
├── Makefile
└── README.md
```

## Development

### Backend layout

Each domain typically has `entity.go`, `repository.go`, and `service.go`. HTTP adapters are in `internal/httpapi/*_handlers.go`.

### Tests

```bash
make test
```

Integration tests need a reachable database:

```bash
export DATABASE_URL='postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable'
cd backend && go test ./tests/...
```

### New migrations

```bash
migrate create -ext sql -dir backend/migrations -seq <name>
```

### Regenerate OpenAPI types

```bash
make openapi
```

## Environment variables (API)

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | Postgres connection string |
| `JWT_SECRET` | Yes | Signing key for access tokens |
| `PORT` | No | Listen port (default `8080`) |
| `REDIS_URL` | No | Redis for refresh tokens / currency cache |
| `CORS_ALLOWED_ORIGINS` | No | Comma-separated origins (default `http://localhost:5173`) |

## Database notes

- Primary keys are **UUIDs** (see `backend/migrations/001_init_schema.up.sql`).
- Migration `002` seeds **10** global default categories (Food, Transport, Housing, etc.).
- Migrations `003`–`007` add tables and columns for budgets, tags, receipts, notifications, personal loans, session metadata, and 2FA. Some of these tables are **not yet exposed** by HTTP handlers; dashboard budget widgets read from the `budgets` table when rows exist, but there is no budget CRUD API in the current codebase.

For table-level detail, read the migration files under `backend/migrations/`.
