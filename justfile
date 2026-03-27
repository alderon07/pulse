set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# ── dev ──────────────────────────────────────────────────

# Run frontend + backend API in parallel (Ctrl-C stops both)
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    trap 'kill 0' EXIT
    echo "▸ backend API → :8080"
    (cd backend && set -a && source .env.local && set +a && exec go run ./cmd/api) &
    echo "▸ frontend    → :3000"
    (cd frontend && exec npm run dev) &
    wait

# Run frontend + backend API + worker
dev-all:
    #!/usr/bin/env bash
    set -euo pipefail
    trap 'kill 0' EXIT
    echo "▸ backend API    → :8080"
    (cd backend && set -a && source .env.local && set +a && exec go run ./cmd/api) &
    echo "▸ backend worker → background"
    (cd backend && set -a && source .env.local && set +a && exec go run ./cmd/worker) &
    echo "▸ frontend       → :3000"
    (cd frontend && exec npm run dev) &
    wait

dev-api:
    cd backend && set -a && source .env.local && set +a && go run ./cmd/api

dev-worker:
    cd backend && set -a && source .env.local && set +a && go run ./cmd/worker

dev-fe:
    cd frontend && npm run dev

# ── backend ──────────────────────────────────────────────

be-setup:
    cd backend && go mod tidy

be-fmt:
    cd backend && gofmt -w ./cmd ./internal

be-vet:
    cd backend && go vet ./...

be-lint:
    cd backend && golangci-lint run

be-test:
    cd backend && go test ./...

# ── frontend ─────────────────────────────────────────────

fe-setup:
    cd frontend && npm install

fe-build:
    cd frontend && npm run build

fe-lint:
    cd frontend && npm run lint

# ── migrations ───────────────────────────────────────────

migrate-up:
    #!/usr/bin/env bash
    set -euo pipefail
    set -a && source backend/.env.local && set +a
    goose -dir backend/migrations postgres "$DATABASE_URL" up

migrate-down:
    #!/usr/bin/env bash
    set -euo pipefail
    set -a && source backend/.env.local && set +a
    goose -dir backend/migrations postgres "$DATABASE_URL" down

migrate-status:
    #!/usr/bin/env bash
    set -euo pipefail
    set -a && source backend/.env.local && set +a
    goose -dir backend/migrations postgres "$DATABASE_URL" status

migrate-create name:
    goose -dir backend/migrations create {{name}} sql

# ── docker ───────────────────────────────────────────────

dc-up:
    cd backend && docker compose up -d --build

dc-down:
    cd backend && docker compose down -v

dc-logs:
    cd backend && docker compose logs -f --tail=100

# ── ci ───────────────────────────────────────────────────

ci: be-fmt be-vet be-test fe-lint
