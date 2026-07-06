# Functional Requirements Specification (FRS)

**Project**: PipelineBuilder  
**Version**: 1.0  
**Date**: 2026-07-07  

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Scope](#2-scope)
3. [User Roles](#3-user-roles)
4. [Functional Requirements](#4-functional-requirements)
   - [4.1 Pipeline Management](#41-pipeline-management)
   - [4.2 Pipeline Execution](#42-pipeline-execution)
   - [4.3 Progress Tracking](#43-progress-tracking)
   - [4.4 Results & Aggregation](#44-results--aggregation)
   - [4.5 Error Reporting](#45-error-reporting)
   - [4.6 Authentication & Security](#46-authentication--security)
   - [4.7 Dashboard & UI](#47-dashboard--ui)
5. [Non-Functional Requirements](#5-non-functional-requirements)
6. [Input / Output Specifications](#6-input--output-specifications)
7. [Validation Rules](#7-validation-rules)
8. [Constraints & Assumptions](#8-constraints--assumptions)

---

## 1. Introduction

PipelineBuilder is a data processing platform that allows operators to define, execute, monitor, and manage multi-source data pipelines through a REST API and a React-based web dashboard. Each pipeline ingests data from one or more sources (CSV files, JSON files, or REST API endpoints), validates and transforms every record, computes aggregate statistics, and exports the processed records as a JSON file.

---

## 2. Scope

This document covers:

- All HTTP API endpoints and their behaviour
- The complete pipeline execution lifecycle
- Validation rules for all user-supplied inputs
- The dashboard's display, polling, and interaction requirements
- Security and rate-limiting requirements

Out of scope: infrastructure provisioning, user account management (there is a single shared API key), and data source content (the system does not control what the external URLs return).

---

## 3. User Roles

| Role | Description | Access |
|---|---|---|
| **Operator** | Authenticated user who submits and manages pipelines | Full API access (read + write) |
| **Viewer** | Unauthenticated user who observes pipeline status | Read-only API access |

Authentication is via a single shared `X-API-Key` header. Any client that provides the correct key is treated as an Operator.

---

## 4. Functional Requirements

### 4.1 Pipeline Management

#### FR-PM-01: Create Pipeline
- The system shall accept a POST request to `/api/v1/pipelines` with a `JobSpec` body.
- On success the system shall persist a new job row with `status = pending` and return a `201 Created` response containing the full `PipelineJob` object including the generated UUID.
- The system shall start pipeline execution asynchronously; the HTTP response must be returned before execution completes.
- The system shall reject the request with `400 Bad Request` if the `JobSpec` fails any validation rule (see §7).
- The system shall require a valid `X-API-Key` header; absence or mismatch returns `401 Unauthorized`.

#### FR-PM-02: List Pipelines
- The system shall return all pipeline jobs ordered by `created_at` descending on GET `/api/v1/pipelines`.
- The response shall include full job details (ID, status, spec, timestamps, record count, error count).
- No authentication is required.

#### FR-PM-03: Get Single Pipeline
- The system shall return a single pipeline job by UUID on GET `/api/v1/pipelines/{id}`.
- If the UUID is malformed the system shall return `400 Bad Request`.
- If no job exists with that ID the system shall return `404 Not Found`.

#### FR-PM-04: Delete Pipeline
- The system shall accept DELETE `/api/v1/pipelines/{id}`.
- If the job is currently running, the system shall first send a cancellation signal to stop it.
- The system shall then delete all database rows for the job (including validation errors and aggregation results via cascade).
- The system shall delete the output file from disk if it exists.
- On success the system shall return `204 No Content`.
- Requires a valid `X-API-Key` header.

#### FR-PM-05: Cancel Running Pipeline
- The system shall accept PATCH `/api/v1/pipelines/{id}/cancel`.
- If the job is currently running, the system shall deliver a cancellation signal to the running goroutine.
- The response shall be `200 OK` with `{ "message": "cancellation signal sent" }`.
- If the job is not currently running, the system shall return `404 Not Found`.
- Requires a valid `X-API-Key` header.

---

### 4.2 Pipeline Execution

#### FR-PE-01: Multi-Source Ingestion
- The system shall support three source types: `csv`, `json`, and `api`.
- Each source shall be fetched concurrently (one goroutine per source).
- A failure to fetch or parse one source shall be logged and the pipeline shall continue processing the remaining sources.
- **CSV**: Each row shall produce one record. The first row is treated as a header.
- **JSON**: The response body must be a JSON array; each element produces one record.
- **API**: The response body must be a JSON object. If a `current_weather` key exists its child fields shall be flattened into the record's data map.

#### FR-PE-02: Record Validation
- The system shall validate every ingested record.
- Validation shall run in a configurable worker pool (`ValidationWorkers`, default 5, max 100).
- Records that pass all rules are forwarded to the transformation stage.
- Records that fail one or more rules have their errors persisted to the `job_errors` table; the record itself is not forwarded.
- Each validation error record shall capture: job ID, record ID, the failing field name, an error message, and a timestamp.

#### FR-PE-03: Record Transformation
- The system shall transform every valid record using a configurable worker pool (`TransformWorkers`, default 5, max 100).
- Transformation rules:
  - String values that can be parsed as `float64` shall be converted to `float64`.
  - Other string values shall be trimmed of leading/trailing whitespace and converted to lowercase.
  - Non-string values shall be passed through unchanged.

#### FR-PE-04: Aggregation
- After all records have been transformed, the system shall compute:
  - Total record count
  - Valid record count
  - Error record count
  - Record count grouped by source type (`csv`, `json`, `api`)
  - For each field whose value is `float64`: min, max, sum, average, and count

#### FR-PE-05: Export
- The system shall write all transformed records to a JSON file at the path specified in `ExportConfig.Path`.
- If the output directory does not exist it shall be created automatically.
- The output shall be a valid JSON array of record objects.
- After the file is written, the aggregation result shall be persisted to the `aggregation_results` table.

#### FR-PE-06: Job Status Transitions
- A job transitions through statuses in this order: `pending → running → completed | failed | cancelled`.
- The system shall set `started_at` when execution begins.
- The system shall set `finished_at` and the final status when execution ends.
- A context cancellation shall result in `cancelled` status.
- An unrecoverable export or DB error shall result in `failed` status.
- Any other completion shall result in `completed` status.

#### FR-PE-07: Cancellation Propagation
- The system shall create a per-job `context.CancelFunc` when execution begins.
- All pipeline stages shall respect `ctx.Done()` and stop processing as soon as possible.
- The `CancelFunc` shall be stored in memory keyed by job ID and removed when the job finishes.

---

### 4.3 Progress Tracking

#### FR-PT-01: Live Progress Endpoint
- The system shall expose GET `/api/v1/pipelines/{id}/progress` returning `ProgressMetrics`.
- While the job is running, metrics shall be served from an in-memory tracker.
- Once the job has finished, metrics shall be fetched from the database snapshot.

#### FR-PT-02: Progress Metrics Content
The `ProgressMetrics` response shall include:
- `job_id` — UUID of the job
- `status` — current job status string
- `processed_count` — total records processed so far
- `error_count` — total validation errors so far
- `percent_complete` — 0–100, or `-1` if total count is unknown
- `records_per_sec` — throughput calculated from elapsed time
- `start_time` — ISO 8601 timestamp
- `end_time` — ISO 8601 timestamp or `null` if still running
- `elapsed_seconds` — wall-clock seconds since start

#### FR-PT-03: In-Memory Tracker
- The tracker shall be thread-safe (protected by `sync.RWMutex` or equivalent).
- The tracker shall accumulate `ProgressEvent` structs emitted by the validation stage (one event per record processed).

---

### 4.4 Results & Aggregation

#### FR-RA-01: Results Endpoint
- The system shall expose GET `/api/v1/pipelines/{id}/results`.
- If aggregation has been computed and persisted, the endpoint shall return the full `AggregationResult`.
- If aggregation is not yet available (job still running or failed before completion), the endpoint shall return `404 Not Found`.

#### FR-RA-02: Aggregation Persistence
- The system shall upsert the aggregation result into `aggregation_results` keyed by `job_id`.
- The persisted data shall include the computed-at timestamp.

---

### 4.5 Error Reporting

#### FR-ER-01: Validation Errors Endpoint
- The system shall expose GET `/api/v1/pipelines/{id}/errors`.
- The response shall be a JSON array of all `ValidationError` rows for the job, ordered by creation time ascending.
- An empty array shall be returned if no errors were recorded.

#### FR-ER-02: Error Persistence
- Validation errors shall be persisted to the `job_errors` table as they are discovered (not batched after completion).

---

### 4.6 Authentication & Security

#### FR-AS-01: API Key Authentication
- The system shall authenticate write operations (POST, PATCH, DELETE) using the `X-API-Key` HTTP header.
- The comparison shall be constant-time to prevent timing attacks.
- Missing or incorrect key returns `401 Unauthorized`.

#### FR-AS-02: Rate Limiting
- The system shall limit each client IP to 100 requests per minute using a token-bucket algorithm.
- Excess requests shall receive `429 Too Many Requests`.
- Old IP entries shall be cleaned up every 10 minutes.

#### FR-AS-03: CORS
- The system shall allow cross-origin requests from any origin with methods GET, POST, PATCH, DELETE, OPTIONS.

#### FR-AS-04: Export Path Safety
- The system shall reject any export path that is absolute or contains `..` path components.
- This prevents directory traversal attacks.

#### FR-AS-05: Server Startup Guard
- The server shall refuse to start if `API_KEY` is empty.
- The server shall refuse to start if the database connection cannot be established.

---

### 4.7 Dashboard & UI

#### FR-UI-01: Dashboard Page
- The dashboard shall display KPI tiles: total jobs, running jobs, completed jobs, failed jobs.
- The dashboard shall display a pie chart showing the status distribution of all jobs.
- The dashboard shall display the 6 most recently created jobs in a summary table.

#### FR-UI-02: Pipeline List Page
- The list page shall display all pipeline jobs as cards.
- The user shall be able to filter jobs by status: all, pending, running, completed, failed, cancelled.
- Each card shall show job ID, status badge, record count, error count, and source count.

#### FR-UI-03: Create Pipeline Page
- The form shall allow the user to add one or more data sources (type + URL).
- The form shall allow the user to set the export path (optional; auto-generated if blank).
- The form shall expose concurrency controls: validation workers, transform workers, ingestion buffer size.
- On successful submission the user shall be redirected to the detail page for the new job.
- On failure an error message shall be displayed.

#### FR-UI-04: Pipeline Detail Page
- The detail page shall display: job ID, status badge, created/started/finished timestamps, action buttons (cancel, delete).
- The detail page shall display a five-stage visual tracker (Ingestion, Validation, Transformation, Aggregation, Export) with state indicators (idle, active, done, error).
- The detail page shall display metric tiles: records processed, records/sec rate, error count, percentage complete.
- The detail page shall display a live rate chart (records/sec over time) while the job is running.
- The detail page shall display aggregation results once the job completes.
- The detail page shall display a validation errors table.
- Cancel and Delete actions shall display a confirmation modal before executing.

#### FR-UI-05: Polling Behaviour
- The pipeline list and dashboard shall refresh data every 5 seconds.
- The progress metrics on the detail page shall refresh every 2 seconds while the job is pending or running.
- Polling shall pause when the browser tab is not visible and resume when it becomes visible.

#### FR-UI-06: Health & Metrics Endpoints
- GET `/health` shall return `{ "status": "ok", "version": "1.0" }` with HTTP 200.
- GET `/metrics` shall return Prometheus-format text with job counts grouped by status.

---

## 5. Non-Functional Requirements

| ID | Category | Requirement |
|---|---|---|
| NFR-01 | Performance | The pipeline runner shall process records concurrently; worker counts are configurable. |
| NFR-02 | Reliability | A crash in one ingestion source shall not halt ingestion from other sources. |
| NFR-03 | Reliability | The server shall recover from panics (via middleware) without restarting. |
| NFR-04 | Scalability | Worker pool sizes and buffer sizes are user-configurable per job. |
| NFR-05 | Security | The API key comparison shall be constant-time. |
| NFR-06 | Security | Export paths must not allow traversal outside the working directory. |
| NFR-07 | Observability | Every request shall be logged with method, path, and duration. |
| NFR-08 | Observability | A Prometheus-compatible `/metrics` endpoint shall expose job status counts. |
| NFR-09 | Graceful Shutdown | The server shall complete in-flight requests within 15 seconds before exiting. |
| NFR-10 | DB Connections | The connection pool shall use max 25 open connections, 5 idle, 30-minute lifetime. |

---

## 6. Input / Output Specifications

### JobSpec (POST /api/v1/pipelines)

```
Sources[]
  Type    string   Required. One of: "csv", "json", "api"
  URL     string   Required. Must be an absolute http(s) URL

Export
  Type    string   Required. One of: "json", "csv"
  Path    string   Required. Relative path, no ".." segments, not absolute

Concurrency (optional — all fields default to 0 which applies server-side defaults)
  ValidationWorkers   int   0–100, default 5
  TransformWorkers    int   0–100, default 5
  IngestionBufferSize int   0–10000, default 100
```

### PipelineJob (response)

```
ID          string     UUID
Status      string     "pending" | "running" | "completed" | "failed" | "cancelled"
Spec        JobSpec    Echo of submitted spec
CreatedAt   timestamp
StartedAt   timestamp | null
FinishedAt  timestamp | null
ErrorCount  int
RecordCount int
```

### Record (output file element)

```json
{
  "id": "uuid-string",
  "source": "https://source-url",
  "source_type": "csv",
  "data": { "field": "value", "numeric_field": 42.0 },
  "is_valid": true,
  "processed_at": "2024-01-01T12:00:00Z"
}
```

---

## 7. Validation Rules

### JobSpec Validation

| Rule | Error condition |
|---|---|
| Sources must not be empty | Zero sources provided |
| Source type must be valid | Type not in `["csv", "json", "api"]` |
| Source URL must be absolute | URL does not start with `http://` or `https://` |
| Export type must be valid | Type not in `["json", "csv"]` |
| Export path must be relative | Path is absolute or starts with `/` |
| Export path must not traverse | Path contains `..` segment |
| ValidationWorkers in range | Value > 100 |
| TransformWorkers in range | Value > 100 |
| IngestionBufferSize in range | Value > 10000 |

### Record Validation (per record, during pipeline execution)

| Rule | Field | Error message |
|---|---|---|
| `id` field is required | `id` | "id is required" |
| `source` field is required | `source` | "source is required" |
| Numeric fields must parse as float64 | `new_cases`, `new_deaths`, `cases`, `deaths`, `height(inches)`, `weight(pounds)`, `price`, `temperature`, `windspeed` | "{field} must be numeric" |

---

## 8. Constraints & Assumptions

- **Single API key**: The system is designed for a single-team / single-operator deployment. There is no per-user auth.
- **Async execution**: Pipeline jobs run asynchronously; callers must poll `/progress` or `/results` for completion.
- **Local file output**: Exported JSON files are written to the local filesystem of the server process. In a distributed deployment, this would need to be replaced with object storage.
- **In-memory tracking**: Live progress metrics are stored in memory. If the server restarts during a running job, live metrics are lost (the job is still readable from DB).
- **External data sources**: The system does not validate the content of external source URLs beyond what it can parse. Network errors are logged and skipped.
- **No pagination**: List endpoints return all records. This is suitable for the expected data volumes of a single-operator deployment.
