import { Card } from '../components/ui/Card'
import { CreatePipelineForm } from '../components/pipeline/CreatePipelineForm'

// Page wrapper for the new-pipeline creation form
export default function CreatePipeline() {
  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">New Pipeline</h1>
        <p className="text-gray-500 text-sm mt-1">Configure sources, export, and concurrency, then start a job</p>
      </div>
      <Card>
        <CreatePipelineForm />
      </Card>
    </div>
  )
}
