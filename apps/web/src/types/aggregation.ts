// Mirrors the backend models.FieldStats for a single numeric column
export interface FieldStats {
  min: number
  max: number
  sum: number
  avg: number
  count: number
}

// Mirrors the backend models.AggregationResult returned by GET .../results
export interface AggregationResult {
  job_id: string
  total_count: number
  valid_count: number
  error_count: number
  by_source: Record<string, number>
  numeric_stats: Record<string, FieldStats>
  computed_at: string
}
