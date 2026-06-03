# SpendSense Backend

HTTP API for SpendSense. Run from this directory:

```bash
docker compose up -d
# set DATABASE_URL, JWT_SECRET, etc. in .env (see repository README)
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/api
```

- Swagger UI: http://localhost:8080/api/docs  
- OpenAPI file: http://localhost:8080/openapi.yaml  
- Canonical schema: [internal/httpapi/openapi.yaml](internal/httpapi/openapi.yaml)

Protected routes expect `Authorization: Bearer <access_token>` unless noted otherwise.

## Health and documentation

| Method | Path | Auth |
|--------|------|------|
| GET | `/health` | No |
| GET | `/openapi.yaml` | No |
| GET | `/api/docs` | No |

## Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Create account |
| POST | `/auth/login` | Login (supports TOTP code when 2FA enabled) |
| POST | `/auth/refresh` | Exchange refresh token for new access token |
| POST | `/auth/logout` | Revoke one refresh token |
| POST | `/auth/logout-all` | Revoke all refresh tokens for the user |
| POST | `/auth/logout-other` | Revoke all refresh tokens except the current session |
| GET | `/auth/me` | Current user profile |
| PATCH | `/auth/me/profile` | Update display name, avatar, etc. |
| PATCH | `/auth/me/preferences` | Update base currency, timezone, locale |
| POST | `/auth/me/password` | Change password |
| GET | `/auth/me/sessions` | List active sessions |
| DELETE | `/auth/me/sessions/{sessionId}` | Revoke a session |
| POST | `/auth/me/2fa/setup` | Start TOTP enrollment |
| POST | `/auth/me/2fa/confirm` | Confirm TOTP with a code |
| POST | `/auth/me/2fa/disable` | Disable 2FA |

### Register

```json
{
  "email": "user@example.com",
  "password": "s3cureP@ssw0rd"
}
```

### Login

```json
{
  "email": "user@example.com",
  "password": "s3cureP@ssw0rd",
  "totp_code": ""
}
```

### Login response (200)

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<refresh-token>",
  "user": { }
}
```

Access tokens expire in about **15 minutes**. Refresh tokens last about **7 days** and are stored hashed server-side.

### Refresh

```json
{
  "email": "user@example.com",
  "refresh_token": "<refresh-token>"
}
```

### Logout (single session)

```json
{
  "refresh_token": "<refresh-token>"
}
```

## Operations

| Method | Path | Description |
|--------|------|-------------|
| POST | `/ops/refresh-tokens/cleanup` | Delete expired refresh tokens (authenticated) |

Response example:

```json
{ "deleted": 42 }
```

## API v1 — resources

Common headers: `Authorization: Bearer <access_token>`, `Content-Type: application/json`.

Amounts are JSON **strings** to preserve decimal precision.

### Expenses

| Method | Path |
|--------|------|
| POST, GET | `/api/v1/expenses` |
| GET, PUT, DELETE | `/api/v1/expenses/{id}` |

List query: `limit`, `pagination` (opaque cursor).

Create body example:

```json
{
  "wallet_id": "…",
  "amount": "12.50",
  "currency": "USD",
  "category_id": "…",
  "date": "2026-05-24",
  "merchant": "Cafe",
  "notes": "latte",
  "is_recurring": false
}
```

DELETE soft-deletes the expense.

### Incomes

| Method | Path |
|--------|------|
| POST, GET | `/api/v1/incomes` |
| GET, PUT, DELETE | `/api/v1/incomes/{id}` |

Create body example:

```json
{
  "wallet_id": "…",
  "category_id": null,
  "source_name": "Salary",
  "amount": "1500.00",
  "currency": "USD",
  "income_date": "2026-05-01",
  "notes": "May paycheck"
}
```

### Categories

| Method | Path |
|--------|------|
| POST, GET | `/api/v1/categories` |
| GET, PUT, DELETE | `/api/v1/categories/{id}` |

### Wallets

| Method | Path |
|--------|------|
| POST, GET | `/api/v1/wallets` |
| GET, PUT, DELETE | `/api/v1/wallets/{id}` |
| POST | `/api/v1/wallets/transfer` |

Create body example:

```json
{
  "name": "Cash Wallet",
  "wallet_type": "CASH",
  "opening_balance": "100.00",
  "currency": "USD"
}
```

`wallet_type`: `CASH`, `MOBILE_WALLET`, `BANK`, `CARD`.

### Dashboard

| Method | Path |
|--------|------|
| GET | `/api/v1/dashboard/summary` |
| GET | `/api/v1/dashboard/widgets` |

Widgets include budget usage rows when matching data exists in the `budgets` table.

### Currencies

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/currencies` | No |
| POST | `/api/v1/currencies/convert` | No |

## OpenAPI code generation

From the repository root:

```bash
make openapi
```

This writes [internal/httpapi/openapi/types.gen.go](internal/httpapi/openapi/types.gen.go).

If generation fails on missing modules:

```bash
cd backend
go get github.com/oapi-codegen/runtime@latest
```

## Tests

```bash
cd backend
go test ./...
```

Integration package:

```bash
export DATABASE_URL='postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable'
go test ./tests/...
```

## Packages without HTTP routes

The following exist in `internal/` or in migrations but are **not** part of the public API yet: budget management (CRUD), tags, receipts, notifications, personal loans, recurring job processing. Do not assume endpoints for these until they appear in `openapi.yaml` and the handler registrations in `internal/httpapi/`.
