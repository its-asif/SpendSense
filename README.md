# SpendSense - Phase 1

Personal expense tracker. One user, no teams, no sharing.

## Phase 1 Scope

- ✅ User Registration & Login (JWT + Redis refresh tokens)
- ✅ Create & List Expenses (cursor pagination)
- ✅ Edit & Soft-Delete Expenses
- ✅ Basic Categories (10 system defaults)
- ✅ CLI for quick logging
- ✅ Web Dashboard

## Tech Stack

| Layer | Choice |
|-------|--------|
| Backend | Go 1.21, Gin, PostgreSQL 15, SQLC, Redis |
| Auth | JWT (15 min) + Redis refresh token (7 days) |
| Frontend | React 18, TypeScript, Tailwind, Shadcn/ui, Recharts, TanStack Query |
| CLI | Go + Cobra + Viper |
| Migrations | golang-migrate |

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15 (or via Docker)
- Redis (or via Docker)

### Setup

1. **Start infrastructure**
   ```bash
   make infra-up
   ```

2. **Install dependencies**
   ```bash
   # Backend
   cd backend && go mod download
   
   # Frontend
   cd ../frontend && npm install
   ```

3. **Run migrations**
   ```bash
   make migrate
   ```

4. **Start backend**
   ```bash
   make backend-run
   ```

5. **Start frontend** (in new terminal)
   ```bash
   make frontend-dev
   ```

6. **Build CLI**
   ```bash
   make cli-build
   ```

## API Documentation

Swagger docs available at `http://localhost:8080/api/docs` once backend is running.

For a full API reference with example request and response bodies see the backend-specific API reference: [backend/README.md](backend/README.md).

## CLI Usage

```bash
# Register
./bin/expense register --email user@example.com

# Login
./bin/expense login --email user@example.com

# Add expense
./bin/expense add --amount 50 --category Food --date today --merchant "Cafe"

# List expenses
./bin/expense list
./bin/expense list --from 2024-01-01 --to 2024-01-31

# Logout
./bin/expense logout
```

Config stored at `~/.expenserc`

## Project Structure

```
.
├── backend/              # Go API
│   ├── cmd/
│   │   ├── api/         # API server
│   │   └── migrate/     # Migration runner
│   ├── internal/
│   │   ├── handler/     # HTTP handlers
│   │   ├── service/     # Business logic
│   │   ├── repository/  # Data access
│   │   ├── model/       # Domain models
│   │   ├── middleware/  # HTTP middleware
│   │   ├── config/      # Configuration
│   │   ├── db/          # Database setup
│   │   └── util/        # Utilities
│   ├── migrations/      # SQL migrations
│   ├── test/            # Integration tests
│   └── go.mod
├── cli/                 # Go CLI
│   ├── cmd/             # Cobra commands
│   ├── internal/        # CLI logic
│   └── main.go
├── frontend/            # React app
│   ├── src/
│   │   ├── components/  # React components
│   │   ├── pages/       # Page components
│   │   ├── hooks/       # Custom hooks
│   │   ├── services/    # API services
│   │   ├── stores/      # Zustand stores
│   │   ├── types/       # TypeScript types
│   │   └── lib/         # Utilities
│   ├── vite.config.ts
│   └── package.json
├── infrastructure/      # Docker Compose
│   └── docker-compose.yml
├── docs/               # Documentation
├── Makefile            # Development commands
└── README.md
```

## Development

### Code Organization

- **Handlers** - HTTP request/response handling
- **Services** - Business logic layer
- **Repositories** - Data access layer
- **Models** - Domain entities

### Testing

```bash
make test
```

Aim for 80%+ coverage in Phase 1.

### Database

Migrations managed with golang-migrate. New migrations:

```bash
migrate create -ext sql -dir backend/migrations -seq <name>
```

## Database Schema (Phase 1)

### users
- id (BIGSERIAL PK)
- email (VARCHAR UNIQUE)
- password_hash (VARCHAR)
- created_at, updated_at (TIMESTAMP)

### categories
- id (BIGSERIAL PK)
- name (VARCHAR)
- is_system (BOOLEAN)
- created_at (TIMESTAMP)

### expenses
- id (BIGSERIAL PK)
- user_id (BIGINT FK → users)
- wallet_id (BIGINT FK → wallets)
- amount (DECIMAL, > 0)
- currency (VARCHAR 3, ISO 4217)
- category_id (BIGINT FK → categories)
- date (DATE)
- merchant (VARCHAR, nullable)
- notes (TEXT, nullable)
- is_deleted (BOOLEAN)
- deleted_at (TIMESTAMP, nullable)
- created_at, updated_at (TIMESTAMP)

### incomes
- id (BIGSERIAL PK)
- user_id (BIGINT FK → users)
- wallet_id (BIGINT FK → wallets)
- category_id (BIGINT FK → categories, nullable)
- source_name (VARCHAR)
- amount (DECIMAL, > 0)
- currency (VARCHAR 3, ISO 4217)
- income_date (DATE)
- notes (TEXT, nullable)
- is_deleted (BOOLEAN)
- deleted_at (TIMESTAMP, nullable)
- created_at, updated_at (TIMESTAMP)

### wallets
- id (BIGSERIAL PK)
- user_id (BIGINT FK → users)
- name (VARCHAR)
- wallet_type (CASH, MOBILE_WALLET, BANK, CARD)
- provider (VARCHAR, nullable)
- opening_balance (DECIMAL)
- current_balance (DECIMAL)
- currency (VARCHAR 3)
- is_active (BOOLEAN)
- created_at, updated_at (TIMESTAMP)

### wallet_transfers
- id (BIGSERIAL PK)
- user_id (BIGINT FK → users)
- from_wallet_id (BIGINT FK → wallets)
- to_wallet_id (BIGINT FK → wallets)
- amount (DECIMAL, > 0)
- fee_amount (DECIMAL, >= 0)
- transfer_date (DATE)
- notes (TEXT, nullable)
- created_at (TIMESTAMP)

Indexes on user_id, category_id, date, (user_id, date DESC)

## Environment Variables

```
PORT=8080
DATABASE_URL=postgres://spendsense:spendsense@localhost:5432/spendsense
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key
ALLOW_FUTURE_DATES=false
```

## Definition of Done (Phase 1)

- [ ] All FR-P1-01 through FR-P1-06 implemented
- [ ] 80%+ test coverage
- [ ] Docker Compose runs Postgres 15 + Redis
- [ ] Swagger docs at /api/docs
- [ ] Cursor pagination verified
- [ ] Decimal amount handling implemented (amount)
- [ ] Income tracking implemented
- [ ] Wallet + transfer + transfer-fee flow implemented
- [ ] Handlers + Service + Repo layers for all FRs
- [ ] README with setup + API examples
