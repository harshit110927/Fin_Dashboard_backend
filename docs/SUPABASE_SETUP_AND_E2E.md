# Full Setup Guide (Local + Supabase) with Dummy Data and End-to-End Testing

This document explains how to run the backend, connect it to a Supabase Postgres database, seed realistic dummy data, and validate the main API flow end-to-end.

## 1) Prerequisites

- Go 1.22+
- PostgreSQL client (`psql`)
- `curl`
- A Supabase project (for hosted DB setup)

## 2) Clone and install dependencies

```bash
git clone <your-repo-url>
cd Fin_Dashboard_backend
go mod tidy
```

## 3) Configure environment

Create a local `.env` from `.env.example`:

```bash
cp .env.example .env
```

### Local Postgres example

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=finance_dashboard
DB_SSLMODE=disable
JWT_SECRET=change_me_to_a_long_random_secret
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
SERVER_PORT=8080
APP_ENV=development
```

### Supabase Postgres example

Use your Supabase project database values from **Project Settings → Database**.

```env
DB_HOST=db.<project-ref>.supabase.co
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<your_supabase_db_password>
DB_NAME=postgres
DB_SSLMODE=require
JWT_SECRET=change_me_to_a_long_random_secret
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
SERVER_PORT=8080
APP_ENV=production
```

> If Supabase pooler settings are used in your project, use the pooler host/port provided by Supabase.

## 4) Run schema migrations

```bash
make migrate
```

This runs all SQL files in `internal/db/migrations/` in order.

## 5) Seed dummy data

```bash
make seed
```

Seed file location: `internal/db/seed_dummy_data.sql`.

### Seeded credentials

- `admin@fin.local` / `Admin@12345`
- `analyst@fin.local` / `Analyst@12345`
- `viewer@fin.local` / `Viewer@12345`

## 6) Run backend

```bash
make run
```

Server starts on `http://localhost:${SERVER_PORT}` (default `8080`).

## 7) End-to-end API test flow (curl)

Set base URL:

```bash
BASE_URL=http://localhost:8080/api/v1
```

### A. Auth flow

#### 1. Login as admin

```bash
curl -s -X POST "$BASE_URL/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@fin.local","password":"Admin@12345"}'
```

Save tokens from response as shell variables:

```bash
ADMIN_ACCESS=<paste_access_token>
ADMIN_REFRESH=<paste_refresh_token>
```

#### 2. Refresh token

```bash
curl -s -X POST "$BASE_URL/auth/refresh" \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$ADMIN_REFRESH\"}"
```

### B. Category flow

#### 1. List categories

```bash
curl -s "$BASE_URL/categories" \
  -H "Authorization: Bearer $ADMIN_ACCESS"
```

#### 2. Create category

```bash
curl -s -X POST "$BASE_URL/categories" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d '{"name":"Equipment","type":"expense"}'
```

#### 3. Update category

```bash
curl -s -X PATCH "$BASE_URL/categories/1" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d '{"name":"Salary Main"}'
```

### C. Record flow

#### 1. Create a financial record

```bash
curl -s -X POST "$BASE_URL/records" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d '{
    "amount": 999.99,
    "type": "expense",
    "category_id": 6,
    "date": "2026-04-01",
    "description": "E2E test expense"
  }'
```

#### 2. List records

```bash
curl -s "$BASE_URL/records?page=1&per_page=10&type=expense" \
  -H "Authorization: Bearer $ADMIN_ACCESS"
```

#### 3. Update mutable fields

```bash
curl -s -X PATCH "$BASE_URL/records/<record_id>" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d '{"description":"Updated E2E description","category_id":8}'
```

#### 4. Void record

```bash
curl -s -X POST "$BASE_URL/records/<record_id>/void" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d '{"reason":"Duplicate entry added during end-to-end test"}'
```

### D. Dashboard flow

```bash
curl -s "$BASE_URL/dashboard/summary?from=2026-04-01&to=2026-04-30" -H "Authorization: Bearer $ADMIN_ACCESS"
curl -s "$BASE_URL/dashboard/trends" -H "Authorization: Bearer $ADMIN_ACCESS"
curl -s "$BASE_URL/dashboard/categories" -H "Authorization: Bearer $ADMIN_ACCESS"
curl -s "$BASE_URL/dashboard/recent?limit=10" -H "Authorization: Bearer $ADMIN_ACCESS"
```

### E. User administration flow

```bash
# List users
curl -s "$BASE_URL/users?page=1&per_page=20" -H "Authorization: Bearer $ADMIN_ACCESS"

# Create user
curl -s -X POST "$BASE_URL/users" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d '{"name":"Ops User","email":"ops@fin.local","password":"OpsUser@12345","role":"viewer"}'

# Update role
curl -s -X PATCH "$BASE_URL/users/<user_id>/role" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d '{"role":"analyst"}'
```

### F. Logout

```bash
curl -s -X POST "$BASE_URL/auth/logout" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS" \
  -d "{\"refresh_token\":\"$ADMIN_REFRESH\"}"
```

## 8) What to verify during E2E

- Login returns both access + refresh tokens.
- Role-based authorization works:
  - viewer cannot create records or users.
  - analyst can read trends but not mutate resources.
  - admin can access all routes.
- Dashboard values update after create/update/void/delete operations.
- `records` update does not allow changing immutable fields (`amount`, `type`).
- Validation errors return structured payloads.

## 9) Helpful troubleshooting

- `pq: password authentication failed`: verify Supabase DB password.
- `connection refused`: verify DB host/port and IP allow list/network path.
- `SSL is required`: set `DB_SSLMODE=require` for Supabase.
- `make migrate`/`make seed` fails: ensure `psql` is installed and `.env` is present.

## 10) Automated E2E via JavaScript

A complete Node.js script is available at `scripts/e2e-test.js` and validates:

- auth login/refresh/logout
- role-based access checks (viewer/analyst/admin)
- category create/update
- record create/list/get/update/void
- dashboard summary/trends/categories/recent
- user create/update-role/update-status/delete

Run it after `make run`:

```bash
BASE_URL=http://localhost:8080/api/v1 node scripts/e2e-test.js
```

If your seed credentials are different, override via env vars:

```bash
ADMIN_EMAIL=admin@fin.local \
ADMIN_PASSWORD=Admin@12345 \
ANALYST_EMAIL=analyst@fin.local \
ANALYST_PASSWORD=Analyst@12345 \
VIEWER_EMAIL=viewer@fin.local \
VIEWER_PASSWORD=Viewer@12345 \
node scripts/e2e-test.js
```
