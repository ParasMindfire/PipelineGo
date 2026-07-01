import type { JobStatus } from '../../types/pipeline'

// Pipeline stages in the order they execute (all run concurrently via channels)
const STAGES = ['Ingestion', 'Validation', 'Transformation', 'Aggregation', 'Export'] as const

interface StageTrackerProps {
  status: JobStatus
  percentComplete: number
}

// Returns per-stage visual state based on the overall job status
function stageClass(status: JobStatus, i: number, pct: number): 'done' | 'active' | 'idle' | 'error' {
  if (status === 'completed') return 'done'
  if (status === 'failed' || status === 'cancelled') return 'error'
  if (status === 'pending') return 'idle'
  // running: infer approximate stage from percent if available, else animate all
  if (pct >= 0) {
    const stageThreshold = ((i + 1) / STAGES.length) * 100
    if (pct >= stageThreshold) return 'done'
  }
  return 'active'
}

// Visual flow tracker showing all five pipeline stages and their current state
export function StageTracker({ status, percentComplete }: StageTrackerProps) {
  const nodeClasses: Record<'done' | 'active' | 'idle' | 'error', string> = {
    done:   'bg-green-500 text-white',
    active: 'bg-blue-500 text-white ring-4 ring-blue-100 animate-pulse',
    idle:   'bg-gray-100 text-gray-400',
    error:  'bg-red-100 text-red-400',
  }
  const labelClasses: Record<'done' | 'active' | 'idle' | 'error', string> = {
    done:   'text-green-600 font-semibold',
    active: 'text-blue-600 font-semibold',
    idle:   'text-gray-400',
    error:  'text-red-400',
  }
  const lineClasses: Record<'done' | 'active' | 'idle' | 'error', string> = {
    done:   'bg-green-400',
    active: 'bg-blue-300',
    idle:   'bg-gray-200',
    error:  'bg-red-200',
  }

  return (
    <div className="flex items-start w-full py-2">
      {STAGES.map((stage, i) => {
        const state = stageClass(status, i, percentComplete)
        return (
          <div key={stage} className="flex-1 flex flex-col items-center">
            {/* Connector + node row */}
            <div className="flex items-center w-full">
              {i > 0 && (
                <div className={`flex-1 h-1 transition-colors ${lineClasses[state]}`} />
              )}
              <div className={`w-9 h-9 rounded-full flex items-center justify-center shrink-0 z-10 text-xs font-bold transition-all ${nodeClasses[state]}`}>
                {state === 'done' ? '✓' : state === 'error' ? '✕' : i + 1}
              </div>
              {i < STAGES.length - 1 && (
                <div className={`flex-1 h-1 transition-colors ${lineClasses[stageClass(status, i + 1, percentComplete)]}`} />
              )}
            </div>
            {/* Stage label */}
            <span className={`mt-2 text-xs text-center transition-colors ${labelClasses[state]}`}>
              {stage}
            </span>
          </div>
        )
      })}
    </div>
  )
}
