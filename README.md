# Fin_Dashboard_backend

A role-based finance dashboard backend built with Go, Gin, and PostgreSQL. It provides secure authentication, financial record management, category management, dashboard analytics, and audit logging. The system is designed for teams where users have different visibility and control levels (`viewer`, `analyst`, `admin`).

## Tech stack

- Go (Gin, sqlx)
- PostgreSQL
- JWT authentication (`golang-jwt/jwt/v5`)
- Password hashing (bcrypt)
- In-memory IP rate limiting middleware

## Prerequisites

- Go **1.22+**
- PostgreSQL **14+**
- `psql` available in shell for migrations

## Setup

1. Clone the repository.
2. Create env file:
   ```bash
   cp .env.example .env
   ```
3. Fill DB and JWT values in `.env`.
4. Run migrations:
   ```bash
   make migrate
   ```
5. Start API server:
   ```bash
   make run
   ```

## Role permissions

| Role | Permissions |
|---|---|
| `viewer` | Read records, summary, categories, recent activity |
| `analyst` | Viewer permissions + trends endpoint |
| `admin` | Full CRUD on users, records, and categories |

## API reference (24 endpoints)

| Method | Endpoint | Role required | Description |
|---|---|---|---|
| POST | `/api/v1/auth/register` | Public | Register a new user (default viewer) |
| POST | `/api/v1/auth/login` | Public | Login and receive access/refresh tokens |
| POST | `/api/v1/auth/refresh` | Public | Exchange refresh token for new access token |
| POST | `/api/v1/auth/logout` | Any authenticated | Revoke refresh token |
| GET | `/api/v1/users` | `admin` | List users (paginated) |
| GET | `/api/v1/users/:id` | `admin` | Get one user |
| POST | `/api/v1/users` | `admin` | Create user |
| PATCH | `/api/v1/users/:id` | `admin` | Update user profile fields |
| PATCH | `/api/v1/users/:id/role` | `admin` | Change user role |
| PATCH | `/api/v1/users/:id/status` | `admin` | Enable/disable user |
| DELETE | `/api/v1/users/:id` | `admin` | Soft-delete user |
| GET | `/api/v1/records` | `viewer`,`analyst`,`admin` | List financial records (filtered, paginated) |
| GET | `/api/v1/records/:id` | `viewer`,`analyst`,`admin` | Get one financial record |
| POST | `/api/v1/records` | `admin` | Create financial record |
| PATCH | `/api/v1/records/:id` | `admin` | Update mutable record fields |
| DELETE | `/api/v1/records/:id` | `admin` | Soft-delete financial record |
| POST | `/api/v1/records/:id/void` | `admin` | Void record while preserving history |
| GET | `/api/v1/dashboard/summary` | `viewer`,`analyst`,`admin` | Income/expense/net summary |
| GET | `/api/v1/dashboard/trends` | `analyst`,`admin` | Monthly trends |
| GET | `/api/v1/dashboard/categories` | `viewer`,`analyst`,`admin` | Category totals split by type |
| GET | `/api/v1/dashboard/recent` | `viewer`,`analyst`,`admin` | Recent activity |
| GET | `/api/v1/categories` | `viewer`,`analyst`,`admin` | List categories |
| POST | `/api/v1/categories` | `admin` | Create category |
| PATCH | `/api/v1/categories/:id` | `admin` | Update category |

## Key design decisions

- **NUMERIC in DB for money**: avoids floating-point precision issues at persistence layer.
- **Soft delete**: records/users are recoverable and traceable (`deleted_at`).
- **Void vs delete**: void keeps record visible as invalidated; delete hides from operational views.
- **Materialized views**: faster dashboard reads (`mvw_monthly_summary`, `mvw_category_totals`).
- **Audit logs**: critical write actions are recorded with actor and data snapshots.
- **Immutable amount/type on update**: prevents semantic mutation of financial entries.

## Assumptions

- Role IDs are fixed as: `viewer=1`, `analyst=2`, `admin=3`.
- Dates are exchanged as `YYYY-MM-DD` strings.
- Auth middleware injects `user_id` and `role` into request context for protected routes.
- Refresh token revocation is persisted in DB and access token is short-lived.
