import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createPipeline } from '../../api/client'
import type { JobSpec, SourceConfig } from '../../types/pipeline'

// Form to configure and submit a new pipeline job, then redirect to its detail page
export function CreatePipelineForm() {
  const navigate = useNavigate()
  const [sources, setSources] = useState<SourceConfig[]>([{ type: 'csv', url: '' }])
  const [exportType, setExportType] = useState<'json' | 'csv'>('json')
  const [exportPath, setExportPath] = useState('')
  const [valWorkers, setValWorkers] = useState(5)
  const [transWorkers, setTransWorkers] = useState(5)
  const [bufferSize, setBufferSize] = useState(100)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Appends a blank source entry
  const addSource = () => setSources(prev => [...prev, { type: 'csv', url: '' }])

  // Removes the source at index i
  const removeSource = (i: number) => setSources(prev => prev.filter((_, idx) => idx !== i))

  // Updates one field in the source at index i
  const updateSource = (i: number, field: keyof SourceConfig, value: string) =>
    setSources(prev => prev.map((s, idx) => (idx === i ? { ...s, [field]: value } : s)))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)
    const spec: JobSpec = {
      sources,
      export: {
        type: exportType,
        path: exportPath || `data/output/job-${Date.now()}.json`,
      },
      concurrency: {
        validation_workers: valWorkers,
        transform_workers: transWorkers,
        ingestion_buffer_size: bufferSize,
      },
    }
    try {
      const job = await createPipeline(spec)
      navigate(`/pipelines/${job.id}`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create pipeline')
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-7">
      {/* Data sources */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <label className="text-sm font-semibold text-gray-700">Data Sources</label>
          <button type="button" onClick={addSource}
            className="text-xs text-blue-600 hover:text-blue-800 font-medium">
            + Add source
          </button>
        </div>
        <div className="space-y-3">
          {sources.map((src, i) => (
            <div key={i} className="flex gap-2 items-center">
              <select value={src.type}
                onChange={e => updateSource(i, 'type', e.target.value)}
                className="w-24 text-sm border border-gray-200 rounded-lg px-2 py-2 focus:outline-none focus:ring-2 focus:ring-blue-300 bg-white">
                <option value="csv">CSV</option>
                <option value="json">JSON</option>
                <option value="api">API</option>
              </select>
              <input
                type="text"
                placeholder="https://example.com/data.csv"
                value={src.url}
                onChange={e => updateSource(i, 'url', e.target.value)}
                required
                className="flex-1 text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-300"
              />
              {sources.length > 1 && (
                <button type="button" onClick={() => removeSource(i)}
                  className="text-gray-300 hover:text-red-400 text-xl leading-none shrink-0">×</button>
              )}
            </div>
          ))}
        </div>
      </section>

      {/* Export config */}
      <section>
        <label className="text-sm font-semibold text-gray-700 block mb-3">Export</label>
        <div className="flex gap-2">
          <select value={exportType}
            onChange={e => setExportType(e.target.value as 'json' | 'csv')}
            className="w-24 text-sm border border-gray-200 rounded-lg px-2 py-2 focus:outline-none focus:ring-2 focus:ring-blue-300 bg-white">
            <option value="json">JSON</option>
            <option value="csv">CSV</option>
          </select>
          <input
            type="text"
            placeholder="data/output/result.json  (auto-generated if blank)"
            value={exportPath}
            onChange={e => setExportPath(e.target.value)}
            className="flex-1 text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-300"
          />
        </div>
      </section>

      {/* Concurrency settings */}
      <section>
        <label className="text-sm font-semibold text-gray-700 block mb-3">Concurrency</label>
        <div className="grid grid-cols-3 gap-4">
          {([
            { label: 'Validation Workers', value: valWorkers, set: setValWorkers },
            { label: 'Transform Workers', value: transWorkers, set: setTransWorkers },
            { label: 'Ingestion Buffer', value: bufferSize, set: setBufferSize },
          ] as const).map(({ label, value, set }) => (
            <div key={label}>
              <label className="text-xs text-gray-400 block mb-1">{label}</label>
              <input
                type="number" min={1} max={100} value={value}
                onChange={e => set(Number(e.target.value))}
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-300"
              />
            </div>
          ))}
        </div>
      </section>

      {error && (
        <p className="text-sm text-red-600 bg-red-50 border border-red-100 rounded-lg px-4 py-3">{error}</p>
      )}

      <button
        type="submit" disabled={loading}
        className="w-full bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white font-semibold py-3 rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
        {loading ? 'Creating pipeline…' : 'Create Pipeline'}
      </button>
    </form>
  )
}
