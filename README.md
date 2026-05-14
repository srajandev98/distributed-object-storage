# Distributed Object Storage

`distributed-object-storage` is an open-source object storage service inspired by S3.  
It is currently in active MVP development, with a working prototype for upload, download, presigned URLs, object versioning, and background replication.

## Current Capabilities

- Upload objects to a bucket/object path
- Download latest object version via presigned URL
- Store object metadata in PostgreSQL
- Generate object version IDs
- Persist replication jobs in PostgreSQL
- Replicate object files to secondary storage nodes with retry and terminal failure
- Idempotent replica writes per `(object_id, node_name)`
- Unit tests for service and replication failure handling

## Architecture (Current)

1. Client uploads object to `/upload/{bucket}/{objectKey}`.
2. Service writes object file to primary storage node (`storage/node1`).
3. Service stores metadata in `objects` and enqueues a durable replication job in `replication_jobs` in the same DB transaction.
4. Background worker claims pending jobs and replicates files to secondary nodes (`storage/node2`, `storage/node3`).
5. Client requests presigned URL from `/presign/{bucket}/{objectKey}` and uses it on `/download/...`.

## Replication Job State Machine

`pending -> running -> completed`

`pending -> running -> pending` (retry path with backoff)

`pending -> running -> failed` (after max attempts)

Notes:

- Worker claims jobs with `FOR UPDATE SKIP LOCKED`.
- State transitions are guarded in SQL (`WHERE status = 'running'` on completion/failure updates).
- Retry delay uses quadratic backoff: `attempt^2` seconds.

## Project Structure

- `cmd/server`: application entrypoint and dependency wiring
- `internal/config`: environment configuration loading
- `internal/httpapi`: HTTP handlers and route registration
- `internal/service`: core use-cases
- `internal/repository`: PostgreSQL data access
- `internal/storage`: local filesystem storage adapter
- `internal/replication`: background replication worker

## Prerequisites

- Go `1.24+`
- PostgreSQL

## Configuration

Create `.env` in repo root:

```env
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=object_storage
DB_HOST=localhost
DB_PORT=5432
DB_SSLMODE=disable
APP_SECRET=replace_with_strong_secret
```

## Database Notes

This project currently expects existing tables like `objects` and `replicas`.  
`replication_jobs` is auto-created at startup.
- A unique index is also created for replica idempotency on `(object_id, node_name)`.

Planned improvement: move all schema creation to migration files under a dedicated `migrations/` directory.

## Run Locally

```bash
go run ./cmd/server
```

Default server address: `http://localhost:8080`

## API (Current)

### Upload

```bash
curl -X POST \
  --data-binary @./sample.txt \
  -H "Content-Type: text/plain" \
  http://localhost:8080/upload/my-bucket/docs/sample.txt
```

### Create Presigned URL

```bash
curl http://localhost:8080/presign/my-bucket/docs/sample.txt
```

Response:

```json
{"url":"/download/my-bucket/docs/sample.txt?expires=...&signature=..."}
```

### Download

Use the returned `url` directly with host prefix:

```bash
curl "http://localhost:8080/download/my-bucket/docs/sample.txt?expires=...&signature=..."
```

## MVP Roadmap

See [PLAN.md](./PLAN.md) for phased MVP milestones.

Near-term priorities:

1. Object key/path sanitization
2. Streaming uploads (avoid full in-memory object reads)
3. Durable schema migrations
4. API key auth and scoped access
5. Observability and test coverage

## Contributing

Contributions are welcome.  
Before opening a PR:

1. Keep changes scoped and focused.
2. Run formatting and tests:

```bash
gofmt -w ./...
GOCACHE=/private/tmp/go-build go test ./...
```

3. Document behavior changes in README/PLAN when applicable.

## License

MIT License. See [LICENSE](./LICENSE).
