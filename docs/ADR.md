# Architecture Decision Records (ADR)

**Project**: PipelineBuilder  
**Version**: 1.0  
**Date**: 2026-07-07  

---

## Table of Contents

1. [ADR-001 — Channel-Based Pipeline Over a Distributed Job Queue](#adr-001--channel-based-pipeline-over-a-distributed-job-queue)
2. [ADR-002 — Single Binary Monolith Over Microservices](#adr-002--single-binary-monolith-over-microservices)
3. [ADR-003 — Raw SQL with pgx Over an ORM](#adr-003--raw-sql-with-pgx-over-an-orm)
4. [ADR-004 — JSONB for JobSpec Storage](#adr-004--jsonb-for-jobspec-storage)
5. [ADR-005 — Interface-Driven Layered Architecture](#adr-005--interface-driven-layered-architecture)
6. [ADR-006 — Per-Job Context with In-Memory Cancel Map](#adr-006--per-job-context-with-in-memory-cancel-map)
7. [ADR-007 — Worker Pool Fan-Out with WaitGroup](#adr-007--worker-pool-fan-out-with-waitgroup)
8. [ADR-008 — sync.RWMutex vs sync.Mutex by Access Pattern](#adr-008--syncrwmutex-vs-syncmutex-by-access-pattern)
9. [ADR-009 — Polling Over WebSocket or SSE for Live Progress](#adr-009--polling-over-websocket-or-sse-for-live-progress)
10. [ADR-010 — nginx Reverse Proxy Instead of Baking API URL](#adr-010--nginx-reverse-proxy-instead-of-baking-api-url)
11. [ADR-011 — API Key Auth Over JWT or OAuth](#adr-011--api-key-auth-over-jwt-or-oauth)
12. [ADR-012 — HTTP Retry with Exponential Backoff in Ingestion Readers](#adr-012--http-retry-with-exponential-backoff-in-ingestion-readers)

---

## ADR-001 — Channel-Based Pipeline Over a Distributed Job Queue

**Date**: 2026-07-07  
**Status**: Accepted

### Context

The pipeline engine needs to pass records between stages — ingestion, validation, transformation, aggregation, and export — with backpressure support and clean cancellation. Two approaches were considered:

- **In-process Go channels**: buffered channels connecting stages, all running inside one process.
- **Distributed job queue** (e.g. Redis Streams, RabbitMQ): each stage publishes to and consumes from an external broker.

### Decision

Use buffered Go channels as the transport between every pipeline stage.

Each `Start*` function accepts an input channel and returns an output channel. The runner wires them:

```
rawCh → validCh → transformedCh → exportCh
               └→ errorCh
```

Stages are connected before any goroutine processes a record. A slow stage applies natural backpressure to the previous one because `chan <- record` blocks when the buffer is full.

### Consequences

**Positive**
- Zero external dependencies for pipeline execution — no broker to run, configure, or monitor.
- Backpressure is automatic and free: a full channel blocks the sender without any polling or delay logic.
- Cancellation is propagated by passing the same `context.Context` into every stage — no inter-process signal needed.
- Channel close is the natural "no more data" signal; every stage's `for range ch` loop exits automatically when the upstream closes.
- The entire pipeline topology is readable in one place (`runner.go`).

**Negative**
- All stages must run in the same process. If the server restarts mid-job, the in-flight records are lost (the job is recoverable from DB but the records are not replayed).
- Cannot scale pipeline workers horizontally across machines without replacing channels with a broker.
- A pipeline goroutine leak (e.g. a stage that never exits) is harder to detect than a queue consumer that stops consuming.

---

## ADR-002 — Single Binary Monolith Over Microservices

**Date**: 2026-07-07  
**Status**: Accepted

### Context

The system has three logical concerns: HTTP API, pipeline execution engine, and data persistence. These could be deployed as separate services (an API service, a worker service, a scheduler) or as a single binary.

### Decision

Compile and deploy everything as a single Go binary. The HTTP server and the pipeline runner share the same process and communicate through Go interfaces and channels — not network calls.

### Consequences

**Positive**
- Deployment is a single Docker image with no inter-service networking.
- The job's `context.CancelFunc` can be stored directly in memory — no distributed locking or message passing needed to cancel a running job.
- End-to-end latency for a request is in microseconds — no serialization or network hop between the API and the runner.
- The test surface is a single binary. Every layer is testable with Go's standard `testing` package and interface mocks.

**Negative**
- The API server and the pipeline runner share a process. A goroutine leak in the runner can degrade HTTP latency.
- Horizontal scaling means running multiple full copies of the binary. Without external coordination, two instances could run the same job concurrently (mitigated by the DB being the source of truth for job state).
- If pipeline workloads become CPU-heavy enough to interfere with HTTP handler latency, the only fix is splitting into separate processes.

---

## ADR-003 — Raw SQL with pgx Over an ORM

**Date**: 2026-07-07  
**Status**: Accepted

### Context

The repository layer executes a small, well-defined set of queries: insert job, get job, list jobs, update status, delete job, save error, get errors, save aggregation, get aggregation, count by status. Options considered:

- **ORM** (GORM, ent): generates SQL from struct annotations.
- **Query builder** (sqlx, squirrel): parameterized helpers with a lighter API.
- **Raw SQL** with `database/sql` and the pgx stdlib driver.

### Decision

Write all queries by hand using positional `$1/$2/...` parameters with the `pgx/v5/stdlib` driver registered under `database/sql`.

### Consequences

**Positive**
- Every SQL statement is visible and auditable directly in the source file — no ORM magic to reverse-engineer.
- No struct tags polluting the model types. `models/` stays free of persistence concerns.
- The repository interface (`PipelineRepo`) provides the same testability that an ORM abstraction would — tests inject a `MockPipelineRepository` that implements the same interface.
- Query performance is predictable; no ORM-generated N+1 queries.
- `INSERT ... ON CONFLICT DO UPDATE` (used for aggregation upserts) is idiomatic SQL and trivial to write; expressing it via an ORM is not.

**Negative**
- Schema changes require manually updating both the `CREATE TABLE` statement and every affected query. An ORM's migration system would track this automatically.
- No compile-time query validation. A typo in a column name surfaces at runtime.
- Boilerplate for scanning rows into structs is repetitive.

---

## ADR-004 — JSONB for JobSpec Storage

**Date**: 2026-07-07  
**Status**: Accepted

### Context

`JobSpec` is a structured Go type containing sources, export config, and concurrency config. Two storage approaches were considered:

- **Normalised columns**: individual columns for each field of `JobSpec` (source URLs spread into a `job_sources` join table, export path as a TEXT column, etc.).
- **JSONB blob**: store the entire `JobSpec` as a serialised JSON blob in one column.

### Decision

Store `JobSpec` as a JSONB column on the `jobs` table. The Go repository layer marshals and unmarshals it on every read and write.

```sql
spec  JSONB NOT NULL DEFAULT '{}'
```

### Consequences

**Positive**
- The `jobs` table schema does not change as `JobSpec` gains or loses fields. Only the Go struct needs updating.
- A variable number of sources per job does not require a `job_sources` join table or a `LEFT JOIN` on every list query.
- PostgreSQL's JSONB operators allow future ad-hoc querying of spec fields (e.g., find all jobs using a specific URL) without a schema migration.

**Negative**
- There is no relational constraint on spec contents — the DB accepts any valid JSON. All spec validation must happen in the application layer (which it does, in `ValidateJobSpec`).
- `spec` fields cannot be indexed individually without a generated column or GIN index.
- Debugging raw DB rows requires parsing JSON mentally rather than reading typed columns.

---

## ADR-005 — Interface-Driven Layered Architecture

**Date**: 2026-07-07  
**Status**: Accepted

### Context

The system has three behavioural layers: HTTP handling (controller), business logic (service), and data access (repository). Dependencies between them needed to be managed so that each layer could be tested in isolation.

### Decision

Every cross-layer dependency is expressed as a Go interface. Concrete types are only named at the composition root (`main.go`).

```
Controller depends on → PipelineService (interface)
Service    depends on → PipelineRepo   (interface)
Service    depends on → JobRunner      (interface)
Runner     depends on → JobStore       (interface)
```

`main.go` constructs all concrete types and passes them down:

```go
repo   := repository.NewPipelineRepository(db)   // concrete
runner := pipelines.NewRunner(repo)               // concrete
svc    := service.NewPipelineService(repo, runner) // concrete, injected
ctrl   := controller.NewPipelineController(svc)   // concrete, injected
```

### Consequences

**Positive**
- Any layer can be tested by injecting a mock that satisfies the interface. Unit tests for service logic use `MockPipelineRepository` without touching PostgreSQL.
- Swapping a concrete implementation (e.g., replacing the file exporter with S3) requires only changing `main.go` and implementing the interface — no changes to callers.
- The dependency graph is acyclic and readable: controller → service → repository.

**Negative**
- More files and types than a direct-call approach. Each layer has a `types.go` file defining its interface.
- Interface changes (adding a method) require updating all mock implementations in the test packages.

---

## ADR-006 — Per-Job Context with In-Memory Cancel Map

**Date**: 2026-07-07  
**Status**: Accepted

### Context

Running jobs need to be stoppable on demand (cancel endpoint) and on deletion (delete endpoint). The question was where to store the signal mechanism and how to deliver it to a running goroutine.

### Decision

For each job that starts, create a `context.WithCancel(context.Background())` and store the returned `CancelFunc` in a `map[string]context.CancelFunc` inside `PipelineService`, keyed by job ID. The `ctx` is passed into `Runner.Run` and from there into every pipeline stage.

```go
type PipelineService struct {
    cancels map[string]context.CancelFunc
    mu      sync.Mutex
    ...
}
```

To cancel a job: look up the `CancelFunc` by ID and call it. The context's `Done()` channel closes, every stage's `select` unblocks, and goroutines exit. To signal the map is guarded: `sync.Mutex` wraps all reads, writes, and deletes.

### Consequences

**Positive**
- Cancellation is instantaneous — no polling, no DB round-trip.
- The `ctx` carries the cancellation signal through the entire call stack without any shared variable.
- When a job's goroutine exits naturally, it removes its own entry from the map — no stale entries.
- The cancel map also serves as the "is this job currently running?" check for the progress endpoint.

**Negative**
- In-memory only. If the server restarts, all `CancelFunc`s are lost. Jobs that were running at restart time stay in `status=running` in the DB forever unless a startup reconciliation pass is added.
- The map + mutex pattern requires careful lock discipline. A missed `Unlock` would deadlock the server.

---

## ADR-007 — Worker Pool Fan-Out with WaitGroup

**Date**: 2026-07-07  
**Status**: Accepted

### Context

Validation and transformation are the most CPU-intensive stages and are the natural candidates for parallelism. The design choices were:

- **Single goroutine** per stage: simple, but leaves CPU cores idle.
- **Fixed thread pool** managed manually: complex lifecycle management.
- **Goroutine-per-record**: no back-pressure, unbounded goroutine count.
- **Worker pool**: a fixed number of goroutines all reading from the same input channel.

### Decision

Each stage (`StartValidation`, `StartTransformation`) spawns exactly `numWorkers` goroutines. All workers read from the same shared input channel. Go's channel semantics guarantee that each record is received by exactly one worker.

```go
var wg sync.WaitGroup
for i := 0; i < numWorkers; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for record := range in {
            // process
        }
    }()
}
go func() { wg.Wait(); close(out) }()
```

When the input channel closes, all workers exit their `for range` loop, `wg.Wait()` unblocks, and the output channel is closed — propagating the "done" signal to the next stage automatically.

### Consequences

**Positive**
- Worker count is a tunable parameter per job, so operators can match it to the machine's core count and workload.
- Channel semantics eliminate any shared record state between workers — no explicit partitioning or locking needed.
- The `wg.Wait()` + `close(out)` pattern guarantees the output channel closes exactly once and only after every worker has finished.
- Backpressure flows naturally: if the output channel is full, workers block on send until the next stage drains it.

**Negative**
- If `numWorkers` is set too high relative to available CPU, goroutine scheduling overhead can hurt throughput instead of helping it.
- Workers within a stage share the same `in` channel — one very slow record can stall that worker's goroutine, reducing effective parallelism.

---

## ADR-008 — sync.RWMutex vs sync.Mutex by Access Pattern

**Date**: 2026-07-07  
**Status**: Accepted

### Context

Several shared data structures are accessed concurrently and need synchronisation. The two options were `sync.Mutex` (exclusive lock for all operations) and `sync.RWMutex` (shared read lock, exclusive write lock). Choosing the wrong one wastes either correctness or throughput.

### Decision

Apply the lock type that matches the read-to-write ratio of each data structure:

| Data structure | Location | Lock type | Reason |
|---|---|---|---|
| `cancels map[string]CancelFunc` | `PipelineService` | `sync.Mutex` | Every access is a write (insert on start, delete on stop). No pure reads exist. |
| `trackers map[string]*Tracker` | `Runner` | `sync.RWMutex` | `GetTracker` is called on every metrics poll (frequent read). Write only on job start and end (rare). |
| `Tracker` internal fields | `metrics.Tracker` | `sync.RWMutex` | `Snapshot()` is called by every progress request (frequent read). `SetFinished()` is called once per job (rare write). |
| `clients map[string]*entry` | Rate limiter | `sync.Mutex` | Every access updates `lastSeen` — a write. No pure reads exist. |

### Consequences

**Positive**
- `GetTracker` and `Snapshot()` — the two hot paths called on every metrics poll — are non-blocking to each other. Multiple concurrent metric requests are served simultaneously without serialisation.
- `PipelineService.cancels` correctly uses an exclusive lock since the lookup-then-delete pattern in `CancelPipeline` is not safe to split across read and write locks.
- The decision is documented per site, so future contributors understand why each lock type was chosen rather than assuming `RWMutex` is always better.

**Negative**
- `sync.RWMutex` has higher overhead than `sync.Mutex` for workloads with infrequent reads. The benefit only appears when concurrent readers are common — which is the case for tracker snapshot reads.

---

## ADR-009 — Polling Over WebSocket or SSE for Live Progress

**Date**: 2026-07-07  
**Status**: Accepted

### Context

The detail page needs to display live progress metrics while a job is running. Three delivery mechanisms were considered:

- **Polling**: the frontend calls `GET /progress` on a timer.
- **Server-Sent Events (SSE)**: a persistent HTTP connection where the server pushes updates.
- **WebSocket**: a bidirectional persistent connection.

### Decision

Use client-side polling. The `useProgress` hook calls `GET /api/v1/pipelines/{id}/progress` every 2 seconds while the job is active (`status === "pending" || "running"`). Polling stops automatically when the status transitions to a terminal state.

```typescript
const { metrics, history } = useProgress(jobId, isActive, 2000)
```

### Consequences

**Positive**
- The `/progress` endpoint is a plain stateless HTTP GET. No connection state to manage on the server.
- Works transparently through nginx proxying and any load balancer without special configuration.
- The 2-second interval is fast enough to feel live for a data pipeline dashboard.
- Polling stops when the job completes — no idle connections to clean up.
- If the server restarts mid-job, the next poll simply reconnects — no reconnect logic needed.

**Negative**
- Each poll is an independent HTTP request with full headers. SSE would be more efficient for sustained high-frequency updates.
- Updates are delayed by up to 2 seconds after a status change. For a pipeline job this is acceptable; for a financial ticker it would not be.
- Polling continues even during brief network interruptions, showing stale data rather than a disconnected state.

---

## ADR-010 — nginx Reverse Proxy Instead of Baking API URL

**Date**: 2026-07-07  
**Status**: Accepted

### Context

The React frontend needs to call the Go backend. The two approaches for telling the frontend where the backend is:

- **Bake the URL at build time**: set `VITE_API_URL=http://backend:8080` and let Vite embed it into the JS bundle.
- **nginx reverse proxy**: leave `VITE_API_URL` empty and configure nginx to forward all `/api/*` requests to the backend container.

### Decision

Leave `VITE_API_URL` empty. Configure nginx in the frontend container to proxy:

```nginx
location /api/ {
    proxy_pass http://backend:8080/api/;
}
```

The frontend's `api/client.ts` prepends `VITE_API_URL ?? ''` — an empty string — so all fetch calls are relative URLs (`/api/v1/pipelines`). nginx resolves them at the network level.

### Consequences

**Positive**
- The frontend Docker image is backend-URL-agnostic. It can be deployed in any environment without rebuilding — only the nginx config changes.
- Requests from the browser arrive at the frontend's origin. No CORS headers are needed at the nginx level (same-origin request). CORS on the Go server covers only direct API access from non-browser clients.
- `VITE_API_KEY` is the only build-time argument needed for the frontend image.

**Negative**
- An extra nginx hop adds a small amount of latency (sub-millisecond on the same Docker network).
- Local development without Docker requires either running nginx locally, using a Vite proxy config (`vite.config.ts → server.proxy`), or setting `VITE_API_URL` explicitly.

---

## ADR-011 — API Key Auth Over JWT or OAuth

**Date**: 2026-07-07  
**Status**: Accepted

### Context

Mutating API endpoints (create, cancel, delete) need to be protected from unauthorised access. Options considered:

- **API Key** (`X-API-Key` header with a shared secret).
- **JWT (JSON Web Token)**: stateless signed tokens with expiry and claims.
- **OAuth 2.0**: delegated authorisation with an external identity provider.

### Decision

Use a single shared API key stored in the `API_KEY` environment variable. Incoming requests are validated by comparing the `X-API-Key` header value using `crypto/subtle.ConstantTimeCompare`.

The server refuses to start if `API_KEY` is unset.

### Consequences

**Positive**
- No token issuance infrastructure, expiry handling, or public-key management.
- `crypto/subtle.ConstantTimeCompare` eliminates timing-based key enumeration — the check takes the same time regardless of how many bytes match.
- Operationally trivial: rotate the key by updating the environment variable and restarting the server.
- Appropriate for a single-operator, single-team deployment where there is no per-user access control requirement.

**Negative**
- The key is a bearer credential. Anyone who obtains it has full write access. Key rotation requires a server restart.
- No per-user identity. All write operations look the same in logs.
- Does not scale to multi-tenant or fine-grained permission scenarios without a complete replacement.

---

## ADR-012 — HTTP Retry with Exponential Backoff in Ingestion Readers

**Date**: 2026-07-07  
**Status**: Accepted

### Context

The three ingestion readers (`CSVReader`, `JSONReader`, `APIReader`) fetch data from external HTTP URLs. Initially all three used `http.DefaultClient.Do` with a single attempt and no timeout. Two problems existed:

1. **Transient failures** (DNS blips, momentary 503s) caused the entire source to be skipped permanently, even though a second attempt milliseconds later would have succeeded.
2. **`http.DefaultClient` has no timeout**. A remote server that accepts the connection but never sends a response would block the ingestion goroutine for the lifetime of the job context.

### Decision

Extract a shared `fetchWithRetry(ctx, url)` helper in `ingestion/fetch.go`. Replace all three readers' direct `http.DefaultClient.Do` calls with this helper.

```
attempt 0 — immediate
  network error or 5xx → wait 1s
attempt 1
  network error or 5xx → wait 2s
attempt 2
  still failing → return error (source is skipped, logged in manager.go)

4xx (401, 404, etc.) → return immediately, no retry
ctx cancelled during backoff → return ctx.Err() immediately
```

Use a package-level `http.Client{Timeout: 15s}` instead of `http.DefaultClient`.

### Consequences

**Positive**
- Transient network errors or momentary server-side 5xx responses are survived transparently.
- The 15-second per-request timeout bounds the worst-case hang per attempt. With 3 attempts and backoff, the maximum blocking time is bounded: `15s + 1s + 15s + 2s + 15s = 48s` per source.
- Context cancellation during any backoff sleep unblocks immediately — the retry loop does not ignore job cancellation.
- 4xx responses are not retried — a 404 or 401 will not succeed on retry and failing fast is the correct behaviour.
- All three readers share one implementation — the retry logic cannot diverge between them.

**Negative**
- A source with a persistent 5xx will delay that ingestion goroutine by `1s + 2s = 3s` of backoff before giving up. This delays the closure of `rawCh` if that source is the slowest.
- Retrying on all network errors without inspecting the error type means a non-retryable client-side error (e.g., bad TLS cert) is still retried twice unnecessarily.

---

*These records reflect the state of the codebase at v1.0. Future decisions that meaningfully change the architecture should be appended as new ADRs rather than modifying existing ones, so the evolution of the design remains traceable.*
