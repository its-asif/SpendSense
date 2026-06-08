# SpendSense — Backend

Go HTTP API for the SpendSense personal finance tracker.

> Full project context and architecture overview: [../README.md](../README.md)

---

## Table of Contents

- [Getting Started](#getting-started)
- [Package Layout](#package-layout)
- [API Reference](#api-reference)
  - [Health & Docs](#health--docs)
  - [Auth](#auth)
  - [Expenses](#expenses)
  - [Incomes](#incomes)
  - [Wallets](#wallets)
  - [Categories](#categories)
  - [Budgets](#budgets)
  - [Dashboard](#dashboard)
  - [Currencies](#currencies)
  - [Recurring Payments](#recurring-payments)
  - [Notifications](#notifications)
  - [Operations](#operations)
- [Design Decisions](#design-decisions)
- [Testing](#testing)
- [Database Migrations](#database-migrations)
- [OpenAPI Code Generation](#openapi-code-generation)

---

## Getting Started

```bash
# 1. Start Postgres + Redis
docker compose up -d

# 2. Configure
cp .env.example .env   # then set DATABASE_URL, JWT_SECRET, etc.

# 3. Apply migrations
migrate -path migrations \
        -database "$DATABASE_URL" up

# 4. Run
go run ./cmd/api
```

| Endpoint | URL |
|----------|-----|
| API root | `http://localhost:8080` |
| Swagger UI | `http://localhost:8080/api/docs` |
| OpenAPI YAML | `http://localhost:8080/openapi.yaml` |

All protected routes require `Authorization: Bearer <access_token>`.

---

## Package Layout

```
internal/
├── httpapi/          Routes, handlers, OpenAPI-generated types
├── auth/             JWT, bcrypt, sessions, TOTP
├── expense/          Expense CRUD + recurring templates
├── income/           Income CRUD
├── wallet/           Wallet CRUD + transfers
├── budget/           Budget CRUD + period logic
├── category/         Category CRUD (default + user-defined, EXPENSE/INCOME kinds)
├── report/           Dashboard aggregation, chart data
├── currency/         FX rate fetch, normalisation, Redis cache
├── notification/     Notification list, read, dismiss
├── recurring/        Recurring payment lifecycle
├── receipt/          Receipt file upload & storage
├── middleware/       CORS, rate-limiting, logging, panic recovery, JWT auth
├── domain/           Shared error codes and DomainError type
├── infra/            pgx pool wrapper, Redis client
└── audit/            Audit log schema (hooks not yet wired)
```

Each domain package follows the same three-file pattern:

| File | Responsibility |
|------|---------------|
| `entity.go` | Plain structs, request/response types, helpers (pagination encoding, etc.) |
| `repository.go` | SQL via `pgx`; implements a `Store` / `DBConnection` interface |
| `service.go` | Business rules: validation, currency conversion, balance accounting |

---

## API Reference

### Health & Docs

| Method | Path | Auth |
|--------|------|------|
| GET | `/health` | No |
| GET | `/openapi.yaml` | No |
| GET | `/api/docs` | No |

---

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Create account |
| POST | `/auth/login` | Login (TOTP code when 2FA enabled) |
| POST | `/auth/refresh` | Exchange refresh token for new access token |
| POST | `/auth/logout` | Revoke one refresh token |
| POST | `/auth/logout-all` | Revoke all tokens for the user |
| POST | `/auth/logout-other` | Revoke all tokens except current session |
| GET | `/auth/me` | Current user profile |
| PATCH | `/auth/me/profile` | Update display name, avatar, etc. |
| PATCH | `/auth/me/preferences` | Update base currency, timezone, locale |
| POST | `/auth/me/password` | Change password |
| GET | `/auth/me/sessions` | List active sessions |
| DELETE | `/auth/me/sessions/{sessionId}` | Revoke one session |
| POST | `/auth/me/2fa/setup` | Start TOTP enrollment (returns provisioning URI + QR) |
| POST | `/auth/me/2fa/confirm` | Confirm TOTP code to activate 2FA |
| POST | `/auth/me/2fa/disable` | Disable 2FA |

#### Register / Login bodies

```json
// POST /auth/register
{ "email": "user@example.com", "password": "s3cureP@ss" }

// POST /auth/login  (totp_code optional when 2FA is off)
{ "email": "user@example.com", "password": "s3cureP@ss", "totp_code": "123456" }
```

#### Login response `200`

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<opaque-token>",
  "user": { "id": "…", "email": "…", "display_name": "…" }
}
```

Access tokens expire in **~15 minutes**. Refresh tokens last **~7 days** and are stored as SHA-256 hashes in Postgres.

#### Refresh

```json
// POST /auth/refresh
{ "email": "user@example.com", "refresh_token": "<opaque-token>" }
```

---

### Expenses

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/expenses` | Create expense |
| GET | `/api/v1/expenses` | List expenses (paginated) |
| GET | `/api/v1/expenses/{id}` | Get by ID |
| PUT | `/api/v1/expenses/{id}` | Update |
| DELETE | `/api/v1/expenses/{id}` | Soft-delete |
| POST | `/api/v1/expenses/{id}/receipt` | Upload receipt (multipart, field `receipt`, max 5 MB) |
| POST | `/api/v1/expenses/recurring/post` | Post one instance from a recurring template |

**List query params:** `limit` (default 20, max 100), `pagination` (opaque cursor).

**Create body:**

```json
{
  "wallet_id":      "uuid",
  "category_id":    "uuid",
  "amount":         12.50,
  "currency":       "USD",
  "date":           "2026-06-01",
  "merchant":       "Corner Cafe",
  "notes":          "team lunch",
  "is_recurring":   false,
  "recurring_rule": null
}
```

> `currency` defaults to `USD` if omitted. If it differs from the wallet's currency the amount is automatically converted via the FX service.

**Post recurring template:**

```json
// POST /api/v1/expenses/recurring/post
{ "expense_id": "<template-uuid>", "date": "2026-06-07" }
```

---

### Incomes

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/incomes` | Create income |
| GET | `/api/v1/incomes` | List incomes (paginated) |
| GET | `/api/v1/incomes/{id}` | Get by ID |
| PUT | `/api/v1/incomes/{id}` | Update |
| DELETE | `/api/v1/incomes/{id}` | Soft-delete |

**Create body:**

```json
{
  "wallet_id":    "uuid",
  "category_id":  "uuid",
  "source_name":  "Salary",
  "amount":       3500.00,
  "currency":     "USD",
  "income_date":  "2026-06-01",
  "notes":        "May paycheck"
}
```

---

### Wallets

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/wallets` | Create wallet |
| GET | `/api/v1/wallets` | List all wallets |
| GET | `/api/v1/wallets/{id}` | Get by ID |
| PUT | `/api/v1/wallets/{id}` | Update |
| DELETE | `/api/v1/wallets/{id}` | Delete |
| POST | `/api/v1/wallets/transfer` | Transfer between wallets |

**Create body:**

```json
{
  "name":            "Cash Wallet",
  "wallet_type":     "CASH",
  "opening_balance": 100.00,
  "currency":        "USD",
  "account_number":  null,
  "account_name":    null
}
```

`wallet_type`: `CASH` · `MOBILE_WALLET` · `BANK` · `CARD`

**Transfer body:**

```json
{
  "from_wallet_id": "uuid",
  "to_wallet_id":   "uuid",
  "amount":         200.00,
  "currency":       "USD",
  "fee_amount":     0.50,
  "exchange_rate":  0,
  "transfer_date":  "2026-06-07",
  "notes":          "savings deposit"
}
```

> If wallets are in different currencies and `exchange_rate` is `0`, the FX service is called automatically.

---

### Categories

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/categories` | Create custom category |
| GET | `/api/v1/categories` | List categories |
| GET | `/api/v1/categories/{id}` | Get by ID |
| PUT | `/api/v1/categories/{id}` | Update (user-owned only) |
| DELETE | `/api/v1/categories/{id}` | Delete (user-owned only) |

**List query params:** `kind` (`EXPENSE` or `INCOME`, default `EXPENSE`).

**Create body:**

```json
{
  "name":  "Gym",
  "kind":  "EXPENSE",
  "icon":  "🏋️",
  "color": "#FF6B6B"
}
```

> Default (global) categories cannot be edited or deleted.

---

### Budgets

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/budgets` | Create budget |
| GET | `/api/v1/budgets` | List budgets |
| GET | `/api/v1/budgets/{id}` | Get by ID |
| PUT | `/api/v1/budgets/{id}` | Update |
| DELETE | `/api/v1/budgets/{id}` | Delete |

**List query params:** `period` (`MONTHLY` · `WEEKLY` · `YEARLY`).  
One MONTHLY budget per category per user — `409` on duplicate.

**Create body:**

```json
{
  "category_id":     "uuid",
  "amount":          500.00,
  "currency":        "USD",
  "period":          "MONTHLY",
  "start_date":      "2026-06-01",
  "rollover_enabled": false
}
```

---

### Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/dashboard/summary` | Spending KPIs and totals |
| GET | `/api/v1/dashboard/widgets` | Monthly budget usage widgets |

---

### Currencies

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/currencies` | No | List available currency codes |
| POST | `/api/v1/currencies/convert` | No | Convert an amount between two currencies |

**Convert body:**

```json
{ "from": "USD", "to": "EUR", "amount": 100 }
```

---

### Recurring Payments

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/recurring-payments` | Create |
| GET | `/api/v1/recurring-payments` | List |
| PUT | `/api/v1/recurring-payments/{id}` | Update |
| DELETE | `/api/v1/recurring-payments/{id}` | Delete |
| POST | `/api/v1/recurring-payments/{id}/pay` | Pay current cycle |

**Create / update body:**

```json
{
  "wallet_id":   "uuid",
  "category_id": "uuid",
  "title":       "Netflix",
  "amount":      15.99,
  "currency":    "USD",
  "interval":    "monthly",
  "start_date":  "2026-06-01",
  "deadline":    "2026-06-15",
  "alert_rule":  "7d",
  "end_date":    null
}
```

`interval`: `daily` · `weekly` · `monthly` · `yearly`  
`alert_rule`: `start` · `1h` · `12h` · `1d` · `7d`

**Pay body:**

```json
{ "payment_date": "2026-06-07", "fine": 0.00 }
```

Paying books an expense, adjusts the wallet balance, and auto-advances `start_date` / `deadline` to the next cycle.

---

### Notifications

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/notifications` | List notifications |
| POST | `/api/v1/notifications/read-all` | Mark all as read |
| POST | `/api/v1/notifications/{id}/read` | Mark one as read |
| POST | `/api/v1/notifications/{id}/dismiss` | Dismiss one |

**List query params:** `limit` (default 30), `unread_only` (`true` / `false`).

**Response:**

```json
{
  "notifications": [ { "id": "…", "message": "…", "is_read": false, "created_at": "…" } ],
  "unread_count": 3
}
```

---

### Operations

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/ops/refresh-tokens/cleanup` | Yes | Delete expired refresh tokens |

```json
// Response
{ "deleted": 42 }
```

---

## Design Decisions

### Interface-based dependency injection

Every repository accepts a `DBConnection` interface, not a concrete `*infra.Database`:

```go
type DBConnection interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    BeginTx(ctx context.Context) (pgx.Tx, error)
}
```

This allows unit tests to substitute a `mockDB` struct without a running database.

### Deterministic mock routing in tests

Mock `queryRowFn` hooks route responses by **inspecting arguments** (e.g. matching a `uuid.UUID` in `args`) rather than a sequential counter. This makes tests resilient to the internal order in which queries are issued:

```go
db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
    if len(args) >= 2 && args[1] == incomeID {
        return &mockRow{val: []any{walletID, 500.0}} // fetch old income
    }
    return &mockRow{val: []any{1000.0}} // fetch wallet balance
}
```

### Wallet balance accounting with deadlock prevention

All balance mutations (create/update/delete expense or income, transfer) run inside a serialisable transaction. Wallets are locked with `SELECT … FOR UPDATE` in **UUID-sorted order** so that concurrent transactions always acquire locks in the same sequence, eliminating circular-wait deadlocks.

### Soft deletes

Expenses and incomes use `is_deleted` + `deleted_at` rather than hard deletes, preserving data integrity for historical reports and budget calculations.

### Cursor-based pagination

List endpoints use opaque cursors (`base64(json{created_at, id})`) rather than `OFFSET`. This keeps pagination stable under concurrent writes and allows `O(log n)` indexed scans.

### Shared `domain.DomainError`

All business-rule violations surface through a single `DomainError` type that carries an error code, a human-readable message, and an HTTP status code. Handlers map this type to structured JSON error responses.

---

## Testing

### Run unit tests

```bash
cd backend
go test -cover ./...
```

Core service packages use `fakeStore` structs for in-memory repository stubs, and `mockDB` / `mockTx` for repository-layer tests:

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

### What is tested

- Validation edge cases (nil IDs, zero amounts, future dates, invalid currency codes)
- Database error propagation (begin-tx failure, `pgx.ErrNoRows`, scan errors)
- Transaction lifecycle (commit, rollback, wallet-balance consistency)
- Business rules (insufficient balance, wrong category owner, duplicate budget, mismatched currency on transfer)
- Pagination encoding and decoding
- Currency normalization and FX conversion paths

### Run integration tests

Integration tests exercise the full HTTP + Postgres stack:

```bash
export DATABASE_URL='postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable'
cd backend && go test ./tests/...
```

### Adding new tests

1. Define a `fake<Domain>Store` implementing the `Store` interface.
2. For repository tests, implement `mockDB` + `mockTx` with `queryRowFn` / `queryRowsFn` hooks.
3. Route responses by argument inspection, not call-count counters.
4. Assert both the success path **and** relevant error paths.

---

## Database Migrations

Migrations live in `backend/migrations/` and are managed by [`golang-migrate`](https://github.com/golang-migrate/migrate).

| # | Name | What it does |
|---|------|-------------|
| 001 | init schema | `users`, `sessions`, `refresh_tokens`, `categories`, `expenses`, `wallets` |
| 002 | seed default categories | 10 global categories (Food, Transport, Housing, …) |
| 003 | budgets wallets reporting | `budgets`, `tags`, `receipts`, `notifications` |
| 004 | income and loan management | `incomes`, `personal_loans` |
| 005 | refresh token metadata | `user_agent`, `ip_address` on refresh tokens |
| 006 | session tracking fields | `last_used_at`, `device_name` on sessions |
| 007 | 2FA fields | `totp_secret`, `totp_enabled` on `users` |
| 008 | category kind | `kind` column + migration of existing categories |
| 009 | recurring payments | `recurring_payments` table |

**Apply / rollback:**

```bash
# Apply all pending
migrate -path migrations -database "$DATABASE_URL" up

# Roll back one step
migrate -path migrations -database "$DATABASE_URL" down 1

# Create a new migration pair
migrate create -ext sql -dir migrations -seq <name>
```

---

## OpenAPI Code Generation

The canonical API contract is `backend/openapi.yaml`. Generated types live in `internal/httpapi/openapi/types.gen.go`.

Regenerate from the repository root:

```bash
make openapi
```

If `oapi-codegen` is missing:

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

If runtime types are missing in the module:

```bash
cd backend
go get github.com/oapi-codegen/runtime@latest
```
