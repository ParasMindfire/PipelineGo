import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  getPipeline, getResults, getErrors, cancelPipeline, deletePipeline,
} from '../api/client'
import { useProgress } from '../hooks/useProgress'
import { Card } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { ConfirmModal } from '../components/ui/ConfirmModal'
import { StageTracker } from '../components/pipeline/StageTracker'
import { RateChart } from '../components/charts/RateChart'
import { ErrorsTable } from '../components/pipeline/ErrorsTable'
import type { PipelineJob, ValidationError } from '../types/pipeline'
import type { AggregationResult } from '../types/aggregation'

type ModalType = 'delete' | 'cancel' | null

// Formats elapsed seconds as "Xm Ys" or just "Xs"
function fmtElapsed(s: number): string {
  const m = Math.floor(s / 60)
  return m > 0 ? `${m}m ${Math.floor(s % 60)}s` : `${Math.floor(s)}s`
}

// Detail view for one pipeline job — live metrics, stage tracker, results, and errors
export default function PipelineDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [job, setJob] = useState<PipelineJob | null>(null)
  const [results, setResults] = useState<AggregationResult | null>(null)
  const [errors, setErrors] = useState<ValidationError[]>([])
  const [jobLoading, setJobLoading] = useState(true)
  const [actionBusy, setActionBusy] = useState(false)
  const [pageError, setPageError] = useState<string | null>(null)
  const [modal, setModal] = useState<ModalType>(null)

  // Poll progress only while the job is still running
  const isActive = job?.status === 'running' || job?.status === 'pending'
  const { metrics, history } = useProgress(id!, isActive, 2000)

  // Loads (or re-loads) the job row, aggregation results, and validation errors
  const loadJob = useCallback(async () => {
    if (!id) return
    try {
      const [j, r, e] = await Promise.all([getPipeline(id), getResults(id), getErrors(id)])
      setJob(j)
      setResults(r)
      setErrors(e)
    } catch (err) {
      setPageError(err instanceof Error ? err.message : 'Failed to load job')
    } finally {
      setJobLoading(false)
    }
  }, [id])

  // Re-poll the job row every 5 s while active so the status badge updates
  useEffect(() => {
    loadJob()
    if (!isActive) return
    const t = setInterval(loadJob, 5000)
    return () => clearInterval(t)
  }, [loadJob, isActive])

  const handleCancel = async () => {
    if (!id) return
    setActionBusy(true)
    setModal(null)
    try { await cancelPipeline(id); await loadJob() }
    catch (err) { setPageError(err instanceof Error ? err.message : 'Cancel failed') }
    finally { setActionBusy(false) }
  }

  const handleDelete = async () => {
    if (!id) return
    setActionBusy(true)
    setModal(null)
    try { await deletePipeline(id); navigate('/pipelines') }
    catch (err) {
      setPageError(err instanceof Error ? err.message : 'Delete failed')
      setActionBusy(false)
    }
  }

  if (jobLoading) return <Spinner />
  if (pageError && !job) return <p className="text-red-600 text-sm bg-red-50 rounded-lg px-4 py-3">{pageError}</p>
  if (!job) return <p className="text-gray-500 text-sm">Job not found.</p>

  // Use live metrics when available, else fall back to DB values
  const processedCount = metrics?.processed_count ?? job.record_count
  const errorCount     = metrics?.error_count     ?? job.error_count
  const rate           = metrics?.records_per_sec  ?? 0
  const elapsed        = metrics?.elapsed_seconds  ?? 0
  const pct            = metrics?.percent_complete ?? (job.status === 'completed' ? 100 : -1)

  return (
    <div className="space-y-6">
      {/* ── Header ── */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-3 mb-1 flex-wrap">
            <Link to="/pipelines" className="text-gray-400 hover:text-gray-600 text-sm">← Pipelines</Link>
            <Badge status={job.status} />
          </div>
          <h1 className="font-mono text-lg font-bold text-gray-900 break-all">{job.id}</h1>
          <p className="text-xs text-gray-400 mt-1">
            Created {new Date(job.created_at).toLocaleString()}
            {job.started_at  && ` · Started ${new Date(job.started_at).toLocaleString()}`}
            {job.finished_at && ` · Finished ${new Date(job.finished_at).toLocaleString()}`}
          </p>
        </div>
        <div className="flex gap-2 shrink-0">
          {isActive && (
            <button
              onClick={() => setModal('cancel')} disabled={actionBusy}
              className="px-4 py-2 text-sm font-medium text-yellow-700 bg-yellow-50 hover:bg-yellow-100 rounded-lg transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
          )}
          <button
            onClick={() => setModal('delete')} disabled={actionBusy}
            className="px-4 py-2 text-sm font-medium text-red-700 bg-red-50 hover:bg-red-100 rounded-lg transition-colors disabled:opacity-50"
          >
            Delete
          </button>
        </div>
      </div>

      {pageError && (
        <p className="text-red-600 text-sm bg-red-50 border border-red-100 rounded-lg px-4 py-3">{pageError}</p>
      )}

      {/* ── Stage Tracker ── */}
      <Card title="Pipeline Stages">
        <StageTracker status={job.status} percentComplete={pct} />
      </Card>

      {/* ── Metric tiles ── */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        {[
          { label: 'Records Processed', value: processedCount.toLocaleString() },
          { label: 'Processing Rate',   value: isActive ? `${Math.round(rate)} r/s` : '—' },
          { label: 'Validation Errors', value: errorCount.toLocaleString(), red: errorCount > 0 },
          { label: 'Progress',          value: pct < 0 ? 'Live' : `${Math.round(pct)}%` },
        ].map(({ label, value, red }) => (
          <div key={label} className="bg-white rounded-xl border border-gray-100 p-5">
            <p className="text-xs text-gray-400">{label}</p>
            <p className={`text-2xl font-bold mt-1 ${red ? 'text-red-500' : 'text-gray-900'}`}>{value}</p>
          </div>
        ))}
      </div>

      {/* ── Progress bar ── */}
      {pct >= 0 && (
        <div>
          <div className="flex justify-between text-xs text-gray-400 mb-1.5">
            <span>
              {job.status === 'running' ? `Elapsed: ${fmtElapsed(elapsed)}` : job.status}
            </span>
            <span>{Math.round(Math.min(pct, 100))}%</span>
          </div>
          <div className="w-full bg-gray-100 rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all duration-700 ${
                job.status === 'failed'     ? 'bg-red-500'  :
                job.status === 'cancelled'  ? 'bg-gray-400' : 'bg-blue-500'
              }`}
              style={{ width: `${Math.min(pct, 100)}%` }}
            />
          </div>
        </div>
      )}

      {/* ── Live processing-rate chart ── */}
      {isActive && (
        <Card title="Processing Rate (records / sec)">
          <RateChart data={history} />
        </Card>
      )}

      {/* ── Source breakdown (sources list from spec) ── */}
      <Card title="Sources">
        <div className="flex flex-wrap gap-2">
          {(job.spec?.sources ?? []).map((src, i) => (
            <span key={i} className="bg-blue-50 text-blue-700 px-3 py-1 rounded-full text-xs font-medium">
              {src.type.toUpperCase()} · {src.url}
            </span>
          ))}
        </div>
      </Card>

      {/* ── Aggregation results ── */}
      {results && (
        <Card title="Aggregation Results">
          <div className="grid grid-cols-3 gap-6 mb-5">
            <div>
              <p className="text-xs text-gray-400">Total Records</p>
              <p className="text-2xl font-bold text-gray-900">{results.total_count.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Valid</p>
              <p className="text-2xl font-bold text-green-600">{results.valid_count.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Errors</p>
              <p className="text-2xl font-bold text-red-500">{results.error_count.toLocaleString()}</p>
            </div>
          </div>

          {/* Records by source */}
          {Object.keys(results.by_source).length > 0 && (
            <div className="mb-5">
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-2">By Source</p>
              <div className="flex flex-wrap gap-2">
                {Object.entries(results.by_source).map(([src, count]) => (
                  <span key={src} className="bg-gray-50 border border-gray-100 text-gray-700 px-3 py-1 rounded-full text-xs">
                    {src}: <strong>{count.toLocaleString()}</strong>
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Numeric field stats */}
          {Object.keys(results.numeric_stats).length > 0 && (
            <div>
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-2">Numeric Stats</p>
              <div className="overflow-x-auto">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="text-gray-400 text-left border-b border-gray-100">
                      {['Field', 'Min', 'Max', 'Avg', 'Count'].map(h => (
                        <th key={h} className="pb-2 pr-4 font-medium">{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {Object.entries(results.numeric_stats).map(([field, s]) => (
                      <tr key={field} className="hover:bg-gray-50">
                        <td className="py-2 pr-4 font-medium text-gray-700">{field}</td>
                        <td className="py-2 pr-4 text-gray-500">{s.min.toFixed(2)}</td>
                        <td className="py-2 pr-4 text-gray-500">{s.max.toFixed(2)}</td>
                        <td className="py-2 pr-4 text-gray-500">{s.avg.toFixed(2)}</td>
                        <td className="py-2 text-gray-500">{s.count.toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </Card>
      )}

      {/* ── Validation errors ── */}
      <Card title={`Validation Errors (${errors.length})`}>
        <ErrorsTable errors={errors} />
      </Card>

      {/* ── Confirm modals ── */}
      {modal === 'delete' && (
        <ConfirmModal
          title="Delete pipeline?"
          message="This will permanently remove the job, its output file, and all recorded errors. This cannot be undone."
          confirmLabel="Delete"
          danger
          onConfirm={handleDelete}
          onCancel={() => setModal(null)}
        />
      )}
      {modal === 'cancel' && (
        <ConfirmModal
          title="Cancel pipeline?"
          message="The running job will be stopped immediately. Partially processed records will not be recoverable."
          confirmLabel="Yes, cancel it"
          onConfirm={handleCancel}
          onCancel={() => setModal(null)}
        />
      )}
    </div>
  )
}
