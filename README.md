# Fin Dashboard Backend

> A production-grade finance dashboard API built with **Go**, **Gin**, and **PostgreSQL** — featuring role-based access control, immutable financial records, audit logging, and pre-aggregated analytics via PostgreSQL materialized views.

---

## Why This Architecture

Most CRUD backends treat financial data like any other data. Finance systems have stricter rules:

- A posted transaction **cannot be edited** — it can only be voided and re-entered
- Nothing is ever truly deleted — every state change is traceable
- Aggregation queries run constantly — they must be fast regardless of record volume
- Every action by every user must leave a permanent trail

This backend enforces all four constraints at the service and database layer — not just in documentation.

---

## Tech Stack

| Layer | Choice | Reason |
|---|---|---|
| Language | Go 1.22 | Explicit error handling maps cleanly to financial validation requirements |
| Framework | Gin | Minimal overhead, clean middleware chaining |
| Database | PostgreSQL 14+ | NUMERIC type, materialized views, partial indexes |
| DB Access | sqlx + raw SQL | Full control over queries — no ORM magic hiding what runs |
| Auth | JWT (golang-jwt/jwt/v5) | Short-lived access tokens + DB-stored revocable refresh tokens |
| Password | bcrypt (cost 12) | Industry standard, resistant to brute force |
| Validation | go-playground/validator/v10 | Struct-tag validation, consistent error messages |
| Config | godotenv | 12-factor app compatible |

---

## Architecture

```
Request
  │
  ▼
┌─────────────────────────────────────────┐
│             Middleware Chain             │
│  RateLimiter → Logger → Auth → RoleGuard│
└──────────────────┬──────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────┐
│              Handler Layer               │
│  Parse · Validate · Call service · Respond│
└──────────────────┬───────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────┐
│              Service Layer               │
│  Business logic · Role enforcement       │
│  Orchestrates repos · Writes audit log   │
└──────┬───────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│            Repository Layer              │
│  Write repo  │  Read repo               │
│  (mutations) │  (views, mat. views)     │
└──────┬───────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│              PostgreSQL                  │
│  Tables · Views · Materialized Views     │
│  Partial Indexes · Audit Log             │
└──────────────────────────────────────────┘
```

**Rule:** Handlers contain zero business logic. Services contain zero SQL. Repositories contain zero business decisions.

---

## Role Model

| Role | Records | Dashboard | Users | Trends |
|---|---|---|---|---|
| `viewer` | Read only | Summary + Categories + Recent | ✗ | ✗ |
| `analyst` | Read only | All dashboard endpoints | ✗ | ✓ |
| `admin` | Full CRUD + Void | All dashboard endpoints | Full CRUD | ✓ |

Role is embedded in the JWT payload and enforced by middleware on every protected route. Service layer performs a secondary check for mutations as defence-in-depth.

---

## Prerequisites

- Go **1.22+**
- PostgreSQL **14+** (or a Supabase project)
- Node.js **18+** (for E2E test runner only)
- `psql` available in PATH (for migrations)

---

## Setup

### 1. Clone and configure

```bash
git clone https://github.com/your-username/fin-dashboard-backend
cd fin-dashboard-backend
cp .env.example .env
```

Fill `.env` with your values:

```env
DB_HOST=your-host
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=postgres
DB_SSLMODE=require          # use 'disable' for local PostgreSQL
DATABASE_URL=postgresql://postgres:password@host:5432/postgres
JWT_SECRET=minimum-32-character-secret-here
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
SERVER_PORT=8080
APP_ENV=development
```

### 2. Run migrations

**Linux / macOS:**
```bash
make migrate
```

**Windows:**
```bat
scripts\migrate.bat
```

Migrations are idempotent — safe to re-run. They create all tables, indexes, views, and seed categories and roles.

### 3. Seed the first admin

Because registration defaults new users to `viewer`, a bootstrap endpoint exists in development mode to create the first admin:

```bash
curl -X POST http://localhost:8080/api/v1/auth/bootstrap
```

Returns `access_token` + `refresh_token` for `admin@finance.dev` / `Admin@Bootstrap1`.

> This endpoint is **disabled** when `APP_ENV != development`.

### 4. Start the server

```bash
make run
# Server starting on port 8080
```

---

## API Reference

All responses follow this envelope:

```json
// Success
{ "success": true, "data": { } }

// Success paginated
{ "success": true, "data": [ ], "meta": { "page": 1, "per_page": 20, "total": 150 } }

// Error
{ "success": false, "error": { "code": "SNAKE_CASE_CODE", "message": "Human readable message" } }
```

### Auth

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/auth/register` | Public | Register (defaults to viewer role) |
| POST | `/api/v1/auth/login` | Public | Returns access + refresh token |
| POST | `/api/v1/auth/refresh` | Public | Exchange refresh token for new access token |
| POST | `/api/v1/auth/logout` | Required | Revokes refresh token in DB |
| POST | `/api/v1/auth/bootstrap` | Dev only | Creates first admin on empty DB |

**Login request/response:**
```json
// POST /api/v1/auth/login
// Request
{ "email": "admin@company.com", "password": "Admin@12345" }

// Response 200
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "user": { "id": "uuid", "name": "Admin", "email": "...", "role": "admin" }
  }
}
```

### Users (Admin only)

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/users` | Paginated list (`?page&per_page`) |
| GET | `/api/v1/users/:id` | Single user |
| POST | `/api/v1/users` | Create user with explicit role |
| PATCH | `/api/v1/users/:id` | Update name or email |
| PATCH | `/api/v1/users/:id/role` | Change role (`viewer`\|`analyst`\|`admin`) |
| PATCH | `/api/v1/users/:id/status` | Activate / deactivate |
| DELETE | `/api/v1/users/:id` | Soft delete (sets `deleted_at`) |

### Financial Records

| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/records` | viewer+ | Filtered, paginated list |
| GET | `/api/v1/records/:id` | viewer+ | Single record with category + creator |
| POST | `/api/v1/records` | admin | Create record |
| PATCH | `/api/v1/records/:id` | admin | Update mutable fields only |
| DELETE | `/api/v1/records/:id` | admin | Soft delete |
| POST | `/api/v1/records/:id/void` | admin | Void with mandatory reason |

**GET /records — query parameters:**

| Param | Type | Description |
|---|---|---|
| `type` | `income`\|`expense` | Filter by transaction type |
| `category_id` | integer | Filter by category |
| `status` | `active`\|`void` | Filter by status |
| `date_from` | `YYYY-MM-DD` | Start of date range |
| `date_to` | `YYYY-MM-DD` | End of date range |
| `sort` | `date_desc`\|`date_asc`\|`amount_desc`\|`amount_asc` | Sort order |
| `page` | integer | Page number (default 1) |
| `per_page` | integer | Page size (default 20) |

**Create record request/response:**
```json
// POST /api/v1/records
// Request
{
  "amount": 75000.00,
  "type": "income",
  "category_id": 1,
  "date": "2026-04-01",
  "description": "April salary payment"
}

// Response 201
{
  "success": true,
  "data": {
    "id": "uuid",
    "amount": 75000,
    "type": "income",
    "category_name": "Salary",
    "date": "2026-04-01",
    "status": "active",
    "created_by_name": "Admin User",
    "created_at": "2026-04-01T10:00:00Z"
  }
}
```

> **Fintech rule enforced:** `amount` and `type` are immutable after creation. A PATCH request containing either field returns `400 IMMUTABLE_FIELD`. Use `POST /records/:id/void` to invalidate a wrong record, then create a corrected one.

### Dashboard

| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/dashboard/summary` | viewer+ | Total income, expenses, net balance |
| GET | `/api/v1/dashboard/trends` | analyst+ | Monthly income vs expense breakdown |
| GET | `/api/v1/dashboard/categories` | viewer+ | Per-category totals split by type |
| GET | `/api/v1/dashboard/recent` | viewer+ | Latest N records (`?limit`, max 50) |

**Summary response:**
```json
{
  "success": true,
  "data": {
    "total_income": 950000.00,
    "total_expenses": 620000.00,
    "net_balance": 330000.00,
    "period": { "from": "2026-04-01", "to": "2026-04-30" }
  }
}
```

Accepts `?from=YYYY-MM-DD&to=YYYY-MM-DD`. Defaults to the current calendar month.

### Categories

| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/categories` | viewer+ | List all active categories |
| POST | `/api/v1/categories` | admin | Create category |
| PATCH | `/api/v1/categories/:id` | admin | Update name or active status |

---

## Database Design

### Tables

| Table | Type | Notes |
|---|---|---|
| `roles` | Static seed | 3 rows, never mutated by app |
| `users` | Read-heavy | Indexed on email and role_id |
| `categories` | Read-heavy | Seeded with 9 defaults |
| `financial_records` | Write-heavy | 5 partial indexes (all `WHERE deleted_at IS NULL`) |
| `audit_logs` | Append-only | `BIGSERIAL` PK for insert speed, JSONB snapshots |
| `refresh_tokens` | Write-heavy | Hashed storage, revocable |

### Views

| View | Type | Purpose |
|---|---|---|
| `vw_record_details` | Regular view | Pre-joins records + categories + users |
| `mvw_monthly_summary` | Materialized | Monthly income/expense totals, refreshed after every write |
| `mvw_category_totals` | Materialized | Per-category totals, refreshed after every write |

Materialized views are refreshed with `REFRESH MATERIALIZED VIEW CONCURRENTLY` — reads are never blocked during refresh.

### Why `NUMERIC(15,2)` not `FLOAT`

```
FLOAT:   100.10 + 200.20 = 300.29999999999998   ← unacceptable in finance
NUMERIC: 100.10 + 200.20 = 300.30               ← exact decimal arithmetic
```

### Why `BIGSERIAL` for audit_log id

Audit logs are insert-only at high volume. Sequential integer IDs are faster to insert and index than random UUIDs. Business tables use UUID for distributed safety; the audit log has no such requirement.

### Partial Indexes

All indexes on `financial_records` use `WHERE deleted_at IS NULL`. Deleted records are excluded from every index, keeping index size small and scans fast as the table grows.

---

## Key Design Decisions

### 1. Void vs Soft Delete

Two separate invalidation mechanisms exist intentionally:

- **Soft delete** (`deleted_at`) — operational removal. The record disappears from all listings and dashboards. Used for genuine duplicates or data entry errors caught immediately.
- **Void** (`status = 'void'`) — accounting invalidation. The record remains visible with `status=void`, is excluded from financial calculations, and requires a mandatory written reason. This is how real accounting systems work — you never erase a posted transaction.

### 2. Immutable Financial Fields

`amount` and `type` cannot be changed after a record is created. The service layer explicitly rejects PATCH requests containing these fields with `IMMUTABLE_FIELD`. This prevents silent financial discrepancies — if a number was wrong, there must be a void record proving it was wrong and a new record proving the correction.

### 3. Audit Log on Every Mutation

Every create, update, void, and delete writes a row to `audit_logs` with:
- Who did it (`actor_id`)
- What changed (`old_data` and `new_data` as JSONB snapshots)
- When (`created_at`)
- From where (`ip_address`)

This is not optional in financial systems. Regulatory requirements in most jurisdictions mandate it.

### 4. Materialized Views for Dashboard

Dashboard queries like "sum all expenses grouped by month for the last 12 months" are expensive on large tables. Rather than application-level caching (which introduces cache invalidation complexity), PostgreSQL materialized views pre-compute and store these results. After every write to `financial_records`, the service triggers a concurrent refresh. Dashboard reads are then simple indexed scans of small pre-aggregated tables.

### 5. Refresh Token Hashing

Refresh tokens are stored as `SHA-256(token)` in the database — never the raw token. If the `refresh_tokens` table were compromised, the attacker gets hashes they cannot use. The actual token only exists in the HTTP response and the client's storage.

### 6. Connection Pool Tuned for Supabase

Supabase free tier allows a maximum of 10 concurrent connections. The pool is set to `MaxOpenConns=10, MaxIdleConns=3, ConnMaxLifetime=5min` to respect this limit and avoid connection exhaustion under concurrent requests.

---

## Running Tests

The E2E test suite covers **101 test cases** across auth, RBAC, record validation, dashboard, and user management. It requires Node.js 18+ and no npm packages.

```bash
# Default (localhost:8080)
node scripts/e2e-test.js

# Custom URL
BASE_URL=http://localhost:8080/api/v1 node scripts/e2e-test.js

# Debug mode (prints every request + response body)
DEBUG=1 node scripts/e2e-test.js

# Stop immediately on first failure
STOP_ON_FAIL=1 node scripts/e2e-test.js
```

**Test categories:**

| Category | Cases | What's covered |
|---|---|---|
| Auth — Login & Tokens | 6 | Login shape, role in token, refresh, invalid refresh |
| Auth — Validation | 11 | Duplicate email, bad format, short password, wrong credentials, garbage JWT |
| RBAC | 13 | Every role against every restricted route |
| Categories | 6 | CRUD, duplicate name, bad type, role enforcement |
| Records — Validation | 6 | Zero amount, negative amount, bad type, missing fields |
| Records — CRUD | 18 | Create, read, filter, sort, paginate, update, immutability ×2, void, double-void, soft delete |
| Dashboard | 14 | Summary math, default params, trend shape, category shape, recent limits |
| Users | 14 | Full flow, inactive login blocked, deleted user login blocked, role validation |
| Logout | 3 | Revoke token, revoked token rejected on refresh |
| Response shape | 3 | Error envelope, success envelope, pagination meta |

---

## Project Structure

```
finance-dashboard/
├── cmd/server/main.go          ← Entry point, dependency wiring
├── internal/
│   ├── config/                 ← Env loading
│   ├── db/
│   │   ├── postgres.go         ← Connection pool setup
│   │   └── migrations/         ← 7 ordered idempotent SQL files
│   ├── middleware/             ← Auth, RoleGuard, Logger, RateLimiter
│   ├── domain/                 ← Pure Go structs, no DB tags
│   ├── handler/                ← HTTP layer only, zero business logic
│   ├── service/                ← All business decisions live here
│   ├── repository/             ← All SQL lives here
│   └── router/                 ← Route registration
├── pkg/
│   ├── jwt/                    ← Token generation and validation
│   ├── password/               ← bcrypt helpers
│   ├── response/               ← Standardised JSON envelope helpers
│   └── validator/              ← Singleton validator instance
├── scripts/
│   ├── e2e-test.js             ← 101-case E2E test runner (Node 18+)
│   └── migrate.bat             ← Windows migration runner
├── .env.example
├── Makefile
└── README.md
```

---

## Makefile Targets

```bash
make run        # go run ./cmd/server/main.go
make build      # go build -o bin/server ./cmd/server/main.go
make migrate    # run all SQL migrations via psql
make tidy       # go mod tidy
make test       # go test ./...
```

---

## Assumptions

| Assumption | Decision |
|---|---|
| Role IDs | Fixed: `viewer=1`, `analyst=2`, `admin=3`. Seeded once, never mutated. |
| Date format | All dates exchanged as `YYYY-MM-DD` strings. |
| Money precision | All amounts are `NUMERIC(15,2)` in the DB. JSON responses return Go `float64` — precision is maintained end-to-end because values never exceed float64's safe integer range for realistic business amounts. |
| Materialized view refresh | Triggered synchronously in the service after each write. For very high write throughput this would move to a queue-based refresh — acceptable tradeoff for this scale. |
| Bootstrap endpoint | One-time use. Disabled in non-development environments. Calling it when users already exist returns `403`. |
| Soft delete scope | Hard deletes never occur on `users`, `financial_records`, or `categories`. `refresh_tokens` are hard-deleted on cleanup to control table growth. |

---

## Seeded Categories

| ID | Name | Type |
|---|---|---|
| 1 | Salary | income |
| 2 | Freelance | income |
| 3 | Investment | income |
| 4 | Rent | expense |
| 5 | Marketing | expense |
| 6 | Utilities | expense |
| 7 | Salaries | expense |
| 8 | Travel | expense |
| 9 | Software | expense |
