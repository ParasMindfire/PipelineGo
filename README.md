# PipelineBuilder

A full-stack data processing platform for ingesting data from multiple sources (CSV, JSON, REST APIs), validating, transforming, aggregating, and exporting the results — with a live-updating React dashboard.

---

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Local Development](#local-development)
  - [Docker](#docker)
- [Environment Variables](#environment-variables)
- [API Reference](#api-reference)
- [Pipeline Execution Flow](#pipeline-execution-flow)
- [Frontend](#frontend)
- [Database Schema](#database-schema)
- [Running Tests](#running-tests)
- [Swagger Docs](#swagger-docs)

---

## Overview

PipelineBuilder lets you define a data pipeline by specifying:

- One or more **data sources** (CSV file URL, JSON file URL, or a REST API endpoint)
- An **export path** where processed records are written as JSON
- Optional **concurrency settings** to tune throughput

Once submitted, the pipeline runs fully asynchronously through five stages — Ingestion → Validation → Transformation → Aggregation → Export — and you can track live progress, view per-record validation errors, and inspect aggregated statistics from the dashboard.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend language | Go |
| HTTP router | [chi](https://github.com/go-chi/chi) |
| Database | PostgreSQL (pgx driver) |
| API documentation | Swagger / swaggo |
| Frontend framework | React 18 + TypeScript |
| Build tool | Vite |
| Styling | Tailwind CSS |
| Charts | Recharts |
| Container orchestration | Docker Compose |

---

## Project Structure

```
PipelineBuilder/
├── apps/
│   ├── server/                  # Go HTTP server
│   │   ├── main.go
│   │   ├── controller/          # HTTP handlers
│   │   ├── service/             # Business logic
│   │   ├── repository/          # Database access
│   │   ├── routes/              # Route definitions + middleware wiring
│   │   ├── middleware/          # CORS, auth, rate limiting, logging
│   │   └── docs/                # Generated Swagger spec
│   └── web/                     # React frontend
│       ├── src/
│       │   ├── pages/           # Dashboard, List, Create, Detail
│       │   ├── components/      # Charts, cards, forms, modals, UI kit
│       │   ├── hooks/           # usePipelines, useProgress
│       │   ├── api/             # Fetch-based API client
│       │   └── types/           # TypeScript interfaces
│       ├── Dockerfile
│       └── nginx.conf
├── packages/
│   └── shared/                  # Go packages shared between server and tests
│       ├── config/              # DB config + connection pool
│       ├── models/              # Core data types (Job, Record, etc.)
│       ├── pipelines/           # Pipeline runner + all 5 stages
│       └── utils/               # Validation helpers
├── tests/                       # Integration and unit tests
├── data/output/                 # Runtime: job output JSON files
├── docker-compose.yaml
├── go.mod
└── .env.example
```

---

## Getting Started

### Prerequisites

- Go 1.26+
- Node.js 20+
- PostgreSQL 16+
- Docker & Docker Compose (for containerised setup)

### Local Development

**1. Clone and install**

```bash
git clone <repo-url>
cd PipelineBuilder
```

**2. Set up environment variables**

```bash
cp .env.example .env
# Edit .env — set DB_PASSWORD and API_KEY at minimum
```

**3. Start PostgreSQL**

Make sure a local Postgres instance is running with the credentials from your `.env`. The server creates tables automatically on first start.

**4. Run the backend**

```bash
go run ./apps/server
```

Server starts on `http://localhost:8080`.

**5. Run the frontend**

```bash
cd apps/web
npm install
npm run dev
```

Frontend starts on `http://localhost:5173`.

---

### Docker

The easiest way to run the full stack.

**1. Create a Docker-specific env file**

```bash
cp .env.example .env.docker
```

Edit `.env.docker` and set:

```env
DB_HOST=postgres          # must be "postgres" (the service name)
DB_PASSWORD=your_password
API_KEY=your-long-random-key
VITE_API_KEY=your-long-random-key   # same as API_KEY
```

**2. Build and start**

```bash
docker compose --env-file .env.docker up --build
```

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| Swagger UI | http://localhost:3000/docs/index.html |
| PostgreSQL | localhost:5432 |

**Useful commands**

```bash
# Run in background
docker compose --env-file .env.docker up -d --build

# View logs
docker compose logs backend
docker compose logs frontend

# Stop everything
docker compose down

# Stop and wipe the database volume
docker compose down -v
```

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DB_HOST` | No | `localhost` | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | No | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | **Yes** | — | PostgreSQL password |
| `DB_NAME` | No | `pipeline_db` | PostgreSQL database name |
| `API_KEY` | **Yes** | — | Secret key for mutating API endpoints |
| `PORT` | No | `8080` | HTTP server port |
| `VITE_API_URL` | No | `""` | Frontend API base URL (empty = nginx proxy) |
| `VITE_API_KEY` | No | `""` | Frontend API key for mutating requests |

The backend refuses to start if `API_KEY` or `DB_PASSWORD` is missing.

---

## API Reference

Base path: `/api/v1/pipelines`  
Auth: `X-API-Key` header required on POST, PATCH, DELETE  
Rate limit: 100 requests / minute / IP

### Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | No | Liveness check |
| GET | `/metrics` | No | Prometheus-format job counts |
| GET | `/api/v1/pipelines` | No | List all pipeline jobs |
| POST | `/api/v1/pipelines` | Yes | Create and start a new pipeline |
| GET | `/api/v1/pipelines/{id}` | No | Get a single job |
| DELETE | `/api/v1/pipelines/{id}` | Yes | Delete a job and its output file |
| PATCH | `/api/v1/pipelines/{id}/cancel` | Yes | Cancel a running job |
| GET | `/api/v1/pipelines/{id}/progress` | No | Live progress metrics |
| GET | `/api/v1/pipelines/{id}/results` | No | Aggregation results (once complete) |
| GET | `/api/v1/pipelines/{id}/errors` | No | All validation errors for a job |

### Create Pipeline — Request Body

```json
{
  "Sources": [
    { "Type": "csv",  "URL": "https://example.com/data.csv" },
    { "Type": "json", "URL": "https://example.com/data.json" },
    { "Type": "api",  "URL": "https://api.example.com/weather" }
  ],
  "Export": {
    "Type": "json",
    "Path": "data/output/my-job.json"
  },
  "Concurrency": {
    "ValidationWorkers": 5,
    "TransformWorkers": 5,
    "IngestionBufferSize": 100
  }
}
```

### Progress Metrics Response

```json
{
  "job_id": "uuid",
  "status": "running",
  "processed_count": 1500,
  "error_count": 12,
  "percent_complete": 75,
  "records_per_sec": 320.4,
  "start_time": "2024-01-01T12:00:00Z",
  "end_time": null,
  "elapsed_seconds": 4.7
}
```

`percent_complete` is `-1` when total record count is unknown.

### Aggregation Results Response

```json
{
  "job_id": "uuid",
  "total_count": 2000,
  "valid_count": 1988,
  "error_count": 12,
  "by_source": { "csv": 1000, "json": 800, "api": 200 },
  "numeric_stats": {
    "temperature": { "min": -5.2, "max": 38.1, "sum": 14200.5, "avg": 21.3, "count": 667 }
  },
  "computed_at": "2024-01-01T12:00:05Z"
}
```

For the full interactive API docs, open `/swagger/index.html` when the server is running.

---

## Pipeline Execution Flow

Each job runs through five stages connected by Go channels:

```
Ingestion → Validation → Transformation → Aggregation → Export
```

| Stage | Concurrency | Description |
|---|---|---|
| **Ingestion** | 1 goroutine per source | Fetches data from each source URL concurrently, emits `Record` structs |
| **Validation** | N workers (configurable) | Checks required fields and numeric field formats; splits records into valid/error channels |
| **Transformation** | N workers (configurable) | Coerces string numbers to `float64`, trims and lowercases other strings |
| **Aggregation** | 1 goroutine (fan-in) | Accumulates totals, per-source counts, and numeric field statistics |
| **Export** | 1 goroutine | Writes all records to a JSON file; persists aggregation result to PostgreSQL |

Jobs move through statuses: `pending → running → completed / failed / cancelled`

Cancellation is cooperative — a context cancel signal propagates through all stages, each of which respects `ctx.Done()`.

---

## Frontend

### Pages

| Route | Description |
|---|---|
| `/dashboard` | KPI tiles, status pie chart, 6 most recent jobs |
| `/pipelines` | Filterable grid of all jobs |
| `/pipelines/new` | Form to create a new pipeline |
| `/pipelines/:id` | Full detail: live metrics, rate chart, stage tracker, errors, results |

### Polling

- Pipeline list: every 5 seconds (pauses when browser tab is hidden)
- Progress metrics: every 2 seconds while job is running or pending

---

## Database Schema

Three tables, all created automatically on server start:

**jobs** — one row per pipeline job  
**job_errors** — one row per validation error, foreign-keyed to `jobs`  
**aggregation_results** — one row per completed job with full stats JSON

All child rows cascade-delete when a job is deleted.

---

## Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests only (requires a running Postgres)
go test ./tests/...
```

---

## Swagger Docs

The Swagger UI is served at `/swagger/index.html`.

To regenerate the spec after changing controller annotations:

```bash
swag init -g apps/server/main.go -o apps/server/docs
```
