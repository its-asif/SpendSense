# Backend API Reference

This file documents the HTTP API for the backend with example request bodies and usage notes.

Auth
----

- POST /auth/register
  - Body:
    ```json
    {
      "email": "user@example.com",
      "password": "s3cureP@ssw0rd"
    }
    ```

- POST /auth/login
  - Body:
    ```json
    {
      "email": "user@example.com",
      "password": "s3cureP@ssw0rd"
    }
    ```
  - Response (200):
    ```json
    {
      "access_token": "<jwt-access-token>",
      "expires_in": 900,
      "refresh_token": "<refresh-token>"
    }
    ```

- POST /auth/refresh
  - Body:
    ```json
    {
      "email": "user@example.com",
      "refresh_token": "<refresh-token>"
    }
    ```
  - Response: new `access_token` and optionally a new `refresh_token`.

- POST /auth/logout
  - Auth: `Authorization: Bearer <access_token>`
  - Body:
    ```json
    {
      "refresh_token": "<refresh-token>"
    }
    ```
  - Description: Revokes the provided refresh token (single session logout).

- POST /auth/logout-all
  - Auth: `Authorization: Bearer <access_token>`
  - Body: none
  - Description: Revokes all refresh tokens for the authenticated user (logout from all sessions).

Ops
----

- POST /ops/refresh-tokens/cleanup
  - Auth: `Authorization: Bearer <access_token>` (recommended for admin/ops)
  - Body: none
  - Response (200):
    ```json
    { "deleted": 42 }
    ```
  - Description: Trigger a one-off cleanup job to purge expired refresh tokens.

Health
------

- GET /health
  - No auth required
  - Response:
    ```json
    { "status": "ok" }
    ```

API v1 Resource Endpoints
------------------------

Common headers
- `Authorization: Bearer <access_token>` for protected endpoints
- `Content-Type: application/json`

Expenses
- POST /api/v1/expenses
  - Body:
    ```json
    {
      "wallet_id": 1,
      "amount": "12.50",
      "currency": "USD",
      "category_id": 3,
      "date": "2026-05-24",
      "merchant": "Cafe",
      "notes": "latte"
    }
    ```

- GET /api/v1/expenses?limit=20&pagination=<pagination>
  - Query params: `limit` (int), `pagination` (opaque)

- GET /api/v1/expenses/:id
  - Response: expense object

- PUT /api/v1/expenses/:id
  - Body (partial or full update):
    ```json
    {
      "merchant": "Coffee Shop",
      "notes": "latte and croissant"
    }
    ```

- DELETE /api/v1/expenses/:id
  - Soft-deletes the expense (marks `is_deleted=true`).

Categories
- POST /api/v1/categories
  - Body:
    ```json
    {
      "name": "Groceries",
      "is_system": false
    }
    ```

- GET /api/v1/categories
- GET /api/v1/categories/:id
- PUT /api/v1/categories/:id
- DELETE /api/v1/categories/:id

Wallets
- POST /api/v1/wallets
  - Body:
    ```json
    {
      "name": "Cash Wallet",
      "wallet_type": "CASH",
      "provider": null,
      "opening_balance": "100.00",
      "currency": "USD"
    }
    ```

- GET /api/v1/wallets
- GET /api/v1/wallets/:id
- PUT /api/v1/wallets/:id
- DELETE /api/v1/wallets/:id

Incomes
- POST /api/v1/incomes
  - Body:
    ```json
    {
      "wallet_id": 1,
      "category_id": null,
      "source_name": "Salary",
      "amount": "1500.00",
      "currency": "USD",
      "income_date": "2026-05-01",
      "notes": "May paycheck"
    }
    ```

- GET /api/v1/incomes?limit=20&pagination=<pagination>
- GET /api/v1/incomes/:id
- PUT /api/v1/incomes/:id
- DELETE /api/v1/incomes/:id

Notes
-----

- Amounts are sent as strings to preserve decimal precision (use server-side decimal parsing).
- Access tokens are short-lived (about 15 minutes); refresh tokens are long-lived and stored hashed server-side.
- If you run the server locally, Swagger UI is available at `/api/docs` when enabled.

If you want the examples in a different format (OpenAPI YAML/JSON or Postman collection), tell me which format and I'll generate it.

Developer: OpenAPI & Swagger
--------------------------------

- Regenerate Go types from the OpenAPI YAML (committed spec):
  - From the repository root run:

```bash
cd backend
make openapi
```

  - This generates the Go types package at [backend/internal/httpapi/openapi/types.gen.go](backend/internal/httpapi/openapi/types.gen.go#L1).

- Run the backend and view Swagger UI:
  - Start the server (example):

```bash
cd backend
go run ./cmd/api
```

  - Open the API docs in your browser: http://localhost:8080/api/docs
  - The raw spec is served at: http://localhost:8080/openapi.yaml

- Troubleshooting / dependencies:
  - The generator uses `oapi-codegen` and the runtime helper. If you run into missing packages, run:

```bash
cd backend
go get github.com/oapi-codegen/runtime@latest
```

  - The `make openapi` target runs the generator with `go run`, so you usually don't need to install the generator globally.

