import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import Dashboard from './pages/Dashboard'
import PipelineList from './pages/PipelineList'
import PipelineDetail from './pages/PipelineDetail'
import CreatePipeline from './pages/CreatePipeline'

// All client-side routes defined in one place
export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard"      element={<Dashboard />} />
          <Route path="pipelines"      element={<PipelineList />} />
          <Route path="pipelines/new"  element={<CreatePipeline />} />
          <Route path="pipelines/:id"  element={<PipelineDetail />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
