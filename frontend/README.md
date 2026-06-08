# SpendSense — Frontend

React + TypeScript web application for the SpendSense personal finance tracker.

> Full project context and architecture overview: [../README.md](../README.md)  
> Backend API reference: [../backend/README.md](../backend/README.md)

---

## Table of Contents

- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Pages](#pages)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [API Layer](#api-layer)
- [State & Auth Flow](#state--auth-flow)
- [Theming](#theming)

---

## Tech Stack

| Tool | Version | Purpose |
|------|---------|---------|
| React | 18 | UI framework |
| TypeScript | 5 | Type safety |
| Vite | 5 | Dev server & bundler |
| Tailwind CSS | 3 | Utility-first styling |
| React Router | 6 | Client-side routing |
| Axios | 1 | HTTP client with interceptors |
| Recharts | 3 | Dashboard charts |
| react-hot-toast | 2 | Toast notifications |

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Browser                        │
│                                                  │
│  ┌──────────┐   ┌──────────┐   ┌─────────────┐  │
│  │  Router  │──▶│  Pages   │──▶│  Components │  │
│  └──────────┘   └────┬─────┘   └─────────────┘  │
│                      │                           │
│              ┌───────▼────────┐                  │
│              │  API Layer     │  src/api/         │
│              │  (axios)       │                  │
│              └───────┬────────┘                  │
│                      │ REST / JSON               │
└──────────────────────┼──────────────────────────┘
                       ▼
             Go HTTP API  :8080
```

### Request / auth flow

```
Page component
  └─► api/*.ts (typed fetch functions)
        └─► src/api/client.ts  (axios instance)
              ├── Request interceptor  — injects Authorization: Bearer <access_token>
              └── Response interceptor — on 401: calls /auth/refresh, retries original
```

Access tokens are short-lived (~15 min). The axios response interceptor transparently refreshes them using the stored refresh token, so the user is never interrupted.

---

## Project Structure

```
frontend/
├── index.html
├── vite.config.ts
├── tailwind.config.js
├── tsconfig.json
└── src/
    ├── main.tsx              # App entry — mounts <App />, sets up Router
    ├── App.tsx               # Route definitions, global session-event listener
    ├── index.css             # Global styles, Tailwind base
    ├── api/                  # One file per resource + shared axios client
    │   ├── client.ts         # Axios instance, auth interceptors, token refresh
    │   ├── auth.ts           # register, login, logout, me, sessions, 2FA
    │   ├── expenses.ts       # CRUD + receipt upload + recurring post
    │   ├── incomes.ts        # CRUD
    │   ├── wallets.ts        # CRUD + transfer
    │   ├── budgets.ts        # CRUD
    │   ├── categories.ts     # CRUD
    │   ├── currencies.ts     # list, convert
    │   ├── dashboard.ts      # summary, widgets
    │   ├── notifications.ts  # list, read, dismiss
    │   └── recurring.ts      # CRUD + pay
    ├── pages/                # Top-level route components
    │   ├── Dashboard.tsx
    │   ├── ExpensesPage.tsx
    │   ├── IncomesPage.tsx
    │   ├── WalletsPage.tsx
    │   ├── BudgetsPage.tsx
    │   ├── CategoriesPage.tsx
    │   ├── RecurringPage.tsx
    │   ├── ReportsPage.tsx
    │   ├── SettingsPage.tsx
    │   └── Login.tsx
    ├── components/           # Reusable UI building blocks
    ├── hooks/                # Custom React hooks
    ├── stores/               # Global state (auth, theme, etc.)
    ├── services/             # Non-API helpers (formatting, date utils)
    ├── types/                # TypeScript interfaces & enums
    ├── data/                 # Static data (e.g. default category icons)
    ├── lib/                  # Utility functions
    └── styles/               # Additional CSS modules
```

---

## Pages

| Route | Component | Description |
|-------|-----------|-------------|
| `/` | `Dashboard.tsx` | KPI cards, spending charts, budget widgets, recent transactions |
| `/expenses` | `ExpensesPage.tsx` | Expense list, create/edit/delete, receipt upload |
| `/incomes` | `IncomesPage.tsx` | Income list, create/edit/delete |
| `/wallets` | `WalletsPage.tsx` | Wallet list, create/edit/delete, wallet-to-wallet transfers |
| `/budgets` | `BudgetsPage.tsx` | Budget list, create/edit/delete, usage progress bars |
| `/categories` | `CategoriesPage.tsx` | Category list, create/edit/delete (user-defined only) |
| `/recurring` | `RecurringPage.tsx` | Recurring payment list, create/edit/delete, pay current cycle |
| `/reports` | `ReportsPage.tsx` | Spending breakdown charts, income vs expense trends |
| `/settings` | `SettingsPage.tsx` | Profile, account, preferences, security, active sessions |
| `/login` | `Login.tsx` | Auth form (login + register), TOTP entry when 2FA is enabled |

---

## Getting Started

### Prerequisites

- Node.js 18+
- A running SpendSense backend on `http://localhost:8080` (see [../backend/README.md](../backend/README.md))

### Install & run

```bash
cd frontend
npm install
npm run dev
# App: http://localhost:5173
```

### Build for production

```bash
npm run build
# Output: frontend/dist/
```

### Preview production build

```bash
npm run preview
```

---

## Environment Variables

Create a `frontend/.env` file (optional — defaults shown):

```env
VITE_API_URL=http://localhost:8080
```

All `VITE_*` variables are inlined at build time by Vite and are accessible in the browser. **Do not store secrets here.**

---

## API Layer

Every resource has a dedicated file in `src/api/` that exports typed async functions:

```ts
// Example: src/api/expenses.ts
export const createExpense = (data: CreateExpenseRequest) =>
  apiClient.post<Expense>('/api/v1/expenses', data);

export const listExpenses = (cursor?: string, limit = 20) =>
  apiClient.get<ExpenseListResponse>('/api/v1/expenses', {
    params: { pagination: cursor, limit },
  });
```

The shared `src/api/client.ts` exports a single Axios instance configured with:

- `baseURL` from `VITE_API_URL`
- **Request interceptor** — reads the access token from `localStorage` and sets `Authorization: Bearer <token>`
- **Response interceptor** — on `401`, attempts to refresh via `POST /auth/refresh`; on success, retries the original request; on failure, redirects to `/login`

---

## State & Auth Flow

Authentication state is held in `src/stores/` and `localStorage`:

| Key | Storage | Content |
|-----|---------|---------|
| `accessToken` | `localStorage` | Short-lived JWT |
| `refreshToken` | `localStorage` | Opaque refresh token |
| `user` | `localStorage` / store | Serialised user profile |

`App.tsx` listens for a global `session-change` custom event dispatched whenever tokens are written or cleared, so components that render user data (e.g. the avatar in the navbar) update in real time without a page reload.

---

## Theming

- Dark / light mode is toggled via a class on `<html>` (`dark` / `light`).
- The user's preference is persisted in `localStorage` and restored on load.
- Tailwind's `darkMode: 'class'` strategy is used; all colour utilities are paired with `dark:` variants.
