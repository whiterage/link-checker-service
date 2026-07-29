# Link Checker Service

A Go service that takes a list of links, checks their availability asynchronously through a
worker pool, and generates a PDF report for any of the sets you've submitted before.

## How it's built

- Go 1.25, standard library plus `gofpdf` for the PDF.
- In-memory repository with a thread-safe `links_num` counter.
- Job queue and worker pool: `Service` pushes tasks onto a channel, `WorkerPool` drains it and
  updates statuses.
- HTTP API on `net/http` with hand-written handlers — no framework.
- Graceful shutdown: catches `SIGINT`/`SIGTERM`, stops accepting new HTTP requests, then waits
  for the workers to drain the queue before exiting.

## Running it

```bash
go run ./cmd/server
```

Listens on `http://localhost:8080` by default.

## API

### `POST /links`

```json
// request
{ "links": ["google.com", "malformedlink.gg"] }

// response
{ "links": { "google.com": "pending", "malformedlink.gg": "pending" }, "links_num": 1, "status": "pending" }
```

### `GET /links/{links_num}`

Returns the current status of every link in that set.

```json
{ "links": { "google.com": "available", "malformedlink.gg": "not_available" }, "links_num": 1, "status": "done" }
```

### `POST /links_list`

```json
{ "links_list": [1, 2] }
```

Returns `application/pdf` — a table of links, statuses, and check timestamps for every set
requested.

## Try it

```bash
curl -i -X POST http://localhost:8080/links \
  -H "Content-Type: application/json" \
  -d '{"links":["google.com","malformedlink.gg"]}'

curl -i http://localhost:8080/links/1

curl -X POST http://localhost:8080/links_list \
  -H "Content-Type: application/json" \
  -d '{"links_list":[1]}' \
  -o report.pdf
```

## Details worth knowing

- **Worker pool size** is set in `cmd/server/main.go` — 4 by default.
- **URL normalization** adds `https://` where it's missing and rejects obviously malformed input.
- **Persistence**: tasks and their statuses are written to `storage/tasks.json` (override with
  `TASK_STORAGE_PATH`). On restart, any tasks that were still pending get requeued automatically.
- **Graceful shutdown**: on `SIGINT`/`SIGTERM`, the server stops taking new HTTP requests first,
  then waits for the worker pool to drain the queue — if that takes too long, the workers are
  force-cancelled instead of hanging the shutdown.

## Tests

```bash
go test ./...
```

Covers URL normalization and the repository layer, alongside the helper functions.
