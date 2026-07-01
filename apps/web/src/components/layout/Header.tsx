import { NavLink } from 'react-router-dom'

const NAV = [
  { to: '/dashboard', label: 'Dashboard', end: true },
  { to: '/pipelines', label: 'Pipelines', end: true },
  { to: '/pipelines/new', label: 'New Pipeline', end: true },
]

// Sticky top navigation bar with active-link highlighting via React Router NavLink
export function Header() {
  return (
    <header className="bg-white border-b border-gray-100 sticky top-0 z-10">
      <div className="max-w-7xl mx-auto px-6 flex items-center h-16 gap-6">
        {/* Brand */}
        <NavLink to="/dashboard" className="flex items-center gap-2 font-bold text-gray-900 shrink-0">
          <span className="w-7 h-7 bg-blue-600 rounded-lg flex items-center justify-center text-white text-xs font-bold">P</span>
          <span>Pipeline Builder</span>
        </NavLink>

        {/* Nav links */}
        <nav className="flex gap-1">
          {NAV.map(({ to, label, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                `px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-blue-50 text-blue-700'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>
      </div>
    </header>
  )
}
