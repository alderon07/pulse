# Pulse Backend

Go backend for heartbeat monitoring with optional job-health signals.

## Quick start

1. Copy env file:

```bash
cp .env.example .env
```

2. Install dependencies and run tests:

```bash
just setup
just test
```

3. Run local stack with Docker:

```bash
docker compose up -d postgres
just dc-migrate-up
just run-api
just run-worker
```

## Endpoints

- `GET /healthz`
- `GET /readyz`
- `GET /v1/ping/{token}`
- `POST /v1/ping/{token}`
- Deprecated alias: `/ping/{token}`
- Authenticated (`Authorization: Bearer <JWT>`):
  - `GET/POST /api/v1/checks`
  - `GET/PATCH /api/v1/checks/{id}`
  - `POST /api/v1/checks/{id}/pause`
  - `POST /api/v1/checks/{id}/resume`
  - `POST /api/v1/checks/{id}/rotate-token`
  - `GET /api/v1/checks/{id}/events`
  - `GET/POST /api/v1/alert-channels`
  - `PATCH /api/v1/alert-channels/{id}`

## Notes

- Ping endpoint sets `Cache-Control: no-store`.
- Token values are redacted from request logs.
- Worker applies overdue transitions (`late` and `down`) and queues alert deliveries.
- API expects HS256 JWTs (`JWT_HS256_SECRET`); `sub` must be a UUID.
