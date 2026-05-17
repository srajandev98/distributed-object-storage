# Distributed Object Storage Plan

## 1) Product Goal

Build an open-source, production-grade distributed object storage service inspired by S3.

This plan is split into:

- **MVP track**: minimal reliable product for real usage
- **Post-MVP track**: advanced capabilities toward S3-comparable breadth

## 2) Current Status

- Upload, download, and presign endpoints are implemented.
- Object version metadata is stored in PostgreSQL.
- Durable replication jobs exist in PostgreSQL.
- Replication retry and terminal failure handling are implemented.
- Replication state transitions are guarded in SQL.
- Unit tests exist for service upload and replication failure paths.
- Object key and bucket path validation is implemented for upload/download/presign routes.
- Streaming upload and request size limits are still pending.

## 3) MVP Scope (What We Must Finish First)

An MVP is complete when the system includes:

- Bucket/object upload and download
- Object versioning
- Presigned download URLs
- Durable metadata and durable replication jobs
- Replication retry with status tracking
- ~~Request validation and safety controls~~
- AuthN/AuthZ with API keys
- Basic observability and health checks
- Dockerized local deployment
- Core tests + operator/developer documentation

## 4) MVP Phases

### Phase 1: Foundation and Data Model

- Add DB migrations for `buckets`, `objects`, `replicas`, `users`, `api_keys`, `audit_logs`.
- Add indexes and constraints (including one latest version per `(bucket, object_key)`).
- ~~Add config loading with validation and startup checks.~~
- ~~Add replication idempotency constraints and job polling indexes.~~

**Progress**
- [x] Config loading with validation and startup checks
- [x] Replication idempotency/job polling indexes
- [x] DB migrations
- [x] Object latest-version constraints

### Phase 2: Correctness and Safety

- ~~Add strict object key validation to prevent path traversal.~~
- Replace full-memory upload reads with streaming write + checksum.
- Add upload request size limits.
- ~~Ensure replication enqueue occurs only after metadata commit.~~
- Add graceful shutdown for HTTP server and workers.

**Progress**
- [x] Object key validation + enforcement on upload/download/presign
- [x] Replication enqueue after metadata commit (durable job model)
- [ ] Streaming upload path
- [ ] Request size limits
- [ ] Graceful shutdown

### Phase 3: API Contract and Auth

- Introduce versioned API paths (`/v1/...`).
- Standardize JSON response and error schema.
- Add API key authentication and scoped permissions.
- Add bucket CRUD endpoints.
- Harden presign generation/verification behavior.

**Progress**
- [ ] Not started

### Phase 4: Replication Reliability

- ~~Replace in-memory queue with durable replication jobs.~~
- ~~Add retry with backoff and terminal failure state.~~
- ~~Track replication status per node.~~
- Add repair/replay job support.

**Progress**
- [x] Durable replication jobs
- [x] Retry/backoff + terminal failure
- [x] Per-node replica status recording
- [ ] Repair/replay support

### Phase 5: Observability and Operations

- Add structured logging with request IDs.
- Add Prometheus metrics (latency, errors, replication lag, queue depth).
- Add `/healthz` and `/readyz`.
- Add Dockerfile and `docker-compose` for local stack.

**Progress**
- [ ] Not started

### Phase 6: Testing and MVP Release Readiness

- Add unit tests for signing, validation, and version selection.
- Add integration tests for upload/download/replication flows.
- Add concurrency and large-object smoke tests.
- Add security checklist and basic threat model.
- Add MVP docs: architecture, API, deployment, contribution guide.

**Progress**
- [x] Unit tests for service upload and replication failure flows
- [ ] Unit tests for signing/version selection coverage
- [ ] Integration tests
- [ ] Concurrency/smoke tests
- [ ] Security checklist/threat model
- [ ] Final MVP docs set

## 5) Immediate Next Work (MVP)

1. Implement streaming upload write path with checksum calculation.
2. Add upload request size limits.
3. Update tests for streaming + limits.
4. Mark Phase 2 fully complete.

## 6) Post-MVP Roadmap (Toward S3-Comparable Scope)

### A. Core Object API Expansion

- `HEAD Object`, `DELETE Object`, `CopyObject`
- `ListObjectsV2` (prefix, delimiter, pagination tokens)
- Batch delete (`DeleteObjects`)
- Conditional requests (`ETag`, `If-Match`, `If-None-Match`)
- Range reads (`Range` header)

### B. Multipart Upload

- Create multipart upload
- Upload part
- List parts
- Complete multipart upload
- Abort multipart upload

### C. Bucket Control Plane

- Bucket list/create/delete
- Bucket policy model
- CORS and lifecycle configuration
- Versioning controls

### D. Security and Identity

- IAM-style policy evaluation
- Short-lived credentials flow
- Signature V4 compatibility
- At-rest encryption abstractions
- Audit logging across control/data plane

### E. Durability and Repair

- Background checksum scrub
- Anti-entropy repair flows
- Configurable replication policies
- Disaster recovery procedures

### F. Scale, Ops, and Compatibility

- High-cardinality metadata/listing tuning
- SLOs, alerting, runbooks
- SDK/CLI compatibility test matrix
- Upgrade/backward-compatibility policy

## 7) Production-Grade Done Criteria

- Reliability targets validated with failure drills
- Security model reviewed and enforced
- SLOs met under representative load
- Observability and operational runbooks complete
- Compatibility claims backed by automated conformance tests
