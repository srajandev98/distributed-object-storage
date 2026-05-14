# Distributed Object Storage MVP Plan

This project currently has a working prototype, but it is not yet MVP-ready for a production-grade open-source release.

## Current Status

- Core happy path exists: upload, presign, download.
- Version metadata is stored in Postgres.
- Local multi-node directories simulate storage nodes.
- Replication jobs are durable in Postgres with retry and terminal failure states.
- Replication state transitions are guarded at the DB update layer.
- Unit tests exist for service upload paths and replication failure behavior.
- No migrations yet; object key validation and streaming uploads are still pending.

## MVP Goal

Ship a single-binary service that supports:

- ~~Bucket and object upload/download~~
- ~~Object versioning~~
- ~~Presigned GET URLs~~
- ~~Asynchronous replication with retry and status tracking~~
- ~~Durable metadata in Postgres~~
- Authentication/authorization (API keys)
- Metrics, logs, health checks
- Dockerized local deployment and clear docs

## Phased Plan

### Phase 1: Foundation and Data Model

- Add DB migrations for `buckets`, `objects`, `replicas`, `users`, `api_keys`, `audit_logs`.
- ~~Add indexes and constraints for replication idempotency and job polling~~.
- Add indexes and constraints (including one latest version per `(bucket, object_key)`).
- ~~Add config loading with validation and startup checks~~.

### Phase 2: Correctness and Safety

- Add strict object key validation to prevent path traversal.
- Stream upload data to disk; avoid full in-memory reads.
- Add request size limits.
- ~~Enqueue replication only after metadata commit (outbox/job-table approach)~~.
- Add graceful shutdown for HTTP server and workers.

### Phase 3: API Contract and Auth

- Introduce versioned API paths (`/v1/...`).
- Standardize JSON responses and error format.
- Add API key authentication and scoped permissions.
- Add bucket CRUD endpoints.
- Harden presign generation/verification.

### Phase 4: Replication Reliability

- ~~Replace in-memory queue with durable replication jobs~~.
- ~~Add retry with backoff and terminal failure state~~.
- ~~Track replication status per node~~.
- Add repair/replay job support.

### Phase 5: Observability and Operations

- Add structured logging with request IDs.
- Add Prometheus metrics (latency, errors, replication lag, queue depth).
- Add `/healthz` and `/readyz`.
- Add Dockerfile and `docker-compose` for local stack.

### Phase 6: Testing and MVP Release Readiness

- ~~Unit tests for service and replication failure flows~~.
- Unit tests for signing, validation, version selection, and path safety.
- Integration tests for upload/download/replication flows.
- Concurrency and large-object smoke tests.
- Security checklist and basic threat model.
- MVP docs: architecture, API, deployment, and contribution guide.

## Priority Backlog (Do First)

1. Path/key sanitization and streaming uploads
2. Migrations and schema constraints
3. Replication repair/replay and observability around failed jobs
4. API key auth
5. Tests, metrics, and Dockerized local deployment
