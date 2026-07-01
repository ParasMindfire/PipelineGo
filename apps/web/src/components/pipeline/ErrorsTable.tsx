import { EmptyState } from '../ui/EmptyState'
import type { ValidationError } from '../../types/pipeline'

interface ErrorsTableProps { errors: ValidationError[] }

// Tabular display of validation errors with field, message, and timestamp columns
export function ErrorsTable({ errors }: ErrorsTableProps) {
  if (errors.length === 0) return <EmptyState message="No validation errors recorded" />

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-xs text-gray-400 border-b border-gray-100">
            <th className="pb-3 pr-4 font-medium">Record ID</th>
            <th className="pb-3 pr-4 font-medium">Field</th>
            <th className="pb-3 pr-4 font-medium">Message</th>
            <th className="pb-3 font-medium">Time</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-50">
          {errors.map((e, i) => (
            <tr key={i} className="hover:bg-gray-50 transition-colors">
              <td className="py-2.5 pr-4 font-mono text-xs text-gray-400 truncate max-w-[140px]">{e.record_id}</td>
              <td className="py-2.5 pr-4 text-orange-600 font-medium">{e.field}</td>
              <td className="py-2.5 pr-4 text-red-600 text-xs">{e.message}</td>
              <td className="py-2.5 text-gray-400 text-xs">{new Date(e.at).toLocaleTimeString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
