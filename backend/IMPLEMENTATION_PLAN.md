# Pulse Backend Implementation Plan

## Scope
Build `pulse/backend` as a Go + Postgres heartbeat monitor with optional job-health signals (`duration_ms`, `output_size`, `success`), API versioning, a separate overdue worker, and robust alert delivery.

## Core Decisions
- Public ping endpoint: `/v1/ping/{token}` (with temporary deprecated alias `/ping/{token}`)
- Management endpoints: `/api/v1/*` with JWT auth (Clerk-compatible claims mapping)
- Migrations: `goose`
- DB query layer: `sqlc`
- API contract: OpenAPI (`openapi/pulse-v1.yaml`) as source of truth
- Separate worker binary for overdue scanning + alert delivery
- Dedicated incidents table with one-open-incident DB constraint
- Notifications: Email + generic webhook
- Drift warnings are non-paging by default

## High-Level Build Order
1. Tooling bootstrap (`mise`, `just`, lint/test/codegen configs)
2. OpenAPI v1 contract and CI contract gates
3. DB schema + constraints + indexes (goose)
4. Domain logic tests (status transitions, incidents, repeat alert, drift rules)
5. Ping ingestion (`GET/POST /v1/ping/{token}`) with atomic DB transaction
6. Overdue worker (`late`/`down` transitions, incident open/resolve)
7. Outbox + notification sending (retry/backoff/dead-letter)
8. Authenticated check/channel management APIs
9. Security and abuse controls (rate limits, token rotation, webhook hardening)
10. Docker dev/CI parity and observability hardening

## Data/State Rules
- `up`: `now <= next_due_at`
- `late`: `next_due_at < now <= next_due_at + grace`
- `down`: `now > next_due_at + grace`
- `paused`: excluded from overdue transitions and alerts
- New checks start with `next_due_at = null` until first ping
- On resume: set `next_due_at = now + interval` to avoid immediate false down

## Caveats / Gotchas / Implementation Details Checklist

### Reliability and correctness
- [ ] Add bounded retry policy for DB deadlock/serialization errors
- [ ] Enforce expand/contract zero-downtime migrations (no long blocking locks)
- [ ] Define exactly-once consumer behavior via stable IDs (system remains at-least-once)
- [ ] Guarantee down/recovered ordering per incident in delivery queue
- [ ] Prevent duplicate recovered alerts via state-transition guard only
- [ ] Add synthetic canary checks to monitor the monitor itself
- [ ] Add audit trail for sensitive actions (token rotate, pause/resume, channel changes)
- [ ] Propagate correlation IDs across API, worker, and webhook payloads

### Performance and scale
- [ ] Keep ping path O(1) and minimal lock scope (`events` + `checks` hot writes)
- [ ] Keep `events` indexing minimal in v1 to protect ingest throughput
- [ ] Tune autovacuum for `events` and `alert_deliveries` high churn
- [ ] Baseline overdue/queue queries with `EXPLAIN ANALYZE`
- [ ] Prevent N+1 queries in checks/events list APIs
- [ ] Configure worker batch size/time budget and tenant fairness
- [ ] Split/limit DB pools for API vs worker contention control
- [ ] Use retry jitter to avoid synchronized retry storms

### Security and abuse
- [ ] Never log raw ping tokens; log only token fingerprints/hashes
- [ ] Throttle invalid-token brute-force attempts by IP
- [ ] Enforce webhook SSRF guards (`https` only, private CIDR deny, timeouts)
- [ ] Define webhook retry policy: retry network/429/5xx, avoid most 4xx retries
- [ ] Support token rotation/revocation with immediate old-token invalidation
- [ ] Cap `meta_json` size and validate signal bounds/body size
- [ ] Redact token values in all HTTP/access logs (token appears in URL path)
- [ ] Redact email/webhook targets in logs/traces and define log retention policy

### Product/ops policy details
- [ ] Decide check delete semantics (soft delete default; hard delete admin-only)
- [ ] Define interval/grace edit behavior (immediate recompute policy)
- [ ] Define backlog backpressure policy (drop/defer warning traffic first)
- [ ] Define notification payload/template versioning (`schema_version`)
- [ ] Define SLOs (ingest latency, overdue detection lag, alert dispatch latency)
- [ ] Document provider partial-outage policy (channel isolation)
- [ ] Add retention/export policies early (events cleanup, customer export path)
- [ ] Create operator runbooks (provider outage, queue lag, migration rollback)
- [ ] Add circuit breaker policy per webhook channel to prevent thrashing under persistent failure
- [ ] Capture bounded request provenance (IP/User-Agent) in event metadata for abuse forensics
- [ ] Add restore-drill validation with API/worker running against restored snapshots
- [ ] Define at-least-once idempotency key storage policy (TTL + cleanup) to avoid unbounded growth

### Docker/runtime specifics
- [ ] Pin Docker base image digests for reproducible builds
- [ ] Decide/publish multi-arch support (`amd64`/`arm64`)
- [ ] Ensure graceful SIGTERM drain in API and worker
- [ ] Enforce UTC across app/DB/containers
- [ ] Ensure single migration runner per deployment
- [ ] Add resource limits in compose for early pressure testing
- [ ] Set `Cache-Control: no-store` on ping endpoints to prevent intermediary caching/replay

## Tooling Baseline
- `mise` for toolchain version pinning
- `just` for workflows/tasks
- `golangci-lint` + `go vet` + `go test`
- `testcontainers-go` for Postgres integration tests
- `oapi-codegen` + `sqlc` generation checks in CI

## Status
- Plan authored and caveats folded in.
- Implementation not yet started.
