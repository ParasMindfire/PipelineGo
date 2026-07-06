import type { JobStatus } from '../types/pipeline'

// Polling intervals (milliseconds)
export const PIPELINES_POLL_MS = 5000
export const PROGRESS_POLL_MS  = 2000

// Number of rate-chart samples kept in memory per job
export const PROGRESS_HISTORY_LIMIT = 30

// Number of recent jobs shown on the Dashboard
export const DASHBOARD_RECENT_COUNT = 6

// Ordered list of pipeline stage names
export const PIPELINE_STAGES = [
  'Ingestion',
  'Validation',
  'Transformation',
  'Aggregation',
  'Export',
] as const

// Status filter options for the Pipelines list page
export const STATUS_FILTERS: (JobStatus | 'all')[] = [
  'all', 'running', 'pending', 'completed', 'failed', 'cancelled',
]

// Default form values for new pipeline creation
export const DEFAULT_VAL_WORKERS    = 5
export const DEFAULT_TRANS_WORKERS  = 5
export const DEFAULT_BUFFER_SIZE    = 100
export const DEFAULT_EXPORT_TYPE    = 'json' as const
export const DEFAULT_EXPORT_PREFIX  = 'data/output/job-'

// Upper bound shown in the concurrency number inputs
export const MAX_WORKERS = 100
