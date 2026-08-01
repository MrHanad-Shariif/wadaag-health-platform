COMPOSE = docker compose -f deploy/docker-compose.yml

.PHONY: up down logs migrate seed test backend-test web-test lint

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

# Re-run migrations without restarting the whole stack (e.g. after adding
# a new migration file while the stack is already up).
migrate:
	$(COMPOSE) run --rm migrate

# Host port 5433 matches the mapping in deploy/docker-compose.yml (5432 may
# already be taken by a native Postgres install on the host).
seed:
	cd backend && DATABASE_URL="postgres://wadaag:wadaag@localhost:5433/wadaag_health?sslmode=disable" go run ./db/seed

test: backend-test web-test

backend-test:
	cd backend && go vet ./... && go test ./...

web-test:
	cd web && npm run lint && npm run build

lint:
	cd backend && go vet ./...
	cd web && npm run lint
