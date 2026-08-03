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

Seed the database — this creates two demo facilities and exactly one account, the system administrator (`administrator@wadaaghealthy.com` / `wadaaghealthy.com@2026`). Every other user (physicians, hospital admins, patients, etc.) is created afterward from the webapp's Authentication module by that admin, or via patient self-signup — nothing else is hardcoded:

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

## Production readiness — not yet done, tracked here so it isn't forgotten

`deploy/docker-compose.yml` is dev-only (hot-reloading dev servers, a hardcoded insecure `JWT_SECRET`/`ENCRYPTION_KEY` dev default, no TLS). Before any real deployment, the following need to happen — from the roadmap's Phase 10 security-hardening pass (`docs/plans/full-feature-roadmap.md`):

- **HTTPS/TLS termination**: nothing in this repo terminates TLS today. Production needs a reverse proxy (Caddy, nginx, or a cloud load balancer) in front of the `backend`/`web` containers handling certificate provisioning and forcing HTTPS — not yet built here; add it as part of standing up the actual production environment, once a real domain/host is chosen.
- **Database encryption at rest**: two layers —
  1. Volume/disk-level encryption is expected to come from whatever managed Postgres hosting is chosen (DigitalOcean Managed Databases, AWS RDS, etc. — these encrypt at rest by default with no application change needed). Pick a host that provides this; don't run production Postgres on an unencrypted raw disk.
  2. Column-level encryption for the most sensitive fields (`patients.national_id`, `patient_medical_history`'s six jsonb columns) is implemented at the application layer via `pgcrypto` — see migration `0024_encrypt_sensitive_columns` and `ENCRYPTION_KEY` in `platform.Config`. **`ENCRYPTION_KEY` must be set to a real, securely-generated secret outside development** (same requirement as `JWT_SECRET`) — losing or rotating it without a re-encryption migration makes the encrypted data unrecoverable. Note the accepted tradeoff: patient search by `national_id` no longer works (encrypted values can't support partial-text match) — search by name/phone still does.
- **File-scanning on upload**: deliberately not implemented. Attachments (Phase 3, `internal/attachments`) accept files from authenticated users with no malware/content scanning. Before real patient documents flow through this in production, add a scan step in `attachments.Service.Upload` — either a self-hosted ClamAV container (no vendor account needed) or a hosted scanning API (needs an account + credentials, a product-owner decision when the time comes).
- **RBAC audit**: a review pass against every endpoint built across Phases 0-9 was done as part of Phase 10 — see that review's findings for anything that needs following up before production.
