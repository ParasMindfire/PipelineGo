import { useState } from 'react'
import { Link } from 'react-router-dom'
import { usePipelines } from '../hooks/usePipelines'
import { PipelineCard } from '../components/pipeline/PipelineCard'
import { Spinner } from '../components/ui/Spinner'
import { EmptyState } from '../components/ui/EmptyState'
import type { JobStatus } from '../types/pipeline'

// All possible filter values including the "show everything" option
const FILTERS: (JobStatus | 'all')[] = ['all', 'running', 'pending', 'completed', 'failed', 'cancelled']

// Filterable grid of all pipeline jobs with a "New Pipeline" shortcut
export default function PipelineList() {
  const { jobs, loading, error } = usePipelines(5000)
  const [filter, setFilter] = useState<JobStatus | 'all'>('all')

  const visible = filter === 'all' ? jobs : jobs.filter(j => j.status === filter)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Pipelines</h1>
          <p className="text-gray-500 text-sm mt-1">{jobs.length} total jobs</p>
        </div>
        <Link
          to="/pipelines/new"
          className="bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold px-5 py-2.5 rounded-xl transition-colors"
        >
          New Pipeline
        </Link>
      </div>

      {/* Status filter pills */}
      <div className="flex gap-2 flex-wrap">
        {FILTERS.map(f => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-3.5 py-1.5 rounded-full text-xs font-semibold capitalize transition-colors ${
              filter === f
                ? 'bg-blue-600 text-white'
                : 'bg-white border border-gray-200 text-gray-600 hover:border-blue-300 hover:text-blue-700'
            }`}
          >
            {f}
          </button>
        ))}
      </div>

      {loading && <Spinner />}
      {error && <p className="text-red-600 text-sm bg-red-50 rounded-lg px-4 py-3">{error}</p>}

      {!loading && !error && (
        visible.length === 0
          ? <EmptyState message={filter === 'all' ? 'No pipeline jobs yet' : `No ${filter} jobs`} />
          : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {visible.map(job => <PipelineCard key={job.id} job={job} />)}
            </div>
          )
      )}
    </div>
  )
}
