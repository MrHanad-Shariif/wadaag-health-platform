# Wadaag Health Platform

A digital health platform connecting physicians, hospitals, laboratories, pharmacies, and insurers to make care coordination and telemedicine work across Somalia and Africa.

Stack: **React** (web) · **Go** (backend, modular monolith) · **PostgreSQL**.

## Repo layout

```
backend/    Go API — internal/<domain> modules (identity, referrals, records, consent, ...)
web/        React + Vite + TypeScript
mobile/     React Native app (built in a later phase)
deploy/     Docker Compose stack for local dev
docs/       API spec (OpenAPI, added as endpoints grow)
```

## Local dev — Docker Compose

Requires Docker Desktop.

```
docker compose -f deploy/docker-compose.yml up --build
```

This brings up Postgres, runs migrations, then starts the backend (`:8080`, hot-reloading via `air`) and the web app (`:5173`, Vite dev server). Postgres is exposed on host port **5433**, not 5432 — a native Postgres install already listening on 5432 will silently swallow connections meant for the container otherwise.

Seed demo accounts (one per role, all sharing password `wadaag-dev-2026`):

```
cd backend && DATABASE_URL="postgres://wadaag:wadaag@localhost:5433/wadaag_health?sslmode=disable" go run ./db/seed
```

If you have `make` available (Git Bash + `choco install make`, or WSL), the same steps are `make up` / `make seed` — see the `Makefile` for the full list of targets (`down`, `logs`, `migrate`, `test`, `lint`).

New/changed migrations aren't picked up by an already-running stack — run `docker compose -f deploy/docker-compose.yml run --rm migrate` (or `make migrate`) after adding one. Also, on Windows, `air`'s file-watcher inside the container can miss changes written from the Windows host to the bind-mounted `backend/` volume (a known Docker Desktop bind-mount limitation) — if a code change doesn't seem to take effect, `docker compose -f deploy/docker-compose.yml restart backend` forces a rebuild.

## Local dev — without Docker

1. Run Postgres 16 locally, create a `wadaag_health` database owned by a `wadaag`/`wadaag` role (or point `DATABASE_URL` at whatever you have).
2. `cd backend && cp .env.example .env` (edit if your DB differs), then apply migrations:
   ```
   migrate -path db/migrations -database "$DATABASE_URL" up
   ```
3. `cd backend && go run ./cmd/api`
4. `cd web && cp .env.example .env && npm install && npm run dev`

## Status

MVP-first build: Digital Referrals, Unified Health Records, Consent & Privacy Controls, and Notifications are the core loop being built now. Telemedicine, Cross-Specialty Consults, Lab Integration, Pharmacy Integration, the offline-first mobile app, and Insurance Readiness are staged in afterward — see the module list under `backend/internal/` for the stubbed domains already reserving their place in the architecture.
