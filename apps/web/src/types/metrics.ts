// Mirrors the backend models.ProgressMetrics returned by GET .../progress
export interface ProgressMetrics {
  job_id: string
  status: string
  processed_count: number
  error_count: number
  /** -1 when the total record count is unknown upfront */
  percent_complete: number
  records_per_sec: number
  start_time: string
  end_time?: string
  elapsed_seconds: number
}

// Single time-series sample accumulated by useProgress for the rate chart
export interface RateDataPoint {
  time: string
  rate: number
  processed: number
}
