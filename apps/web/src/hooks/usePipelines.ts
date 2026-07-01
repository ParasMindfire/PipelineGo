import { useState, useEffect, useCallback } from 'react'
import { listPipelines } from '../api/client'
import type { PipelineJob } from '../types/pipeline'

// Polls GET /api/v1/pipelines every intervalMs, pausing when the tab is hidden
export function usePipelines(intervalMs = 5000) {
  const [jobs, setJobs] = useState<PipelineJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    // Skip fetch when the browser tab is backgrounded
    if (document.visibilityState === 'hidden') return
    try {
      const data = await listPipelines()
      setJobs(data)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch pipelines')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
    const id = setInterval(refresh, intervalMs)
    // Also refresh immediately when the user switches back to this tab
    document.addEventListener('visibilitychange', refresh)
    return () => {
      clearInterval(id)
      document.removeEventListener('visibilitychange', refresh)
    }
  }, [refresh, intervalMs])

  return { jobs, loading, error, refetch: refresh }
}
