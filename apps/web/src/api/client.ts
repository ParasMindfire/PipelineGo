import type { PipelineJob, JobSpec, ValidationError } from '../types/pipeline'
import type { ProgressMetrics } from '../types/metrics'
import type { AggregationResult } from '../types/aggregation'

// Base URL empty in dev — Vite proxy forwards /api to localhost:8080
const BASE_URL = import.meta.env.VITE_API_URL ?? ''
const API_KEY = import.meta.env.VITE_API_KEY ?? ''

// Builds headers — Content-Type only on mutating requests (GET has no body, adding it triggers CORS preflight)
function headers(withAuth = false): HeadersInit {
  const h: Record<string, string> = {}
  if (withAuth) {
    h['Content-Type'] = 'application/json'
    if (API_KEY) h['X-API-Key'] = API_KEY
  }
  return h
}

// Parses the response as JSON or throws with the server error message
async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error((body as { error?: string }).error ?? res.statusText)
  }
  return res.json() as Promise<T>
}

// Returns all pipeline jobs, most recently created first
export async function listPipelines(): Promise<PipelineJob[]> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines`, { headers: headers() })
  return handleResponse<PipelineJob[]>(res)
}

// Returns a single pipeline job by its UUID
export async function getPipeline(id: string): Promise<PipelineJob> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines/${id}`, { headers: headers() })
  return handleResponse<PipelineJob>(res)
}

// Creates a new pipeline job from a JobSpec; requires VITE_API_KEY
export async function createPipeline(spec: JobSpec): Promise<PipelineJob> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines`, {
    method: 'POST',
    headers: headers(true),
    body: JSON.stringify(spec),
  })
  return handleResponse<PipelineJob>(res)
}

// Permanently deletes a job and its output file; requires VITE_API_KEY
export async function deletePipeline(id: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines/${id}`, {
    method: 'DELETE',
    headers: headers(true),
  })
  if (res.status === 204) return
  await handleResponse<void>(res)
}

// Sends a cancellation signal to a running job; requires VITE_API_KEY
export async function cancelPipeline(id: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines/${id}/cancel`, {
    method: 'PATCH',
    headers: headers(true),
  })
  await handleResponse<{ message: string }>(res)
}

// Returns live metrics from the in-memory tracker, or a DB snapshot when done
export async function getProgress(id: string): Promise<ProgressMetrics> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines/${id}/progress`, { headers: headers() })
  return handleResponse<ProgressMetrics>(res)
}

// Returns aggregation stats for a completed job, or null when not yet computed
export async function getResults(id: string): Promise<AggregationResult | null> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines/${id}/results`, { headers: headers() })
  if (res.status === 404) return null
  return handleResponse<AggregationResult>(res)
}

// Returns all validation errors recorded for a job
export async function getErrors(id: string): Promise<ValidationError[]> {
  const res = await fetch(`${BASE_URL}/api/v1/pipelines/${id}/errors`, { headers: headers() })
  return handleResponse<ValidationError[]>(res)
}
