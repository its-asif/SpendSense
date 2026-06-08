# SpendSense

A personal finance tracker for a single user — track expenses, incomes, budgets, wallets, and recurring payments with a full REST API, a React web app, and an optional CLI.

> **This document reflects what is fully implemented and runnable today.** Roadmap items are clearly marked.

---

## Table of Contents

- [Architecture](#architecture)
- [Feature Overview](#feature-overview)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [API Overview](#api-overview)
- [Design Decisions](#design-decisions)
- [Testing](#testing)
- [Database](#database)
- [Environment Variables](#environment-variables)
- [Makefile Targets](#makefile-targets)
- [Roadmap](#roadmap)

---

## Architecture

```mermaid
graph TD
    Browser["Web Browser\n(React + Vite)"]
    CLI["CLI\n(Cobra / Viper)"]
    API["Go HTTP API\n(net/http · ServeMux)"]
    Auth["auth\nJWT · TOTP · Sessions"]
    Domain["Domain Services\nexpense · income · wallet\nbudget · category · report\nrecurring · notification"]
    DB["PostgreSQL 16\n(pgx)"]
    Redis["Redis 7\n(optional)\nRefresh tokens · Currency cache"]
    ExFX["External FX API\nopen.er-api.com"]

    Browser -->|REST / JSON| API
    CLI -->|REST / JSON| API
    API --> Auth
    API --> Domain
    Auth --> DB
    Auth --> Redis
    Domain --> DB
    Domain --> ExFX
```

### Request lifecycle

```
Client
  └─► ServeMux (Go net/http)
        └─► Middleware chain
              │  CORS · Rate-limit · Request-log · Panic-recovery · JWT-auth
              └─► Handler (internal/httpapi/)
                    └─► Service layer (internal/<domain>/service.go)
                          └─► Repository layer (internal/<domain>/repository.go)
                                └─► PostgreSQL (pgx pool)
```

---

## Feature Overview

### Backend

| Area | Details |
|------|---------|
| **Auth** | Register, login (optional TOTP), refresh, logout (single / all / others), sessions list & revoke, password change, profile & preferences, 2FA setup/confirm/disable |
| **Expenses** | Full CRUD, soft-delete, optional receipt upload, recurring expense templates & one-click posting |
| **Incomes** | Full CRUD, soft-delete |
| **Wallets** | Full CRUD, wallet-to-wallet transfers with optional FX conversion |
| **Categories** | Full CRUD, default (read-only) and user-defined, EXPENSE / INCOME kinds |
| **Budgets** | Full CRUD, monthly/weekly/yearly periods, rollover setting, one budget per category per period |
| **Reports / Dashboard** | Spending summary, KPIs, chart data, budget-usage widgets |
| **Recurring Payments** | Create / update / delete / pay (auto-advances next cycle, books expense) |
| **Notifications** | List, mark-read, dismiss |
| **Receipt uploads** | `multipart/form-data`, stored as file URL linked to expense |
| **Currencies** | List available codes, convert between currencies (cached in Redis) |
| **Ops** | Health check, expired refresh-token cleanup |

### Frontend (React / TypeScript)

- Dashboard with KPI cards, spending charts, budget widgets, recent transaction list
- Full pages for Expenses, Incomes, Wallets, Categories, Budgets, Recurring Payments, Reports
- Settings: profile, account, general preferences, security, active sessions
- JWT access token / refresh token auth with axios interceptors
- Dark / light theme toggle
- Real-time notification dropdown

### CLI (Go / Cobra)

- `auth` — register, login, me, refresh, logout, logout-all
- `expense` — add, list (with `--from` / `--to` filters)
- `category`, `wallet`, `income` — list
- `config` — view / set API URL
- Config file: `~/.expenserc` · env: `SPENDSENSE_API_URL` · flag: `--api-url`

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| API server | Go 1.26, `net/http`, `ServeMux` |
| Database | PostgreSQL 16 (pgx v5 driver) |
| Cache / sessions | Redis 7 (go-redis v9) — optional |
| Auth tokens | JWT (golang-jwt v5) — access ~15 min, refresh ~7 days |
| 2FA | TOTP (pquerna/otp), QR via skip2/go-qrcode |
| API contract | OpenAPI 3 YAML + oapi-codegen generated types |
| Frontend | React 18, TypeScript, Vite 5, Tailwind CSS 3, Recharts |
| CLI | Go, Cobra, Viper |
| Migrations | golang-migrate (sequential SQL files) |
| Tests | Go standard `testing` package, interface-based mocks |

---

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 18+
- Docker & Docker Compose
- [`golang-migrate`](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) CLI

### 1 · Start infrastructure

```bash
cd backend
docker compose up -d
# Postgres  → localhost:5432  (user/pass/db: spendsense)
# Redis     → localhost:6379
# Adminer   → http://localhost:8081
```

### 2 · Configure the backend

Create `backend/.env`:

```env
PORT=8080
DATABASE_URL=postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=change-me-in-production
CORS_ALLOWED_ORIGINS=http://localhost:5173
```

### 3 · Apply database migrations

```bash
migrate -path backend/migrations \
        -database "postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable" \
        up
```

### 4 · Run the API

```bash
cd backend
go run ./cmd/api
# API:       http://localhost:8080
# Swagger:   http://localhost:8080/api/docs
# OpenAPI:   http://localhost:8080/openapi.yaml
```

### 5 · Run the web app

```bash
cd frontend
npm install
npm run dev
# App: http://localhost:5173
```

Optional: create `frontend/.env` with `VITE_API_URL=http://localhost:8080`.

### 6 · Build & use the CLI (optional)

```bash
make cli-build
./bin/spendsense auth login
./bin/spendsense expense list
```

---

## Project Structure

```
.
├── backend/
│   ├── cmd/api/                 # main.go — wires everything, starts server
│   ├── internal/
│   │   ├── httpapi/             # HTTP handlers, route registration, OpenAPI types
│   │   ├── auth/                # JWT, bcrypt, sessions, TOTP
│   │   ├── expense/             # expense entity, service, repository
│   │   ├── income/              # income entity, service, repository
│   │   ├── wallet/              # wallet + transfer entity, service, repository
│   │   ├── budget/              # budget entity, service, repository
│   │   ├── category/            # category entity, service, repository
│   │   ├── report/              # dashboard aggregation, chart data
│   │   ├── currency/            # FX rate fetching, normalization, caching
│   │   ├── notification/        # notification entity, service, repository
│   │   ├── recurring/           # recurring-payment entity, service, repository
│   │   ├── receipt/             # receipt upload storage
│   │   ├── middleware/          # CORS, rate-limit, logging, recovery, JWT
│   │   ├── domain/              # shared error types & codes
│   │   ├── infra/               # Database pool, Redis client wrappers
│   │   └── audit/               # audit log hooks (schema-only)
│   ├── migrations/              # 009 sequential SQL up/down files
│   ├── tests/                   # integration tests (require DATABASE_URL)
│   ├── assets/                  # uploaded receipt files
│   ├── openapi.yaml             # source-of-truth API contract
│   └── docker-compose.yml       # Postgres, Redis, Adminer
├── frontend/
│   └── src/
│       ├── api/                 # axios API clients per resource
│       ├── pages/               # Dashboard, Expenses, Incomes, Wallets, …
│       ├── components/          # Shared UI building blocks
│       ├── hooks/               # Custom React hooks
│       ├── stores/              # State management
│       └── types/               # TypeScript types
├── cli/
│   └── cmd/                     # Cobra command groups
├── Makefile
└── README.md  ← you are here
```

---

## API Overview

All protected routes require `Authorization: Bearer <access_token>`.  
Full interactive docs: `http://localhost:8080/api/docs` (with the server running).  
Detailed endpoint reference with request/response bodies: [backend/README.md](backend/README.md).

| Group | Base path |
|-------|-----------|
| Auth | `/auth/…` |
| Expenses | `/api/v1/expenses` |
| Incomes | `/api/v1/incomes` |
| Wallets | `/api/v1/wallets` |
| Categories | `/api/v1/categories` |
| Budgets | `/api/v1/budgets` |
| Reports | `/api/v1/dashboard/…` |
| Currencies | `/api/v1/currencies` |
| Recurring Payments | `/api/v1/recurring-payments` |
| Notifications | `/api/v1/notifications` |
| Ops | `/health`, `/ops/…` |

---

## Design Decisions

### Domain-driven package layout

Each domain (`expense`, `income`, `wallet`, …) owns three files:

| File | Responsibility |
|------|---------------|
| `entity.go` | Plain Go structs, request/response types, pagination helpers |
| `repository.go` | SQL queries via `pgx`; implements a `Store` interface |
| `service.go` | Business logic (validation, currency normalisation, balance accounting) |

HTTP adapters live in `internal/httpapi/` and call service methods only — they never touch the database directly.

### Interface-based dependency injection

Repositories accept a `DBConnection` interface rather than a concrete `*infra.Database`. This enables unit tests to swap in a lightweight `mockDB` without a real Postgres instance.

```go
type DBConnection interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    BeginTx(ctx context.Context) (pgx.Tx, error)
}
```

### Deterministic mock routing

Test mocks route query calls by **inspecting arguments** (e.g. matching a specific `uuid.UUID` in `args`) rather than a fragile sequential counter. This makes tests order-independent and resilient to internal refactoring.

### Wallet balance accounting

Every balance-mutating operation (create/update/delete expense or income, wallet transfer) runs inside a **serialisable transaction** that acquires row-level `FOR UPDATE` locks on involved wallets in a **consistent UUID-sorted order** to prevent deadlocks.

### Soft deletes

Expenses and incomes are soft-deleted (`is_deleted = TRUE`). This preserves historical accuracy in budgets and reports while removing records from all normal list queries.

### Currency handling

Incomes and expenses store an explicit `currency` field. When the transaction currency differs from the wallet currency, the service calls `currency.Service.Convert` (backed by an external FX API, cached in Redis) and stores the **converted amount** in the wallet's native currency.

### Pagination

List endpoints use **opaque cursor-based pagination** (base64-encoded JSON `{created_at, id}`) rather than page offsets to remain stable under concurrent inserts.

### Auth & sessions

- **Access tokens**: short-lived JWTs (~15 min), verified in middleware.
- **Refresh tokens**: stored as `SHA-256` hashes in Postgres with device/IP metadata; support per-session revocation.
- **2FA**: TOTP-based; setup flow returns a provisioning URI + QR image. Login requires the TOTP code when 2FA is enabled.

---

## Testing

### Unit tests

Core services use `fakeStore` / `mockDB` implementations of the repository interfaces, requiring no database:

```bash
cd backend
go test -cover ./...
```

**Current coverage targets (≥ 80%):**

| Package | Coverage |
|---------|---------|
| `internal/auth` | 86.4% |
| `internal/budget` | 82.1% |
| `internal/category` | 86.3% |
| `internal/currency` | 65.3% |
| `internal/domain` | 100.0% |
| `internal/expense` | 83.5% |
| `internal/income` | 78.4% |
| `internal/middleware` | 86.7% |
| `internal/notification` | 45.5% |
| `internal/report` | 80.6% |
| `internal/wallet` | 77.4% |

### What the mocks cover

- Happy-path CRUD
- Validation edge cases (zero amounts, future dates, invalid currencies, nil IDs)
- Database error propagation (transaction begin failure, `ErrNoRows`, scan errors)
- Business-rule violations (insufficient wallet balance, wrong category owner, duplicate budgets)

### Integration tests

Integration tests hit a real Postgres instance and test the full HTTP stack:

```bash
# 1. Start Postgres
cd backend && docker compose up -d

# 2. Apply migrations
migrate -path migrations \
        -database "postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable" \
        up

# 3. Run integration suite
export DATABASE_URL='postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable'
cd backend && go test ./tests/...
```

### Adding new tests

Follow the established pattern:

1. Define a `fake<Domain>Store` that implements the `Store` interface.
2. For repository tests, implement `mockDB` / `mockTx` with `queryRowFn` / `queryRowsFn` hooks.
3. Route mock responses by **argument inspection**, not sequence counters.

---

## Database

### Migration history

| # | Migration | Tables / changes |
|---|-----------|-----------------|
| 001 | Init schema | `users`, `sessions`, `refresh_tokens`, `categories`, `expenses`, `wallets` |
| 002 | Seed defaults | 10 global default categories (Food, Transport, Housing, …) |
| 003 | Budgets & reporting | `budgets`, `tags`, `receipts`, `notifications` |
| 004 | Income & loans | `incomes`, `personal_loans` (schema only) |
| 005 | Refresh token metadata | `user_agent`, `ip_address` columns |
| 006 | Session tracking | `last_used_at`, `device_name` |
| 007 | 2FA | `totp_secret`, `totp_enabled` on `users` |
| 008 | Category kind | `kind` column (`EXPENSE` / `INCOME`) + back-fill |
| 009 | Recurring payments | `recurring_payments` table |

### Key schema choices

- **UUID primary keys** throughout (`gen_random_uuid()`).
- **Soft-delete** via `is_deleted` + `deleted_at` on expenses and incomes.
- **Row-level locking** (`SELECT … FOR UPDATE`) on wallet balance updates.
- **Indexes** on `(user_id, is_deleted)` for all frequently filtered tables.

### Creating a new migration

```bash
migrate create -ext sql -dir backend/migrations -seq <descriptive_name>
# Writes: NNNN_<descriptive_name>.up.sql  and  NNNN_<descriptive_name>.down.sql
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | **Yes** | — | Postgres connection string |
| `JWT_SECRET` | **Yes** | — | HMAC-SHA256 signing key for access tokens |
| `PORT` | No | `8080` | HTTP listen port |
| `REDIS_URL` | No | — | Redis connection string; enables refresh-token storage and currency caching |
| `CORS_ALLOWED_ORIGINS` | No | `http://localhost:5173` | Comma-separated allowed origins |

---

## Makefile Targets

| Target | Command | Description |
|--------|---------|-------------|
| `make test` | `go test ./...` in `backend/` | Run all unit & integration tests |
| `make openapi` | `oapi-codegen` | Regenerate Go types from `backend/openapi.yaml` |
| `make cli-build` | `go build` | Build `bin/spendsense` from `cli/` |
| `make cli-run` | cli-build + `--help` | Build CLI and print help |

---

## Roadmap

Items below are **not yet implemented** and represent planned future work.

### Near-term

- [ ] **Tags** — Schema exists (`tags` table in migration 003); service and HTTP routes pending.
- [ ] **Personal Loans** — Schema exists (`personal_loans` in migration 004); business logic pending.
- [ ] **`testcontainers-go`** — Run integration tests against a fresh ephemeral Postgres container in CI without a separate Docker service.
- [ ] **Audit log** — `internal/audit` package exists; event-recording hooks not yet wired to handlers.

### Medium-term

- [ ] **Multi-currency reporting** — Aggregate expenses across wallets by converting to a user-defined base currency.
- [ ] **Budget alerts** — Push notification when a budget reaches a configurable threshold (e.g. 80%).
- [ ] **CSV / PDF export** — Export expense and income history.
- [ ] **Recurring payment reminders** — Notify before upcoming recurring payment deadlines.

### Longer-term

- [ ] **Mobile app** — React Native client reusing the same REST API.
- [ ] **OpenID Connect login** — Google / GitHub OAuth2 as an alternative to password auth.
- [ ] **Shared expenses** — Optional multi-user mode for splitting costs between accounts.
