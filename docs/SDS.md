# Software Design Specification (SDS)

**Project**: PipelineBuilder  
**Version**: 1.0  
**Date**: 2026-07-07  

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Architecture](#2-architecture)
3. [Backend Design](#3-backend-design)
   - [3.1 Layered Architecture](#31-layered-architecture)
   - [3.2 Dependency Injection](#32-dependency-injection)
   - [3.3 Middleware Stack](#33-middleware-stack)
   - [3.4 Data Models](#34-data-models)
   - [3.5 Repository Layer](#35-repository-layer)
   - [3.6 Service Layer](#36-service-layer)
   - [3.7 Controller Layer](#37-controller-layer)
4. [Pipeline Engine Design](#4-pipeline-engine-design)
   - [4.1 Channel-Based Fan-Out / Fan-In](#41-channel-based-fan-out--fan-in)
   - [4.2 Stage Design](#42-stage-design)
   - [4.3 Concurrency Model](#43-concurrency-model)
   - [4.4 Cancellation](#44-cancellation)
   - [4.5 Progress Tracking](#45-progress-tracking)
5. [Database Design](#5-database-design)
6. [Frontend Design](#6-frontend-design)
   - [6.1 Component Hierarchy](#61-component-hierarchy)
   - [6.2 State Management](#62-state-management)
   - [6.3 API Client](#63-api-client)
   - [6.4 Polling Strategy](#64-polling-strategy)
7. [Security Design](#7-security-design)
8. [Infrastructure & Deployment](#8-infrastructure--deployment)
9. [Error Handling Strategy](#9-error-handling-strategy)
10. [Key Design Decisions](#10-key-design-decisions)

---

## 1. System Overview

PipelineBuilder is a full-stack data processing platform composed of three runtime components:

| Component | Technology | Responsibility |
|---|---|---|
| **Backend API** | Go + chi | HTTP server, pipeline orchestration, data persistence |
| **PostgreSQL** | PostgreSQL 16 | Job state, validation errors, aggregation results |
| **Frontend** | React 18 + Vite + nginx | Dashboard, job creation form, live monitoring |

In production (Docker Compose) these three components run as separate containers on a shared Docker network. The frontend nginx instance proxies all `/api/` requests to the backend, eliminating CORS complexity and avoiding baking the backend URL into the JS bundle at build time.

---

## 2. Architecture

### High-Level Component Diagram

```
Browser
  │  HTTP (port 3000)
  ▼
┌──────────────────────────────┐
│  nginx (frontend container)  │
│  - Serves /dist (React SPA)  │
│  - Proxies /api/* → backend  │
│  - Proxies /swagger/* → backend │
└──────────────┬───────────────┘
               │ HTTP (Docker internal)
               ▼
┌──────────────────────────────┐
│  Go HTTP Server (port 8080)  │
│  ┌────────────────────────┐  │
│  │  chi Router            │  │
│  │  + Middleware Stack    │  │
│  ├────────────────────────┤  │
│  │  PipelineController    │  │
│  ├────────────────────────┤  │
│  │  PipelineService       │  │
│  ├────────────────────────┤  │
│  │  PipelineRepository    │  │
│  └────────────────────────┘  │
│  Pipeline Runner (goroutines)│
└──────────────┬───────────────┘
               │ pgx (TCP)
               ▼
┌──────────────────────────────┐
│  PostgreSQL 16               │
│  jobs / job_errors /         │
│  aggregation_results         │
└──────────────────────────────┘
```

### Request Flow (Create Pipeline)

```
POST /api/v1/pipelines
  → nginx proxy
  → chi router
  → Recoverer → Logger → CORS → RateLimit → VersionGate → APIKeyAuth
  → PipelineController.CreatePipeline()
  → ValidateJobSpec()
  → PipelineService.CreatePipeline()
      → PipelineRepository.CreateJob()   [DB INSERT]
      → go Runner.Run()                  [async goroutine]
  ← 201 Created { PipelineJob }
```

---

## 3. Backend Design

### 3.1 Layered Architecture

The backend follows a strict three-layer architecture. Each layer communicates only with the layer directly below it.

```
Controller  →  Service  →  Repository
    │              │
    │         Pipeline Engine
    │              │
    └──────────────┴──── Database (via Repository)
```

All inter-layer dependencies are expressed as Go interfaces, making each layer independently testable.

### 3.2 Dependency Injection

`main.go` is the composition root. It constructs all dependencies and wires them together before starting the server:

```go
db    := config.InitDB(cfg)
repo  := repository.NewPipelineRepository(db)
runner := pipelines.NewRunner(repo)
svc   := service.NewPipelineService(repo, runner)
ctrl  := controller.NewPipelineController(svc)
router := routes.NewRouter(ctrl, apiKey)
```

No package imports another package's concrete type — only interfaces are imported across layers.

### 3.3 Middleware Stack

Applied globally in this order for every request:

| Order | Middleware | Purpose |
|---|---|---|
| 1 | `Recoverer` | Catches panics, writes 500, keeps server alive |
| 2 | `Logger` | Logs method, path, duration |
| 3 | `CORS` | Allows all origins for all methods |
| 4 | `RateLimit` | 100 req/min per IP, token-bucket per IP map |
| 5 | `VersionGate` | Validates `{version}` path param — only `v1` accepted |
| 6 | `APIKeyAuth` | Applied only on POST / PATCH / DELETE routes |

The rate limiter stores a `map[string]*rate.Limiter` keyed by IP. A background goroutine cleans stale entries every 10 minutes to prevent memory growth.

### 3.4 Data Models

All shared data types live in `packages/shared/models/`.

#### Core Types

```go
// JobStatus is an integer enum stored in the DB
type JobStatus int
const (
    StatusPending   JobStatus = 0
    StatusRunning   JobStatus = 1
    StatusCompleted JobStatus = 2
    StatusFailed    JobStatus = 3
    StatusCancelled JobStatus = 4
)

type PipelineJob struct {
    ID          string
    Status      JobStatus
    Spec        JobSpec
    CreatedAt   time.Time
    StartedAt   *time.Time
    FinishedAt  *time.Time
    ErrorCount  int
    RecordCount int
}

type JobSpec struct {
    Sources     []SourceConfig
    Export      ExportConfig
    Concurrency ConcurrencyConfig
}

type Record struct {
    ID          string
    Source      string
    SourceType  string
    Data        map[string]any
    IsValid     bool
    ProcessedAt time.Time
}
```

`JobSpec` is stored as JSONB in PostgreSQL, serialised and deserialised by the repository using `pgx`'s native JSONB support.

### 3.5 Repository Layer

**Interface**: `PipelineRepository`  
**Implementation**: `pgxRepository` (wraps a `*pgxpool.Pool`)

All SQL is written directly (no ORM). Queries use positional parameters (`$1`, `$2`, …). The repository returns a domain-level sentinel error `ErrJobNotFound` when a SELECT returns no rows, decoupling the service from `pgx`-specific error types.

Key design choices:
- `CreateJob` uses `pgx.QueryRow` and reads back the full row to return a populated struct.
- `SaveAggregation` uses `INSERT ... ON CONFLICT DO UPDATE` (upsert) to handle re-runs.
- `DeleteJob` relies on `ON DELETE CASCADE` foreign keys — deleting the `jobs` row automatically deletes `job_errors` and `aggregation_results`.
- Tables are created with `CREATE TABLE IF NOT EXISTS` on every startup — no separate migration tool required.

### 3.6 Service Layer

**Interface**: `PipelineService`  
**Implementation**: `pipelineService`

The service layer holds the only mutable in-process state: a `map[string]context.CancelFunc` protected by a `sync.Mutex`, used to cancel running jobs on demand.

```go
type pipelineService struct {
    repo    PipelineRepository
    runner  Runner
    mu      sync.Mutex
    cancels map[string]context.CancelFunc
}
```

`CreatePipeline` workflow:
1. Validates the `JobSpec` (source URLs, export path, concurrency bounds).
2. Calls `repo.CreateJob` to persist the initial `pending` row.
3. Creates a cancellable context and stores the `CancelFunc` in `cancels`.
4. Launches `runner.Run(ctx, job)` in a goroutine.
5. Returns the newly created job immediately.

`DeletePipeline` workflow:
1. Calls `CancelPipeline` to stop execution if running.
2. Calls `repo.DeleteJob` (cascade handles child rows).
3. Removes the output file from disk (best-effort, ignores "not found" errors).

`GetProgress` workflow:
1. Checks if the job's `CancelFunc` is still in the map (i.e., it's currently running).
2. If yes, returns a live snapshot from the in-memory `Tracker`.
3. If no, fetches the final snapshot from the database.

### 3.7 Controller Layer

**Struct**: `PipelineController`

Controllers are thin. Each handler:
1. Parses and validates path parameters (UUID format check).
2. Decodes the request body if applicable.
3. Calls one service method.
4. Writes the JSON response.

All JSON responses use `json.NewEncoder(w).Encode(payload)`. Write errors (client disconnect) are logged but do not affect server state.

---

## 4. Pipeline Engine Design

### 4.1 Channel-Based Fan-Out / Fan-In

The pipeline is a linear DAG of stages connected by buffered Go channels. Each stage reads from its input channel and writes to its output channel. The runner wires all channels together and launches all goroutines before returning.

```
SourceA ─┐
SourceB ─┼──► rawCh ──► validCh ──► transformCh ──► exportCh
SourceC ─┘               │
                          └──► errorCh
```

All channels are buffered with a size equal to `ConcurrencyConfig.IngestionBufferSize` (default 100). Buffering decouples stages so a slow consumer does not immediately block a fast producer.

### 4.2 Stage Design

#### Ingestion

```
func StartIngestion(ctx, sources, out chan<- Record)
```

- One goroutine per source, coordinated by a `sync.WaitGroup`.
- Goroutines send records to the shared `out` channel.
- When all goroutines finish, `out` is closed.
- Source types are mapped to reader implementations:
  - `CSVReader` — `encoding/csv` decoder
  - `JSONReader` — `encoding/json` array decoder
  - `APIReader` — `encoding/json` object decoder with `current_weather` flattening

#### Validation

```
func StartValidation(ctx, in <-chan Record, valid chan<- Record, errs chan<- ValidationError, progress chan<- ProgressEvent, workers int)
```

- `workers` goroutines all read from the same `in` channel (fan-out across workers).
- Validated records → `valid`, errors → `errs`.
- Emits one `ProgressEvent{Processed: 1}` per record, `ProgressEvent{Errors: 1}` per error, to the progress channel.
- When all workers finish, closes both `valid` and `errs`.

#### Transformation

```
func StartTransformation(ctx, in <-chan Record, out chan<- Record, workers int)
```

- Same fan-out pattern as validation.
- Applies `Transform(record)` which iterates `record.Data` and applies coercions.
- Never produces errors (validation stage already eliminated invalid records).

#### Aggregation

```
func StartAggregation(ctx, in <-chan Record, out chan<- Record, errs <-chan ValidationError, errCount int) (aggCh <-chan AggregationResult)
```

- Single goroutine (fan-in point).
- Reads from `in`, accumulates stats into an `AggregationResult`, and forwards each record to `out`.
- After `in` is closed, sends the final `AggregationResult` to `aggCh` and closes `out`.

#### Export

```
type Exporter struct { repo PipelineRepository }
func (e *Exporter) Run(ctx, jobID, path string, records <-chan Record, agg <-chan AggregationResult)
```

- Opens the output file, writes `[` header.
- Reads records from channel, writes each as JSON with comma separators.
- Writes `]` footer and closes file.
- Reads from `agg` channel to get the aggregation result.
- Calls `repo.SaveAggregation` to persist it.

### 4.3 Concurrency Model

```
main goroutine
└── Runner goroutine (1)
    ├── Ingestion goroutine × len(sources)
    ├── Validation goroutine × ValidationWorkers
    ├── Transformation goroutine × TransformWorkers
    ├── Aggregation goroutine (1)
    ├── Error collector goroutine (1)    — saves ValidationErrors to DB
    ├── Progress tracker goroutine (1)  — accumulates ProgressEvents
    └── Export goroutine (1)
```

The runner blocks on all goroutines completing (via `sync.WaitGroup` or channel drain) before marking the job finished.

Default worker counts:

| Stage | Default | Max |
|---|---|---|
| Validation | 5 | 100 |
| Transformation | 5 | 100 |
| Ingestion buffer | 100 items | 10 000 |

### 4.4 Cancellation

Every stage receives `ctx context.Context`. Channel reads are wrapped in a `select`:

```go
select {
case record, ok := <-in:
    if !ok { return }
    // process record
case <-ctx.Done():
    return
}
```

This ensures goroutines exit promptly when the job is cancelled. The runner detects `ctx.Err() == context.Canceled` and sets the job status to `cancelled`.

### 4.5 Progress Tracking

```go
type Tracker struct {
    mu        sync.RWMutex
    processed int64
    errors    int64
    startTime time.Time
}

func (t *Tracker) Snapshot() ProgressMetrics { ... }
```

The tracker receives `ProgressEvent` values on a channel and accumulates them in a single goroutine (no lock contention on write). `Snapshot()` acquires a read lock and computes derived fields (`records_per_sec`, `elapsed_seconds`) at read time.

---

## 5. Database Design

### Tables

#### `jobs`
```sql
CREATE TABLE IF NOT EXISTS jobs (
    id           TEXT PRIMARY KEY,
    status       INTEGER NOT NULL DEFAULT 0,
    spec         JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL,
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    error_count  INTEGER NOT NULL DEFAULT 0,
    record_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
```

- `status` stored as integer for efficient filtering (`idx_jobs_status`).
- `spec` stored as JSONB — allows flexible schema evolution without migrations.

#### `job_errors`
```sql
CREATE TABLE IF NOT EXISTS job_errors (
    id         BIGSERIAL PRIMARY KEY,
    job_id     TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    record_id  TEXT NOT NULL,
    field      TEXT NOT NULL,
    message    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_job_errors_job_id ON job_errors(job_id);
```

- `ON DELETE CASCADE` means deleting a job automatically purges its errors.
- `idx_job_errors_job_id` makes the `GetErrors(jobID)` query O(errors_for_job) rather than O(all_errors).

#### `aggregation_results`
```sql
CREATE TABLE IF NOT EXISTS aggregation_results (
    job_id      TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    data        JSONB NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL
);
```

- `data` is a JSONB blob containing the full `AggregationResult` struct.
- Upserted via `INSERT ... ON CONFLICT (job_id) DO UPDATE`.

### Connection Pool

Configured in `config.InitDB`:

| Setting | Value |
|---|---|
| MaxOpenConns | 25 |
| MaxIdleConns | 5 |
| ConnMaxLifetime | 30 minutes |

---

## 6. Frontend Design

### 6.1 Component Hierarchy

```
App
└── Router
    ├── /dashboard       → Dashboard
    │     ├── KPI tiles (inline)
    │     ├── StatusPieChart
    │     └── PipelineCard × 6
    ├── /pipelines       → PipelineList
    │     ├── Filter tabs (inline)
    │     └── PipelineCard × N
    ├── /pipelines/new   → CreatePipeline
    │     └── CreatePipelineForm
    │           ├── Source rows (dynamic)
    │           ├── Export config
    │           └── Concurrency controls
    └── /pipelines/:id   → PipelineDetail
          ├── Header + action buttons
          ├── StageTracker
          ├── Metric tiles (inline)
          ├── RateChart
          ├── AggregationResults (inline)
          └── ErrorsTable
```

All layout is wrapped in `Layout` which renders the navigation `Header`.

### 6.2 State Management

No external state management library is used. Each page manages its own state with `useState` and `useEffect`. Shared data (job list) is fetched independently by each page that needs it.

Custom hooks encapsulate all polling logic:

```typescript
// Returns jobs array, loading state, error, and a manual refetch trigger
const { jobs, loading, error, refetch } = usePipelines(5000)

// Returns live ProgressMetrics and time-series history (max 30 samples)
const { metrics, history, error } = useProgress(jobId, isActive, 2000)
```

### 6.3 API Client

`src/api/client.ts` is a thin wrapper over the browser Fetch API. It:
- Prepends `VITE_API_URL` (empty by default — nginx proxy handles routing).
- Adds `Content-Type: application/json` and `X-API-Key` header only on mutating requests.
- Calls `res.json()` on error responses to extract the server error message.
- Throws an `Error` with the server message on non-2xx responses.

```typescript
const BASE_URL = import.meta.env.VITE_API_URL ?? ''
const API_KEY  = import.meta.env.VITE_API_KEY  ?? ''
```

### 6.4 Polling Strategy

```
usePipelines:
  setInterval(fetchAll, 5000)
  → pauses on document.hidden
  → resumes on visibilitychange

useProgress:
  setInterval(fetchOne, 2000)   ← only when active=true
  → active = status is "pending" or "running"
  → appends to history array (capped at 30 samples)
  → pauses on document.hidden
```

Pausing on tab hide prevents unnecessary network requests when the user is not looking at the page.

---

## 7. Security Design

### Authentication

API key is stored server-side as an environment variable (`API_KEY`). The server refuses to start without it. Incoming `X-API-Key` headers are compared using `subtle.ConstantTimeCompare` from `crypto/subtle` to prevent timing-based key enumeration.

### Rate Limiting

Per-IP token-bucket rate limiter using `golang.org/x/time/rate`:
- Each IP gets its own `rate.Limiter` with a burst of 100 and refill of 100/minute.
- Limiter instances are lazily created and stored in a `sync.Map` (or `sync.RWMutex`-protected map).
- A cleanup goroutine removes entries older than 10 minutes every 10 minutes.

### Input Validation

| Input | Validation |
|---|---|
| `{id}` path param | Parsed with `google/uuid` — returns 400 on failure |
| Source URLs | Must parse as absolute URL with `http` or `https` scheme |
| Export paths | Must be relative, must not contain `..` |
| Concurrency values | Numeric bounds checked before pipeline launch |

### CORS

Applied globally. In a production deployment behind nginx, CORS on the Go server is largely irrelevant (requests arrive from the same origin via proxy). It is included for direct API access from non-browser clients.

---

## 8. Infrastructure & Deployment

### Docker Compose Services

| Service | Image | Port |
|---|---|---|
| `postgres` | `postgres:16-alpine` | 5432 (internal) |
| `backend` | Built from `apps/server/Dockerfile` | 8080 |
| `frontend` | Built from `apps/web/Dockerfile` | 3000 → 80 |

### Backend Dockerfile (Multi-Stage)

```
Stage 1 — builder (golang:1.26-alpine)
  COPY go.mod go.sum → go mod download
  COPY . .
  go build -ldflags="-w -s" -o /server ./apps/server

Stage 2 — runtime (alpine:3.21)
  COPY /server from builder
  mkdir data/output
  EXPOSE 8080
  CMD ["./server"]
```

Build context is the **repo root** because `go.mod` lives there and imports both `apps/server` and `packages/shared`.

### Frontend Dockerfile (Multi-Stage)

```
Stage 1 — builder (node:20-alpine)
  npm ci
  COPY . .
  ARG VITE_API_KEY → ENV VITE_API_KEY
  npm run build  (VITE_API_URL intentionally left empty)

Stage 2 — serve (nginx:1.27-alpine)
  COPY dist → /usr/share/nginx/html
  COPY nginx.conf → /etc/nginx/conf.d/default.conf
  EXPOSE 80
```

`VITE_API_URL` is left empty because nginx proxies `/api/*` to the backend container at the network level — no URL baking required. `VITE_API_KEY` is a client-visible key (it ends up in the JS bundle) and is therefore acceptable to pass as a build arg.

### Service Startup Order

```
postgres (healthcheck: pg_isready)
  → backend (depends_on: postgres healthy)
    → frontend (depends_on: backend)
```

### Volumes

| Volume | Mounted At | Purpose |
|---|---|---|
| `postgres_data` | `/var/lib/postgresql/data` | Persist DB across restarts |
| `pipeline_output` | `/app/data/output` | Persist job output files across restarts |

---

## 9. Error Handling Strategy

### Backend

| Scenario | Handling |
|---|---|
| Panic in any handler | `Recoverer` middleware catches it, returns 500 |
| DB connection failure at startup | `run()` returns exit code 1 |
| DB error in handler | Handler returns 500 with JSON `{ "error": "..." }` |
| Job not found | Repository returns `ErrJobNotFound`; service propagates; controller returns 404 |
| Invalid UUID in path | Controller returns 400 before calling service |
| Rate limit exceeded | Middleware returns 429 |
| API key missing/wrong | Auth middleware returns 401 |
| Source fetch error | Logged; ingestion continues from remaining sources |
| Export write error | Runner marks job `failed`; error logged |
| Context cancelled | Runner marks job `cancelled` |

### Frontend

| Scenario | Handling |
|---|---|
| API request fails | Hook or page catches error; renders error message in UI |
| Poll returns error | Error stored in hook state; displayed inline |
| Create form submission fails | Error banner displayed below form |
| Delete/cancel fails | Error shown in modal |

---

## 10. Key Design Decisions

### Why Go channels instead of a job queue (e.g. Redis)?

For the target workload (single-operator, moderate data volumes), in-process channels provide sufficient throughput with zero external dependencies. A distributed queue would add operational complexity without clear benefit. If horizontal scaling is required, the service layer's `PipelineRepository` interface can be swapped for a queue-backed implementation without changing the pipeline stages.

### Why no ORM?

The query set is small and well-defined. Hand-written SQL with `pgx` is faster, more predictable, and easier to audit for correctness than ORM-generated queries. The repository interface provides the same testability benefit that an ORM abstraction would.

### Why JSONB for JobSpec?

`JobSpec` is a structured but potentially evolving type. Storing it as JSONB means the jobs table schema does not need to change as `JobSpec` gains new fields. The Go struct handles serialization; the DB treats it as opaque.

### Why nginx proxy instead of baking VITE_API_URL?

Vite bakes `import.meta.env` values into the JS bundle at build time. If the backend URL changes, the frontend image would need to be rebuilt. By leaving `VITE_API_URL` empty and using nginx to proxy `/api/`, the frontend image is URL-agnostic and can be deployed anywhere without rebuilding.

### Why poll instead of WebSocket for live progress?

WebSocket adds a stateful connection that complicates load balancing, proxying, and reconnection logic. The 2-second poll interval is fast enough for a live dashboard experience and avoids all of that complexity. If sub-second latency becomes a requirement, the `/progress` endpoint can be extended with SSE (Server-Sent Events) with minimal frontend changes.

### Why store JobStatus as an integer?

Integer comparisons are faster and more compact than string comparisons in PostgreSQL. The index on `status` benefits from the fixed-width integer type. The human-readable string form is always derived by the Go `String()` method on the `JobStatus` type before being serialized to JSON.
