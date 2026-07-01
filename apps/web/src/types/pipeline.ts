// Mirrors the backend models.JobStatus string values
export type JobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

// Input source configuration for a pipeline job
export interface SourceConfig {
  type: 'csv' | 'json' | 'api'
  url: string
}

// Output export configuration for a pipeline job
export interface ExportConfig {
  type: 'json' | 'csv'
  path: string
}

// Worker and buffer concurrency settings for a pipeline job
export interface ConcurrencyConfig {
  validation_workers: number
  transform_workers: number
  ingestion_buffer_size: number
}

// Full job specification sent in POST /api/v1/pipelines body
export interface JobSpec {
  sources: SourceConfig[]
  export: ExportConfig
  concurrency: ConcurrencyConfig
}

// Mirrors the backend models.PipelineJob database record
export interface PipelineJob {
  id: string
  status: JobStatus
  spec: JobSpec
  created_at: string
  started_at?: string
  finished_at?: string
  error_count: number
  record_count: number
}

// Mirrors the backend models.ValidationError
export interface ValidationError {
  job_id: string
  record_id: string
  field: string
  message: string
  at: string
}
