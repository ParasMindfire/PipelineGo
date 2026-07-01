import { Link } from 'react-router-dom'
import { Badge } from '../ui/Badge'
import type { PipelineJob } from '../../types/pipeline'

interface PipelineCardProps { job: PipelineJob }

// Clickable summary card for a pipeline job shown in the list view
export function PipelineCard({ job }: PipelineCardProps) {
  return (
    <Link to={`/pipelines/${job.id}`} className="block">
      <div className="bg-white border border-gray-100 rounded-xl p-5 hover:shadow-md hover:border-blue-100 transition-all">
        <div className="flex items-center justify-between mb-3">
          <span className="font-mono text-xs text-gray-400 truncate max-w-[200px]">{job.id}</span>
          <Badge status={job.status} />
        </div>
        <div className="grid grid-cols-3 gap-4 text-sm">
          <div>
            <p className="text-gray-400 text-xs">Records</p>
            <p className="font-semibold text-gray-800">{job.record_count.toLocaleString()}</p>
          </div>
          <div>
            <p className="text-gray-400 text-xs">Errors</p>
            <p className={`font-semibold ${job.error_count > 0 ? 'text-red-500' : 'text-gray-800'}`}>
              {job.error_count.toLocaleString()}
            </p>
          </div>
          <div>
            <p className="text-gray-400 text-xs">Sources</p>
            <p className="font-semibold text-gray-800">{job.spec?.sources?.length ?? 0}</p>
          </div>
        </div>
        <p className="text-xs text-gray-400 mt-3">
          Created {new Date(job.created_at).toLocaleString()}
        </p>
      </div>
    </Link>
  )
}
