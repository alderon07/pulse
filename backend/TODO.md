# TODO / Progress Tracker

## Overall Status
- Progress: `35%`
- Current phase: `Backend foundation + management API scaffold`

## Completed
- [x] Create backend project scaffold (`cmd/api`, `cmd/worker`, `internal/*`)
- [x] Add migrations baseline (`users`, `checks`, `events`, `incidents`, `alert_channels`, `alert_deliveries`, `ping_idempotency`)
- [x] Implement ping ingestion endpoints (`GET/POST /v1/ping/{token}`)
- [x] Implement deprecated ping alias (`/ping/{token}`) with deprecation headers
- [x] Add no-store cache headers for ping endpoints
- [x] Add token redaction in request logs
- [x] Implement overdue worker transitions (`up -> late -> down`) and incident open/resolve flow
- [x] Implement outbox claim/send status lifecycle (`queued/sending/sent/failed/dead_letter`)
- [x] Implement webhook sender guards (HTTPS-only, SSRF/private-range checks, timeout, HMAC option)
- [x] Add circuit breaker for webhook channels
- [x] Add idempotency key support + cleanup path
- [x] Add JWT middleware (`HS256`) and user-scoped `/api/v1` endpoints
- [x] Implement check management handlers (`create/list/get/patch/pause/resume/rotate-token/events`)
- [x] Implement alert channel handlers (`create/list/patch`)
- [x] Add OpenAPI v1 scaffold and update it for implemented routes
- [x] Add Dockerfiles + docker-compose + local env template
- [x] Add root/project `.gitignore`
- [x] Add minimal unit tests (`status logic`, `ping parser precedence/validation`)

## In Progress
- [ ] Expand automated tests from unit-only to integration coverage

## Next Up (High Priority)
- [ ] Replace HS256 JWT path with Clerk/JWKS verification flow
- [ ] Add integration tests (`testcontainers-go`) for:
  - [ ] concurrent pings and row-lock behavior
  - [ ] single-open-incident invariant
  - [ ] overdue scanner and repeat alert logic
  - [ ] delivery retry/backoff/dead-letter behavior
- [ ] Implement real email provider integration (Postmark/SendGrid adapter)
- [ ] Add warning drift evaluation + minimum-sample gating in write path
- [ ] Add management API pagination upgrade to cursor-based responses
- [ ] Add check soft-delete endpoint and enforce delete semantics

## Next Up (Medium Priority)
- [ ] Add audit events for sensitive actions (pause/resume, rotate-token, channel edits)
- [ ] Add brute-force invalid-token telemetry + alerting thresholds
- [ ] Add retention job for old `events` and idempotency cleanup scheduling
- [ ] Add queue backpressure policy (shed warning events before paging alerts)
- [ ] Add OpenAPI validation/codegen checks in CI

## Ops / Reliability Checklist
- [ ] Define SLOs for ping ingest latency and alert dispatch latency
- [ ] Add Prometheus metrics + OpenTelemetry traces
- [ ] Add runbooks (provider outage, backlog spike, migration rollback)
- [ ] Run restore drill against a backup snapshot with api+worker online

## Notes
- API currently requires `JWT_HS256_SECRET` for `/api/v1/*`.
- `go test ./...` currently passes.
- Worker and API are both runnable in current scaffold.
