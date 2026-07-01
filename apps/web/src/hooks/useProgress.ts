import { useState, useEffect, useCallback } from 'react'
import { getProgress } from '../api/client'
import type { ProgressMetrics, RateDataPoint } from '../types/metrics'

// Polls progress for a job and accumulates time-series data for the rate chart
export function useProgress(jobId: string, active: boolean, intervalMs = 2000) {
  const [metrics, setMetrics] = useState<ProgressMetrics | null>(null)
  const [history, setHistory] = useState<RateDataPoint[]>([])
  const [error, setError] = useState<string | null>(null)

  const poll = useCallback(async () => {
    if (document.visibilityState === 'hidden') return
    try {
      const data = await getProgress(jobId)
      setMetrics(data)
      // Keep last 30 samples so the chart doesn't grow unbounded
      setHistory(prev => [
        ...prev.slice(-29),
        {
          time: new Date().toLocaleTimeString(),
          rate: Math.round(data.records_per_sec),
          processed: data.processed_count,
        },
      ])
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch progress')
    }
  }, [jobId])

  useEffect(() => {
    poll()
    if (!active) return
    const id = setInterval(poll, intervalMs)
    return () => clearInterval(id)
  }, [poll, active, intervalMs])

  return { metrics, history, error }
}
