import { Link } from 'react-router-dom'
import { usePipelines } from '../hooks/usePipelines'
import { Card } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { StatusPieChart } from '../components/charts/StatusPieChart'
import type { JobStatus } from '../types/pipeline'

// Single KPI tile used in the top summary row
function MetricTile({ label, value, bg }: { label: string; value: number; bg: string }) {
  return (
    <div className={`rounded-xl p-5 ${bg}`}>
      <p className="text-sm font-medium opacity-75">{label}</p>
      <p className="text-3xl font-bold mt-1">{value.toLocaleString()}</p>
    </div>
  )
}

// Dashboard overview with KPI counts, status pie chart, and recent job list
export default function Dashboard() {
  const { jobs, loading, error } = usePipelines(5000)

  if (loading) return <Spinner />
  if (error) return <p className="text-red-600 text-sm bg-red-50 rounded-lg px-4 py-3">{error}</p>

  // Count jobs per status
  const counts = jobs.reduce((acc, j) => {
    acc[j.status] = (acc[j.status] ?? 0) + 1
    return acc
  }, {} as Record<JobStatus, number>)

  const recent = jobs.slice(0, 6)

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-500 text-sm mt-1">Overview of all pipeline jobs</p>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <MetricTile label="Total Jobs"  value={jobs.length}              bg="bg-gray-800 text-white" />
        <MetricTile label="Running"     value={counts.running   ?? 0}    bg="bg-blue-600 text-white" />
        <MetricTile label="Completed"   value={counts.completed ?? 0}    bg="bg-green-600 text-white" />
        <MetricTile label="Failed"      value={(counts.failed ?? 0) + (counts.cancelled ?? 0)} bg="bg-red-500 text-white" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Status distribution */}
        <Card title="Job Status Distribution">
          <StatusPieChart
            pending={counts.pending   ?? 0}
            running={counts.running   ?? 0}
            completed={counts.completed ?? 0}
            failed={(counts.failed ?? 0) + (counts.cancelled ?? 0)}
          />
        </Card>

        {/* Recent jobs */}
        <Card title="Recent Jobs">
          {recent.length === 0 ? (
            <div className="py-8 text-center text-sm text-gray-400">
              No jobs yet.{' '}
              <Link to="/pipelines/new" className="text-blue-600 hover:underline">Create one →</Link>
            </div>
          ) : (
            <div className="space-y-1">
              {recent.map(job => (
                <Link
                  key={job.id}
                  to={`/pipelines/${job.id}`}
                  className="flex items-center justify-between rounded-lg px-3 py-2.5 hover:bg-gray-50 transition-colors group"
                >
                  <div className="min-w-0">
                    <p className="font-mono text-xs text-gray-500 truncate">{job.id}</p>
                    <p className="text-xs text-gray-400 mt-0.5">
                      {job.record_count.toLocaleString()} records · {new Date(job.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <Badge status={job.status} />
                </Link>
              ))}
              {jobs.length > 6 && (
                <Link to="/pipelines" className="block text-right text-xs text-blue-600 hover:underline mt-2 pr-3">
                  View all {jobs.length} jobs →
                </Link>
              )}
            </div>
          )}
        </Card>
      </div>
    </div>
  )
}
