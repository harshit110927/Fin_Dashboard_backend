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
┌──────────────────────────────────────────────────────────┐
│                    Middleware Chain                        │
│  RequestID → RequestLogger → RateLimiter → Auth → RoleGuard│
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                     Handler Layer                         │
│       Parse · Validate · Call service · Respond          │
│              Zero business logic here                     │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                     Service Layer                         │
│   Business logic · Role enforcement (defence-in-depth)   │
│   Orchestrates repositories · Writes audit log           │
│          Transport-agnostic — no Gin imports             │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                  Repository Layer (interfaces)            │
│    Write path (mutations)  │  Read path (views)          │
│    record_repo             │  dashboard_repo             │
│    user_repo               │  vw_record_details          │
│    audit_repo (append-only)│  mvw_monthly_summary        │
│    token_repo              │  mvw_category_totals        │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                       PostgreSQL                          │
│   Tables · Regular Views · Materialized Views            │
│   Partial Indexes · JSONB Audit Snapshots                │
└──────────────────────────────────────────────────────────┘
```

**Three strict rules:**
- Handlers contain zero business logic
- Services contain zero SQL and zero Gin imports
- Repositories contain zero business decisions

Services depend on **repository interfaces**, not concrete types — every data access behaviour is independently testable without a live database.

---

## Role Model

| Role | Records | Dashboard | Users | Trends | Audit History |
|---|---|---|---|---|---|
| `viewer` | Read only | Summary + Categories + Recent | ✗ | ✗ | ✗ |
| `analyst` | Read only | All endpoints | ✗ | ✓ | ✗ |
| `admin` | Full CRUD + Void | All endpoints | Full CRUD | ✓ | ✓ |

Role is embedded in the JWT payload and enforced at two layers: the `RoleGuard` middleware on every route, and the service layer as defence-in-depth. If this service is ever called from a non-HTTP context, the business rule still holds.

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
GIN_MODE=release
RATE_LIMIT_RPM=100
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

Registration defaults all new users to `viewer`. A one-time bootstrap endpoint creates the first admin when the users table is empty:

```bash
curl -X POST http://localhost:8080/api/v1/auth/bootstrap
```

Returns `access_token` + `refresh_token` for `admin@finance.dev` / `Admin@Bootstrap1`.
Use the token to create additional users via `POST /api/v1/users`.

> This endpoint returns `403` if any users exist, and is **disabled entirely** when `APP_ENV != development`.

### 4. Start the server

```bash
make run
# Server starting on port 8080
```

The server starts with graceful shutdown enabled. On `SIGINT` or `SIGTERM` it waits up to 10 seconds for in-flight requests to complete before exiting — ensuring mid-flight financial writes are never interrupted.

---

## API Reference

All responses follow this envelope without exception:

```json
// Success
{ "success": true, "data": { } }

// Success paginated
{ "success": true, "data": [ ], "meta": { "page": 1, "per_page": 20, "total": 150 } }

// Error
{ "success": false, "error": { "code": "SNAKE_CASE_CODE", "message": "Human readable message" } }
```

### System

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/health` | None | Liveness probe — DB ping status + request ID |

```json
// GET /health — Response 200
{
  "status": "ok",
  "db": "ok",
  "timestamp": "2026-04-01T10:00:00Z",
  "request_id": "uuid"
}
```

### Auth

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/auth/register` | Public | Register (defaults to viewer role) |
| POST | `/api/v1/auth/login` | Public | Returns access + refresh token |
| POST | `/api/v1/auth/refresh` | Public | Exchange refresh token for new access token |
| POST | `/api/v1/auth/logout` | Required | Revokes refresh token in DB |
| POST | `/api/v1/auth/bootstrap` | Dev only | Creates first admin on empty DB |

```json
// POST /api/v1/auth/login — Request
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
| POST | `/api/v1/records/:id/void` | admin | Void with mandatory reason (min 10 chars) |
| GET | `/api/v1/records/:id/history` | admin | Full audit trail for a single record |

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
| `per_page` | integer | Page size (default 20, max 100) |

```json
// POST /api/v1/records — Request
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

> **Fintech rule enforced:** `amount` and `type` are immutable after creation. A PATCH request containing either field returns `400 IMMUTABLE_FIELD`. Void the record and create a corrected one.

### Dashboard

| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/dashboard/summary` | viewer+ | Total income, expenses, net balance |
| GET | `/api/v1/dashboard/trends` | analyst+ | Monthly income vs expense breakdown |
| GET | `/api/v1/dashboard/categories` | viewer+ | Per-category totals split by type |
| GET | `/api/v1/dashboard/recent` | viewer+ | Latest N records (`?limit`, max 50) |

```json
// GET /api/v1/dashboard/summary?from=2026-04-01&to=2026-04-30
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

`net_balance` is computed in the service layer — guaranteeing exact arithmetic with no floating-point accumulation across rows. Defaults to current calendar month when `from`/`to` are omitted.

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
| `roles` | Static seed | 3 rows, never mutated by the application |
| `users` | Read-heavy | Partial index on email, index on role_id |
| `categories` | Read-heavy | Seeded with 9 defaults, admin-extensible |
| `financial_records` | Write-heavy | 5 partial indexes, all `WHERE deleted_at IS NULL` |
| `audit_logs` | Append-only | `BIGSERIAL` PK for insert speed, `JSONB` before/after snapshots |
| `refresh_tokens` | Write-heavy | SHA-256 hashed, revocable, hard-deleted on expiry cleanup |

### Views

| View | Type | Purpose |
|---|---|---|
| `vw_record_details` | Regular view | Pre-joins records + categories + users for all record reads |
| `mvw_monthly_summary` | Materialized | Monthly income/expense totals, refreshed after every write |
| `mvw_category_totals` | Materialized | Per-category totals, refreshed after every write |

Materialized views are refreshed with `REFRESH MATERIALIZED VIEW CONCURRENTLY` — active reads are never blocked during refresh. This requires a unique index on each materialized view, defined in the migrations.

### Why `NUMERIC(15,2)` not `FLOAT`

```
FLOAT:   100.10 + 200.20 = 300.29999999999998   ← unacceptable in finance
NUMERIC: 100.10 + 200.20 = 300.30               ← exact decimal arithmetic
```

### Why `BIGSERIAL` for audit_log id

Audit logs are insert-only at high volume. Sequential integer IDs are faster to insert and index than random UUIDs — sequential inserts cause no page splits in the B-tree index. Business tables use UUID for distributed uniqueness; the audit log has no such requirement.

### Partial Indexes

All five indexes on `financial_records` use `WHERE deleted_at IS NULL`. As records are soft-deleted, they are excluded from every index automatically — keeping index size proportional to active data, not total historical data.

---

## Key Design Decisions

### 1. Void vs Soft Delete

Two separate invalidation mechanisms exist intentionally:

- **Soft delete** (`deleted_at`) — operational removal. The record disappears from all listings and dashboard calculations.
- **Void** (`status = 'void'`) — accounting invalidation. The record stays visible with `status=void`, excluded from financial totals, and requires a mandatory written reason. A posted transaction cannot simply vanish.

Both can be applied independently. A record can be voided without being deleted.

### 2. Immutable Financial Fields

`amount` and `type` cannot be changed after a record is created. The handler detects these fields in the raw request body before struct binding. The service layer enforces the same rule independently as defence-in-depth. If the HTTP layer is ever bypassed, the business rule still holds.

### 3. Audit Log on Every Mutation

Every create, update, void, and delete writes a row to `audit_logs` with the actor ID, IP address, and full JSONB snapshots before and after the change. The `GET /records/:id/history` endpoint exposes this trail via the API.

### 4. Materialized Views for Dashboard

Rather than application-level caching — which introduces invalidation complexity and stale-read risk — PostgreSQL materialized views pre-compute and physically store aggregation results. After every write the service triggers a concurrent refresh. Dashboard reads hit small, pre-aggregated tables with unique indexes.

### 5. Interface-Based Repositories

Every service depends on a repository interface, not a concrete struct. Business logic is testable without a live database. Compile-time interface compliance checks in `interfaces.go` catch any drift between interface and implementation at build time, not at runtime.

### 6. Refresh Token Hashing

Refresh tokens are stored as `SHA-256(token)` — never the raw value. If the `refresh_tokens` table were compromised, the attacker obtains hashes that cannot be reversed to usable tokens.

### 7. Graceful Shutdown

The server handles `SIGINT` and `SIGTERM` with a 10-second drain window. In a financial system, a mid-flight record create followed by an audit log write must not be interrupted — both succeed or neither does.

### 8. Typed Error System

All domain errors are defined as `*apperr.AppError` values in `pkg/apperr` with a machine-readable `SNAKE_CASE_CODE`, human-readable message, and HTTP status code. PostgreSQL constraint codes are translated in `pkg/dberr` before they reach any handler. No raw error strings or magic status codes are scattered across the codebase.

### 9. Request Correlation

Every request receives a unique `X-Request-ID` header (generated if not supplied by the client). The ID is logged with every request line and returned in every response. Log correlation across client and server requires no external tracing infrastructure.

### 10. Migration Strategy

Migrations use ordered raw SQL files executed via `psql` for zero extra tooling dependencies. Each file is idempotent (`IF NOT EXISTS`, `ON CONFLICT DO NOTHING`).

**Tradeoff:** In production this would use [goose](https://github.com/pressly/goose) or [golang-migrate](https://github.com/golang-migrate/migrate) for versioned rollback and migration state tracking. Excluded here to keep setup dependency-free.

---

## Running Tests

The E2E test suite covers **109 test cases** with no external npm dependencies — only Node.js 18+ built-in `fetch`.

```bash
# Default (localhost:8080)
node scripts/e2e-test.js

# Custom target
BASE_URL=http://localhost:8080/api/v1 node scripts/e2e-test.js

# Debug mode — prints every request + full response body
DEBUG=1 node scripts/e2e-test.js

# Halt on first failure
STOP_ON_FAIL=1 node scripts/e2e-test.js
```

| Category | Cases | What's covered |
|---|---|---|
| Auth — Login & Tokens | 9 | Login shape, JWT role claim, refresh, invalid refresh |
| Auth — Validation | 11 | Duplicate email, bad format, short password, wrong credentials, garbage/malformed JWT |
| RBAC | 13 | Every role against every restricted route |
| Categories | 6 | CRUD, duplicate name, bad type, role enforcement |
| Records — Validation | 6 | Zero amount, negative, bad type, missing fields, invalid category |
| Records — CRUD | 18 | Create, read, filter ×4, sort, paginate, update, immutability ×2, void, double-void, soft delete |
| Health & History | 5 | DB ping, X-Request-ID header, audit trail shape, RBAC on history |
| Dashboard | 14 | Summary math, default params, trend shape, category shape, limit enforcement |
| Users | 14 | Full lifecycle, inactive login blocked, deleted user blocked, role validation |
| Logout | 3 | Token revocation, revoked token rejected on refresh |
| Response shape | 3 | Error envelope, success envelope, pagination meta |

Go unit tests cover the service layer independently:

```bash
go test ./internal/service/...
```

---

## Project Structure

```
finance-dashboard/
├── cmd/server/main.go              ← Entry point, dependency wiring, graceful shutdown
├── internal/
│   ├── config/config.go            ← Env loading, typed Config struct
│   ├── db/
│   │   ├── postgres.go             ← Connection pool (tuned for Supabase free tier)
│   │   └── migrations/             ← 7 ordered idempotent SQL files
│   ├── middleware/
│   │   ├── request_id.go           ← Injects X-Request-ID for log correlation
│   │   ├── request_logger.go       ← Logs method, path, status, latency, IP, request_id
│   │   ├── rate_limiter.go         ← Configurable req/min per IP, in-memory
│   │   ├── auth.go                 ← JWT validation, sets user_id + role in Gin context
│   │   └── role_guard.go           ← Role enforcement, 403 INSUFFICIENT_PERMISSIONS
│   ├── domain/                     ← Pure Go structs — no DB tags mixed with JSON tags
│   ├── handler/
│   │   ├── health_handler.go       ← /health with DB ping
│   │   └── *.go                    ← HTTP only — parse, validate, call service, respond
│   ├── service/
│   │   ├── record_service_test.go  ← Unit tests for immutability, date range, limits
│   │   └── *.go                    ← All business logic, transport-agnostic
│   ├── repository/
│   │   ├── interfaces.go           ← Repository interfaces + compile-time checks
│   │   └── *.go                    ← PostgreSQL implementations via sqlx, raw SQL
│   └── router/router.go            ← All 27 routes with correct middleware chains
├── pkg/
│   ├── apperr/errors.go            ← Typed domain errors (code + message + HTTP status)
│   ├── dberr/dberr.go              ← PostgreSQL error code → AppError translation
│   ├── jwt/jwt.go                  ← Token generation and validation
│   ├── password/bcrypt.go          ← Hash + Verify helpers
│   ├── response/response.go        ← Standardised JSON envelope helpers
│   └── validator/validator.go      ← Singleton validator instance
├── scripts/
│   ├── e2e-test.js                 ← 109-case E2E runner, Node 18+, zero npm deps
│   └── migrate.bat                 ← Windows migration runner
├── .env.example
├── Makefile
└── README.md
```

---

## Makefile Targets

```bash
make run        # go run ./cmd/server/main.go
make build      # go build -o bin/server ./cmd/server/main.go
make migrate    # run all SQL migrations via psql (Linux/macOS)
make tidy       # go mod tidy
make test       # go test ./...
```

Windows: use `scripts\migrate.bat` instead of `make migrate`.

---

## Assumptions

| Assumption | Decision |
|---|---|
| Role IDs | Fixed: `viewer=1`, `analyst=2`, `admin=3`. Seeded once, never mutated. |
| Date format | All dates as `YYYY-MM-DD` strings. UTC throughout. |
| Money precision | `NUMERIC(15,2)` in DB. JSON returns Go `float64` — safe for realistic business amounts. |
| Mat. view refresh | Triggered after each write. For very high write throughput would move to async queue refresh. |
| Bootstrap endpoint | One-time use. Disabled outside development. Returns `403` if users already exist. |
| Hard delete scope | Never on `users`, `financial_records`, or `categories`. `refresh_tokens` are hard-deleted on expiry cleanup. |
| Rate limiting | In-memory per-IP, not shared across instances. Multi-instance would use Redis. Configurable via `RATE_LIMIT_RPM`. |
| Error translation | All PostgreSQL constraint errors translated to typed `AppError` values. Raw `pq.Error` never reaches API responses. |

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
