import { Outlet } from 'react-router-dom'
import { Header } from './Header'

// Root layout shared by all pages — header + scrollable content area
export function Layout() {
  return (
    <div className="min-h-screen bg-gray-50">
      <Header />
      <main className="max-w-7xl mx-auto px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
